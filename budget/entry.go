package budget

import (
	"fmt"
	"time"
)

type Entry struct {
	Amount        int           `json:"amount"`
	OperationType OperationType `json:"type"`
	Timestamp     time.Time     `json:"timestamp"`
	Description   string        `json:"description"`
}

type OperationType int

const (
	Deposit OperationType = iota
	Withdraw
)

var operationNames = map[OperationType]string{
	Deposit:  "deposit",
	Withdraw: "withdraw",
}

func (o OperationType) String() string {
	name, exists := operationNames[o]
	if exists {
		return name
	} else {
		return "unknown"
	}
}

func (e *Entry) Print() {
	date := e.Timestamp.Format(time.DateOnly)
	fmt.Printf("%s %10s %-12s %-15s\n", date, FormatAmount(e.Amount), e.OperationType, e.Description)
}

func FormatAmount(value int) string {
	sign := ""
	if value < 0 {
		sign = "-"
		value = -value
	}
	return fmt.Sprintf("%s%d.%02d", sign, value/100, value%100)
}

func NewEntry(amount int, operationType OperationType, description string) *Entry {
	return &Entry{amount, operationType, time.Now(), description}
}
