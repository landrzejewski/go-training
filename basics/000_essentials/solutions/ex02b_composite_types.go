package solutions

import (
	"slices"
	"strings"
)

// ---- Exercise 2b.1: slices share memory --------------------------------------------------------

// RemoveAt returns a new slice without the element at index i and never modifies nums. For
// an index out of range it returns an unchanged copy.
func RemoveAt(nums []int, i int) []int {
	if i < 0 || i >= len(nums) {
		return slices.Clone(nums)
	}
	// The tempting one-liner append(nums[:i], nums[i+1:]...) writes into nums' own array
	// and so changes the caller's slice. Build the result in fresh memory instead.
	out := make([]int, 0, len(nums)-1)
	out = append(out, nums[:i]...)
	return append(out, nums[i+1:]...)
}

// ---- Exercise 2b.2: counting with a map --------------------------------------------------------

// WordCount counts how often each word appears in text. Words are separated by whitespace
// and compared case-insensitively. Punctuation counts as part of a word.
func WordCount(text string) map[string]int {
	counts := make(map[string]int)
	for _, w := range strings.Fields(strings.ToLower(text)) {
		counts[w]++ // a missing key reads as 0, so no "does it exist?" check is needed
	}
	return counts
}

// ---- Exercise 2b.3: structs, slices and maps together ------------------------------------------

// Student is a name with a grade.
type Student struct {
	Name  string
	Grade int
}

// GroupByGrade returns, for every grade, the names of the students who got it, in input order.
func GroupByGrade(students []Student) map[int][]string {
	groups := make(map[int][]string)
	for _, s := range students {
		groups[s.Grade] = append(groups[s.Grade], s.Name) // appending to a nil slice works
	}
	return groups
}
