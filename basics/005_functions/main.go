// Command 005_functions is module 005 of the Go course: functions.
//
// Functions are first-class values: they have types, can be stored, passed, returned and can
// close over variables. Go deliberately has no overloading, no default parameter values and no
// named arguments - one name is one function.
//
//	go run ./basics/005_functions
package main

import (
	"cmp"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
)

func main() {
	declarations()    // parameters, pass-by-value, no overloading or defaults
	multipleReturns() // (T, error), named results, naked return
	variadics()       // ...T, spreading a slice
	functionValues()  // function types, dispatch tables, method values
	closures()        // capture by reference, factories, functional options
	panicAndRecover() // panic, recover, runtime panics, MustXxx
}

// declarations shows how parameters are declared and that every argument is passed by value.
func declarations() {
	fmt.Println("--- declarations ---")

	// Consecutive parameters of the same type may share it: add(a, b int).
	fmt.Println("add(2, 3) =", add(2, 3))

	// No overloading: a float version needs a different name (Go: Parse, ParseInt, ParseFloat).
	//	func add(a, b float64) float64 // ERROR: add redeclared in this block
	fmt.Println("addFloats(2, 3) =", addFloats(2, 3))

	// Unnamed parameters are legal when the body does not use them (handy for interfaces).
	fmt.Println(ignore(1, "x"))

	// Recursion needs no special syntax. There is no tail-call optimisation, but goroutine
	// stacks grow dynamically (from 2 KB up to 1 GB), so deep recursion is usually fine.
	fmt.Println("factorial(10) =", factorial(10))

	// ALL arguments are passed by value - always. A struct is copied, a pointer is copied.
	big := bigStruct{}
	byValue(big)
	fmt.Println("after byValue:   Data[0] =", big.Data[0], "(the copy was modified)")
	byPointer(&big)
	fmt.Println("after byPointer: Data[0] =", big.Data[0])

	// A slice is a small header (pointer, len, cap); the header is copied, the elements are
	// shared. So the callee can change elements, but its append is invisible (module 006).
	s := []int{1, 2, 3}
	setFirst(s)
	fmt.Println("callee changed an element:", s)
	tryAppend(s)
	fmt.Println("callee's append is lost:  ", s)

	// No default values and no named arguments - use a config struct or functional options
	// (see closures):
	//	func greet(name string, greeting string = "Hello") // ERROR: syntax error: unexpected = in parameter list; possibly missing comma or )
	//	add(a: 1, b: 2)                                     // ERROR: syntax error: unexpected : in argument list; possibly missing comma or )
}

func add(a, b int) int { return a + b }

func addFloats(a, b float64) float64 { return a + b }

func ignore(int, string) string { return "unnamed parameters: arguments deliberately unused" }

func factorial(n int) int {
	if n <= 1 {
		return 1
	}
	return n * factorial(n-1)
}

type bigStruct struct{ Data [1024]int }

func byValue(s bigStruct) { s.Data[0] = 99 }

func byPointer(s *bigStruct) { s.Data[0] = 99 }

func setFirst(s []int) { s[0] = 99 }

func tryAppend(s []int) { s = append(s, 4); _ = s }

// multipleReturns shows several results, the (value, error) convention and named results.
func multipleReturns() {
	fmt.Println("\n--- multiple return values ---")

	// The dominant convention: (result, error), with the error LAST.
	if q, err := divide(10, 4); err == nil {
		fmt.Println("divide(10, 4) =", q)
	}
	if _, err := divide(1, 0); err != nil {
		fmt.Println("divide(1, 0) ->", err)
	}

	// The number of variables must match the number of results; discard with _.
	//	q := divide(10, 4) // ERROR: assignment mismatch: 1 variable but divide returns 2 values
	q, _ := divide(10, 4)
	fmt.Println("discarding the error with _:", q)

	// The second common shape is comma-ok (value, bool): map reads, type assertions, receives.

	// Named results document what each result means when the types alone do not.
	dir, file := splitPath("/usr/local/bin/go")
	fmt.Printf("splitPath: dir=%q file=%q\n", dir, file)

	// Named results are variables, so a deferred closure can read or change them before the
	// function returns. This is the only way to turn a panic into an error (see panicAndRecover).
	_, _ = tracedDivide(9, 3)
	_, _ = tracedDivide(9, 0)

	// A naked return returns the current named results. Legal, but in anything longer than a
	// few lines it hides what is returned - most style guides discourage it.
	d, l := nakedReturn(21)
	fmt.Printf("naked return gave (%d, %q)\n", d, l)

	// A multi-value call can be passed straight on - but only as the SOLE argument.
	fmt.Println(splitPath("/a/b"))
	//	fmt.Println(splitPath("/a/b"), "extra") // ERROR: multiple-value splitPath("/a/b") (value of type (dir string, file string)) in single-value context
}

