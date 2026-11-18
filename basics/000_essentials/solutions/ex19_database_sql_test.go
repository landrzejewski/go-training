package solutions

// The SQL tests need the PostgreSQL from docker-compose.yml and are SKIPPED without it. The
// DSN environment variable points them at another database.
//
//	docker-compose up -d
//	go test -v -run 'Product|Transfer' ./basics/000_essentials/solutions

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"
)

func ExampleMemoryProductStore() {
	ctx := context.Background()
	store := NewMemoryProductStore()
	for _, p := range []Product{{Name: "book", Price: 1500}, {Name: "pen", Price: 250}, {Name: "mug", Price: 800}} {
		added, err := store.Add(ctx, p)
		fmt.Println(added, err)
	}
	_, err := store.Add(ctx, Product{Name: "pen", Price: 1})
	fmt.Println(err, errors.Is(err, ErrProductExists))

	p, err := store.ByID(ctx, 2)
	fmt.Println(p, err)
	_, err = store.ByID(ctx, 99)
	fmt.Println(err, errors.Is(err, ErrProductNotFound))

	cheap, _ := store.Cheaper(ctx, 800)
	fmt.Println(cheap)
	// Output:
	// {1 book 1500} <nil>
	// {2 pen 250} <nil>
	// {3 mug 800} <nil>
	// add "pen": product already exists true
	// {2 pen 250} <nil>
	// product 99: product not found true
	// [{2 pen 250} {3 mug 800}]
}

// testProductStoreContract runs the same checks against any ProductStore (19.1, 19.2, 20.2).
func testProductStoreContract(t *testing.T, store ProductStore) {
	t.Helper()
	ctx := t.Context()
	var added []Product
	for _, p := range []Product{{Name: "book", Price: 1500}, {Name: "pen", Price: 250}, {Name: "mug", Price: 800}} {
		got, err := store.Add(ctx, p)
		if err != nil || got.ID == 0 || got.Name != p.Name || got.Price != p.Price {
			t.Fatalf("Add(%v) = %v, %v; want it back with an ID", p, got, err)
		}
		added = append(added, got)
	}
	if _, err := store.Add(ctx, Product{Name: "pen", Price: 1}); !errors.Is(err, ErrProductExists) {
		t.Errorf("Add of a used name = %v, want ErrProductExists", err)
	}
	if got, err := store.ByID(ctx, added[1].ID); err != nil || got != added[1] {
		t.Errorf("ByID(%d) = %v, %v; want %v", added[1].ID, got, err, added[1])
	}
	if _, err := store.ByID(ctx, 99999); !errors.Is(err, ErrProductNotFound) {
		t.Errorf("ByID of a missing id = %v, want ErrProductNotFound", err)
	}
	cheap, err := store.Cheaper(ctx, 800)
	if err != nil || len(cheap) != 2 || cheap[0].Name != "pen" || cheap[1].Name != "mug" {
		t.Errorf("Cheaper(800) = %v, %v; want [pen mug]", cheap, err)
	}
	if none, err := store.Cheaper(ctx, 100); err != nil || len(none) != 0 {
		t.Errorf("Cheaper(100) = %v, %v; want nothing", none, err)
	}
}

func TestMemoryProductStore(t *testing.T) {
	testProductStoreContract(t, NewMemoryProductStore())
}

func TestSQLProductStore(t *testing.T) {
	db := openTestDB(t)
	mustExec(t, db, `drop table if exists ex19_products;
		create table ex19_products (
			id    bigserial primary key,
			name  text   not null unique,
			price bigint not null
		)`)
	testProductStoreContract(t, NewSQLProductStore(db))
}

func TestTransfer(t *testing.T) {
	db := openTestDB(t)
	ctx := t.Context()
	mustExec(t, db, `drop table if exists ex19_accounts;
		create table ex19_accounts (id bigint primary key, balance bigint not null);
		insert into ex19_accounts (id, balance) values (1, 100), (2, 50)`)

	steps := []struct {
		name             string
		from, to, amount int64
		wantErr          error // nil = success
		wantFrom, wantTo int64 // the balances of accounts 1 and 2 afterwards
	}{
		{"ok", 1, 2, 30, nil, 70, 80},
		{"too much", 1, 2, 500, ErrInsufficientFunds, 70, 80},
		{"unknown sender", 9, 2, 10, ErrAccountNotFound, 70, 80},
		{"unknown receiver: the debit is rolled back", 1, 9, 10, ErrAccountNotFound, 70, 80},
		{"back again", 2, 1, 80, nil, 150, 0},
	}
	for _, s := range steps {
		err := Transfer(ctx, db, s.from, s.to, s.amount)
		if (s.wantErr == nil && err != nil) || (s.wantErr != nil && !errors.Is(err, s.wantErr)) {
			t.Errorf("%s: Transfer = %v, want %v", s.name, err, s.wantErr)
		}
		if a, b := balance(t, db, 1), balance(t, db, 2); a != s.wantFrom || b != s.wantTo {
			t.Errorf("%s: balances = %d, %d; want %d, %d", s.name, a, b, s.wantFrom, s.wantTo)
		}
	}
	if err := Transfer(ctx, db, 1, 2, 0); err == nil {
		t.Error("Transfer of 0 returned no error")
	}
}

// testDSN is the database the tests use: $DSN, or the one from docker-compose.yml.
func testDSN() string {
	if v := os.Getenv("DSN"); v != "" {
		return v
	}
	return "postgres://admin:admin@localhost:5432/users?sslmode=disable"
}

// openTestDB connects to PostgreSQL or SKIPS the test: a missing database is not a bug in the
// code under test. The pool is closed when the test ends.
func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("postgres", testDSN())
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		t.Skipf("PostgreSQL is not reachable (docker-compose up -d): %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func mustExec(t *testing.T, db *sql.DB, query string) {
	t.Helper()
	if _, err := db.ExecContext(t.Context(), query); err != nil {
		t.Fatal(err)
	}
}

func balance(t *testing.T, db *sql.DB, id int64) int64 {
	t.Helper()
	var b int64
	if err := db.QueryRowContext(t.Context(), `select balance from ex19_accounts where id = $1`, id).Scan(&b); err != nil {
		t.Fatal(err)
	}
	return b
}
