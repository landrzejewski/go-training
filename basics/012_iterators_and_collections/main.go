// Command 012_iterators_and_collections is module 012 of the Go course: range-over-function
// iterators and the slices, maps and cmp packages.
//
// slices, maps and cmp arrived in Go 1.21, range over functions and iter in 1.23. Together they
// replaced most hand-written loops and the whole of the old sort API - if you learned Go before
// 1.21, this module has the most that is new to you.
//
//	go run ./basics/012_iterators_and_collections
package main

import (
	"cmp"
	"fmt"
	"iter"
	"maps"
	"slices"
	"strings"
	"unique"
)

func main() {
	rangeOverFunc() // iter.Seq/Seq2, yield, early break, cleanup, a tree walk
	iterPull()      // turning a push iterator into a pull one
	slicesPackage() // search, sort, modify, compare, chunk, iterator bridges
	mapsPackage()   // Keys/Values are iterators, Clone is shallow, DeleteFunc, Collect
	cmpPackage()    // Compare and NaN, Or for defaults and multi-key sorts, min/max
	pipelines()     // lazy adapters over infinite sources, unique.Make
}

// rangeOverFunc shows Go 1.23 iterators: functions that a for-range loop can drive.
func rangeOverFunc() {
	fmt.Println("--- range over a function ---")

	// for x := range f works when f has one of exactly three shapes:
	//	func(yield func() bool)       for range f
	//	func(yield func(V) bool)      for v := range f      = iter.Seq[V]
	//	func(yield func(K, V) bool)   for k, v := range f   = iter.Seq2[K, V]
	// iter.Seq and Seq2 are plain generic type definitions (not aliases) - no magic, no interface.
	// range calls f with a yield closure holding the loop body; f calls yield once per element.
	fmt.Print("iter.Seq[int]: ")
	for v := range count(5) {
		fmt.Print(v, " ")
	}
	fmt.Println()
	//	for i, v := range count(5) {} // ERROR: range over count(5) (value of func type iter.Seq[int]) permits only one iteration variable

	fmt.Print("iter.Seq2[string, int]: ")
	for k, v := range pairs([]string{"a", "b", "c"}, []int{1, 2, 3}) {
		fmt.Printf("%s=%d ", k, v)
	}
	fmt.Println()

	// break, return, goto (or a labelled continue out of the loop) make yield return false - the
	// iterator MUST stop then. Ignoring the result and yielding again is the one real hazard:
	//	func(yield func(int) bool) { for i := range n { yield(i) } } // then break out of it:
	//	// panic: runtime error: range function continued iteration after function for loop body returned false
	// So always write: if !yield(v) { return }.
	fmt.Print("breaking out at 3: ")
	for v := range count(100) {
		if v == 3 {
			break
		}
		fmt.Print(v, " ")
	}
	fmt.Println("<- the iterator returned; it did not run 100 times")

	// Iterators are lazy and need not walk a collection at all: lines of a file, a DB cursor,
	// a tree walk, an infinite sequence.
	fmt.Print("words, lazily: ")
	for w := range words("  range over a function  ") {
		fmt.Printf("%q ", w)
	}
	fmt.Println()

	// An iterator holding a resource releases it with defer - which runs whether the loop
	// completes or the consumer breaks out early.
	for v := range withCleanup(10) {
		if v == 2 {
			break
		}
	}

	// A tree walk: before 1.23 this needed a callback, an explicit stack or a goroutine+channel.
	t := &tree{Value: 5,
		Left:  &tree{Value: 3, Left: &tree{Value: 1}},
		Right: &tree{Value: 8, Left: &tree{Value: 7}}}
	fmt.Print("in-order tree walk: ")
	for v := range t.inOrder() {
		fmt.Print(v, " ")
	}
	fmt.Println()

	// Standard-library naming: All yields index/key-value pairs, Values just values, Backward
	// reverses, Sorted collects and sorts, Collect materialises.
}

// count is the simplest possible iterator.
func count(n int) iter.Seq[int] {
	return func(yield func(int) bool) {
		for i := range n {
			if !yield(i) {
				return // the consumer broke out: stop immediately
			}
		}
	}
}

