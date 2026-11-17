package solutions

import (
	"fmt"
	"time"
)

// duck is declared here, in another file, long after Chorus was written - and Chorus works
// with it unchanged. That is the point of interfaces.
type duck struct{}

func (duck) Sound() string { return "Quack" }

func ExampleChorus() {
	fmt.Println(Chorus([]Speaker{Dog{}, Cat{}, Cow{}, duck{}}))
	// Output: Woof Meow Moo Quack
}

func ExampleGreeting() {
	at := func(hour int) FixedClock {
		return FixedClock{T: time.Date(2026, time.January, 1, hour, 0, 0, 0, time.UTC)}
	}
	// In production code: Greeting(RealClock{}). Its output depends on the time, so it is
	// not part of this example.
	fmt.Println(Greeting(at(9)))
	fmt.Println(Greeting(at(15)))
	fmt.Println(Greeting(at(21)))
	// Output:
	// Good morning
	// Good afternoon
	// Good evening
}

func ExampleTotalSize() {
	items := []any{"abc", []int{1, 2}, 10, File{Name: "a.txt", Bytes: 100}, 3.5, nil}
	fmt.Println(TotalSize(items)) // 3 + 2 + 10 + 100; the float64 and nil count as 0
	// Output: 115
}
