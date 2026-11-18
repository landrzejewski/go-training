package solutions

import "fmt"

func ExampleRemoveAt() {
	nums := []int{10, 20, 30, 40}
	fmt.Println(RemoveAt(nums, 1))
	fmt.Println(RemoveAt(nums, 9))
	fmt.Println(nums) // the input is untouched
	// Output:
	// [10 30 40]
	// [10 20 30 40]
	// [10 20 30 40]
}

func ExampleWordCount() {
	fmt.Println(WordCount("The cat and the hat"))
	// Output: map[and:1 cat:1 hat:1 the:2]
}

func ExampleGroupByGrade() {
	students := []Student{{"Ann", 4}, {"Bob", 5}, {"Cid", 3}, {"Eve", 4}}
	fmt.Println(GroupByGrade(students)) // fmt prints map keys in sorted order
	// Output: map[3:[Cid] 4:[Ann Eve] 5:[Bob]]
}