func pairs[K comparable, V any](keys []K, values []V) iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		for i := range min(len(keys), len(values)) {
			if !yield(keys[i], values[i]) {
				return
			}
		}
	}
}

// words yields the space-separated words of s without building a slice.
func words(s string) iter.Seq[string] {
	return func(yield func(string) bool) {
		for {
			s = strings.TrimLeft(s, " ")
			if s == "" {
				return
			}
			word, rest, _ := strings.Cut(s, " ")
			if !yield(word) {
				return
			}
			s = rest
		}
	}
}

func withCleanup(n int) iter.Seq[int] {
	return func(yield func(int) bool) {
		fmt.Println("(resource acquired)")
		defer fmt.Println("(resource released - even on an early break)")
		for i := range n {
			if !yield(i) {
				return
			}
		}
	}
}

type tree struct {
	Value       int
	Left, Right *tree
}

func (t *tree) inOrder() iter.Seq[int] {
	return func(yield func(int) bool) {
		var walk func(*tree) bool
		walk = func(n *tree) bool {
			if n == nil {
				return true
			}
			return walk(n.Left) && yield(n.Value) && walk(n.Right) // false stops the whole walk
		}
		walk(t)
	}
}

// iterPull shows iter.Pull, which turns a push iterator (it calls you) into a pull one (you ask).
func iterPull() {
	fmt.Println("\n--- iter.Pull ---")

	// Pull returns next() (V, bool) and stop(); Pull2 is the two-value version. You MUST call
	// stop - defer it - or the iterator's coroutine state is never released. stop twice is
	// safe; next after stop returns the zero value and false.
	next, stop := iter.Pull(count(5))
	defer stop()
	a, _ := next()
	b, _ := next()
	c, ok := next()
	fmt.Printf("pulled on demand: %d %d, then %d ok=%t\n", a, b, c, ok)

	// The classic reason to need pull: merging two sorted sequences needs look-ahead on both.
	left := slices.Values([]int{1, 4, 9})
	right := slices.Values([]int{2, 3, 10})
	fmt.Println("merging two sorted sequences:", slices.Collect(mergeSorted(left, right)))

	next2, stop2 := iter.Pull(count(10))
	stop2()
	_, ok2 := next2()
	fmt.Printf("after stop(), next() reports ok=%t\n", ok2)

	// Pull uses runtime coroutines, far cheaper than the old goroutine+channel trick, but still
	// much more expensive than a plain range - use it only when you must control the pace.
}

func mergeSorted[T cmp.Ordered](a, b iter.Seq[T]) iter.Seq[T] {
	return func(yield func(T) bool) {
		nextA, stopA := iter.Pull(a)
		defer stopA()
		nextB, stopB := iter.Pull(b)
		defer stopB()

		va, okA := nextA()
		vb, okB := nextB()
		for okA || okB {
			if okA && (!okB || va <= vb) {
				if !yield(va) {
					return
				}
				va, okA = nextA()
			} else {
				if !yield(vb) {
					return
				}
				vb, okB = nextB()
			}
		}
	}
}

