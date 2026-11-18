package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/lib/pq" // the PostgreSQL driver; see step 1
)

// ============================================================================================
// 019 - database/sql and the repository pattern
// ============================================================================================
//
// What you build: the users of 018 stored in PostgreSQL, with database/sql - the standard
// library's way to talk to SQL databases. It is not an ORM: you write the SQL yourself.
//
//	1. connecting     a driver, the connection pool, and a check that the database answers
//	2. queries        ExecContext, QueryRowContext, QueryContext - and which one to use when
//	3. safe values    placeholders against SQL injection, and NULL
//	4. transactions   several changes that are kept together or not at all
//	5. a repository   UserRepository from 018 on PostgreSQL - and the handlers stay the same
//
// Steps 2-4 need PostgreSQL (`docker-compose up -d`); without it they say "skipped". Step 5
// also runs the in-memory storage from 018, which needs nothing.

// databaseSQL runs the steps of section 019.
func databaseSQL() {
	fmt.Println("--- 019 database/sql and the repository pattern ---")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	db := connectDemo(ctx) // 1. connecting - nil when PostgreSQL is not reachable
	if db != nil {
		defer db.Close() // the pool is closed once, when the program is done with it
	}
	queriesDemo(ctx, db)     // 2. queries
	safeValuesDemo(ctx, db)  // 3. safe values
	transactionDemo(ctx, db) // 4. transactions
	repositoryDemo(ctx, db)  // 5. a repository

	// More: basics/019_database_sql_and_repository
}

// --------------------------------------------------------------------------------------------
// Step 1 - connecting
// --------------------------------------------------------------------------------------------

// connectDemo is step 1. database/sql defines the interface; a DRIVER implements it for one
// database. A driver registers itself in its init() function, so importing it is enough.
// When you use nothing else from the driver package, import it with a blank name:
//
//	import _ "github.com/lib/pq"
//
// (This file also uses pq.Error, so it imports lib/pq by name.) The other popular PostgreSQL
// driver is pgx; GORM uses it in 020.
func connectDemo(ctx context.Context) *sql.DB {
	step("1. connecting: open the pool, then ping the database")
	db, err := openPostgres(ctx)
	if err != nil {
		printNoDatabase(err)
		return nil
	}
	// *sql.DB is a POOL of connections, safe for concurrent use: open ONE for the whole program.
	// Opening one per request is the classic mistake.
	show("PostgreSQL answered; the pool allows up to %d open connections", db.Stats().MaxOpenConnections)
	return db
}

// openPostgres opens the connection pool and checks that the database answers.
func openPostgres(ctx context.Context) (*sql.DB, error) {
	// sql.Open does NOT connect: it only sets up the pool, and fails only for a driver name that
	// nobody registered or a malformed DSN. "postgres" was registered by lib/pq's init().
	db, err := sql.Open("postgres", dsn())
	if err != nil {
		return nil, err
	}
	// The pool's defaults are wrong for a server: unlimited open connections, only 2 idle ones.
	db.SetMaxOpenConns(25)                 // stay well below the server's max_connections
	db.SetMaxIdleConns(25)                 // reconnecting is expensive, idle connections are cheap
	db.SetConnMaxLifetime(5 * time.Minute) // lets failovers and load balancers rebalance
	db.SetConnMaxIdleTime(2 * time.Minute) // gives back connections left over from a burst
	pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil { // the first real connection: fail fast at start
		db.Close()
		return nil, err
	}
	return db, nil
}

const defaultDSN = "postgres://admin:admin@localhost:5432/users?sslmode=disable"

// dsn returns the connection string (the DSN, "data source name"): the DSN environment
// variable if set, otherwise the database from docker-compose.yml. 020 uses it too.
func dsn() string {
	if v := os.Getenv("DSN"); v != "" {
		return v
	}
	return defaultDSN
}

// printNoDatabase explains why the steps that need a database are skipped. 020 uses it too.
func printNoDatabase(err error) {
	show("NOTE: PostgreSQL is not reachable, so the steps that need it are skipped:")
	show("  %s", oneLine(err))
	show("  Start it with `docker-compose up -d`, or set DSN to another database.")
}

// oneLine puts a (possibly long, multi-line) driver error on one short line.
func oneLine(err error) string {
	line := strings.Join(strings.Fields(err.Error()), " ")
	if len(line) > 100 {
		line = line[:100] + "..."
	}
	return line
}

// --------------------------------------------------------------------------------------------
// Step 2 - queries
// --------------------------------------------------------------------------------------------

