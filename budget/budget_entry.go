package budget

import (
	"fmt"
	"time"
)

type Entry struct {
	Amount float64
	OperationType OperationType
	Timestamp time.Time
	Description string
}

var operationName = map[OperationType]string{
	DepositOperation: "DEPOSIT",
	WithdrawOperation: "WITHDRAW",
}

func (e *Entry) PrintSummary() {
	date := fmt.Sprintf("%v", e.Timestamp.Format(time.DateOnly))
	fmt.Printf("%v %10.2f %-10v %-20v\n", date, e.Amount, operationName[e.OperationType], e.Description)
}

func NewEntry(amount float64, operationType OperationType, description string) *Entry {
	return &Entry{amount, operationType, time.Now(), description}
}

type OperationType int

const (
	DepositOperation = iota
	WithdrawOperation
)
