// Package db is a flat-file database that stores gob-encoded records in one
// binary file and keeps an in-memory index (id -> offset, length) that is
// persisted next to it. Every operation goes through a single goroutine, so
// Database is safe for concurrent use without further locking.
//
// DatabaseTest runs the operations once; DatabaseExercise exposes them as a
// small HTTP API.
package db

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"training.pl/go/examples/common"
)

const stateFileSuffix = ".state"

// Sentinel errors returned by the CRUD operations; test them with errors.Is.
var (
	ErrNotFound      = errors.New("db: record not found")
	ErrAlreadyExists = errors.New("db: record already exists")
)

// Typed constants instead of bare strings: a typo in an action name is now a
// compile error rather than a silently dropped command (the caller would then
// wait on <-reply forever).
type action string

const (
	actionInsert action = "insert"
	actionFind   action = "find"
	actionUpdate action = "update"
	actionDelete action = "delete"
)

type command struct {
	action action
	id     int64
	input  any
	output any
	reply  chan result
}

type result struct {
	record *Record
	err    error
}

// Record describes where one value lives in the data file.
type Record struct {
	ID     int64
	Offset int64
	Length int64
}

// Database is a handle to an open data file. Create it with Open and release
// it with Close.
type Database struct {
	file        *os.File
	commands    chan command
	state       *databaseState
	idGenerator IDGenerator
	// done is closed by run() once the command channel has been drained - that is
	// how Close() knows nobody is still writing to the file or to state.
	done chan struct{}
	// closeOnce keeps Close idempotent: DatabaseExercise may call it both from
	// the shutdown path and from a defer, and close() on an already closed
	// channel panics.
	closeOnce sync.Once
	closeErr  error
}

// databaseState is the index persisted in the .state file. Its fields are
// exported because encoding/gob ignores unexported ones.
type databaseState struct {
	Records map[int64]*Record
	LastID  int64
}

// NOTE ON A KNOWN LIMITATION: space is never reclaimed. delete removes only the
// index entry, and update always appends at the end of the file, so the file
// grows without bound. Exercise 8 in notes.md asks for reuse of freed space -
// implementing it means keeping a free list of (offset, length) holes and
// picking one that fits, plus compaction when fragmentation gets bad.

// Open opens (or creates) the data file at path and loads the index stored in
// path + ".state". The id generator is seeded from the index so that numbering
// continues where the previous run stopped.
func Open(path string, gen IDGenerator) (*Database, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, fmt.Errorf("db: opening %s: %w", path, err)
	}

	state := &databaseState{Records: make(map[int64]*Record)}
	data, err := os.ReadFile(path + stateFileSuffix)
	switch {
	case errors.Is(err, os.ErrNotExist):
		// A fresh database: no index yet.
	case err != nil:
		file.Close()
		return nil, fmt.Errorf("db: reading state: %w", err)
	default:
		if err := common.FromBytes(data, state); err != nil {
			file.Close()
			return nil, fmt.Errorf("db: decoding state: %w", err)
		}
		if state.Records == nil {
			state.Records = make(map[int64]*Record)
		}
	}
	// Seed the generator from the stored state. Without this LastID was a dead
	// field - written to disk but never read back - and Sequence restarted from
	// zero on every run, so after a restart ids collided with existing records.
	gen.seed(state.LastID)

	d := &Database{
		file:        file,
		commands:    make(chan command, 100),
		state:       state,
		idGenerator: gen,
		done:        make(chan struct{}),
	}
	go d.run()
	return d, nil
}

// Close drains pending commands, persists the index and closes the data file.
// It is safe to call more than once; later calls return the first result.
func (d *Database) Close() error {
	d.closeOnce.Do(func() {
		// close(d.commands) on its own does NOT wait for run() - without <-d.done
		// the run goroutine could still be writing to the file and mutating state
		// while Close closes the file and serialises that same state (a data race
		// plus a write to a closed descriptor). Buffered commands would be dropped
		// too.
		close(d.commands)
		<-d.done

		// Order matters: save the state first (it uses d.file.Name()), then close
		// the file.
		d.closeErr = errors.Join(d.saveState(), d.file.Close())
	})
	return d.closeErr
}

