/*
Napisz aplikację do rejestrowania wpływów/wydatków na potrzeby budżetu domowego.
Aplikacja powinna rejestrować kwotę, rodzaj operacji, timestamp i jej opis (podane jako argumenty wiersza poleceń)
oraz generować raport/tabelę w terminalu. Raport powinien zawierać wszystkie operacje i podsumowanie/saldo końcowe.
Aplikacja powinna zapisywać dane wprowadzone przez użytkownika w pliku tekstowym (json).
*/

package exercises

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strconv"
)

const budgetFile = "budget.json"

type HomeBudget struct {
	BudgetEntries []BudgetEntry `json:"entries"`
}

func Load() (budget *HomeBudget) {
	budget = &HomeBudget{}
	bytes, err := os.ReadFile(budgetFile)
	if err != nil {
		panic("Loading budget failed")
	}
	err = json.Unmarshal(bytes, budget)
	if err != nil {
		panic("Unmarshalling budget failed")
	}
	return
}

func (b *HomeBudget) Save() {
	bytes, err := json.MarshalIndent(b, "", "\t")
	if err != nil {
		panic("Marshalling budget failed")
	}
	err = os.WriteFile(budgetFile, bytes, 0644)
	if err != nil {
		panic("Saving budget failed")
	}
}

func (b *HomeBudget) PrintSummary() {
	totalBalance := 0.0
	for _, entry := range b.BudgetEntries {
		if entry.OperationType == Deposit {
			totalBalance += entry.Amount
		} else {
			totalBalance -= entry.Amount
		}
		entry.PrintSummary()
	}
	fmt.Printf("--------------------------------------------------\n")
	fmt.Printf("Total Balance: %.2f\n", totalBalance)
}

func (b *HomeBudget) Add(entry *BudgetEntry) {
	b.BudgetEntries = append(b.BudgetEntries, *entry)
}

func entryFromArgs(args []string) (*BudgetEntry, error) {
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

func (b *HomeBudget) FromArgs() {
	args := os.Args[1:]
	if len(args) == 2 {
		entry, err := entryFromArgs(args)
		if err != nil {
			panic("Invalid budget entry")
		}
		b.Add(entry)
	}
}

func Budget() (budget *HomeBudget) {
	budget = Load()
	return
}
