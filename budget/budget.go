/*
Napisz aplikację do rejestrowania wpływów/wydatków na potrzeby budżetu domowego.
Aplikacja powinna rejestrować kwotę, rodzaj operacji, timestamp i jej opis (podane jako argumenty wiersza poleceń)
oraz generować raport/tabelę w terminalu. Raport powinien zawierać wszystkie operacje i podsumowanie/saldo końcowe.
Aplikacja powinna zapisywać dane wprowadzone przez użytkownika w pliku tekstowym (json).
*/

package budget

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"strconv"
)

type HomeBudget struct {
	Entries []Entry `json:"entries"`
}

func (b *HomeBudget) AddEntry(entry *Entry) {
	b.Entries = append(b.Entries, *entry)
}

func (b *HomeBudget) PrintSummary() {
	totalBalance := 0.0
	for _, entry := range b.Entries {
		if entry.OperationType == DepositOperation {
			totalBalance += entry.Amount
		} else {
			totalBalance -= entry.Amount
		}
		entry.PrintSummary()
	}
	fmt.Println("-----------------------------------------------")
	fmt.Printf("Total balance: %.2f\n", totalBalance)
}

const BUDGET_FILE string = "budget.json"

func load() (homeBudget *HomeBudget) {
	homeBudget = &HomeBudget{}
	bytesRead, _ := os.ReadFile(BUDGET_FILE)
	json.Unmarshal(bytesRead, &homeBudget)
	return
}

func save(homeBudget *HomeBudget) {
	bytes, _ := json.MarshalIndent(homeBudget, "", "  ")
	os.WriteFile(BUDGET_FILE, bytes, 0644)
}

func entryFromArgs(args []string) (*Entry, error) {
	amount, err := strconv.ParseFloat(args[0], 64)
	if err != nil || amount == 0 {
		return nil, errors.New("invalid amount")
	}
	operationType := DepositOperation
	if amount < 0 {
		amount = math.Abs(amount)
		operationType = WithdrawOperation
	}
	description := args[1]
	return NewEntry(amount, OperationType(operationType), description), nil
}

func processArgs(homeBudget *HomeBudget) {
	args := os.Args[1:]
	if (len(args) >= 2) {
		entry, err := entryFromArgs(args)
		if err == nil {
			homeBudget.AddEntry(entry)
			save(homeBudget)
		}
	}

}

func Budget() {
	homeBudget := load()
	processArgs(homeBudget)
	homeBudget.PrintSummary()
}
