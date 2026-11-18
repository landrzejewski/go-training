package library

import (
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// fakeClock is a Clock the test can move forward.
type fakeClock struct{ now time.Time }

func (c *fakeClock) Now() time.Time          { return c.now }
func (c *fakeClock) Advance(d time.Duration) { c.now = c.now.Add(d) }

func newClock() *fakeClock {
	return &fakeClock{now: time.Date(2026, time.March, 1, 10, 0, 0, 0, time.UTC)}
}

// newTestLibrary returns a library with three books. t.Helper makes failures point at the
// caller, not at this function.
func newTestLibrary(t *testing.T, clock Clock) *Library {
	t.Helper()
	l := New(clock)
	for _, b := range []Book{
		{ID: "b1", Title: "The Go Programming Language", Author: "Donovan", Year: 2015},
		{ID: "b2", Title: "Dune", Author: "Herbert", Year: 1965},
		{ID: "b3", Title: "Go in Action", Author: "Kennedy", Year: 2015},
		{ID: "b4", Title: "Solaris", Author: "Lem", Year: 1961},
	} {
		if err := l.AddBook(b); err != nil {
			t.Fatalf("AddBook(%q): %v", b.ID, err)
		}
	}
	return l
}

func TestAddBookErrors(t *testing.T) {
	l := newTestLibrary(t, newClock())
	tests := []struct {
		name string
		book Book
		want error
	}{
		{"blank id", Book{Title: "No ID"}, ErrInvalidBook},
		{"blank title", Book{ID: "x", Title: "  "}, ErrInvalidBook},
		{"duplicate", Book{ID: "b1", Title: "Again"}, ErrDuplicateBook},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := l.AddBook(tt.book); !errors.Is(err, tt.want) {
				t.Errorf("AddBook(%+v) error = %v, want %v", tt.book, err, tt.want)
			}
		})
	}
}

func TestBorrowAndReturn(t *testing.T) {
	clock := newClock()
	l := newTestLibrary(t, clock)

	loan, err := l.Borrow("ann", "b2")
	if err != nil {
		t.Fatalf("Borrow: %v", err)
	}
	if want := clock.now.Add(LoanPeriod); !loan.Due.Equal(want) {
		t.Errorf("Due = %v, want %v", loan.Due, want)
	}

	clock.Advance(LoanPeriod + 2*24*time.Hour + 5*time.Hour) // 2 full days and 5 hours late
	fee, err := l.Return("b2")
	if err != nil {
		t.Fatalf("Return: %v", err)
	}
	if fee != 2*FeePerDay {
		t.Errorf("fee = %d, want %d", fee, 2*FeePerDay)
	}

	if _, err := l.Return("b2"); !errors.Is(err, ErrNotOnLoan) {
		t.Errorf("second Return error = %v, want ErrNotOnLoan", err)
	}
}

func TestBorrowErrors(t *testing.T) {
	l := newTestLibrary(t, newClock())
	if _, err := l.Borrow("ann", "nope"); !errors.Is(err, ErrBookNotFound) {
		t.Errorf("unknown book: error = %v, want ErrBookNotFound", err)
	}

	for _, id := range []string{"b1", "b2", "b3"} {
		if _, err := l.Borrow("ann", id); err != nil {
			t.Fatalf("Borrow(%q): %v", id, err)
		}
	}
	if _, err := l.Borrow("bob", "b1"); !errors.Is(err, ErrAlreadyOnLoan) {
		t.Errorf("book on loan: error = %v, want ErrAlreadyOnLoan", err)
	}

	_, err := l.Borrow("ann", "b4") // a fourth book is over the limit
	var limitErr *LoanLimitError
	if !errors.As(err, &limitErr) {
		t.Fatalf("over the limit: error = %v, want *LoanLimitError", err)
	}
	if limitErr.MemberID != "ann" || limitErr.Limit != MaxLoans {
		t.Errorf("LoanLimitError = %+v", *limitErr)
	}
}

