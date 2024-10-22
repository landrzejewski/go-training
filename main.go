package main

import "training.pl/examples/concurrency"

func main() {
	/*
		budget := exercises.Budget()
		var newEntry = exercises.NewBudgetEntry(100.0, exercises.Withdraw, "Cinema")
		budget.Add(newEntry)
		budget.FromArgs()
		budget.PrintSummary()
		budget.Save()
	*/

	concurrency.Concurrency()
	concurrency.ProducerConsumerClassic()
}
