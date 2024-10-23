/*
Stwórsz bazę danych opartą o plik płaski przechowującą dane w postaci binarnej (https://gobyexample.com/reading-files).
Baza powinna umożliwiać wykonywanie następujących operacje: ADD, READ, UPDATE, DELETE na podstwie podango id rekordu.
W celu uzyskania lepszej wydajnoci, wprowadź indeksowanie pozycji rekordu w pliku oraz pamięć podręczną (wykorzystaj mapy).
Pomyśl o optymalnym sposobie usuwania rekordów i ponownym wykorzystaniem miejsca po usuniętym rekordzie.
*/

package concurrency

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"os"
	"slices"
	"strconv"
	"sync"
	"sync/atomic"
	"training.pl/examples/utils"
)

type IdGenerator interface {
	next() int64
}

type Sequence struct {
	counter int64
}

func (g *Sequence) next() int64 {
	counter := atomic.AddInt64(&g.counter, 1)
	return counter
}

type Record struct {
	Id     int64
	Offset int64
	Length int64
}

const stateFileExtension = ".state"

type Database struct {
	file        *os.File
	mutex       sync.RWMutex
	records     []Record
	idGenerator IdGenerator
}

func (db *Database) saveSnapshot() {
	buffer, _ := utils.ToBytes(db.records)
	_ = os.WriteFile(db.file.Name()+stateFileExtension, buffer, 0666)
}

func (db *Database) Close() error {
	return db.file.Close()
}

func (db *Database) Insert(input interface{}) (*Record, error) {
	buffer, err := utils.ToBytes(input)
	if err != nil {
		return nil, err
	}
	db.mutex.Lock()
	defer db.mutex.Unlock()
	offset, err := db.file.Seek(0, 2)
	if err != nil {
		return nil, err
	}
	length, err := db.file.WriteAt(buffer, offset)
	if err != nil {
		return nil, err
	}
	record := Record{db.idGenerator.next(), offset, int64(length)}
	db.records = append(db.records, record)
	db.saveSnapshot()
	return &record, nil
}

func (db *Database) GetById(id int64, output interface{}) error {
	db.mutex.RLock()
	defer db.mutex.RUnlock()
	idx := slices.IndexFunc(db.records, func(record Record) bool { return record.Id == id })
	if idx != -1 {
		record := db.records[idx]
		buffer := make([]byte, record.Length)
		_, err := db.file.ReadAt(buffer, record.Offset)
		if err != nil {
			return err
		}
		err = utils.FromBytes(buffer, output)
		if err != nil {
			return err
		}
		return nil
	}
	return errors.New("record not found")
}

func Db(filePath string) (*Database, error) {
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		return nil, err
	}

	buffer, err := os.ReadFile(filePath + stateFileExtension)
	var records []Record
	if err != nil {
		records = make([]Record, 0)
	} else {
		err = utils.FromBytes(buffer, &records)
		if err != nil {
			return nil, err
		}
	}
	db := &Database{file, sync.RWMutex{}, records, &Sequence{0}}
	return db, nil
}

type User struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	IsActive  bool   `json:"isActive"`
}

func UsersDatabase() {
	db, _ := Db("users.db")
	defer db.Close()
	router := gin.Default()

	router.Use(func(ctx *gin.Context) {
		ctx.Set("db", db)
	})

	router.POST("/users", createUser)
	router.GET("/users/:id", getUserById)
	router.Run(":8080")
}

type CreateUserResponse struct {
	Id int64
}

func getDb(c *gin.Context) *Database {
	db, _ := c.Get("db")
	return db.(*Database)
}

func createUser(c *gin.Context) {
	var user User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	record, _ := getDb(c).Insert(&user)
	c.Header("Location", fmt.Sprintf("/users/%v", record.Id))
	c.JSON(http.StatusCreated, &CreateUserResponse{record.Id})
}

func getUserById(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user id"})
		return
	}
	user := User{}
	err = getDb(c).GetById(id, &user)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	c.JSON(http.StatusOK, user)
}