func TestLateFee(t *testing.T) {
	due := time.Date(2026, time.March, 15, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name     string
		returned time.Time
		want     int
	}{
		{"early", due.Add(-time.Hour), 0},
		{"exactly on time", due, 0},
		{"less than a day late", due.Add(23 * time.Hour), 0},
		{"one day late", due.Add(24 * time.Hour), FeePerDay},
		{"ten and a half days late", due.Add(252 * time.Hour), 10 * FeePerDay},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := LateFee(due, tt.returned); got != tt.want {
				t.Errorf("LateFee = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestOverdue(t *testing.T) {
	clock := newClock()
	l := newTestLibrary(t, clock)
	mustBorrow(t, l, "ann", "b1")
	clock.Advance(48 * time.Hour)
	mustBorrow(t, l, "bob", "b2")
	clock.Advance(LoanPeriod + time.Hour) // both loans are overdue now; b1 is the older one
	mustBorrow(t, l, "cid", "b3")         // a fresh loan: not overdue

	var got []string
	for loan := range l.Overdue() {
		got = append(got, loan.BookID)
		if _, err := l.Return(loan.BookID); err != nil { // calling the library inside the loop must not deadlock
			t.Fatalf("Return inside Overdue loop: %v", err)
		}
	}
	if fmt.Sprint(got) != "[b1 b2]" {
		t.Errorf("Overdue = %v, want [b1 b2]", got)
	}
}

func TestSearch(t *testing.T) {
	l := newTestLibrary(t, newClock())
	var titles []string
	for _, b := range l.Search("GO") {
		titles = append(titles, b.Title)
	}
	if want := "[Go in Action The Go Programming Language]"; fmt.Sprint(titles) != want {
		t.Errorf("Search(GO) = %v, want %v", titles, want)
	}
	if got := l.Search("lem"); len(got) != 1 || got[0].ID != "b4" {
		t.Errorf("Search(lem) = %v, want only b4", got)
	}
}

func TestSaveAndLoad(t *testing.T) {
	clock := newClock()
	l := newTestLibrary(t, clock)
	mustBorrow(t, l, "ann", "b2")

	path := filepath.Join(t.TempDir(), "library.json") // removed automatically after the test
	if err := l.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}
	loaded, err := Load(path, clock)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(loaded.Search("")) != 4 {
		t.Errorf("loaded %d books, want 4", len(loaded.Search("")))
	}
	if _, err := loaded.Borrow("bob", "b2"); !errors.Is(err, ErrAlreadyOnLoan) {
		t.Errorf("the loan of b2 was not restored: %v", err)
	}

	if _, err := Load(filepath.Join(t.TempDir(), "missing.json"), clock); err == nil {
		t.Error("Load of a missing file returned no error")
	}
}

// TestConcurrentBorrow: many members try to borrow the same book at once. Exactly one must
// succeed. Run with go test -race.
func TestConcurrentBorrow(t *testing.T) {
	l := newTestLibrary(t, newClock())
	var (
		wg        sync.WaitGroup
		mu        sync.Mutex
		successes int
	)
	for i := range 50 {
		wg.Go(func() {
			_, err := l.Borrow(fmt.Sprintf("member-%d", i), "b2")
			if err != nil && !errors.Is(err, ErrAlreadyOnLoan) {
				t.Errorf("unexpected error: %v", err)
			}
			if err == nil {
				mu.Lock()
				successes++
				mu.Unlock()
			}
		})
	}
	wg.Wait()
	if successes != 1 {
		t.Errorf("%d members borrowed the same book, want exactly 1", successes)
	}
}

func mustBorrow(t *testing.T, l *Library, member, book string) {
	t.Helper()
	if _, err := l.Borrow(member, book); err != nil {
		t.Fatalf("Borrow(%q, %q): %v", member, book, err)
	}
}

func ExampleLibrary() {
	clock := &fakeClock{now: time.Date(2026, time.March, 1, 10, 0, 0, 0, time.UTC)}
	lib := New(clock)
	_ = lib.AddBook(Book{ID: "b1", Title: "Dune", Author: "Herbert", Year: 1965})

	loan, _ := lib.Borrow("ann", "b1")
	fmt.Println("due:", loan.Due.Format("2006-01-02"))

	_, err := lib.Borrow("bob", "b1")
	fmt.Println(err)

	clock.Advance(LoanPeriod + 3*24*time.Hour)
	for loan := range lib.Overdue() {
		fmt.Println("overdue:", loan.BookID, "borrowed by", loan.MemberID)
	}
	fee, _ := lib.Return("b1")
	fmt.Printf("late fee: %d.%02d\n", fee/100, fee%100)
	// Output:
	// due: 2026-03-15
	// borrow "b1": book already on loan
	// overdue: b1 borrowed by ann
	// late fee: 1.50
}