func divide(dividend, divisor float64) (float64, error) {
	if divisor == 0 {
		return 0, errors.New("division by zero")
	}
	return dividend / divisor, nil
}

func splitPath(path string) (dir, file string) {
	i := strings.LastIndex(path, "/")
	if i < 0 {
		return ".", path
	}
	return path[:i], path[i+1:]
}

func tracedDivide(a, b float64) (result float64, err error) {
	defer func() {
		// runs after the return values are set, and can inspect them
		if err != nil {
			fmt.Printf("  [trace] %v / %v failed: %v\n", a, b, err)
		} else {
			fmt.Printf("  [trace] %v / %v = %v\n", a, b, result)
		}
	}()
	return divide(a, b)
}

func nakedReturn(n int) (doubled int, label string) {
	doubled = n * 2
	label = "doubled"
	return // returns (doubled, label)
}

// variadics shows functions that accept any number of trailing arguments.
func variadics() {
	fmt.Println("\n--- variadic functions ---")

	// The LAST parameter may be ...T; inside the function it is an ordinary []T.
	fmt.Println("sumAll()      =", sumAll())
	fmt.Println("sumAll(1,2,3) =", sumAll(1, 2, 3))

	// Spread an existing slice with s...
	nums := []int{4, 5, 6}
	fmt.Println("sumAll(nums...) =", sumAll(nums...))

	// With no arguments the parameter is a NIL slice (len 0, == nil).
	reportVariadic()
	reportVariadic(1)

	// Spreading does NOT copy: the callee gets a header over YOUR array and can change it.
	shared := []int{1, 2, 3}
	clobberFirst(shared...)
	fmt.Println("after a callee wrote to the spread slice:", shared)

	// Fixed and spread arguments cannot be mixed - build one slice first.
	//	sumAll(1, 2, nums...) // ERROR: have (number, number, []int...) want (...int)
	combined := append([]int{1, 2}, nums...)
	fmt.Println("sumAll(combined...) =", sumAll(combined...))

	// ...any is what fmt.Println uses.
	fmt.Println(describe("mixed", 1, "two", 3.0, true))

	// A []string cannot be spread into ...any - []string is not []any. Convert element-wise.
	words := []string{"a", "b"}
	//	describe("x", words...) // ERROR: cannot use words (variable of type []string) as []any value in argument to describe
	asAny := make([]any, len(words))
	for i, w := range words {
		asAny[i] = w
	}
	fmt.Println(describe("converted", asAny...))

	// go vet's printf check knows variadic format functions and flags verb/argument mismatches.
}

func sumAll(values ...int) (sum int) {
	for _, v := range values {
		sum += v
	}
	return sum
}

func reportVariadic(values ...int) {
	fmt.Printf("  inside: %v len=%d isNil=%t\n", values, len(values), values == nil)
}

func clobberFirst(values ...int) {
	if len(values) > 0 {
		values[0] = 999
	}
}

func describe(prefix string, values ...any) string {
	parts := make([]string, 0, len(values))
	for _, v := range values {
		parts = append(parts, fmt.Sprintf("%v(%T)", v, v))
	}
	return prefix + ": " + strings.Join(parts, ", ")
}

