package budget

import (
	"fmt"
	"time"
)

type BudgetEntry struct {
	Amount        float64		`json:"amount"`
	OperationType OperationType `json:"operationType"`
	Timestamp     time.Time		`json:"timestamp"`
	Description	  string		`json:"description"`
}

type OperationType = int

const (
	Deposit OperationType = iota
	Withdraw
)

var operationName = map[OperationType]string{
	Deposit:  "Deposit",
	Withdraw: "Withdraw",
}

func (e *BudgetEntry) Print() {
	date := fmt.Sprintf("%v", e.Timestamp.Format(time.DateOnly))
	fmt.Printf("%v %10.2f %-12v %-20v\n", date, e.Amount, operationName[e.OperationType], e.Description)
}

func NewBudgetEntry(amount float64, operationType OperationType, description string) *BudgetEntry {
	return &BudgetEntry{amount, operationType, time.Now(), description}
}