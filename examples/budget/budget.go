// Package budget records household deposits and withdrawals, prints a report
// with the closing balance, and persists the entries in a JSON file.
//
// Typical use from a command:
//
//	b := budget.New(budget.DefaultPath)
//	if err := b.Load(); err != nil { ... }
//	entry, err := budget.ParseEntry(os.Args[1:])
//	b.Add(entry)
//	b.Print()
//	err = b.Save()
package budget

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// DefaultPath is the JSON file used when the caller has no better idea.
const DefaultPath = "budget.json"

// Budget is the list of entries together with the file they are stored in.
type Budget struct {
	Entries []Entry `json:"entries"`
	path    string
}

// New returns an empty budget bound to path. Call Load to read what the file
// already contains.
func New(path string) *Budget {
	if path == "" {
		path = DefaultPath
	}
	return &Budget{path: path}
}

// Add appends an entry.
func (b *Budget) Add(entry Entry) {
	b.Entries = append(b.Entries, entry)
}

// Balance returns deposits minus withdrawals in minor units.
func (b *Budget) Balance() int64 {
	// Computed in integers - no error accumulation.
	var balanceMinor int64
	for _, entry := range b.Entries {
		if entry.OperationType == Deposit {
			balanceMinor += entry.AmountMinor
		} else {
			balanceMinor -= entry.AmountMinor
		}
	}
	return balanceMinor
}

// Print writes every entry followed by the closing balance to stdout.
func (b *Budget) Print() {
	for _, entry := range b.Entries {
		entry.Print()
	}
	fmt.Println("----------------------------------------------------")
	fmt.Printf("Balance: %v\n", FormatMinor(b.Balance()))
}

// Save writes the budget to its file atomically: the JSON goes to a temporary
// file in the same directory, which is then renamed over the target. A crash
// half-way through leaves either the old file or the new one, never a
// truncated mix of both.
func (b *Budget) Save() error {
	data, err := json.MarshalIndent(b, "", "\t")
	if err != nil {
		return fmt.Errorf("encoding the budget: %w", err)
	}

	dir := filepath.Dir(b.path)
	tmp, err := os.CreateTemp(dir, filepath.Base(b.path)+".*.tmp")
	if err != nil {
		return fmt.Errorf("creating a temporary file in %s: %w", dir, err)
	}
	// On any failure below the temporary file is removed; after a successful
	// rename it no longer exists and Remove is a harmless no-op.
	defer os.Remove(tmp.Name())

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("writing %s: %w", tmp.Name(), err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("closing %s: %w", tmp.Name(), err)
	}
	if err := os.Rename(tmp.Name(), b.path); err != nil {
		return fmt.Errorf("replacing %s: %w", b.path, err)
	}
	return nil
}

// Load replaces the entries with the contents of the budget file. A missing
// file is not an error - on the first run we start from an empty budget.
func (b *Budget) Load() error {
	data, err := os.ReadFile(b.path)
	if errors.Is(err, os.ErrNotExist) {
		b.Entries = nil
		return nil
	}
	if err != nil {
		return fmt.Errorf("reading %s: %w", b.path, err)
	}
	var loaded Budget
	if err := json.Unmarshal(data, &loaded); err != nil {
		return fmt.Errorf("parsing %s: %w", b.path, err)
	}
	b.Entries = loaded.Entries
	return nil
}

// ParseEntry builds an entry from command-line style arguments: an amount such
// as "12.34" (a deposit) or "-12.34" (a withdrawal) followed by a description.
// It returns an error instead of silently doing nothing - a typo in the amount
// (or the wrong argument count) used to produce neither an entry nor any
// message.
func ParseEntry(args []string) (Entry, error) {
	if len(args) != 2 {
		return Entry{}, fmt.Errorf("expected 2 arguments (amount description), got %d", len(args))
	}
	minor, err := parseAmountToMinor(args[0])
	if err != nil {
		return Entry{}, err
	}

	operationType := Deposit
	if minor < 0 {
		operationType = Withdraw
		minor = -minor
	}
	return NewEntry(minor, operationType, args[1]), nil
}

// parseAmountToMinor converts "[-|+]units[.cents]" to minor units without going
// through float64. Both parts must be plain decimal digits and cents may have
// at most two of them: "5", "5.5" (5.50) and "5.05" are valid; "5.", "5.123",
// "1e3" and "0x10" are not.
func parseAmountToMinor(value string) (int64, error) {
	original := value
	negative := strings.HasPrefix(value, "-")
	value = strings.TrimPrefix(strings.TrimPrefix(value, "-"), "+")

	units, cents, found := strings.Cut(value, ".")
	if !found {
		cents = "0"
	}
	if !isDigits(units) || !isDigits(cents) {
		return 0, fmt.Errorf("invalid amount %q: expected digits[.digits]", original)
	}
	// Pad to two digits: "5" -> "50" (i.e. 50 grosze); "123" is an error.
	switch len(cents) {
	case 1:
		cents += "0"
	case 2:
	default:
		return 0, fmt.Errorf("invalid amount %q: expected at most 2 digits after the dot", original)
	}

	unitsValue, err := strconv.ParseInt(units, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid amount %q: %w", original, err)
	}
	centsValue, err := strconv.ParseInt(cents, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid amount %q: %w", original, err)
	}

	// ParseInt guarantees unitsValue fits in int64, but *100 can still overflow.
	if unitsValue > (math.MaxInt64-centsValue)/100 {
		return 0, fmt.Errorf("invalid amount %q: too large", original)
	}
	minor := unitsValue*100 + centsValue
	if negative {
		minor = -minor
	}
	return minor, nil
}

// isDigits reports whether s is non-empty and consists of ASCII digits only.
// strconv.ParseInt alone would also accept a leading sign and underscores.
func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
