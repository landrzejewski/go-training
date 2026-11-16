// Command 004_control_flow is module 004 of the Go course: if, switch, for, range, labels, goto
// and defer.
//
// Go has exactly three keywords for branching and looping: if, switch and for. There is no while,
// no do-while, no separate for-each and no ternary. Two rules apply everywhere: the condition must
// be a bool, and the opening brace stays on the same line (semicolon insertion, module 001a).
//
//	go run ./basics/004_control_flow
package main

import (
	"errors"
	"fmt"
	"io"
	"iter"
	"os"
)

func main() {
	ifStatements()      // mandatory braces, init statements, early return
	switchStatements()  // expression and tagless switch, fallthrough, break
	forLoops()          // all five forms, fresh loop variables (Go 1.22)
	rangeForms()        // what can be ranged over, copies, iterators (Go 1.23)
	labelsAndGoto()     // labelled break/continue, the fences around goto
	deferStatements()   // LIFO, argument evaluation, named results, loop pitfall
	puttingItTogether() // one realistic function using nearly all of it
}

// ifStatements shows if/else, the init statement and the early-return style.
func ifStatements() {
	fmt.Println("--- if statements ---")

	n := 42
	if n > 0 {
		fmt.Println("positive")
	} else if n < 0 {
		fmt.Println("negative")
	} else {
		fmt.Println("zero")
	}

	// No parentheses, but braces are mandatory even for one statement - which makes the
	// `goto fail` class of bug impossible. `if (n > 0) {}` is legal but unidiomatic.
	//	if n > 0 fmt.Println("x") // ERROR: syntax error: unexpected name fmt, expected {

	// The condition must be a bool.
	//	if n {} // ERROR: non-boolean condition in if statement

	// The init statement: its variables live in the whole if/else chain and vanish afterwards,
	// so err cannot leak into the enclosing scope. The most common idiom in Go.
	if value, err := lookup("go"); err != nil {
		fmt.Println("lookup failed:", err)
	} else {
		fmt.Printf("init statement: value=%q\n", value)
	}
	//	fmt.Println(err) // ERROR: undefined: err
	if _, err := lookup("rust"); err != nil {
		fmt.Println("lookup failed:", err)
	}

	// `} else {` must share one line: a newline after } inserts a semicolon and ends the if.
	//	}
	//	else { // ERROR: syntax error: unexpected keyword else, expected }

	// if is a statement, not an expression - there is no `x := if c {1} else {2}`.

	// Prefer the early return: guards first, the happy path last and unindented.
	fmt.Println("early return: classify(15) =", classify(15), "| classify(-1) =", classify(-1))
}

func lookup(key string) (string, error) {
	data := map[string]string{"go": "a programming language"}
	if v, ok := data[key]; ok {
		return v, nil
	}
	return "", fmt.Errorf("key %q not found", key)
}

// classify reads as a series of guards followed by the real result - no nested else.
func classify(n int) string {
	if n < 0 {
		return "negative"
	}
	if n == 0 {
		return "zero"
	}
	if n%15 == 0 {
		return "fizzbuzz"
	}
	return "positive"
}

// switchStatements shows the expression switch, the tagless switch and their surprises.
func switchStatements() {
	fmt.Println("\n--- switch statements ---")

	// Expression switch: several values per case, and NO fall-through - no break needed.
	// Values need not be constants; cases are tried top to bottom and the first match wins.
	for _, day := range []string{"Saturday", "Monday", "Caturday"} {
		switch day {
		case "Saturday", "Sunday":
			fmt.Printf("  %s: weekend\n", day)
		case "Monday", "Tuesday", "Wednesday", "Thursday", "Friday":
			fmt.Printf("  %s: weekday\n", day)
		default: // may appear anywhere, conventionally last; without it no match does nothing
			fmt.Printf("  %s: not a day\n", day)
		}
	}

	// An init statement works exactly as in if.
	switch n := len("gopher"); n {
	case 6:
		fmt.Println("init statement in a switch: length is", n)
	default:
		fmt.Println("unexpected length", n)
	}

	// Tagless switch (= switch true) replaces an if/else-if chain - idiomatic and often clearer.
	for _, score := range []int{95, 72, 41} {
		var grade string
		switch {
		case score >= 90:
			grade = "A"
		case score >= 80:
			grade = "B"
		case score >= 60:
			grade = "C"
		default:
			grade = "F"
		}
		fmt.Printf("  score %d -> %s\n", score, grade)
	}

	// The type switch `switch v := x.(type)` dispatches on an interface's dynamic type
	// (module 008).

	// fallthrough enters the NEXT body without testing its condition. It must be the last
	// statement of a case, not in the final case nor a type switch. Rare, usually a mistake.
	switch 2 {
	case 1:
		fmt.Println("  one")
	case 2:
		fmt.Println("  two")
		fallthrough
	case 3:
		fmt.Println("  three (reached by fallthrough, although 2 != 3)")
	case 4:
		fmt.Println("  four (NOT reached - fallthrough goes one case deep)")
	}

	// An empty case does NOT fall through - the opposite of C. List the values on one case.
	switch 1 {
	case 1:
		// deliberately empty: the switch simply ends
	case 2:
		fmt.Println("  unreachable")
	}

	// break inside a switch leaves the SWITCH, not the enclosing loop. Use a label
	// (labelsAndGoto) - a real gotcha when coming from other languages.
	for i := range 3 {
		switch i {
		case 1:
			break // leaves the switch only
		}
		fmt.Printf("  iteration %d still runs after the switch's break\n", i)
	}
}

