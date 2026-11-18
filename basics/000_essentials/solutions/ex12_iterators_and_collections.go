package solutions

import (
	"cmp"
	"iter"
	"maps"
	"slices"
)

// ---- Exercise 12.1: an iterator ----------------------------------------------------------------

// Range yields start, start+step, start+2*step, ... while the value is below end (or above
// end for a negative step), like range() in Python. A step of 0 yields nothing.
func Range(start, end, step int) iter.Seq[int] {
	return func(yield func(int) bool) {
		if step == 0 {
			return // it would never reach end
		}
		for i := start; (step > 0 && i < end) || (step < 0 && i > end); i += step {
			if !yield(i) {
				return // the loop was left with break: stop producing values
			}
		}
	}
}

// ---- Exercise 12.2: sorting by two keys --------------------------------------------------------

// Person is a name with an age.
type Person struct {
	Name string
	Age  int
}

// SortPeople sorts people in place by age, youngest first. People of the same age are sorted
// by name.
func SortPeople(people []Person) {
	slices.SortFunc(people, func(a, b Person) int {
		if c := cmp.Compare(a.Age, b.Age); c != 0 {
			return c // the ages differ: they decide
		}
		return cmp.Compare(a.Name, b.Name) // same age: fall back to the name
	})
	// Shorter, with the same meaning: return cmp.Or(cmp.Compare(a.Age, b.Age), cmp.Compare(a.Name, b.Name))
}

// ---- Exercise 12.3: maps and sorting together --------------------------------------------------

// TopWords returns the n most frequent words in counts, most frequent first; words with the
// same count are in alphabetical order. If n is larger than the number of words, it returns
// all of them.
func TopWords(counts map[string]int, n int) []string {
	words := slices.Collect(maps.Keys(counts)) // maps.Keys is an iterator; Collect makes a slice
	slices.SortFunc(words, func(a, b string) int {
		if c := cmp.Compare(counts[b], counts[a]); c != 0 { // b before a: descending
			return c
		}
		return cmp.Compare(a, b)
	})
	return words[:min(max(n, 0), len(words))] // never slice past the end (or below 0)
}
