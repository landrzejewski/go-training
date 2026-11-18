// Package library is the reference solution for Project 1 in basic_exercises.md: a small
// library that lends books to members, charges late fees and saves its state as JSON.
//
// It is safe for concurrent use: every method locks the same mutex.
package library

import (
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"maps"
	"os"
	"slices"
	"strings"
	"sync"
	"time"
)

// The rules of the library.
const (
	LoanPeriod = 14 * 24 * time.Hour // how long a member may keep a book
	MaxLoans   = 3                   // books one member may have at the same time
	FeePerDay  = 50                  // late fee in cents for every full day after the due date
)

// Sentinel errors. The methods wrap them with context; callers check them with errors.Is.
var (
	ErrInvalidBook   = errors.New("invalid book")
	ErrDuplicateBook = errors.New("duplicate book")
	ErrBookNotFound  = errors.New("book not found")
	ErrAlreadyOnLoan = errors.New("book already on loan")
	ErrNotOnLoan     = errors.New("book not on loan")
)

// LoanLimitError is returned by Borrow when the member already has MaxLoans books. It is a
// type rather than a sentinel because callers may want to show the limit (errors.As).
type LoanLimitError struct {
	MemberID string
	Limit    int
}

func (e *LoanLimitError) Error() string {
	return fmt.Sprintf("member %s already has %d books on loan", e.MemberID, e.Limit)
}

// Clock tells the library what time it is. Tests pass a fake clock to travel in time.
type Clock interface {
	Now() time.Time
}

// RealClock is the Clock for production use.
type RealClock struct{}

func (RealClock) Now() time.Time { return time.Now() }

// Book is one title in the catalogue.
type Book struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Author string `json:"author"`
	Year   int    `json:"year"`
}

// Loan records who has a book and until when.
type Loan struct {
	BookID   string    `json:"book_id"`
	MemberID string    `json:"member_id"`
	Borrowed time.Time `json:"borrowed"`
	Due      time.Time `json:"due"`
}

// Library holds the catalogue and the current loans. Create one with New or Load.
type Library struct {
	mu    sync.Mutex
	clock Clock
	books map[string]Book // by book ID
	loans map[string]Loan // by book ID: a book has at most one active loan
}

// New returns an empty library that uses clock to tell the time.
func New(clock Clock) *Library {
	return &Library{
		clock: clock,
		books: make(map[string]Book),
		loans: make(map[string]Loan),
	}
}

// AddBook adds b to the catalogue. The ID and title must not be blank, and the ID must be new.
func (l *Library) AddBook(b Book) error {
	if strings.TrimSpace(b.ID) == "" || strings.TrimSpace(b.Title) == "" {
		return fmt.Errorf("add book %q: id and title are required: %w", b.ID, ErrInvalidBook)
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if _, exists := l.books[b.ID]; exists {
		return fmt.Errorf("add book %q: %w", b.ID, ErrDuplicateBook)
	}
	l.books[b.ID] = b
	return nil
}

// Borrow lends the book to the member for LoanPeriod and returns the new loan.
func (l *Library) Borrow(memberID, bookID string) (Loan, error) {
	l.mu.Lock()
	defer l.mu.Unlock() // one lock around check AND update: two goroutines cannot both win

	if _, ok := l.books[bookID]; !ok {
		return Loan{}, fmt.Errorf("borrow %q: %w", bookID, ErrBookNotFound)
	}
	if _, onLoan := l.loans[bookID]; onLoan {
		return Loan{}, fmt.Errorf("borrow %q: %w", bookID, ErrAlreadyOnLoan)
	}
	if l.loanCount(memberID) >= MaxLoans {
		return Loan{}, fmt.Errorf("borrow %q: %w", bookID, &LoanLimitError{MemberID: memberID, Limit: MaxLoans})
	}

	now := l.clock.Now()
	loan := Loan{BookID: bookID, MemberID: memberID, Borrowed: now, Due: now.Add(LoanPeriod)}
	l.loans[bookID] = loan
	return loan, nil
}

// loanCount returns how many books the member has. The caller must hold l.mu.
func (l *Library) loanCount(memberID string) int {
	n := 0
	for _, loan := range l.loans {
		if loan.MemberID == memberID {
			n++
		}
	}
	return n
}

// Return ends the loan of the book and returns the late fee in cents (0 when on time).
func (l *Library) Return(bookID string) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	loan, ok := l.loans[bookID]
	if !ok {
		return 0, fmt.Errorf("return %q: %w", bookID, ErrNotOnLoan)
	}
	delete(l.loans, bookID)
	return LateFee(loan.Due, l.clock.Now()), nil
}

