package budget

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strconv"
)

const budgetFile = "budget.json"

type Budget struct {
	Entries []BudgetEntry `json:"entries"`
}

func (b *Budget) Add(entry *BudgetEntry) {
	b.Entries = append(b.Entries, *entry)
}

func (b *Budget) Print() {
	balance := 0.0
	for _, entry := range b.Entries {
		if entry.OperationType == Deposit {
			balance += entry.Amount
		} else {
			balance -= entry.Amount
		}
		entry.Print()
	}
	fmt.Println("----------------------------------------------------")
	fmt.Printf("Balance: %.2f\n", balance)
}

func (b *Budget) Save() {
	bytes, err := json.MarshalIndent(b, "", "\t")
	if err != nil {
		panic("Failed to marshall budget")
	}
	if os.WriteFile(budgetFile, bytes, 0644) != nil {
		panic("Failed to save budget")
	}
}

func (b *Budget) Load() (budget *Budget) {
	budget = &Budget{}
	bytes, err := os.ReadFile(budgetFile)
	if err != nil {
		panic("Failed to load budget")
	}
	if json.Unmarshal(bytes, budget) != nil {
		panic("Failed to unmarshall budget")
	}
	return
}

func (b *Budget) EntryFromArgs() {
	args := os.Args[1:]
	if len(args) == 2 {
		entry, err := parseArgs(args)
		if err == nil {
			b.Add(entry)
		}
	}
}

func parseArgs(args []string) (*BudgetEntry, error) {
	amount, err := strconv.ParseFloat(args[0], 64)
	if err != nil {
		return nil, err
	}
	operationType := Deposit
	if amount < 0 {
		operationType = Withdraw
		amount = math.Abs(amount)
	}
	description := args[1]
	return NewBudgetEntry(amount, operationType, description), nil
}