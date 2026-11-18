package solutions

import (
	"fmt"
	"slices"
	"strings"
)

func ExampleLargest() {
	fmt.Println(Largest([]int{3, 9, 2}))
	fmt.Println(Largest([]string{"pear", "apple", "plum"})) // strings compare alphabetically
	fmt.Println(Largest([]float64{}))
	// Output:
	// 9 <nil>
	// plum <nil>
	// 0 largest: empty slice
}

func ExampleGroupBy() {
	fruits := []string{"apple", "avocado", "banana", "blueberry", "cherry"}
	fmt.Println(GroupBy(fruits, func(s string) string { return s[:1] }))

	isEven := func(n int) bool { return n%2 == 0 }
	fmt.Println(GroupBy([]int{1, 2, 3, 4, 5}, isEven))

	// GroupByGrade from exercise 2b.3 is GroupBy with K = int - but it keeps only the names.
	students := []Student{{"Ann", 4}, {"Bob", 5}, {"Eve", 4}}
	fmt.Println(GroupBy(students, func(s Student) int { return s.Grade }))
	// Output:
	// map[a:[apple avocado] b:[banana blueberry] c:[cherry]]
	// map[false:[1 3 5] true:[2 4]]
	// map[4:[{Ann 4} {Eve 4}] 5:[{Bob 5}]]
}

func ExampleSet() {
	a := NewSet(1, 2, 3, 2) // T inferred as int; the duplicate 2 is stored once
	b := NewSet(3, 4)
	fmt.Println(a.Len(), a.Has(2), a.Has(4))

	union := a.Union(b)
	items := union.Items()
	slices.Sort(items) // Items has no order, so sort before printing
	fmt.Println(items)

	var words Set[string] // the zero value works
	for _, w := range strings.Fields("to be or not to be") {
		words.Add(w)
	}
	fmt.Println(words.Len())
	// Output:
	// 3 true false
	// [1 2 3 4]
	// 4
}