// slicesPackage shows the slices package (Go 1.21, iterator bridges in 1.23).
func slicesPackage() {
	fmt.Println("\n--- the slices package ---")

	// It replaces almost every hand-written slice loop:
	//	searching  Contains, ContainsFunc, Index, IndexFunc, BinarySearch, BinarySearchFunc
	//	ordering   Sort, SortFunc, SortStableFunc, IsSorted, Min, Max, MinFunc, MaxFunc
	//	comparing  Equal, EqualFunc, Compare, CompareFunc
	//	modifying  Insert, Delete, DeleteFunc, Replace, Reverse, Compact, CompactFunc, Clip, Grow
	//	copying    Clone, Concat, Repeat (1.23)
	//	iterators  All, Values, Backward, Collect, Sorted, SortedFunc, AppendSeq, Chunk (1.23)
	s := []int{5, 2, 8, 2, 1}
	fmt.Printf("Contains(8)=%t Index(2)=%d IndexFunc(>4)=%d\n",
		slices.Contains(s, 8), slices.Index(s, 2), slices.IndexFunc(s, func(n int) bool { return n > 4 }))

	// slices.Sort (pdqsort) is faster and clearer than sort.Slice, which needed a closure and
	// reflection - there is no reason left to use sort.Slice.
	sorted := slices.Clone(s)
	slices.Sort(sorted)
	fmt.Printf("Sort: %v Min=%d Max=%d IsSorted=%t\n", sorted, slices.Min(s), slices.Max(s), slices.IsSorted(sorted))

	// SortFunc takes a THREE-WAY comparison (negative/zero/positive), not a less bool - pair it
	// with cmp.Compare. It is not stable; SortStableFunc keeps equal elements in input order.
	// "pear" and "kiwi" have equal length: here they happen to stay in input order...
	fruit := []string{"banana", "fig", "apple", "pear", "kiwi"}
	byLength := slices.Clone(fruit)
	slices.SortFunc(byLength, func(a, b string) int { return cmp.Compare(len(a), len(b)) })
	fmt.Println("SortFunc by length:", byLength)
	// ...while a second key decides it explicitly (cmp.Or: the first non-zero comparison).
	multi := slices.Clone(fruit)
	slices.SortFunc(multi, func(a, b string) int {
		return cmp.Or(cmp.Compare(len(a), len(b)), cmp.Compare(a, b))
	})
	fmt.Println("SortFunc by length, then alphabetically:", multi)

	// Delete, Insert, Replace, Compact modify in place and RETURN the new slice - use the
	// result. Since Go 1.22 Delete also zeroes the vacated tail, fixing the pointer leak of the
	// manual append(s[:i], s[i+1:]...) idiom.
	fmt.Println("Sorted+Compact (dedup):", slices.Compact(slices.Sorted(slices.Values(s))))
	fmt.Println("Delete(1, 3): ", slices.Delete(slices.Clone(s), 1, 3))
	fmt.Println("Insert(2, 99):", slices.Insert(slices.Clone(s), 2, 99))
	fmt.Println("DeleteFunc(even):", slices.DeleteFunc(slices.Clone(s), func(n int) bool { return n%2 == 0 }))
	reversed := slices.Clone(s)
	slices.Reverse(reversed)
	fmt.Println("Reverse:", reversed, "Concat:", slices.Concat([]int{0}, s), "Repeat:", slices.Repeat([]int{1, 2}, 3))

	// Clip trims capacity to length, releasing the tail of a large backing array.
	big := make([]int, 3, 1000)
	fmt.Printf("Clip: cap %d -> %d\n", cap(big), cap(slices.Clip(big)))

	// Equal is far faster than reflect.DeepEqual - and, unlike it, treats nil and empty as equal.
	fmt.Println("Equal(nil, []int{}) =", slices.Equal([]int(nil), []int{}))
	fmt.Println("Compare([1 2], [1 3]) =", slices.Compare([]int{1, 2}, []int{1, 3}))

	fmt.Print("Chunk(s, 2): ")
	for chunk := range slices.Chunk(s, 2) {
		fmt.Print(chunk, " ")
	}
	fmt.Print("\nAll: ")
	for i, v := range slices.All(s) {
		fmt.Printf("%d:%d ", i, v)
	}
	fmt.Print("\nBackward: ")
	for _, v := range slices.Backward(s) {
		fmt.Print(v, " ")
	}
	fmt.Println()
}