// functionValues shows that functions are values with types.
func functionValues() {
	fmt.Println("\n--- functions as values ---")

	// A function type is its signature: func(int) int. Parameter names do not matter.
	// A named function type documents the signature and - being a defined type - can have
	// methods. That is exactly how http.HandlerFunc turns a function into an interface.
	var double transform = func(n int) int { return n * 2 }
	fmt.Println("double(21) =", double(21), "| double.twice(3) =", double.twice(3))

	// The zero value is nil, and calling a nil function panics.
	var nilFunc func()
	fmt.Println("nil function value: isNil =", nilFunc == nil)

	// Function values can only be compared with nil:
	//	double == double // ERROR: invalid operation: double == double (func can only be compared to nil)

	// A map of functions is the idiomatic dispatch table.
	ops := map[string]func(int, int) int{
		"add": func(a, b int) int { return a + b },
		"mul": func(a, b int) int { return a * b },
		"sub": func(a, b int) int { return a - b },
	}
	for _, name := range slices.Sorted(maps.Keys(ops)) {
		fmt.Printf("  ops[%q](6, 3) = %d\n", name, ops[name](6, 3))
	}

	// Method value x.M: a function with the receiver already bound.
	c := &counter{}
	inc := c.inc
	inc()
	inc()
	fmt.Println("method value c.inc bound the receiver: n =", c.get())

	// Method expression T.M: the receiver becomes the first parameter.
	get := counter.get
	fmt.Println("method expression counter.get(*c) =", get(*c))

	// Higher-order functions. The standard library has slices.SortFunc/IndexFunc, but no
	// Map/Filter/Reduce - a plain for loop is usually clearer (module 012 shows iterators).
	numbers := []int{5, 2, 8, 1}
	sorted := slices.Clone(numbers)
	slices.SortFunc(sorted, func(a, b int) int { return cmp.Compare(b, a) }) // descending
	fmt.Println("slices.SortFunc descending:", sorted)
	fmt.Println("slices.IndexFunc(n > 4) =", slices.IndexFunc(numbers, func(n int) bool { return n > 4 }))

	fmt.Print("forEach: ")
	forEach(numbers, func(i, v int) { fmt.Printf("%d:%d ", i, v) })
	fmt.Println()
}

type transform func(int) int

func (t transform) twice(n int) int { return t(t(n)) }

type counter struct{ n int }

func (c *counter) inc()    { c.n++ }
func (c counter) get() int { return c.n }

func forEach(values []int, task func(index, value int)) {
	for i, v := range values {
		task(i, v)
	}
}

// closures shows function literals that capture variables from the enclosing scope.
func closures() {
	fmt.Println("\n--- closures ---")

	// A captured variable escapes to the heap and lives as long as the closure does,
	// so state survives the factory's return. Each call to the factory makes new state.
	nextUser := idGenerator("user")
	nextOrder := idGenerator("order")
	fmt.Println(nextUser(), nextUser(), nextUser(), "|", nextOrder(), nextOrder())

	// Capture is BY REFERENCE: the closure sees later changes...
	x := 1
	show := func() { fmt.Println("  the closure sees x =", x) }
	show()
	x = 42
	show()

	// ...and can write back.
	total := 0
	addToTotal := func(n int) { total += n }
	for _, n := range []int{1, 2, 3} {
		addToTotal(n)
	}
	fmt.Println("  closure wrote to an outer variable: total =", total)

	// Closures created in the same scope SHARE a variable.
	shared := 0
	var sharers []func() int
	for range 3 {
		sharers = append(sharers, func() int { shared++; return shared })
	}
	fmt.Println("  three closures over ONE variable:", sharers[0](), sharers[1](), sharers[2]())

	// A fresh variable per iteration gives each closure its own. Since Go 1.22 the loop
	// variable itself is fresh per iteration, so capturing i directly would work too.
	var owners []func() int
	for i := range 3 {
		own := i * 10
		owners = append(owners, func() int { return own })
	}
	fmt.Println("  three closures over their OWN variable:", owners[0](), owners[1](), owners[2]())

	// Functional options: Go's answer to default and named parameters.
	def := newServer("localhost")
	custom := newServer("example.com", withPort(8443), withTLS(), withTimeout(5))
	fmt.Printf("  defaults: %+v\n", *def)
	fmt.Printf("  options:  %+v\n", *custom)

	// Memoisation: the cache is captured state.
	fib := memoFib()
	fmt.Println("  memoised fib(50) =", fib(50))

	// Watch lifetimes: a long-lived closure keeps everything it captured alive.
}

