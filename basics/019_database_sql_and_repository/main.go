// Command 019_database_sql_and_repository is module 019 of the Go course: database/sql and the
// repository pattern.
//
// database/sql is not an ORM and not a query builder: it is a thin, driver-agnostic layer with a
// connection pool, prepared statements, transactions and row scanning. You write the SQL.
//
// It needs the PostgreSQL from the repository's docker-compose.yml. The DSN environment variable
// overrides the default connection string. If the database is unreachable the module says so and
// still runs everything that needs no database, including the in-memory repository.
//
//	docker-compose up -d
//	go run ./basics/019_database_sql_and_repository
//	DSN='postgres://admin:admin@localhost:5433/users?sslmode=disable' go run ./basics/019_database_sql_and_repository
package main

import (
	"cmp"
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
	"sync"
	"time"

	_ "github.com/lib/pq" // blank import: its init() calls sql.Register("postgres", ...)
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	db := connectOrExplain() // nil when PostgreSQL is unreachable: live parts are skipped
	if db != nil {
		defer db.Close() // the pool is closed at shutdown, after everything that uses it
	}

	poolBasics(db)                  // sql.Open does not connect, PingContext does, Stats
	poolTuning(db)                  // the four pool settings and their bad defaults
	queryingAndScanning(ctx, db)    // Exec vs QueryRow vs Query, the rows contract
	parametersAndInjection(ctx, db) // placeholders, = any($1), allow-listed identifiers
	nullsAndErrors(ctx, db)         // sql.Null[T], *string, COALESCE, error translation
	transactions(ctx, db)           // BeginTx + defer Rollback, read-only transactions
	repositoryPattern(ctx, db)      // one interface, a PostgreSQL and an in-memory version
}

const (
	defaultDSN  = "postgres://admin:admin@localhost:5432/users?sslmode=disable"
	pingTimeout = 2 * time.Second
)

// dsn returns the connection string: a compiled-in default the DSN environment variable
// overrides - module 017's configuration precedence in miniature (handy when 5432 is taken).
func dsn() string {
	if v, ok := os.LookupEnv("DSN"); ok && v != "" {
		return v
	}
	return defaultDSN
}

// connectOrExplain opens and verifies the pool, or explains why it could not and returns nil.
func connectOrExplain() *sql.DB {
	db, err := connect()
	if err != nil {
		fmt.Println("NOTE: PostgreSQL is not reachable, so the live parts are skipped.")
		fmt.Println("      " + shorten(err))
		fmt.Println("      Start it with:  docker-compose up -d")
		fmt.Println("      DSN: " + dsn() + "  (set the DSN env var to point elsewhere)")
		fmt.Println()
		return nil
	}
	fmt.Println("PostgreSQL is reachable; running every topic against it.")
	fmt.Println()
	return db
}

func connect() (*sql.DB, error) {
	db, err := sql.Open("postgres", dsn())
	if err != nil {
		return nil, fmt.Errorf("open: %w", err) // only a bad DSN or an unknown driver gets here
	}
	tune(db)
	ctx, cancel := context.WithTimeout(context.Background(), pingTimeout)
	defer cancel()
	if err := db.PingContext(ctx); err != nil { // fail fast at boot, not on the first request
		db.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}
	return db, nil
}

// poolBasics shows that *sql.DB is a pool, not a connection.
func poolBasics(db *sql.DB) {
	fmt.Println("--- sql.DB is a pool, not a connection ---")

	// sql.Open only looks up the driver and returns a POOL that connects lazily; lib/pq does not
	// even parse the DSN yet. So Open succeeding tells you almost nothing...
	bogus, err := sql.Open("postgres", "postgres://nobody@127.0.0.1:1/none?sslmode=disable")
	fmt.Printf("sql.Open on a host that is not listening: err=%v  <- it succeeded\n", err)
	// ...and PingContext is what actually connects. Call it once at startup, with a timeout.
	pingCtx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	fmt.Println("PingContext is where it fails:", shorten(bogus.PingContext(pingCtx)))
	bogus.Close()

	// The other Open-time failure: no driver registered under that name.
	_, err = sql.Open("mysql", "whatever")
	fmt.Println("sql.Open with an unregistered driver:", shorten(err))

	// A *sql.DB is safe for concurrent use and long-lived: one per database for the whole
	// program. Opening one per request is the classic bug - you lose the pool. db.Close() closes
	// the POOL: it belongs in shutdown, after the HTTP server has drained (module 018).
	// Use only the Context variants (QueryContext, ExecContext, BeginTx...): the others use
	// context.Background(), so a query keeps running after its client has gone.
	if db == nil {
		return
	}
	s := db.Stats() // the pool's live state; a climbing WaitCount means MaxOpenConns is too low
	fmt.Printf("pool stats: open=%d inUse=%d idle=%d waitCount=%d\n",
		s.OpenConnections, s.InUse, s.Idle, s.WaitCount)
}