func (d *Database) saveState() error {
	data, err := common.ToBytes(d.state)
	if err != nil {
		return err
	}
	return os.WriteFile(d.file.Name()+stateFileSuffix, data, 0o644)
}

// run is the only goroutine that touches the file and the index.
func (d *Database) run() {
	defer close(d.done)
	for cmd := range d.commands {
		var record *Record
		var err error
		switch cmd.action {
		case actionInsert:
			record, err = d.create(cmd.input)
		case actionFind:
			record, err = d.read(cmd.id, cmd.output)
		case actionUpdate:
			record, err = d.update(cmd.id, cmd.input)
		case actionDelete:
			err = d.delete(cmd.id)
		default:
			// Without this branch an unknown action produced no reply and the
			// caller blocked on <-reply forever.
			err = fmt.Errorf("db: unknown action %q", cmd.action)
		}
		cmd.reply <- result{record, err}
	}
}

// appendBytes writes data at the end of the file and returns its offset.
func (d *Database) appendBytes(data []byte) (offset int64, err error) {
	info, err := d.file.Stat()
	if err != nil {
		return 0, err
	}
	offset = info.Size()
	if _, err := d.file.WriteAt(data, offset); err != nil {
		return 0, err
	}
	return offset, nil
}

func (d *Database) create(object any) (*Record, error) {
	data, err := common.ToBytes(object)
	if err != nil {
		return nil, err
	}
	// Peek before consuming the id: a failed insert must not burn a number.
	id := d.idGenerator.peek()
	if _, exists := d.state.Records[id]; exists {
		return nil, fmt.Errorf("%w: id %d", ErrAlreadyExists, id)
	}
	offset, err := d.appendBytes(data)
	if err != nil {
		return nil, err
	}
	id = d.idGenerator.next()
	record := &Record{ID: id, Offset: offset, Length: int64(len(data))}
	d.state.Records[id] = record
	d.state.LastID = id
	if err := d.saveState(); err != nil {
		// Keep the index consistent with what is on disk in the .state file.
		delete(d.state.Records, id)
		return nil, err
	}
	return record, nil
}

func (d *Database) read(id int64, object any) (*Record, error) {
	record, exists := d.state.Records[id]
	if !exists {
		return nil, fmt.Errorf("%w: id %d", ErrNotFound, id)
	}
	data := make([]byte, record.Length)
	if _, err := d.file.ReadAt(data, record.Offset); err != nil {
		return nil, err
	}
	if err := common.FromBytes(data, object); err != nil {
		return nil, err
	}
	return record, nil
}

func (d *Database) delete(id int64) error {
	record, exists := d.state.Records[id]
	if !exists {
		return fmt.Errorf("%w: id %d", ErrNotFound, id)
	}
	delete(d.state.Records, id)
	if err := d.saveState(); err != nil {
		d.state.Records[id] = record
		return err
	}
	return nil
}

func (d *Database) update(id int64, object any) (*Record, error) {
	old, exists := d.state.Records[id]
	if !exists {
		return nil, fmt.Errorf("%w: id %d", ErrNotFound, id)
	}
	data, err := common.ToBytes(object)
	if err != nil {
		return nil, err
	}
	offset, err := d.appendBytes(data)
	if err != nil {
		return nil, err
	}
	// Build the new record first and swap it into the index only once the
	// on-disk index has been written: a failed saveState leaves the old record
	// (and the old bytes, which are still in the file) in place.
	record := &Record{ID: id, Offset: offset, Length: int64(len(data))}
	d.state.Records[id] = record
	if err := d.saveState(); err != nil {
		d.state.Records[id] = old
		return nil, err
	}
	return record, nil
}

func (d *Database) execute(cmd command) (*Record, error) {
	cmd.reply = make(chan result, 1)
	d.commands <- cmd
	r := <-cmd.reply
	return r.record, r.err
}