// mapsPackage shows the maps package (Go 1.21, iterator functions in 1.23).
func mapsPackage() {
	fmt.Println("\n--- the maps package ---")

	// Keys(m), Values(m) -> iter.Seq; All(m) -> iter.Seq2; Collect(seq), Insert(m, seq);
	// Clone, Copy(dst, src), Equal, EqualFunc, DeleteFunc.
	m := map[string]int{"banana": 3, "apple": 5, "cherry": 1}

	// Keys and Values return ITERATORS, not slices (the old x/exp/maps returned slices). Map
	// order is random, so slices.Sorted is usually the right way to materialise them.
	fmt.Println("slices.Sorted(maps.Keys):  ", slices.Sorted(maps.Keys(m)))
	fmt.Println("slices.Sorted(maps.Values):", slices.Sorted(maps.Values(m)))

	// SortedFunc takes an iter.Seq, not a Seq2 - so sort the keys with a custom comparison.
	byLen := slices.SortedFunc(maps.Keys(m), func(a, b string) int {
		return cmp.Or(cmp.Compare(len(a), len(b)), cmp.Compare(a, b))
	})
	fmt.Println("SortedFunc over maps.Keys, by length:", byLen)

	fmt.Print("maps.All (random order): ")
	for k, v := range maps.All(m) {
		fmt.Printf("%s=%d ", k, v)
	}
	fmt.Print("\nsorted key order:        ")
	for _, k := range slices.Sorted(maps.Keys(m)) {
		fmt.Printf("%s=%d ", k, m[k])
	}
	fmt.Println()

	clone := maps.Clone(m)
	fmt.Println("Clone then Equal:", maps.Equal(m, clone))
	clone["durian"] = 9
	fmt.Printf("after adding to the clone: Equal=%t (len %d vs %d)\n", maps.Equal(m, clone), len(m), len(clone))

	// Clone is SHALLOW: a slice or map value is shared with the original (module 006).
	nested := map[string][]int{"a": {1, 2}}
	shallow := maps.Clone(nested)
	shallow["a"][0] = 99
	fmt.Println("Clone is shallow - the original changed too:", nested)

	// Deleting during a range is allowed and defined (a deleted entry not yet reached is not
	// produced); an entry ADDED during a range may or may not appear. So DeleteFunc is safe.
	scores := maps.Clone(m)
	maps.DeleteFunc(scores, func(_ string, v int) bool { return v < 3 })
	fmt.Println("DeleteFunc(v < 3):", slices.Sorted(maps.Keys(scores)))

	dst := map[string]int{"extra": 0}
	maps.Copy(dst, m)
	fmt.Println("Copy into an existing map:", len(dst), "entries")
	fmt.Println("maps.Collect from an iter.Seq2:", maps.Collect(pairs([]string{"x", "y"}, []int{10, 20})))
}

// cmpPackage shows cmp.Compare, cmp.Less, cmp.Or and the min/max builtins.
func cmpPackage() {
	fmt.Println("\n--- the cmp package and ordering ---")

	// cmp.Ordered is the constraint for < (integers, floats, strings). It replaces
	// golang.org/x/exp/constraints.Ordered - use the standard one.
	fmt.Printf("cmp.Compare: (1,2)=%d (2,2)=%d (3,2)=%d  cmp.Less(\"a\",\"b\")=%t\n",
		cmp.Compare(1, 2), cmp.Compare(2, 2), cmp.Compare(3, 2), cmp.Less("a", "b"))

	// cmp.Compare (and cmp.Less) define a TOTAL order with NaN before everything, so sorting
	// floats is well defined even with NaNs - a naive a < b is not.
	nan := 0.0 / zero()
	floats := []float64{3, nan, 1, 2}
	slices.SortFunc(floats, cmp.Compare)
	fmt.Println("sorting with NaN using cmp.Compare:", floats)
	fmt.Printf("naive comparison is inconsistent: NaN < 1 = %t, NaN > 1 = %t\n", nan < 1, nan > 1)

	// cmp.Or (Go 1.22) returns its first non-zero argument: defaults...
	fmt.Printf("cmp.Or(\"\", \"\", \"localhost\") = %q, cmp.Or(0, 8080) = %d\n",
		cmp.Or("", "", "localhost"), cmp.Or(0, 8080))

	// ...and, the neat use, chaining comparisons for a multi-key sort.
	staff := []employee{
		{"Ada", "eng", 120}, {"Alan", "eng", 150}, {"Grace", "ops", 150}, {"Barbara", "eng", 150},
	}
	slices.SortFunc(staff, func(a, b employee) int {
		return cmp.Or(
			cmp.Compare(a.Dept, b.Dept),      // department ascending
			-cmp.Compare(a.Salary, b.Salary), // then salary DESCENDING
			cmp.Compare(a.Name, b.Name),      // then name ascending
		)
	})
	for _, e := range staff {
		fmt.Printf("  %-8s %-4s %d\n", e.Name, e.Dept, e.Salary)
	}

	// min and max are variadic builtins since Go 1.21, for any ordered type - do not write your own.
	fmt.Printf("min(3,1,2)=%d max(3,1,2)=%d min(\"b\",\"a\")=%q\n", min(3, 1, 2), max(3, 1, 2), min("b", "a"))
}

