package budget

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const DefaultPath = "budget.json"

type Budget struct {
	Entries []Entry `json:"entries"`
	path    string
}

func NewBudget(path string) *Budget {
	if path == "" {
		path = DefaultPath
	}
	return &Budget{path: path}
}

func (b *Budget) Append(entry *Entry) {
	b.Entries = append(b.Entries, *entry)
}

func (b *Budget) Balance() int {
	var balance int
	for _, entry := range b.Entries {
		switch entry.OperationType {
		case Deposit:
			balance += entry.Amount
		case Withdraw:
			balance -= entry.Amount
		default:
			fmt.Println("\n--- unknown operation type:", entry.OperationType)
		}
	}
	return balance
}

func (b *Budget) PrintSummary() {
	for _, entry := range b.Entries {
		entry.Print()
	}
	fmt.Println("------------------------------------------")
	fmt.Println("Balance:", FormatAmount(b.Balance()))
}

func (b *Budget) Save() error {
	data, err := json.MarshalIndent(b, "", "\t")
	if err != nil {
		return fmt.Errorf("could not serialize budget: %w", err)
	}
	dir := filepath.Dir(b.path)
	tmp, err := os.CreateTemp(dir, filepath.Base(b.path)+"budget.tmp")
	if err != nil {
		return fmt.Errorf("could not create temporary file: %w", err)
	}
	defer func(name string) {
		_ = os.Remove(name)
	}(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("could not write to temporary file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("could not close temporary file: %w", err)
	}
	if err := os.Rename(tmp.Name(), b.path); err != nil {
		return fmt.Errorf("could not rename temporary file: %w", err)
	}
	return nil
}

func (b *Budget) Load() error {
	data, err := os.ReadFile(b.path)
	if err != nil {
		return fmt.Errorf("could not read budget: %w", err)
	}
	if err := json.Unmarshal(data, b); err != nil {
		return fmt.Errorf("could not unmarshal budget: %w", err)
	}
	return nil
}