// forLoops shows every shape of Go's only loop keyword.
func forLoops() {
	fmt.Println("\n--- for loops ---")

	// 1. Three-clause, C style. All clauses are optional; no parentheses, braces mandatory.
	fmt.Print("  three-clause: ")
	for i := 0; i < 5; i++ {
		fmt.Print(i, " ")
	}
	fmt.Println()

	// 2. Condition only - Go's "while".
	fmt.Print("  while-style:  ")
	n := 1
	for n < 20 {
		fmt.Print(n, " ")
		n *= 2
	}
	fmt.Println()

	// 3. Infinite, left with break.
	fmt.Print("  infinite+break: ")
	count := 0
	for {
		count++
		if count > 3 {
			break
		}
		fmt.Print(count, " ")
	}
	fmt.Println()

	// 4. range over a collection (rangeForms covers every operand).
	fmt.Print("  range slice:  ")
	for i, v := range []string{"a", "b", "c"} {
		fmt.Printf("%d:%s ", i, v)
	}
	fmt.Println()

	// 5. Go 1.22: range over an integer - 0..n-1, or just "repeat n times".
	fmt.Print("  range int:    ")
	for i := range 5 {
		fmt.Print(i, " ")
	}
	for range 3 {
		fmt.Print("* ")
	}
	fmt.Println()

	// continue jumps to the post statement; break leaves the innermost loop. Both take labels.
	fmt.Print("  odd numbers via continue: ")
	for i := range 10 {
		if i%2 == 0 {
			continue
		}
		fmt.Print(i, " ")
	}
	fmt.Println()

	// Go 1.22: the loop variable is FRESH in every iteration. Before, all closures (or
	// goroutines) captured one shared variable and saw its final value - the most common Go bug
	// ever. The old behaviour still applies to modules declaring go 1.21 or earlier in go.mod.
	// (Three-clause on purpose: `range 3` is itself new in 1.22, so it has no "before".)
	var closures []func() int
	for i := 0; i < 3; i++ {
		closures = append(closures, func() int { return i })
	}
	fmt.Print("  closures over the loop variable: ")
	for _, c := range closures {
		fmt.Print(c(), " ")
	}
	fmt.Println("<- 0 1 2 since Go 1.22; it was 3 3 3 before")

	// Older code gets a per-iteration copy by shadowing - `i := i` as the first statement of
	// the body. That workaround is now redundant.
}

