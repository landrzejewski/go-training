package examples

import (
	"errors"
	"fmt"
)

// MonetaryAmount is an amount of money in one currency. The value is stored in
// MINOR UNITS (grosze, cents) as an integer - never as a float64.
//
// Why: a binary float cannot represent 0.01, 0.10 or 0.20 exactly (the classic
// 0.1 + 0.2 != 0.3). Summing a ledger accumulates those errors, and %.2f
// formatting hides them - the balance quietly drifts away from the truth.
// For anything more serious, use a decimal package such as shopspring/decimal.
type MonetaryAmount struct {
	minorUnits int64 // e.g. 12345 == 123.45 PLN
	currency   string
}

// ErrCurrencyMismatch is returned by Add and Subtract when the two amounts are
// in different currencies. Sentinel errors are named Err... by convention so
// that callers can test them with errors.Is.
var ErrCurrencyMismatch = errors.New("currency mismatch")

// NewMonetaryAmount builds an amount from major and minor units, e.g.
// (123, 45, "PLN") for 123.45 PLN.
//
// The sign of the amount is taken from units: minorUnits is always applied in
// the direction of that sign, so (-1, 50, "PLN") is -1.50 and not -0.50 (a naive
// units*100+minorUnits would get every negative amount wrong). minorUnits must
// be in the range 0..99.
func NewMonetaryAmount(units, minorUnits int64, currency string) MonetaryAmount {
	if minorUnits < 0 || minorUnits > 99 {
		panic(fmt.Sprintf("examples: minor units out of range: %d", minorUnits))
	}
	total := units*100 + minorUnits
	if units < 0 {
		total = units*100 - minorUnits
	}
	return MonetaryAmount{minorUnits: total, currency: currency}
}

// String formats the amount as "-123.45 PLN". The value receiver makes both
// MonetaryAmount and *MonetaryAmount satisfy fmt.Stringer.
func (ma MonetaryAmount) String() string {
	sign := ""
	value := ma.minorUnits
	if value < 0 {
		sign = "-"
		value = -value
	}
	return fmt.Sprintf("%s%d.%02d %s", sign, value/100, value%100, ma.currency)
}

// Add adds amount to ma in place. It returns ErrCurrencyMismatch, and leaves ma
// untouched, when the currencies differ.
func (ma *MonetaryAmount) Add(amount MonetaryAmount) error {
	if err := ma.checkCurrency(amount); err != nil {
		return err
	}
	ma.minorUnits += amount.minorUnits
	return nil
}

// Subtract subtracts amount from ma in place. It returns ErrCurrencyMismatch,
// and leaves ma untouched, when the currencies differ.
func (ma *MonetaryAmount) Subtract(amount MonetaryAmount) error {
	if err := ma.checkCurrency(amount); err != nil {
		return err
	}
	ma.minorUnits -= amount.minorUnits
	return nil
}

// checkCurrency is the one rule both operations share.
func (ma *MonetaryAmount) checkCurrency(other MonetaryAmount) error {
	if ma.currency != other.currency {
		return fmt.Errorf("%w: %s vs %s", ErrCurrencyMismatch, ma.currency, other.currency)
	}
	return nil
}

// MonetaryAmountDemo shows integer-based money arithmetic and the error
// returned when currencies are mixed.
func MonetaryAmountDemo() {
	amount := NewMonetaryAmount(0, 10, "PLN")      // 0.10 PLN
	otherAmount := NewMonetaryAmount(0, 20, "PLN") // 0.20 PLN

	// On float64 this sum would come out as 0.30000000000000004.
	if err := amount.Add(otherAmount); err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Println(amount) // 0.30 PLN

	// Mixing currencies returns an error rather than panicking. Check err != nil
	// first, then classify - `if errors.Is(err, X)` on its own would silently
	// swallow every other kind of error.
	err := amount.Add(NewMonetaryAmount(1, 0, "EUR"))
	switch {
	case err == nil:
		fmt.Println("unexpected success:", amount)
	case errors.Is(err, ErrCurrencyMismatch):
		fmt.Println("cannot add EUR to PLN:", err)
	default:
		fmt.Println("unexpected error:", err)
	}
}