// poolTuning shows the four pool settings - their defaults are wrong for a server.
func poolTuning(db *sql.DB) {
	fmt.Println("\n--- tuning the pool ---")

	// tune (below) sets 25 / 25 / 5m / 2m. Run a few queries concurrently so the pool works.
	if db == nil {
		fmt.Println("(live part skipped: no database)")
		return
	}
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			var n int
			_ = db.QueryRowContext(ctx, "select 1").Scan(&n)
		})
	}
	wg.Wait()
	s := db.Stats()
	fmt.Printf("after 8 concurrent queries: maxOpen=%d open=%d idle=%d waitCount=%d waitDuration=%v\n",
		s.MaxOpenConnections, s.OpenConnections, s.Idle, s.WaitCount, s.WaitDuration)

	// More connections is not faster: past what the database runs in parallel they only add
	// contention. Raise MaxOpenConns only when WaitCount and WaitDuration say you are queueing.
}

func tune(db *sql.DB) {
	// Default UNLIMITED: a spike opens more than PostgreSQL's max_connections (100) and they all
	// fail. Stay well under the server's limit, divided by the number of instances.
	db.SetMaxOpenConns(25)
	// Default 2: a busy server closes and reopens constantly. Idle connections are cheap, a
	// TCP+TLS handshake is not - set it equal to MaxOpenConns.
	db.SetMaxIdleConns(25)
	// Default forever: a finite lifetime lets load balancers rebalance and failovers or rolling
	// database upgrades complete. Minutes, not hours.
	db.SetConnMaxLifetime(5 * time.Minute)
	// Default forever (Go 1.15+): reaps connections left idle after a burst.
	db.SetConnMaxIdleTime(2 * time.Minute)
}

type user struct {
	ID    int64
	Name  string
	Email string
}

// queryingAndScanning shows the three query methods - picking the wrong one is the commonest bug.
func queryingAndScanning(ctx context.Context, db *sql.DB) {
	fmt.Println("\n--- querying and scanning ---")
	if db == nil {
		fmt.Println("(skipped: no database)")
		return
	}
	if err := resetSchema(ctx, db); err != nil {
		fmt.Println("setup failed:", err)
		return
	}

	// ExecContext: INSERT/UPDATE/DELETE/DDL. Returns a sql.Result with RowsAffected.
	res, err := db.ExecContext(ctx,
		`insert into m019_users (name, email) values ($1,$2), ($3,$4), ($5,$6)`,
		"Ada", "ada@example.com", "Alan", "alan@example.com", "Grace", "grace@example.com")
	if err != nil {
		fmt.Println("insert failed:", err)
		return
	}
	affected, _ := res.RowsAffected()
	fmt.Println("ExecContext insert: rowsAffected =", affected)

	// PostgreSQL has no LastInsertId - use INSERT ... RETURNING id with QueryRow.
	var newID int64
	_ = db.QueryRowContext(ctx, `insert into m019_users (name, email) values ($1,$2) returning id`,
		"Barbara", "barbara@example.com").Scan(&newID)
	fmt.Println("INSERT ... RETURNING id ->", newID)
	if _, err := res.LastInsertId(); err != nil {
		fmt.Println("LastInsertId on PostgreSQL:", shorten(err))
	}

	// QueryRowContext: exactly one row. Row closes itself; Scan returns sql.ErrNoRows if none.
	const byName = `select id, name, email from m019_users where name = $1`
	var u user
	err = db.QueryRowContext(ctx, byName, "Ada").Scan(&u.ID, &u.Name, &u.Email)
	fmt.Printf("QueryRowContext found: %+v (err=%v)\n", u, err)
	err = db.QueryRowContext(ctx, byName, "Nobody").Scan(&u.ID, &u.Name, &u.Email)
	fmt.Println("missing row: errors.Is(err, sql.ErrNoRows) =", errors.Is(err, sql.ErrNoRows))

	// QueryContext: many rows. *sql.Rows HOLDS A CONNECTION until closed - the one that bites.
	rows, err := db.QueryContext(ctx, `select id, name, email from m019_users order by id`)
	if err != nil {
		fmt.Println("query failed:", err)
		return
	}
	// 1. ALWAYS close, even on the happy path. There is no finalizer: a forgotten Close holds
	// the connection until the context ends, and a handler doing it per request drains the pool.
	defer rows.Close()
	cols, _ := rows.Columns() // the column names, when the shape is not known ahead of time
	fmt.Println("rows.Columns():", cols)
	var users []user
	for rows.Next() { // 2. Next advances and reports whether there is a row
		var u user
		if err := rows.Scan(&u.ID, &u.Name, &u.Email); err != nil { // 3. scan each row
			fmt.Println("scan failed:", err)
			return
		}
		users = append(users, u)
	}
	// 4. Next returning false may mean an ERROR, not "done": without rows.Err() a mid-iteration
	// network failure looks like a successful, silently truncated result.
	if err := rows.Err(); err != nil {
		fmt.Println("iteration failed:", err)
		return
	}
	fmt.Printf("QueryContext returned %d rows:\n", len(users))
	for _, u := range users {
		fmt.Printf("  %d %-8s %s\n", u.ID, u.Name, u.Email)
	}

	// Never query INSIDE a rows.Next() loop on the same pool: the outer Rows holds a connection,
	// so with MaxOpenConns(1) you deadlock. Collect the ids, then query once with = any($1).
}