// LateFee returns FeePerDay for every full day between due and returned.
func LateFee(due, returned time.Time) int {
	if !returned.After(due) {
		return 0
	}
	fullDays := int(returned.Sub(due) / (24 * time.Hour)) // Duration / Duration = a plain number
	return fullDays * FeePerDay
}

// Search returns the books whose title or author contains query, ignoring case, sorted by title.
func (l *Library) Search(query string) []Book {
	q := strings.ToLower(query)
	l.mu.Lock()
	defer l.mu.Unlock()

	var found []Book
	for _, b := range l.books {
		if strings.Contains(strings.ToLower(b.Title), q) || strings.Contains(strings.ToLower(b.Author), q) {
			found = append(found, b)
		}
	}
	slices.SortFunc(found, func(a, b Book) int { return cmp.Compare(a.Title, b.Title) })
	return found
}

// Overdue yields the loans whose due date has passed, the most overdue first.
//
// The iterator copies the overdue loans while holding the lock and yields them after
// releasing it. Yielding under the lock would deadlock as soon as the loop body called
// another Library method, such as Return.
func (l *Library) Overdue() iter.Seq[Loan] {
	return func(yield func(Loan) bool) {
		for _, loan := range l.overdueSnapshot() {
			if !yield(loan) {
				return
			}
		}
	}
}

func (l *Library) overdueSnapshot() []Loan {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.clock.Now()
	var overdue []Loan
	for _, loan := range l.loans {
		if now.After(loan.Due) {
			overdue = append(overdue, loan)
		}
	}
	slices.SortFunc(overdue, func(a, b Loan) int { return a.Due.Compare(b.Due) })
	return overdue
}

// snapshot is the JSON shape of a saved library.
type snapshot struct {
	Books []Book `json:"books"`
	Loans []Loan `json:"loans"`
}

// Save writes the catalogue and the loans to path as indented JSON, sorted by ID so the
// file does not change between runs when the data does not.
func (l *Library) Save(path string) error {
	l.mu.Lock()
	snap := snapshot{Books: sortedValues(l.books), Loans: sortedValues(l.loans)}
	l.mu.Unlock() // no need to hold the lock while encoding and writing

	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return fmt.Errorf("save library: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("save library: %w", err)
	}
	return nil
}

// Load reads a library saved by Save. Every loan must refer to a book in the file.
func Load(path string, clock Clock) (*Library, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("load library: %w", err)
	}
	var snap snapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return nil, fmt.Errorf("load library: %w", err)
	}

	l := New(clock)
	for _, b := range snap.Books {
		if err := l.AddBook(b); err != nil { // reuse the validation of AddBook
			return nil, fmt.Errorf("load library: %w", err)
		}
	}
	for _, loan := range snap.Loans {
		if _, ok := l.books[loan.BookID]; !ok {
			return nil, fmt.Errorf("load library: loan of %q: %w", loan.BookID, ErrBookNotFound)
		}
		l.loans[loan.BookID] = loan
	}
	return l, nil
}

// sortedValues returns the values of m ordered by key. It is generic, so it works for both
// map[string]Book and map[string]Loan.
func sortedValues[K cmp.Ordered, V any](m map[K]V) []V {
	out := make([]V, 0, len(m))
	for _, k := range slices.Sorted(maps.Keys(m)) {
		out = append(out, m[k])
	}
	return out
}
