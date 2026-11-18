package solutions

import "fmt"

func ExampleSwapInts() {
	a, b := 1, 2
	SwapInts(&a, &b)
	fmt.Println(a, b)
	// Output: 2 1
}

func ExampleAddBonus() {
	players := []Player{{"Ann", 10}, {"Bob", 7}}
	AddBonus(players, 5)
	fmt.Println(players)
	// Output: [{Ann 15} {Bob 12}]
}

func ExampleFindPlayer() {
	players := []Player{{"Ann", 10}, {"Bob", 7}}
	if p := FindPlayer(players, "Bob"); p != nil {
		p.Score = 100 // changes the element inside the slice
	}
	fmt.Println(players, FindPlayer(players, "Zoe") == nil)
	// Output: [{Ann 10} {Bob 100}] true
}

func ExampleValues() {
	var list *Node // the empty list
	fmt.Println(Values(list))
	for i := range 3 {
		list = Prepend(list, i)
	}
	fmt.Println(Values(list))
	// Output:
	// []
	// [2 1 0]
}