// rangeForms shows what range accepts and what it hands to the loop body.
func rangeForms() {
	fmt.Println("\n--- range ---")

	// The loop variables depend on the operand:
	//	array, slice  i, v   index, element (a COPY)
	//	string        i, r   byte offset, rune (decodes UTF-8)
	//	map           k, v   key, value (random order)
	//	channel       v      each received value, until closed
	//	integer       i      0..n-1                (Go 1.22)
	//	func(yield)   ...    what the iterator yields (Go 1.23)
	// You may take fewer variables than offered, never more.

	// The element is a COPY: writing to v is a no-op on the slice. Write through the index.
	type item struct{ Name string }
	items := []item{{Name: "a"}, {Name: "b"}}
	for _, v := range items {
		v.Name = "modified"
	}
	fmt.Printf("  writing to the range variable: %v\n", items)
	for i := range items {
		items[i].Name = "modified"
	}
	fmt.Printf("  writing through the index:     %v\n", items)

	fmt.Print("  string: ")
	for i, r := range "Gęś" {
		fmt.Printf("(%d,%c) ", i, r)
	}
	fmt.Println("<- offsets jump: ę and ś take two bytes each")

	m := map[string]int{"a": 1, "b": 2}
	fmt.Print("  map: ")
	for k, v := range m {
		fmt.Printf("%s=%d ", k, v)
	}
	fmt.Println()

	// A channel is received from until it is closed - if nobody closes it, the loop deadlocks.
	ch := make(chan int, 3)
	for i := range 3 {
		ch <- i * 10
	}
	close(ch)
	fmt.Print("  channel: ")
	for v := range ch {
		fmt.Print(v, " ")
	}
	fmt.Println()

	// A nil slice or nil map ranges zero times; a nil channel blocks forever.
	var nilSlice []int
	var nilMap map[string]int
	rounds := 0
	for range nilSlice {
		rounds++
	}
	for range nilMap {
		rounds++
	}
	fmt.Printf("  nil slice + nil map: %d iterations, no panic\n", rounds)

	// The range expression is evaluated ONCE: appending inside the loop does not extend it.
	grow := []int{1, 2, 3}
	appended := 0
	for range grow {
		grow = append(grow, 99)
		appended++
	}
	fmt.Printf("  appending inside the loop: %d iterations, len is now %d\n", appended, len(grow))

	// An ARRAY (not a slice) is copied before the loop starts.
	arr := [3]int{1, 2, 3}
	for i, v := range arr {
		arr[2] = 99
		if i == 2 {
			fmt.Printf("  array range sees the original %d, not the modified %d\n", v, arr[2])
		}
	}

	// Go 1.23: range over a function. range calls it with a yield callback; yield returns
	// false once the loop body breaks. iter.Seq / iter.Seq2 name the signatures (module 012).
	fmt.Print("  range over an iterator: ")
	for v := range countdown(3) {
		fmt.Print(v, " ")
	}
	fmt.Print("| two values: ")
	for i, s := range enumerate([]string{"x", "y"}) {
		fmt.Printf("(%d,%s) ", i, s)
	}
	fmt.Print("| break early: ")
	for v := range countdown(10) {
		if v < 8 {
			break // yield returns false and the iterator stops
		}
		fmt.Print(v, " ")
	}
	fmt.Println()
}

// countdown returns an iterator yielding n, n-1, ..., 1.
func countdown(n int) iter.Seq[int] {
	return func(yield func(int) bool) {
		for i := n; i > 0; i-- {
			if !yield(i) { // false: the loop body broke out
				return
			}
		}
	}
}

// enumerate returns a two-value iterator over a slice.
func enumerate[T any](s []T) iter.Seq2[int, T] {
	return func(yield func(int, T) bool) {
		for i, v := range s {
			if !yield(i, v) {
				return
			}
		}
	}
}

// labelsAndGoto shows labelled break and continue, and how Go fences goto in.
func labelsAndGoto() {
	fmt.Println("\n--- labels, break, continue and goto ---")

	// A label names a statement; labels are function-scoped and have their own namespace.
	// break Label leaves the labelled for/switch/select - the standard nested-loop escape.
	// (gofmt outdents labels by one level.)
search:
	for i := range 3 {
		for j := range 3 {
			if i*3+j == 5 {
				fmt.Printf("  labelled break: found 5 at (%d,%d)\n", i, j)
				break search // leaves BOTH loops; a plain break would leave only the inner one
			}
		}
	}

	// The only way to leave a loop from inside a switch.
	fmt.Print("  labelled break out of a switch: ")
loop:
	for i := range 5 {
		switch {
		case i == 3:
			break loop
		default:
			fmt.Print(i, " ")
		}
	}
	fmt.Println()

	// continue Label starts the next iteration of the labelled LOOP (not of a switch).
rows:
	for _, row := range [][]int{{1, 2}, {3, 0}, {4, 5}} {
		for _, v := range row {
			if v == 0 {
				continue rows // next row, not next element
			}
		}
		fmt.Printf("  labelled continue: row %v has no zeros\n", row)
	}

	// goto jumps within the function.
	i := 0
retry:
	i++
	if i < 3 {
		goto retry
	}
	fmt.Printf("  goto as a retry loop: i=%d\n", i)

	// But it cannot jump over a declaration in scope at the label, nor into a block - which
	// rules out C's uninitialised-variable hazards. Outside generated code and hand-written
	// state machines, a labelled break or an extracted function is nearly always better.
	//	goto skip // ERROR: goto skip jumps over declaration of v at ...
	//	v := 1
	//	skip:
	//	fmt.Println(v)
	//
	//	goto inside // ERROR: goto inside jumps into block starting at ...
	//	{ inside: fmt.Println("x") }

	// An unused label is an error, like an unused variable:
	//	unused: // ERROR: label unused defined and not used
}