// queriesDemo is step 2: the three ways to run SQL. Always use the ...Context methods, so that
// a cancelled request also stops its query.
func queriesDemo(ctx context.Context, db *sql.DB) {
	step("2. queries: Exec for changes, QueryRow for one row, Query for many")
	if db == nil {
		show("(skipped: no database)")
		return
	}
	if err := resetUsersTable(ctx, db); err != nil {
		show("setup failed: %v", err)
		return
	}

	// ExecContext: statements that return no rows - INSERT, UPDATE, DELETE, CREATE.
	// $1, $2 ... are PLACEHOLDERS for the values; step 3 explains why.
	res, err := db.ExecContext(ctx, `insert into e019_users (name, email) values ($1, $2), ($3, $4)`,
		"Ada", "ada@example.com", "Alan", "alan@example.com")
	if err != nil {
		show("insert failed: %v", err)
		return
	}
	n, _ := res.RowsAffected()
	show("ExecContext: rows affected = %d", n)

	// QueryRowContext: exactly one row, read with Scan (column by column, in order).
	// PostgreSQL has no LastInsertId - ask for the new id with RETURNING.
	var id int64
	err = db.QueryRowContext(ctx, `insert into e019_users (name, email) values ($1, $2) returning id`,
		"Grace", "grace@example.com").Scan(&id)
	show("INSERT ... RETURNING id -> %d %v", id, err)

	// "No such row" is not an empty result but an error: sql.ErrNoRows.
	var name string
	err = db.QueryRowContext(ctx, `select name from e019_users where id = $1`, 999).Scan(&name)
	show("no row -> errors.Is(err, sql.ErrNoRows): %t", errors.Is(err, sql.ErrNoRows))

	// QueryContext: many rows. Four steps, none of them optional.
	rows, err := db.QueryContext(ctx, `select id, name, email from e019_users order by id`)
	if err != nil {
		show("query failed: %v", err)
		return
	}
	defer rows.Close() // 1. rows hold a connection from the pool until they are closed
	for rows.Next() {  // 2. move to the next row
		var u User
		if err := rows.Scan(&u.ID, &u.Name, &u.Email); err != nil { // 3. read it
			show("scan failed: %v", err)
			return
		}
		show("row: %+v", u)
	}
	if err := rows.Err(); err != nil { // 4. Next returned false: the end, or an error?
		show("iteration failed: %v", err)
	}
}

// resetUsersTable recreates the demo table, so that every run prints the same output. The
// e019_ prefix keeps it apart from the tables of modules 019 and 020.
func resetUsersTable(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `
		drop table if exists e019_users;
		create table e019_users (
			id       bigserial primary key,
			name     text not null,
			email    text not null unique,
			nickname text
		)`)
	return err
}

// --------------------------------------------------------------------------------------------
// Step 3 - safe values
// --------------------------------------------------------------------------------------------

// safeValuesDemo is step 3; it uses the rows that step 2 inserted.
func safeValuesDemo(ctx context.Context, db *sql.DB) {
	step("3. safe values: placeholders against SQL injection, and NULL")
	if db == nil {
		show("(skipped: no database)")
		return
	}

	// SQL INJECTION: building SQL out of user input ("... where name = '" + name + "'") lets the
	// input change the query. With a placeholder the value travels separately from the SQL
	// text, so the classic attack is just a strange name that matches nothing. Placeholders work
	// for VALUES only; a table or column name must come from a fixed list in your code.
	payload := `x' or '1'='1`
	var count int
	_ = db.QueryRowContext(ctx, `select count(*) from e019_users where name = $1`, payload).Scan(&count)
	show("where name = $1 with %q -> %d rows", payload, count)

	// NULL: a plain string cannot hold it, and Scan fails. Use a pointer (nil means NULL) or
	// sql.Null[string] - or make the column NOT NULL when you own the schema.
	var nickname *string
	err := db.QueryRowContext(ctx, `select nickname from e019_users where name = $1`, "Grace").Scan(&nickname)
	show("NULL into a *string -> nil: %t %v", nickname == nil, err)
}

// --------------------------------------------------------------------------------------------
// Step 4 - transactions
// --------------------------------------------------------------------------------------------

// transactionDemo is step 4. A TRANSACTION groups several changes: either all of them are kept
// (Commit) or none (Rollback). It also uses the rows of step 2.
func transactionDemo(ctx context.Context, db *sql.DB) {
	step("4. transactions: all changes or none")
	if db == nil {
		show("(skipped: no database)")
		return
	}
	before := countRows(ctx, db)
	err := addUsersAtomically(ctx, db, []User{
		{Name: "Linus", Email: "linus@example.com"},
		{Name: "Copy", Email: "ada@example.com"}, // a duplicate: this insert fails
	})
	show("a transaction with a duplicate -> %v; rows before=%d after=%d (Linus was rolled back)",
		err, before, countRows(ctx, db))
}

// addUsersAtomically inserts all users in ONE transaction: all of them, or none.
func addUsersAtomically(ctx context.Context, db *sql.DB, users []User) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	// Rollback right after BeginTx: every early return - and a panic - undoes the transaction.
	// After a successful Commit it does nothing.
	defer tx.Rollback()
	for _, u := range users {
		// Inside the transaction use tx, never db: db would take another connection, outside it.
		_, err := tx.ExecContext(ctx, `insert into e019_users (name, email) values ($1, $2)`, u.Name, u.Email)
		if err != nil {
			return translateSQLError(err)
		}
	}
	return tx.Commit()
}