// parametersAndInjection shows why values travel as placeholders and what they cannot do.
func parametersAndInjection(ctx context.Context, db *sql.DB) {
	fmt.Println("\n--- parameters and SQL injection ---")

	// NEVER concatenate user input into SQL - not with +, not with Sprintf, not "just an int":
	//	q := "select * from users where name = '" + name + "'"
	// name = `x' or '1'='1` returns every row; `x'; drop table users; --` does what it says.
	// A placeholder sends the statement and the values SEPARATELY: the value is never parsed
	// as SQL. It is not escaping, it is a different transport. The syntax is driver-specific:
	// $1 (PostgreSQL: lib/pq, pgx), ? (MySQL, SQLite), :name (Oracle).
	if db == nil {
		fmt.Println("(live part skipped: no database)")
	} else {
		payload := `x' or '1'='1`
		var count, total int
		err := db.QueryRowContext(ctx,
			`select count(*) from m019_users where name = $1`, payload).Scan(&count)
		_ = db.QueryRowContext(ctx, `select count(*) from m019_users`).Scan(&total)
		fmt.Printf("parameterised %q -> %d of %d rows (err=%v)\n", payload, count, total, err)

		// `where id in ($1)` binds ONE value. PostgreSQL: = any($1) with an array (pq.Array);
		// elsewhere build the right number of placeholders - placeholders, never values.
		names, err := namesByIDs(ctx, db, int64Array{1, 2})
		fmt.Printf("`where id = any($1)` binds a whole list: %v (err=%v)\n", names, err)
	}

	// Placeholders bind VALUES, never structure: `select * from $1` is invalid, `order by $1`
	// sorts by a constant. Vary identifiers and keywords only through an allow-list.
	for _, candidate := range []string{"name", "email", "id; drop table m019_users"} {
		col, err := safeColumn(candidate)
		fmt.Printf("allow-list %-28q -> %q (err=%v)\n", candidate, col, err)
	}
}

func namesByIDs(ctx context.Context, db *sql.DB, ids int64Array) ([]string, error) {
	rows, err := db.QueryContext(ctx, `select name from m019_users where id = any($1) order by id`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var names []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			return nil, err
		}
		names = append(names, n)
	}
	return names, rows.Err()
}

// safeColumn validates an identifier against a fixed set. Never interpolate anything else,
// even from an "internal" caller.
func safeColumn(name string) (string, error) {
	if slices.Contains([]string{"id", "name", "email"}, name) {
		return name, nil
	}
	return "", fmt.Errorf("column %q is not sortable", name)
}

// int64Array implements driver.Valuer so a slice can be bound to `= any($1)`
// (lib/pq's pq.Array does this; hand-rolling it shows what it does).
type int64Array []int64

// driver.Value is a DEFINED type (type Value any), not an alias, so the method must name it
// exactly - Value() (any, error) would not satisfy the interface. This line makes the compiler check.
var _ driver.Valuer = int64Array(nil)