type employee struct {
	Name   string
	Dept   string
	Salary int
}

// zero hides the constant from the compiler: 0.0 / 0.0 written directly does not compile
// ("invalid operation: division by zero").
func zero() float64 { return 0 }

// pipelines shows lazy iterator adapters and unique.Make.
func pipelines() {
	fmt.Println("\n--- iterator pipelines and unique ---")

	// An adapter takes a Seq and returns a Seq. Adapters compose and the chain stays lazy:
	// nothing runs until the consumer asks, and everything stops when it breaks. This is where
	// Map/Filter/Take earn their place - over an iterator they avoid every intermediate slice
	// and work on infinite or expensive sources.
	evens := take(filterSeq(fibonacci(), func(n int) bool { return n%2 == 0 }), 6)
	fmt.Println("first 6 even Fibonacci numbers, from an INFINITE source:", slices.Collect(evens))

	inventory := map[string]int{"apple": 5, "banana": 0, "cherry": 12}
	labels := slices.Sorted(mapSeq2(maps.All(inventory), func(k string, v int) string {
		return fmt.Sprintf("%s(%d)", k, v)
	}))
	fmt.Println("iter.Seq2 -> iter.Seq -> sorted:", labels)

	calls := 0
	counted := func(yield func(int) bool) {
		for i := range 1_000_000 {
			calls++
			if !yield(i) {
				return
			}
		}
	}
	fmt.Print("a chain that stops early does no extra work: ")
	for v := range take(iter.Seq[int](counted), 3) {
		fmt.Print(v, " ")
	}
	fmt.Printf("<- the source produced only %d values, not a million\n", calls)

	// A caution: each element costs a closure call per stage. For a hot loop over a slice
	// already in memory a plain for is faster - use iterators for composition and non-slices.

	// unique.Make (Go 1.23) INTERNS a comparable value: equal values share one allocation and
	// the Handle compares by pointer - O(1) whatever the size. For many repeated strings or
	// structs: symbol tables, parsed identifiers, metric labels. weak.Pointer (Go 1.24), a
	// reference the GC may clear, is its companion for caches.
	a := unique.Make("a very long repeated identifier string")
	b := unique.Make("a very long repeated identifier string")
	c := unique.Make("a different one")
	fmt.Printf("unique.Make: a == b is %t, a == c is %t, a.Value() = %q\n", a == b, a == c, a.Value())
}

// filterSeq is a local copy of module 010's filter adapter.
func filterSeq[T any](seq iter.Seq[T], keep func(T) bool) iter.Seq[T] {
	return func(yield func(T) bool) {
		for v := range seq {
			if keep(v) && !yield(v) {
				return
			}
		}
	}
}

func take[T any](seq iter.Seq[T], n int) iter.Seq[T] {
	return func(yield func(T) bool) {
		if n <= 0 {
			return
		}
		taken := 0
		for v := range seq {
			if !yield(v) {
				return
			}
			if taken++; taken >= n {
				return
			}
		}
	}
}

func mapSeq2[K, V, R any](seq iter.Seq2[K, V], f func(K, V) R) iter.Seq[R] {
	return func(yield func(R) bool) {
		for k, v := range seq {
			if !yield(f(k, v)) {
				return
			}
		}
	}
}

// fibonacci is an infinite source - it only ends when the consumer stops.
func fibonacci() iter.Seq[int] {
	return func(yield func(int) bool) {
		a, b := 0, 1
		for {
			if !yield(a) {
				return
			}
			a, b = b, a+b
		}
	}
}
