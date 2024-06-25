package budget

import "fmt"

type HomeBudget struct {
	entries []Entry
}

func (b *HomeBudget) AddEntry(entry *Entry) {
	b.entries = append(b.entries, *entry)
}

func (b *HomeBudget) PrintSummary() {
	totalBalance := 0.0
	for _, entry := range b.entries {
		if entry.OperationType == DepositOperation {
			totalBalance += entry.Amount
		} else {
			totalBalance -= entry.Amount
		}
		entry.PrintSummary()
	}
	fmt.Println("------------------------------------------------------------------------")
	fmt.Printf("Total balance: %.2f\n", totalBalance)
}