func (a int64Array) Value() (driver.Value, error) {
	parts := make([]string, len(a))
	for i, v := range a {
		parts[i] = fmt.Sprint(v)
	}
	return "{" + strings.Join(parts, ",") + "}", nil
}

// nullsAndErrors shows the ways to scan SQL NULL and how to translate database errors.
func nullsAndErrors(ctx context.Context, db *sql.DB) {
	fmt.Println("\n--- NULL, and mapping errors ---")

	// NULL has no Go equivalent. Prefer NOT NULL DEFAULT ... when you own the schema: a nullable
	// column is a decision every reader has to handle forever.
	if db == nil {
		fmt.Println("(skipped: no database)")
		return
	}
	if _, err := db.ExecContext(ctx, `insert into m019_users (name, email, nickname) values ($1,$2,$3)`,
		"Nullable", "null@example.com", nil); err != nil {
		fmt.Println("insert failed:", err)
		return
	}
	const q = `select nickname from m019_users where name = $1`

	var plain string // 1. a plain string fails
	err := db.QueryRowContext(ctx, q, "Nullable").Scan(&plain)
	fmt.Println("into a string:         ", shorten(err))

	var ns sql.NullString // 2. NullString/NullInt64/NullTime...: explicit, but every read is if v.Valid
	_ = db.QueryRowContext(ctx, q, "Nullable").Scan(&ns)
	fmt.Printf("into sql.NullString:    value=%q valid=%t\n", ns.String, ns.Valid)

	var generic sql.Null[string] // 3. Go 1.22: one generic type replaces the whole family
	_ = db.QueryRowContext(ctx, q, "Nullable").Scan(&generic)
	fmt.Printf("into sql.Null[string]:  value=%q valid=%t\n", generic.V, generic.Valid)

	var ptr *string // 4. a pointer: NULL scans as nil - natural to use, matches encoding/json
	_ = db.QueryRowContext(ctx, q, "Nullable").Scan(&ptr)
	fmt.Println("into *string:           nil =", ptr == nil)

	var coalesced string // 5. COALESCE: let the query decide, when the zero value IS the default
	_ = db.QueryRowContext(ctx, `select coalesce(nickname, '(none)') from m019_users where name = $1`,
		"Nullable").Scan(&coalesced)
	fmt.Printf("with COALESCE:          %q\n", coalesced)

	// Errors worth naming: sql.ErrNoRows (see queryingAndScanning), sql.ErrTxDone (a tx used
	// after Commit/Rollback), sql.ErrConnDone (a *sql.Conn used after it went back to the pool).
	if tx, err := db.BeginTx(ctx, nil); err == nil {
		_ = tx.Rollback()
		_, err = tx.ExecContext(ctx, `select 1`)
		fmt.Println("finished transaction: errors.Is(err, sql.ErrTxDone) =", errors.Is(err, sql.ErrTxDone))
	}

	// Driver errors (a unique violation is SQLSTATE 23505) are translated into DOMAIN errors at
	// the repository boundary, so database/sql never leaks into business logic.
	_, err = db.ExecContext(ctx, `insert into m019_users (name, email) values ($1,$2)`,
		"Dup", "ada@example.com")
	fmt.Println("duplicate email ->", translateError(err))
}

// Domain errors (module 009). Callers match them with errors.Is and never see a driver error.
var (
	errNotFound  = errors.New("not found")
	errDuplicate = errors.New("already exists")
)

// translateError maps driver errors to domain errors. Driver knowledge stops here.
func translateError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) { // a sentinel: match it with errors.Is
		return errNotFound
	}
	// Matching the text keeps this module free of a pq import; real code uses errors.As with
	// *pq.Error and checks Code == "23505".
	if strings.Contains(err.Error(), "23505") || strings.Contains(err.Error(), "duplicate key") {
		return errDuplicate
	}
	return err
}