func idGenerator(prefix string) func() string {
	id := 0 // captured; outlives this function
	return func() string {
		id++
		return fmt.Sprintf("%s-%03d", prefix, id)
	}
}

type server struct {
	host    string
	port    int
	timeout int
	tls     bool
}

type option func(*server)

func withPort(p int) option    { return func(s *server) { s.port = p } }
func withTimeout(t int) option { return func(s *server) { s.timeout = t } }
func withTLS() option          { return func(s *server) { s.tls = true } }

func newServer(host string, opts ...option) *server {
	s := &server{host: host, port: 80, timeout: 30} // the defaults live here
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func memoFib() func(int) int {
	cache := map[int]int{}
	var fib func(int) int // declared first so the literal can call itself
	fib = func(n int) int {
		if n < 2 {
			return n
		}
		if v, ok := cache[n]; ok {
			return v
		}
		v := fib(n-1) + fib(n-2)
		cache[n] = v
		return v
	}
	return fib
}

// panicAndRecover shows panics, how to stop one, and when (rarely) to use them.
func panicAndRecover() {
	fmt.Println("\n--- panic and recover ---")

	// panic(v) unwinds the stack running deferred functions; unrecovered, it prints a stack
	// trace and exits with status 2. recover() inside a DEFERRED function stops the unwinding.
	// Typical use: a boundary (HTTP handler, worker loop) turns a panic into an error.
	fmt.Println(" ", safeDivide(10, 2))
	fmt.Println(" ", safeDivide(10, 0))

	// panic is NOT an exception mechanism: expected failures are error values; panic is for
	// bugs. A library must not let a panic escape on bad input.

	// The runtime panics by itself for these (they arrive as runtime.Error):
	for _, f := range []func(){
		func() { var p *counter; p.inc() },                    // nil pointer dereference
		func() { s := []int{1}; _ = s[5] },                    // index out of range
		func() { a, b := 1, 0; _ = a / b },                    // integer divide by zero
		func() { var i any = "text"; _ = i.(int) },            // failed type assertion
		func() { ch := make(chan int); close(ch); close(ch) }, // close of closed channel
	} {
		fmt.Println("  runtime panic:", catchPanic(f))
	}

	// Some failures are fatal errors, not panics, and cannot be recovered at all:
	// concurrent map writes, "all goroutines are asleep - deadlock!", stack exhaustion.

	// recover only works when called directly by a deferred function of the panicking frame.
	fmt.Println(" ", recoverInWrongPlace())

	// MustXxx (regexp.MustCompile, template.Must) panics on input that is effectively a
	// constant - a failure means the program itself is wrong.
	fmt.Println("  mustPositive(5) =", mustPositive(5))
	fmt.Println("  mustPositive(-1) ->", catchPanic(func() { mustPositive(-1) }))

	// A panic in ANY goroutine kills the whole program; a recover in another goroutine does
	// not help. Every goroutine needs its own recover (module 011).
}

func safeDivide(a, b int) (msg string) {
	defer func() {
		if r := recover(); r != nil {
			msg = fmt.Sprintf("%d/%d recovered: %v", a, b, r) // overwrite the named result
		}
	}()
	return fmt.Sprintf("%d/%d = %d", a, b, a/b)
}

// catchPanic runs f and returns whatever it panicked with.
func catchPanic(f func()) (recovered any) {
	defer func() {
		// Only overwrite on a real panic; an unconditional `recovered = recover()` would also
		// run on the normal path and replace "did not panic" with nil.
		if r := recover(); r != nil {
			recovered = r
		}
	}()
	f()
	return "did not panic"
}

func recoverInWrongPlace() string {
	if r := recover(); r != nil { // not deferred: always nil
		return fmt.Sprintf("recovered %v", r)
	}
	return "recover() outside a deferred function returns nil and does nothing"
}

func mustPositive(n int) int {
	if n <= 0 {
		panic(fmt.Sprintf("mustPositive: %d is not positive", n))
	}
	return n
}
