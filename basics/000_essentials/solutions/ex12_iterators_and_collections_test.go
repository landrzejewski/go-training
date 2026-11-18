package solutions

import (
	"fmt"
	"slices"
)

func ExampleRange() {
	var got []int
	for i := range Range(0, 10, 3) {
		got = append(got, i)
	}
	fmt.Println(got)

	got = nil
	for i := range Range(5, 0, -2) {
		got = append(got, i)
	}
	fmt.Println(got)

	got = nil
	for i := range Range(0, 1_000_000, 1) {
		if i == 4 {
			break // the iterator stops right away; the other numbers are never produced
		}
		got = append(got, i)
	}
	fmt.Println(got)

	fmt.Println(slices.Collect(Range(1, 4, 1)), slices.Collect(Range(1, 4, 0)))
	// Output:
	// [0 3 6 9]
	// [5 3 1]
	// [0 1 2 3]
	// [1 2 3] []
}

func ExampleSortPeople() {
	people := []Person{{"Cid", 31}, {"Bob", 25}, {"Ann", 31}, {"Eve", 19}}
	SortPeople(people)
	fmt.Println(people)
	// Output: [{Eve 19} {Bob 25} {Ann 31} {Cid 31}]
}

func ExampleTopWords() {
	counts := WordCount("the cat and the hat and the bat")
	fmt.Println(TopWords(counts, 2))
	fmt.Println(TopWords(counts, 4)) // bat, cat and hat all appear once: alphabetical
	fmt.Println(TopWords(counts, 100))
	// Output:
	// [the and]
	// [the and bat cat]
	// [the and bat cat hat]
}