func countRows(ctx context.Context, db *sql.DB) int {
	var n int
	_ = db.QueryRowContext(ctx, `select count(*) from e019_users`).Scan(&n)
	return n
}

// --------------------------------------------------------------------------------------------
// Step 5 - a repository
// --------------------------------------------------------------------------------------------

// repositoryDemo is step 5: the REPOSITORY PATTERN. The handlers of 018 know only the
// UserRepository interface, so the storage behind it can change. The in-memory version needs
// no database, so it always runs - it is also what fast unit tests use. The PostgreSQL version
// must keep the same CONTRACT: the same answers and the same errors for the same calls.
func repositoryDemo(ctx context.Context, db *sql.DB) {
	step("5. a repository: one interface, the same answers from every storage")
	exerciseRepo(ctx, NewMemoryUserRepo(), "MemoryUserRepo (from 018)")
	if db == nil {
		show("PostgresUserRepo: skipped, no database")
		return
	}
	if err := resetUsersTable(ctx, db); err != nil {
		show("setup failed: %v", err)
		return
	}
	exerciseRepo(ctx, NewPostgresUserRepo(db), "PostgresUserRepo (the same contract, the same output)")

	// Swapping the storage needs no change in a single handler:
	api := usersGin(NewPostgresUserRepo(db))
	show("usersGin (018b) on PostgreSQL: POST -> %s", callAPI(api, "POST", "/users", `{"name":"Ann","email":"ann@example.com"}`))
	// id 3: in PostgreSQL even a failed insert (the duplicate above) uses up a sequence value.
}

// exerciseRepo is the CONTRACT of UserRepository, run in the same way against every
// implementation. 020 runs it a third time, on GORM.
func exerciseRepo(ctx context.Context, repo UserRepository, label string) {
	show("%s:", label)
	ada := User{Name: "Ada", Email: "ada@example.com"}
	err := repo.Create(ctx, &ada)
	show("  Create        -> id=%d err=%v", ada.ID, err)
	err = repo.Create(ctx, &User{Name: "Ada again", Email: "ada@example.com"})
	show("  Create again  -> errors.Is(err, ErrDuplicate): %t", errors.Is(err, ErrDuplicate))
	found, err := repo.ByID(ctx, ada.ID)
	show("  ByID          -> %+v err=%v", found, err)
	_, err = repo.ByID(ctx, 999)
	show("  ByID(999)     -> errors.Is(err, ErrNotFound): %t", errors.Is(err, ErrNotFound))
	all, err := repo.List(ctx)
	show("  List          -> %d user(s) err=%v", len(all), err)
	show("  Delete        -> %v", repo.Delete(ctx, ada.ID))
	show("  Delete again  -> errors.Is(err, ErrNotFound): %t", errors.Is(repo.Delete(ctx, ada.ID), ErrNotFound))
}

// PostgresUserRepo is UserRepository on database/sql.
type PostgresUserRepo struct{ db *sql.DB }

// The compiler checks that PostgresUserRepo satisfies UserRepository (008).
var _ UserRepository = (*PostgresUserRepo)(nil)

func NewPostgresUserRepo(db *sql.DB) *PostgresUserRepo { return &PostgresUserRepo{db: db} }

func (r *PostgresUserRepo) Create(ctx context.Context, u *User) error {
	err := r.db.QueryRowContext(ctx, `insert into e019_users (name, email) values ($1, $2) returning id`,
		u.Name, u.Email).Scan(&u.ID)
	return translateSQLError(err)
}

func (r *PostgresUserRepo) ByID(ctx context.Context, id int64) (User, error) {
	var u User
	err := r.db.QueryRowContext(ctx, `select id, name, email from e019_users where id = $1`, id).
		Scan(&u.ID, &u.Name, &u.Email)
	if err != nil {
		return User{}, translateSQLError(err) // sql.ErrNoRows becomes ErrNotFound
	}
	return u, nil
}

func (r *PostgresUserRepo) List(ctx context.Context) ([]User, error) {
	rows, err := r.db.QueryContext(ctx, `select id, name, email from e019_users order by id`)
	if err != nil {
		return nil, translateSQLError(err)
	}
	defer rows.Close()
	users := []User{}
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Name, &u.Email); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (r *PostgresUserRepo) Delete(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, `delete from e019_users where id = $1`, id)
	if err != nil {
		return translateSQLError(err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound // deleting nothing is not an SQL error, but it is "not found" for us
	}
	return nil
}

// translateSQLError turns driver errors into the domain errors of 018, so that no caller ever
// sees a database error. 23505 is PostgreSQL's code for a unique violation.
func translateSQLError(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if pqErr, ok := errors.AsType[*pq.Error](err); ok && pqErr.Code == "23505" {
		return ErrDuplicate
	}
	return err
}
