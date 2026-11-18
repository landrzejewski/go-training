package solutions

import (
	"cmp"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"sync"

	"github.com/lib/pq" // also registers the "postgres" driver for database/sql
)

// --- 19.1 A product store behind an interface ---

// Product is one item for sale. The price is in cents: money is never a float64.
type Product struct {
	ID    int64
	Name  string
	Price int64
}

// The domain errors of ProductStore. Callers check them with errors.Is.
var (
	ErrProductNotFound = errors.New("product not found")
	ErrProductExists   = errors.New("product already exists")
)

// ProductStore is the repository of products. Implementations wrap only the errors above,
// never a database error type, so callers cannot tell which one they talk to.
type ProductStore interface {
	// Add stores p and returns it with its new ID. A used name gives ErrProductExists.
	Add(ctx context.Context, p Product) (Product, error)
	// ByID returns the product with the given ID, or ErrProductNotFound.
	ByID(ctx context.Context, id int64) (Product, error)
	// Cheaper returns the products with Price <= maxPrice, cheapest first, then by ID.
	Cheaper(ctx context.Context, maxPrice int64) ([]Product, error)
}

// The compiler checks every implementation (19.1, 19.2, 20.2).
var (
	_ ProductStore = (*MemoryProductStore)(nil)
	_ ProductStore = (*SQLProductStore)(nil)
	_ ProductStore = (*GormProductStore)(nil)
)

// MemoryProductStore is ProductStore in memory, safe for concurrent use.
type MemoryProductStore struct {
	mu       sync.RWMutex
	nextID   int64
	products map[int64]Product
}

func NewMemoryProductStore() *MemoryProductStore {
	return &MemoryProductStore{products: map[int64]Product{}}
}

func (s *MemoryProductStore) Add(_ context.Context, p Product) (Product, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, existing := range s.products {
		if existing.Name == p.Name {
			return Product{}, fmt.Errorf("add %q: %w", p.Name, ErrProductExists)
		}
	}
	s.nextID++
	p.ID = s.nextID
	s.products[p.ID] = p
	return p, nil
}

func (s *MemoryProductStore) ByID(_ context.Context, id int64) (Product, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.products[id]
	if !ok {
		return Product{}, fmt.Errorf("product %d: %w", id, ErrProductNotFound)
	}
	return p, nil
}

func (s *MemoryProductStore) Cheaper(_ context.Context, maxPrice int64) ([]Product, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []Product{}
	for _, p := range s.products {
		if p.Price <= maxPrice {
			out = append(out, p)
		}
	}
	slices.SortFunc(out, func(a, b Product) int {
		return cmp.Or(cmp.Compare(a.Price, b.Price), cmp.Compare(a.ID, b.ID))
	})
	return out, nil
}

// --- 19.2 The same store on database/sql ---

// SQLProductStore is ProductStore on PostgreSQL. It expects this table:
//
//	create table ex19_products (
//		id    bigserial primary key,
//		name  text   not null unique,
//		price bigint not null
//	)
type SQLProductStore struct{ db *sql.DB }

func NewSQLProductStore(db *sql.DB) *SQLProductStore { return &SQLProductStore{db: db} }

func (s *SQLProductStore) Add(ctx context.Context, p Product) (Product, error) {
	// PostgreSQL has no LastInsertId: RETURNING gives the new id back.
	err := s.db.QueryRowContext(ctx,
		`insert into ex19_products (name, price) values ($1, $2) returning id`, p.Name, p.Price).Scan(&p.ID)
	if isUniqueViolation(err) {
		return Product{}, fmt.Errorf("add %q: %w", p.Name, ErrProductExists)
	}
	if err != nil {
		return Product{}, fmt.Errorf("add %q: %w", p.Name, err)
	}
	return p, nil
}

func (s *SQLProductStore) ByID(ctx context.Context, id int64) (Product, error) {
	var p Product
	err := s.db.QueryRowContext(ctx, `select id, name, price from ex19_products where id = $1`, id).
		Scan(&p.ID, &p.Name, &p.Price)
	if errors.Is(err, sql.ErrNoRows) {
		return Product{}, fmt.Errorf("product %d: %w", id, ErrProductNotFound)
	}
	if err != nil {
		return Product{}, fmt.Errorf("product %d: %w", id, err)
	}
	return p, nil
}

func (s *SQLProductStore) Cheaper(ctx context.Context, maxPrice int64) ([]Product, error) {
	rows, err := s.db.QueryContext(ctx,
		`select id, name, price from ex19_products where price <= $1 order by price, id`, maxPrice)
	if err != nil {
		return nil, fmt.Errorf("cheaper than %d: %w", maxPrice, err)
	}
	defer rows.Close() // rows hold a pooled connection until closed
	out := []Product{}
	for rows.Next() {
		var p Product
		if err := rows.Scan(&p.ID, &p.Name, &p.Price); err != nil {
			return nil, fmt.Errorf("cheaper than %d: %w", maxPrice, err)
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil { // the loop ended - because of an error?
		return nil, fmt.Errorf("cheaper than %d: %w", maxPrice, err)
	}
	return out, nil
}

// isUniqueViolation reports whether err is PostgreSQL's unique_violation (SQLSTATE 23505).
func isUniqueViolation(err error) bool {
	pqErr, ok := errors.AsType[*pq.Error](err)
	return ok && pqErr.Code == "23505"
}

// --- 19.3 A money transfer in a transaction ---

// The errors of Transfer.
var (
	ErrAccountNotFound   = errors.New("account not found")
	ErrInsufficientFunds = errors.New("insufficient funds")
)

// Transfer moves amount from one account to another. Both updates happen in ONE transaction:
// either both are kept or neither is. It expects this table:
//
//	create table ex19_accounts (id bigint primary key, balance bigint not null)
func Transfer(ctx context.Context, db *sql.DB, from, to, amount int64) error {
	if amount <= 0 {
		return fmt.Errorf("transfer: amount must be positive, got %d", amount)
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("transfer: %w", err)
	}
	defer tx.Rollback() // undoes everything on every early return; a no-op after Commit

	// FOR UPDATE locks the row until the transaction ends, so two transfers running at the
	// same time cannot both spend the same money.
	var balance int64
	err = tx.QueryRowContext(ctx, `select balance from ex19_accounts where id = $1 for update`, from).Scan(&balance)
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("transfer from %d: %w", from, ErrAccountNotFound)
	}
	if err != nil {
		return fmt.Errorf("transfer: %w", err)
	}
	if balance < amount {
		return fmt.Errorf("transfer %d from %d (balance %d): %w", amount, from, balance, ErrInsufficientFunds)
	}

	if _, err := tx.ExecContext(ctx,
		`update ex19_accounts set balance = balance - $1 where id = $2`, amount, from); err != nil {
		return fmt.Errorf("transfer: %w", err)
	}
	res, err := tx.ExecContext(ctx, `update ex19_accounts set balance = balance + $1 where id = $2`, amount, to)
	if err != nil {
		return fmt.Errorf("transfer: %w", err)
	}
	if n, err := res.RowsAffected(); err != nil || n == 0 {
		// The money already left the first account - the deferred Rollback puts it back.
		return fmt.Errorf("transfer to %d: %w", to, ErrAccountNotFound)
	}
	return tx.Commit()
}