// deferStatements shows when deferred calls run and what they capture.
func deferStatements() {
	fmt.Println("\n--- defer ---")

	// defer runs a call when the surrounding FUNCTION returns - by return, by falling off the
	// end, or by a panic unwinding. Go's answer to finally/RAII. Deferred calls run LIFO,
	// matching acquire-in-order, release-in-reverse.
	func() {
		for i := range 3 {
			defer fmt.Printf("  deferred %d\n", i)
		}
		fmt.Println("  function body finished")
	}()

	// Arguments (and the receiver) are evaluated AT the defer statement. So defer mu.Unlock()
	// is safe, but defer p.Close() on a p reassigned later closes the original.
	func() {
		i := 1
		defer fmt.Printf("  argument evaluated at defer time: i was %d\n", i)
		i = 99
		_ = i
	}()
	// A deferred closure reads variables when it RUNS.
	func() {
		i := 1
		defer func() { fmt.Printf("  a closure reads i at run time: i is %d\n", i) }()
		i = 99
	}()

	// A deferred closure runs after return has set the named results, and can change them -
	// how recover turns a panic into an error (module 005).
	fmt.Println("  named result modified by a defer: doubleResult(5) =", doubleResult(5))
	fmt.Println("  cleanup during a panic:", panicWithCleanup())

	// defer is FUNCTION-scoped, not block-scoped: in a loop the calls pile up until the
	// function returns - `defer f.Close()` in a loop leaks file descriptors.
	deferInLoopWrong()
	// Fix: move the body into its own function (or call Close explicitly).
	deferInLoopRight()

	// The canonical shape: acquire, check the error, defer the release immediately.
	// For a written file, ignoring Close's error can hide a failed write - check it.
	if err := readConfig("/nonexistent"); err != nil {
		fmt.Println("  readConfig on a missing file:", err)
	}

	// Since Go 1.14 most defers are open-coded and nearly free - do not contort code to avoid one.
}

func doubleResult(n int) (result int) {
	defer func() { result *= 2 }() // runs after `return n` has set result = n
	return n
}

func panicWithCleanup() (msg string) {
	defer func() {
		if r := recover(); r != nil {
			msg = fmt.Sprintf("recovered from %v, and the defer still ran", r)
		}
	}()
	defer fmt.Println("  this defer runs although the function panics")
	panic("something went wrong")
}

func deferInLoopWrong() {
	for i := range 3 {
		// With `defer file.Close()` all three files would stay open until the function returns.
		defer fmt.Printf("  wrong: cleanup %d runs only at function exit\n", i)
	}
	fmt.Println("  wrong: loop finished, nothing cleaned up yet")
}

func deferInLoopRight() {
	for i := range 3 {
		func() {
			defer fmt.Printf("  right: cleanup %d at the end of its own iteration\n", i)
		}()
	}
}

func readConfig(path string) (err error) {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open config: %w", err)
	}
	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("close config: %w", cerr)
		}
	}()
	if _, err = io.ReadAll(f); err != nil { // a real version would decode the bytes
		return fmt.Errorf("read config: %w", err)
	}
	return nil
}

// puttingItTogether runs analyse, which uses a tagless switch, an early return, a labelled break,
// range over a string and a defer - and still reads top to bottom with shallow nesting.
func puttingItTogether() {
	fmt.Println("\n--- putting it together ---")

	for _, in := range []string{"go2rust", "12345", "hello", "ab!cd", ""} {
		first, kind, err := analyse(in)
		if err != nil {
			fmt.Printf("  %-9q -> error: %v (kind set to %q by the defer)\n", in, err, kind)
			continue
		}
		firstStr := "none"
		if first != 0 {
			firstStr = string(first)
		}
		fmt.Printf("  %-9q -> kind=%-12s firstDigit=%s\n", in, kind, firstStr)
	}
}

var errEmpty = errors.New("empty input")

// analyse reports the first digit in s and what kind of characters s contains.
func analyse(s string) (first rune, kind string, err error) {
	defer func() {
		if err != nil {
			kind = "unknown"
		}
	}()

	if len(s) == 0 {
		return 0, "", errEmpty
	}

	letters, digits := 0, 0
scan:
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9':
			digits++
			if first == 0 {
				first = r
			}
		case r == '!':
			break scan // stop the whole scan at the first '!'
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z':
			letters++
		}
	}

	switch {
	case digits > 0 && letters > 0:
		kind = "alphanumeric"
	case digits > 0:
		kind = "numeric"
	case letters > 0:
		kind = "alphabetic"
	default:
		kind = "other"
	}
	return first, kind, nil
}