// transactions shows the BeginTx / defer Rollback / Commit idiom.
func transactions(ctx context.Context, db *sql.DB) {
	fmt.Println("\n--- transactions ---")
	if db == nil {
		fmt.Println("(skipped: no database)")
		return
	}

	// A transaction lives on *sql.Tx, never on the pool - the pool has no Commit at all:
	//	db.Commit() // ERROR: db.Commit undefined (type *sql.DB has no field or method Commit)

	before := count(ctx, db)
	if err := insertPair(ctx, db, "Committed A", "Committed B", true); err != nil {
		fmt.Println("commit path failed:", err)
	}
	afterCommit := count(ctx, db)
	err := insertPair(ctx, db, "Rolled A", "Rolled B", false)
	afterRollback := count(ctx, db)
	fmt.Printf("rows before=%d afterCommit=%d afterRollback=%d\n", before, afterCommit, afterRollback)
	fmt.Println("the rollback path returned:", err) // both of its inserts vanished: atomicity

	// TxOptions set isolation and read-only. PostgreSQL's default is READ COMMITTED. At
	// SERIALIZABLE it may abort with a serialization failure (40001) that is expected and
	// RETRYABLE - code using it needs a retry loop.
	ro, err := db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelReadCommitted})
	if err == nil {
		var n int
		_ = ro.QueryRowContext(ctx, `select count(*) from m019_users`).Scan(&n)
		_, writeErr := ro.ExecContext(ctx, `insert into m019_users (name, email) values ('x','x@x')`)
		fmt.Printf("a read-only transaction reads (%d rows) but cannot write: %v\n", n, shorten(writeErr))
		_ = ro.Rollback()
	}
}

// insertPair inserts two rows atomically, failing deliberately when commit is false.
func insertPair(ctx context.Context, db *sql.DB, a, b string, commit bool) error {
	// BeginTx takes the context: cancelling it rolls the transaction back.
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	// The whole safety net in one line, right after BeginTx: every early return and every panic
	// releases the transaction. After a successful Commit it is a no-op (returns sql.ErrTxDone).
	defer tx.Rollback()

	// A *sql.Tx holds ONE connection for its whole life: keep it short, never hold it across an
	// HTTP call, and never use the parent db inside it - that is a different connection.
	for _, name := range []string{a, b} {
		if _, err := tx.ExecContext(ctx, `insert into m019_users (name, email) values ($1,$2)`,
			name, name+"@example.com"); err != nil {
			return fmt.Errorf("insert %s: %w", name, err)
		}
	}
	if !commit {
		return errors.New("deliberate failure before commit: both inserts are discarded")
	}
	return tx.Commit()
}

func count(ctx context.Context, db *sql.DB) int {
	var n int
	_ = db.QueryRowContext(ctx, `select count(*) from m019_users`).Scan(&n)
	return n
}

// repositoryPattern runs one contract against two implementations of the same interface.
func repositoryPattern(ctx context.Context, db *sql.DB) {
	fmt.Println("\n--- the repository pattern ---")

	// The in-memory half always runs, database or not - which is the entire point: unit tests
	// need no Docker. (A testcontainer is the right tool for testing the SQL itself.)
	exerciseRepository(ctx, newMemoryRepo(), "in-memory (no database needed)")
	if db == nil {
		fmt.Println("PostgreSQL: skipped, no database")
	} else if err := resetSchema(ctx, db); err == nil {
		exerciseRepository(ctx, &postgresRepo{db: db}, "PostgreSQL")
	}
	// Both return the SAME domain errors, so the caller never learns which one it talks to.
	// Module 020 implements this interface a third time, with GORM.
}

// userRepository is owned by the code that USES it (module 008: define interfaces where they
// are consumed) and speaks the domain's language: no SQL, no driver types.
// Pitfalls: a generic Repository[T] with Save/Find for every entity (aggregates need different
// operations), methods taking a WhereClause (a query builder in disguise), and HIDING
// transactions - expose WithTx(ctx, func(userRepository) error) or pass a tx explicitly.
// A method like FindByAnyFieldWithSorting is the interface telling you something.
type userRepository interface {
	Create(ctx context.Context, u *user) error
	ByID(ctx context.Context, id int64) (*user, error)
	List(ctx context.Context) ([]user, error)
	Delete(ctx context.Context, id int64) error
}

// Compile-time proof that both satisfy the interface (module 008).
var (
	_ userRepository = (*postgresRepo)(nil)
	_ userRepository = (*memoryRepo)(nil)
)

