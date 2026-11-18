package solutions

import (
	"errors"
	"fmt"
	"strconv"
)

// ---- Exercise 9.1: wrapping and a sentinel -----------------------------------------------------

// ErrAgeOutOfRange is returned (wrapped) by ParseAge for numbers outside 0-150.
var ErrAgeOutOfRange = errors.New("age out of range")

// ParseAge parses s as an age between 0 and 150.
func ParseAge(s string) (int, error) {
	age, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("parse age %q: %w", s, err) // %w keeps the strconv error inside
	}
	if age < 0 || age > 150 {
		return 0, fmt.Errorf("parse age %d: %w", age, ErrAgeOutOfRange)
	}
	return age, nil
}

// ---- Exercise 9.2: a custom error type ---------------------------------------------------------

// InsufficientFundsError carries the numbers a caller needs to react to the failure.
type InsufficientFundsError struct {
	Balance, Amount float64
}

func (e *InsufficientFundsError) Error() string {
	return fmt.Sprintf("insufficient funds: balance %.2f, requested %.2f", e.Balance, e.Amount)
}

// Missing returns how much money is lacking.
func (e *InsufficientFundsError) Missing() float64 { return e.Amount - e.Balance }

// Withdraw returns the balance after taking amount out of it. A non-positive amount is a
// plain error; an amount larger than the balance is a wrapped *InsufficientFundsError.
func Withdraw(balance, amount float64) (float64, error) {
	if amount <= 0 {
		return 0, errors.New("withdraw: amount must be positive")
	}
	if amount > balance {
		return 0, fmt.Errorf("withdraw: %w", &InsufficientFundsError{Balance: balance, Amount: amount})
	}
	return balance - amount, nil
}

// ---- Exercise 9.3: errors through two layers ---------------------------------------------------

// ErrMissingSetting is returned (wrapped) by ReadSetting when the key does not exist.
var ErrMissingSetting = errors.New("missing setting")

// ReadSetting returns settings[key] converted to an int.
func ReadSetting(settings map[string]string, key string) (int, error) {
	raw, ok := settings[key]
	if !ok {
		return 0, fmt.Errorf("setting %q: %w", key, ErrMissingSetting)
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("setting %q: %w", key, err)
	}
	return n, nil
}

// Port reads the "port" setting and checks that it is between 1 and 65535. Every layer adds
// its own context with %w, so errors.Is still finds ErrMissingSetting two levels down.
func Port(settings map[string]string) (int, error) {
	p, err := ReadSetting(settings, "port")
	if err != nil {
		return 0, fmt.Errorf("config: %w", err)
	}
	if p < 1 || p > 65535 {
		return 0, fmt.Errorf("config: port %d out of range 1-65535", p)
	}
	return p, nil
}
