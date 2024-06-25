package budget

import "fmt"

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