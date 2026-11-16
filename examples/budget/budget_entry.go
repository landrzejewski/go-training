package budget

import (
	"fmt"
	"time"
)

// Entry is one deposit or withdrawal recorded in the budget.
type Entry struct {
	// The amount in GROSZE (minor units) as a non-negative integer. float64
	// cannot represent 0.01/0.10/0.20 exactly, so summing the ledger would
	// accumulate error and %.2f formatting would hide it.
	AmountMinor   int64         `json:"amountMinor"`
	OperationType OperationType `json:"operationType"`
	Timestamp     time.Time     `json:"timestamp"`
	Description   string        `json:"description"`
}

// OperationType tells a deposit from a withdrawal.
//
// A DEFINED type, not an alias. The previous `type OperationType = int` made
// OperationType and int the same type: any int passed as an operation kind, and
// there was no way to attach a String() method.
type OperationType int

// The two kinds of operation.
const (
	Deposit OperationType = iota
	Withdraw
)

var operationName = map[OperationType]string{
	Deposit:  "Deposit",
	Withdraw: "Withdraw",
}

func (o OperationType) String() string {
	if name, exists := operationName[o]; exists {
		return name
	}
	return fmt.Sprintf("OperationType(%d)", int(o))
}

// FormatMinor renders minor units as an amount with two decimal places.
func FormatMinor(minor int64) string {
	sign := ""
	if minor < 0 {
		sign = "-"
		minor = -minor
	}
	return fmt.Sprintf("%s%d.%02d", sign, minor/100, minor%100)
}

// Print writes the entry as one row of the report.
func (e Entry) Print() {
	date := e.Timestamp.Format(time.DateOnly)
	fmt.Printf("%v %10v %-12v %-20v\n", date, FormatMinor(e.AmountMinor), e.OperationType, e.Description)
}

// NewEntry creates an entry stamped with the current time.
func NewEntry(amountMinor int64, operationType OperationType, description string) Entry {
	return Entry{AmountMinor: amountMinor, OperationType: operationType, Timestamp: time.Now(), Description: description}
}