// exerciseRepository is the CONTRACT, run identically against any implementation.
func exerciseRepository(ctx context.Context, repo userRepository, label string) {
	fmt.Println(label + ":")
	u := &user{Name: "Repo Ada", Email: "repo-ada@example.com"}
	if err := repo.Create(ctx, u); err != nil {
		fmt.Println("  Create:", err)
		return
	}
	fmt.Printf("  Create           -> id=%d\n", u.ID)
	dup := &user{Name: "Repo Dup", Email: "repo-ada@example.com"}
	fmt.Println("  Create duplicate -> errors.Is(err, errDuplicate) =",
		errors.Is(repo.Create(ctx, dup), errDuplicate))
	found, err := repo.ByID(ctx, u.ID)
	fmt.Printf("  %-16s -> %+v (err=%v)\n", fmt.Sprintf("ByID(%d)", u.ID), deref(found), err)
	_, err = repo.ByID(ctx, 999999)
	fmt.Println("  ByID(999999)     -> errors.Is(err, errNotFound) =", errors.Is(err, errNotFound),
		"<- NOT sql.ErrNoRows")
	list, err := repo.List(ctx)
	fmt.Printf("  List             -> %d user(s) (err=%v)\n", len(list), err)
	fmt.Printf("  %-16s -> %v\n", fmt.Sprintf("Delete(%d)", u.ID), repo.Delete(ctx, u.ID))
	fmt.Println("  Delete again     -> errors.Is(err, errNotFound) =",
		errors.Is(repo.Delete(ctx, u.ID), errNotFound))
}

func deref(u *user) user {
	if u == nil {
		return user{}
	}
	return *u
}

// postgresRepo is the database/sql implementation.
type postgresRepo struct{ db *sql.DB }

func (r *postgresRepo) Create(ctx context.Context, u *user) error {
	err := r.db.QueryRowContext(ctx, `insert into m019_users (name, email) values ($1,$2) returning id`,
		u.Name, u.Email).Scan(&u.ID)
	return translateError(err) // driver knowledge stops here
}

func (r *postgresRepo) ByID(ctx context.Context, id int64) (*user, error) {
	var u user
	err := r.db.QueryRowContext(ctx, `select id, name, email from m019_users where id = $1`, id).
		Scan(&u.ID, &u.Name, &u.Email)
	if err != nil {
		return nil, translateError(err) // sql.ErrNoRows becomes errNotFound
	}
	return &u, nil
}

func (r *postgresRepo) List(ctx context.Context) ([]user, error) {
	rows, err := r.db.QueryContext(ctx, `select id, name, email from m019_users order by id`)
	if err != nil {
		return nil, translateError(err)
	}
	defer rows.Close()
	var out []user
	for rows.Next() {
		var u user
		if err := rows.Scan(&u.ID, &u.Name, &u.Email); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (r *postgresRepo) Delete(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, `delete from m019_users where id = $1`, id)
	if err != nil {
		return translateError(err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return errNotFound // deleting nothing is a domain-level "not found"
	}
	return nil
}

// memoryRepo is the in-memory implementation: the whole test double in thirty lines.
type memoryRepo struct {
	mu     sync.RWMutex
	nextID int64
	users  map[int64]user
}

func newMemoryRepo() *memoryRepo { return &memoryRepo{users: map[int64]user{}} }

func (r *memoryRepo) Create(ctx context.Context, u *user) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, existing := range r.users {
		if existing.Email == u.Email {
			return errDuplicate // the same domain error the SQL version returns
		}
	}
	r.nextID++
	u.ID = r.nextID
	r.users[u.ID] = *u
	return nil
}

func (r *memoryRepo) ByID(ctx context.Context, id int64) (*user, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	u, ok := r.users[id]
	if !ok {
		return nil, errNotFound
	}
	return &u, nil
}

func (r *memoryRepo) List(ctx context.Context) ([]user, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]user, 0, len(r.users))
	for _, u := range r.users {
		out = append(out, u)
	}
	slices.SortFunc(out, func(a, b user) int { return cmp.Compare(a.ID, b.ID) })
	return out, nil
}

func (r *memoryRepo) Delete(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.users[id]; !ok {
		return errNotFound
	}
	delete(r.users, id)
	return nil
}

// resetSchema drops and recreates the demo table, so the module can be re-run. The table name
// carries the module number so it cannot clash with module 020 or the examples' tables.
func resetSchema(ctx context.Context, db *sql.DB) error {
	const schema = `
		drop table if exists m019_users;
		create table m019_users (
			id       bigserial primary key,
			name     varchar(100) not null,
			email    varchar(100) not null unique,
			nickname varchar(100)
		);`
	if _, err := db.ExecContext(ctx, schema); err != nil {
		return fmt.Errorf("create schema: %w", err)
	}
	return nil
}

// shorten puts a (multi-line) driver error on one short line, so the output stays readable.
func shorten(err error) string {
	if err == nil {
		return "<nil>"
	}
	line := strings.Join(strings.Fields(err.Error()), " ")
	if len(line) > 110 {
		line = line[:110] + "..."
	}
	return line
}