// Create stores input under a fresh id and returns its record.
func (d *Database) Create(input any) (*Record, error) {
	return d.execute(command{action: actionInsert, input: input})
}

// Read decodes the value stored under id into output, which must be a pointer
// to a zero value: gob omits zero-valued fields on encoding and leaves them
// untouched on decoding, so reusing a populated struct keeps stale fields. It
// returns ErrNotFound for an unknown id.
func (d *Database) Read(id int64, output any) (*Record, error) {
	return d.execute(command{action: actionFind, id: id, output: output})
}

// Delete removes the record with the given id, returning ErrNotFound when
// there is none.
func (d *Database) Delete(id int64) error {
	_, err := d.execute(command{action: actionDelete, id: id})
	return err
}

// Update replaces the value stored under id, returning ErrNotFound when there
// is none.
func (d *Database) Update(id int64, input any) (*Record, error) {
	return d.execute(command{action: actionUpdate, id: id, input: input})
}

// User is the sample record type used by the demos.
type User struct {
	FirstName string
	LastName  string
	Age       int16
	IsActive  bool
}

// DatabaseTest runs one create/update/read/delete cycle against users.db.
func DatabaseTest() {
	db, err := Open("users.db", &Sequence{})
	if err != nil {
		log.Println(err)
		return
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Println("close failed:", err)
		}
	}()

	user := User{"Jan", "Kowalski", 25, true}
	record, err := db.Create(&user)
	// Without checking the error the next line would dereference a nil Record.
	if err != nil {
		log.Println("Create failed:", err)
		return
	}
	fmt.Println(record)
	id := record.ID

	user.IsActive = false
	record, err = db.Update(id, &user)
	fmt.Println(record, err)

	loadedUser := &User{}
	record, err = db.Read(id, loadedUser)
	fmt.Println(record, err, loadedUser)

	fmt.Println(db.Delete(id))
}

// DatabaseExercise serves users.db over HTTP on :8080 until SIGINT/SIGTERM,
// then shuts the server down gracefully and persists the index.
func DatabaseExercise() {
	db, err := Open("users.db", &Sequence{})
	if err != nil {
		log.Println(err)
		return
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Println("close failed:", err)
		}
	}()

	h := &userHandler{db: db}
	router := gin.Default()
	router.POST("/users", h.createUser)
	router.GET("/users/:id", h.getUser)
	router.PUT("/users/:id", h.updateUser)
	router.DELETE("/users/:id", h.deleteUser)

	server := &http.Server{Addr: ":8080", Handler: router}

	// signal.NotifyContext cancels ctx on the first signal; the second signal
	// (after stop() restores the default handler) kills the process outright.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Println("server stopped:", err)
			stop()
		}
	}()

	<-ctx.Done()
	stop()

	// Shutdown stops accepting new connections and waits for in-flight
	// requests, so no database command is lost; the deferred Close then
	// persists the index. os.Exit here would skip all of that.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Println("shutdown failed:", err)
	}
}

// userHandler groups the HTTP handlers around the database they use.
type userHandler struct {
	db *Database
}

// CreateUserResponse is the body of a successful POST /users.
type CreateUserResponse struct {
	ID int64
}

// respondError maps a database error to an HTTP status.
func respondError(c *gin.Context, err error) {
	if errors.Is(err, ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
}

func (h *userHandler) createUser(c *gin.Context) {
	var user User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	record, err := h.db.Create(&user)
	if err != nil {
		respondError(c, err)
		return
	}
	c.Header("Location", fmt.Sprintf("/users/%d", record.ID))
	c.JSON(http.StatusCreated, CreateUserResponse{ID: record.ID})
}

func (h *userHandler) getUser(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var user User
	if _, err := h.db.Read(id, &user); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, user)
}

func (h *userHandler) updateUser(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var user User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := h.db.Update(id, &user); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *userHandler) deleteUser(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := h.db.Delete(id); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
