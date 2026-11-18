// Command 000_essentials is the gentle on-ramp to the Go course: modules 001-014 (the language)
// and 018-020 (building, securing and storing a service), plus Gin as the popular alternative
// to net/http routing.
//
// The full modules are thorough: every corner case, every trap, every Go version. This program
// is the opposite - one short section per module with only what you need on day one. Read a
// section, run the program, compare with the output, and when a topic clicks open the full
// module named in the "More:" line at the end of the section. Interfaces (008) and generics
// (010) get a little more room, because almost everything after them builds on those two ideas.
//
// This file holds modules 001-014. Modules 018-020 have one file per section, s018_http.go to
// s020_gorm.go, and services.go is their map. The sections table below is the table of
// contents for all of them, and the command-line arguments choose which sections run. In
// package main nothing is imported by other packages, so the upper/lower case of names does
// not matter here; the files still use Upper names for the main types and functions, the way
// a library would.
//
// Sections 019 and 020 need PostgreSQL (docker-compose.yml in the repository root). Without it
// they print a note and run only the parts that need no database.
//
//	go run ./basics/000_essentials             # every section
//	go run ./basics/000_essentials list        # the table of contents
//	go run ./basics/000_essentials 008 18c     # only these sections
//	go test -v ./basics/000_essentials
//	docker-compose up -d                       # PostgreSQL for 019 and 020
package main

import (
	"bufio"
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"iter"
	"log/slog"
	"maps"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"
)

// section is one entry of the table of contents: the module number, what it covers, and the
// function that runs it.
type section struct {
	id    string
	topic string
	run   func()
}

// sections is the table of contents, in teaching order. Run every section, or only some:
//
//	go run ./basics/000_essentials              # everything
//	go run ./basics/000_essentials list         # this table
//	go run ./basics/000_essentials 8            # only 008 (leading zeros are optional)
//	go run ./basics/000_essentials 018c 11      # these two, in this order
//	go run ./basics/000_essentials 18           # only 018
//	go run ./basics/000_essentials '18*'        # 018, 018b, 018c and 018d
//	go run ./basics/000_essentials 2            # 002a and 002b: there is no plain 002
var sections = []section{
	{"001", "variables, zero values, scope, constants, iota", basics},
	{"002a", "numbers, strings, bytes and runes, conversions", basicTypes},
	{"002b", "arrays, slices, maps, structs", compositeTypes},
	{"003", "arithmetic, comparison, logic, assignment", operators},
	{"004", "if, switch, for, range, defer", controlFlow},
	{"005", "multiple results, variadics, closures, panic", functions},
	{"006", "& and *, pass by value vs by pointer", pointers},
	{"007", "methods, receivers, constructors, embedding", structsAndMethods},
	{"008", "implicit interfaces, values and aliasing, embedding and shadowing", interfaces},
	{"009", "errors as values, wrapping, Is/As, which kind to use, pitfalls", errorHandling},
	{"010", "type parameters, constraints, generic types", generics},
	{"011", "goroutines, channels, select, mutex/atomic, context, errors", concurrency},
	{"012", "range over func, Seq2, lazy adapters, slices, maps, cmp", iteratorsAndCollections},
	{"013", "fmt, strings, time, json, files, slog", stdlib},
	{"014", "go test - the tests are in main_test.go", testingIntro},

	// one file per section: s018_http.go ... s020_gorm.go (the map is services.go)
	{"018", "net/http: Handler, ServeMux routing, JSON, middleware, http.Server, shutdown", httpServices},
	{"018b", "Gin: routes, params, binding and validation, groups, middleware", ginBasics},
	{"018c", "security: headers, bcrypt, JWT, 401/403, cookies, CSRF, CORS, rate limits, TLS", httpSecurity},
	{"018d", "Gin security: trusted proxies, auth groups, body limits, BasicAuth, cookies, XSS", ginSecurity},
	{"019", "database/sql: the pool, queries, placeholders, NULL, transactions, repository", databaseSQL},
	{"020", "GORM: models and tags, AutoMigrate, CRUD traps, Preload vs N+1, Transaction", gormBasics},
}

func main() {
	selected, err := selectSections(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		printSections(os.Stderr)
		os.Exit(2)
	}
	if selected == nil { // "list"
		printSections(os.Stdout)
		return
	}
	for _, s := range selected {
		s.run()
	}
}

// selectSections turns the command-line arguments into the sections to run: every section for
// no arguments, and nil for "list". Otherwise each argument names the section with exactly
// that number ("018"), or - with a trailing * ("018*"), or when no number matches exactly
// ("002") - every section whose number starts with it. Leading zeros may be left out ("8" is
// "008"), and a section named twice runs once.
func selectSections(args []string) ([]section, error) {
	if len(args) == 0 {
		return sections, nil
	}
	if len(args) == 1 && slices.Contains([]string{"list", "help", "-h", "--help"}, args[0]) {
		return nil, nil
	}
	var selected []section
	seen := map[string]bool{}
	for _, arg := range args {
		name, wildcard := strings.CutSuffix(strings.ToLower(arg), "*")
		name = padSectionNumber(name)
		exact := !wildcard && slices.ContainsFunc(sections, func(s section) bool { return s.id == name })
		found := false
		for _, s := range sections {
			if s.id == name || (!exact && strings.HasPrefix(s.id, name)) {
				found = true
				if !seen[s.id] {
					seen[s.id] = true
					selected = append(selected, s)
				}
			}
		}
		if !found {
			return nil, fmt.Errorf("no section %q", arg)
		}
	}
	return selected, nil
}

// padSectionNumber pads the number at the start of arg to three digits: "8" -> "008",
// "18c" -> "018c". Anything else is returned unchanged.
func padSectionNumber(arg string) string {
	digits := len(arg) - len(strings.TrimLeft(arg, "0123456789"))
	if digits == 0 || digits >= 3 {
		return arg
	}
	return strings.Repeat("0", 3-digits) + arg
}

// printSections prints the table of contents and how to use it.
func printSections(w io.Writer) {
	fmt.Fprintln(w, "usage: go run ./basics/000_essentials [list | SECTION...]")
	fmt.Fprintln(w, "  no SECTION runs every section; 18 runs 018; '18*' runs 018, 018b, 018c and 018d")
	for _, s := range sections {
		fmt.Fprintf(w, "  %-5s %s\n", s.id, s.topic)
	}
}

// ============================================================================================
// 001 - variables, zero values, scope, constants, iota
// ============================================================================================

// basics covers the building blocks of every Go file.
func basics() {
	fmt.Println("--- 001 variables, constants, scope ---")

	// Every file starts with "package name"; imports follow. Comments are // to the end of the
	// line or /* ... */. A comment directly above a declaration is its documentation.

	// Two ways to declare a variable. var works everywhere; := (short declaration) works only
	// inside functions and takes the type from the value.
	var name string = "Gopher"
	var age = 13 // type inferred: int
	city := "Warsaw"
	fmt.Println(" ", name, age, city)

	// Every type has a ZERO VALUE, so a variable is never "uninitialised".
	var (
		i int     // 0
		f float64 // 0
		b bool    // false
		s string  // "" - an empty string, never nil
		p *int    // nil
	)
	fmt.Printf("  zero values: %d %g %t %q %v\n", i, f, b, s, p)

	// An unused local variable (or import) is a compile error, not a warning.
	//	x := 1 // ERROR: declared and not used: x

	// Visibility is decided by the first letter: Upper is exported (usable from other packages),
	// lower is private to the package. There are no public/private keywords.

	// Scope is the block { }. An inner := declares a NEW variable that hides (shadows) the outer.
	x := 1
	if true {
		x := 2 // a different x, alive only inside this block
		fmt.Println("  inner x:", x)
	}
	fmt.Println("  outer x:", x) // still 1

	// The classic trap: := inside a block when you meant = . The error never leaves the block.
	var err error
	if true {
		n, err := strconv.Atoi("42x") // := declares a NEW err here
		fmt.Println("  inside the block:", n, err)
	}
	fmt.Println("  outside, err is still:", err) // <nil>

	// Constants are fixed at compile time. iota counts 0, 1, 2 ... inside a const block - that
	// is the Go way to write an enum.
	const pi = 3.14159
	fmt.Println("  pi:", pi, " Monday:", Monday, " Wednesday:", Wednesday)

	// More: basics/001a_comments_packages_variables_scope_shadowing, basics/001b_constants_iota_init
}

// Weekday is an enum: a defined type plus a block of iota constants.
type Weekday int

const (
	Monday    Weekday = iota // 0
	Tuesday                  // 1 - the expression repeats, iota grows by one
	Wednesday                // 2
)

// ============================================================================================
// 002a - basic types
// ============================================================================================

// basicTypes covers numbers, booleans, strings and conversions.
func basicTypes() {
	fmt.Println("--- 002a basic types ---")

	// The everyday types: int, float64, bool, string. int is 64 bits on every machine you will
	// meet; float64 is the default for decimals. (There are also int8..int64, uint, float32...)
	var count int = 7
	var price float64 = 9.99
	var ok bool = true
	var title string = "Go"
	fmt.Println(" ", count, price, ok, title)

	// No implicit conversions - not even int to float64. Convert explicitly with T(v).
	//	total := price * count // ERROR: invalid operation: price * count (mismatched types float64 and int)
	total := price * float64(count)
	fmt.Printf("  total: %.2f, as int: %d (the fraction is dropped)\n", total, int(total))

	// A string is an immutable sequence of BYTES in UTF-8. len counts bytes, not characters.
	word := "żółw" // "turtle" in Polish: 4 characters, 7 bytes
	fmt.Println("  len (bytes):", len(word), " characters:", utf8.RuneCountInString(word))

	// Indexing gives a byte. range decodes UTF-8 and gives runes (characters) and byte offsets.
	fmt.Println("  word[0] is a byte:", word[0])
	fmt.Print("  range:")
	for i, r := range word {
		fmt.Printf(" %d:%c", i, r)
	}
	fmt.Println()

	// Strings and numbers convert through strconv - NOT through string(n), which treats n as a
	// character code: string(rune(65)) is "A", not "65".
	n, err := strconv.Atoi("123")
	fmt.Println("  Atoi:", n+1, err)
	_, err = strconv.Atoi("12a")
	fmt.Println("  Atoi error:", err)
	f, _ := strconv.ParseFloat("2.5", 64) // _ ignores the error - only OK in tiny examples
	fmt.Println("  Itoa:", strconv.Itoa(42)+"!", " ParseFloat:", f*2)

	// More: basics/002a_basic_types
}

// ============================================================================================
// 002b - arrays, slices, maps, structs
// ============================================================================================

// compositeTypes covers the four ways to group values.
func compositeTypes() {
	fmt.Println("--- 002b arrays, slices, maps, structs ---")

	// ARRAY: the length is part of the type. Rarely used directly.
	arr := [3]int{1, 2, 3}

	// SLICE: a growable view of an array - what you use 99% of the time.
	nums := []int{1, 2, 3}
	nums = append(nums, 4, 5) // append may move the data, so ALWAYS assign the result back
	fmt.Println("  array:", arr, " slice:", nums, " len:", len(nums), " cap:", cap(nums))

	// make preallocates: length 0, room (capacity) for 10.
	squares := make([]int, 0, 10)
	for i := range 5 {
		squares = append(squares, i*i)
	}
	fmt.Println("  squares:", squares, " squares[1:3]:", squares[1:3])

	// Slicing does NOT copy: both slices see the same underlying array.
	part := squares[1:3]
	part[0] = 100
	fmt.Println("  after part[0] = 100, squares is:", squares)

	// Need an independent copy? slices.Clone.
	independent := slices.Clone(squares)
	independent[0] = -1
	fmt.Println("  the clone changed, the original did not:", squares[0])

	// MAP: key -> value. A missing key gives the zero value; the comma-ok form tells "missing"
	// apart from "present but zero".
	ages := map[string]int{"Ann": 31, "Bob": 25}
	ages["Cid"] = 40
	delete(ages, "Bob")
	if age, ok := ages["Bob"]; !ok {
		fmt.Println("  Bob is missing, the lookup gave the zero value:", age)
	}
	fmt.Println("  map:", ages, " len:", len(ages)) // fmt prints maps sorted by key
	// Ranging over a map visits keys in RANDOM order - sort the keys if order matters (012).
	// A nil map (var m map[string]int) can be read, but writing to it panics: use make or {}.

	// STRUCT: named fields grouped into one value.
	type point struct {
		X, Y int
	}
	pt := point{X: 1, Y: 2} // name the fields - clearer, and safe if fields are reordered
	pt.X = 10
	fmt.Printf("  struct: %+v\n", pt)

	// More: basics/002b_composite_types
}

// ============================================================================================
// 003 - operators
// ============================================================================================

// operators covers what is different from other C-like languages.
func operators() {
	fmt.Println("--- 003 operators ---")

	// + - * / % as usual. Integer division truncates toward zero.
	fmt.Println("  7/2 =", 7/2, " 7%2 =", 7%2, " -7/2 =", -7/2, " 7.0/2 =", 7.0/2)

	// Comparison (== != < <= > >=) and logic (&& || !). && and || stop as soon as the answer
	// is known (short-circuit). + also joins strings.
	age := 20
	fmt.Println("  adult:", age >= 18 && age < 65, " greeting:", "Hello, "+"Go")

	// ++ and -- are STATEMENTS, not expressions, and only postfix.
	age++
	//	y := age++ // ERROR: syntax error: unexpected ++ at end of statement
	age += 5 // compound assignment: += -= *= /= %=

	// There is no ternary ?: operator - write an if.
	status := "minor"
	if age >= 18 {
		status = "adult"
	}
	fmt.Println("  age:", age, " status:", status)

	// Tuple assignment: swap without a temporary variable.
	a, b := 1, 2
	a, b = b, a
	fmt.Println("  swapped:", a, b)

	// More: basics/003_operators
}

// ============================================================================================
// 004 - control flow
// ============================================================================================

// controlFlow covers if, switch, for, range and defer.
func controlFlow() {
	fmt.Println("--- 004 control flow ---")

	// if: no parentheses, braces are mandatory. The optional init statement runs first; its
	// variables live only inside the if/else.
	if n, err := strconv.Atoi("42"); err == nil {
		fmt.Println("  parsed:", n)
	} else {
		fmt.Println("  error:", err)
	}

	// switch: cases do NOT fall through, so no break is needed. A case can list several values.
	day := "sat"
	switch day {
	case "sat", "sun":
		fmt.Println("  weekend")
	default:
		fmt.Println("  workday")
	}

	// A switch with no value is a clean if / else-if chain.
	temp := 23
	switch {
	case temp < 10:
		fmt.Println("  cold")
	case temp < 25:
		fmt.Println("  pleasant")
	default:
		fmt.Println("  hot")
	}

	// for is the ONLY loop keyword. Its shapes:
	fmt.Print("  classic:")
	for i := 0; i < 3; i++ {
		fmt.Print(" ", i)
	}
	fmt.Println()

	n := 1
	for n < 100 { // like "while"
		n *= 2
	}
	fmt.Println("  first power of 2 >= 100:", n)

	fmt.Print("  range over an int:")
	for i := range 3 { // 0, 1, 2
		fmt.Print(" ", i)
	}
	fmt.Println()
	// for { ... } loops forever; leave it with break or return.

	// range over a slice gives the index and a COPY of the element. _ discards what you skip.
	fmt.Print("  range over a slice:")
	fruits := []string{"apple", "banana", "cherry"}
	for i, f := range fruits {
		if f == "banana" {
			continue // jump to the next iteration
		}
		fmt.Printf(" %d=%s", i, f)
	}
	fmt.Println()

	// defer schedules a call for when the surrounding function returns, in LIFO order. It is
	// how Go releases resources: open something, then immediately defer its Close.
	deferDemo()

	// More: basics/004_control_flow
}

func deferDemo() {
	defer fmt.Println("  deferred #1 (runs last)")
	defer fmt.Println("  deferred #2 (runs first)")
	fmt.Println("  function body")
}

// ============================================================================================
// 005 - functions
// ============================================================================================

// functions covers declarations, multiple results, variadics, closures and panic.
func functions() {
	fmt.Println("--- 005 functions ---")

	// Types come after names. Arguments are passed BY VALUE (copied). There is no overloading
	// and there are no default parameter values.
	fmt.Println("  add:", add(2, 3))

	// Multiple results. The idiom is (result, error): error last, checked right away.
	q, err := divide(10, 4)
	fmt.Println("  divide:", q, err)
	if _, err := divide(1, 0); err != nil {
		fmt.Println("  divide error:", err)
	}

	// Variadic: ...T collects any number of arguments into a slice; slice... spreads one back.
	values := []int{1, 2, 3}
	fmt.Println("  sumAll:", sumAll(), sumAll(1, 2), sumAll(values...))

	// Functions are values: store them in variables, pass them in, return them.
	multiply := func(a, b int) int { return a * b } // an anonymous function
	fmt.Println("  apply:", apply(add, 3, 4), apply(multiply, 3, 4))

	// A closure captures variables from the surrounding scope - and keeps them alive.
	next := counter()
	fmt.Println("  counter:", next(), next(), next())

	// panic stops normal execution. It is for bugs and impossible states, NOT for expected
	// failures - return an error for those. A deferred recover() can catch a panic.
	fmt.Println("  safeDivide:", safeDivide(1, 0))

	// More: basics/005_functions
}

func add(a, b int) int { return a + b }

func divide(a, b float64) (float64, error) {
	if b == 0 {
		return 0, errors.New("division by zero")
	}
	return a / b, nil
}

func sumAll(nums ...int) int { // nums is a []int
	total := 0
	for _, n := range nums {
		total += n
	}
	return total
}

// apply takes a function as a parameter; func(int, int) int is a type like any other.
func apply(op func(int, int) int, a, b int) int { return op(a, b) }

// counter returns a function that remembers n between calls.
func counter() func() int {
	n := 0
	return func() int {
		n++
		return n
	}
}

// safeDivide turns a run-time panic into a normal result. The result is NAMED so that the
// deferred function can still set it after the panic.
func safeDivide(a, b int) (result string) {
	defer func() {
		if r := recover(); r != nil {
			result = fmt.Sprint("recovered: ", r)
		}
	}()
	return strconv.Itoa(a / b) // integer division by zero panics
}

// ============================================================================================
// 006 - pointers
// ============================================================================================

// pointers covers & and *, and when a function can change its argument.
func pointers() {
	fmt.Println("--- 006 pointers ---")

	// A pointer holds the ADDRESS of a value. &x takes the address, *p reads or writes through it.
	x := 10
	p := &x // p has type *int
	*p = 20
	fmt.Println("  x:", x, " *p:", *p)

	// Everything is passed by value: a function gets a copy - unless you pass a pointer.
	n := 1
	incrementCopy(n)
	fmt.Println("  after incrementCopy:", n) // 1 - only the copy changed
	incrementPointer(&n)
	fmt.Println("  after incrementPointer:", n) // 2

	// The zero value of a pointer is nil. Dereferencing nil compiles but panics at run time.
	var np *int
	fmt.Println("  np == nil:", np == nil)
	//	*np = 1 // panic: runtime error: invalid memory address or nil pointer dereference

	// new(T) allocates a zeroed T and returns *T. For structs, &T{...} is more common.
	ip := new(int)
	*ip = 5
	box := &labelBox{Label: "draft"}
	box.Label = "final" // Go dereferences for you: no (*box).Label, no ->
	fmt.Println("  new(int):", *ip, " struct via pointer:", box.Label)

	// No pointer arithmetic. Returning the address of a local variable is SAFE: the compiler
	// moves it to the heap when needed, and the garbage collector frees it later.
	fmt.Println("  pointer from a function:", *newInt(7))

	// Slices and maps are small headers that point at shared data. Passing one copies the
	// header, not the data, so a function CAN change their elements.
	nums := []int{1, 2, 3}
	doubleAll(nums)
	fmt.Println("  slice changed inside a function:", nums)

	// When to use a pointer: the function must MODIFY the value, or the value is large.
	// Otherwise pass values - they are simpler and safer.

	// More: basics/006_pointers_and_memory
}

type labelBox struct{ Label string }

func incrementCopy(n int)     { n++ }
func incrementPointer(n *int) { *n++ }
func newInt(v int) *int       { return &v }

func doubleAll(nums []int) {
	for i := range nums {
		nums[i] *= 2
	}
}

// ============================================================================================
// 007 - structs and methods
// ============================================================================================

// structsAndMethods covers methods, receivers, constructors and embedding.
func structsAndMethods() {
	fmt.Println("--- 007 structs and methods ---")

	// A method is a function with a RECEIVER - the value before the name. Go has no classes:
	// data lives in a struct, behaviour in methods declared next to it.
	acc := NewAccount("Ann", 100) // constructor: a plain function, by convention NewXxx
	acc.Deposit(50)
	if err := acc.Withdraw(500); err != nil {
		fmt.Println("  withdraw failed:", err)
	}
	fmt.Printf("  %s has %.2f\n", acc.Owner, acc.Balance())

	// Pointer receiver (a *Account) - the method works on the ORIGINAL and can change it.
	// Value receiver   (c Celsius)  - the method gets a COPY: fine for small, read-only types.
	// Rule of thumb: if one method needs a pointer receiver, give all methods of the type one.
	temp := Celsius(21.5)
	fmt.Printf("  %.1f°C = %.1f°F (methods work on any defined type, not only structs)\n", temp, temp.Fahrenheit())

	// Lowercase fields (balance) are private to the package, so other packages must go
	// through the methods - that is how Go encapsulates.

	// EMBEDDING: a field with a type but no name. Its fields and methods are PROMOTED to the
	// outer type. This is composition, not inheritance.
	savings := SavingsAccount{Account: Account{Owner: "Bob", balance: 1000}, Rate: 0.05}
	savings.Deposit(100) // promoted from Account
	savings.AddInterest()
	fmt.Printf("  savings of %s: %.2f\n", savings.Owner, savings.Balance())

	// More: basics/007_structs_and_methods
}

type Account struct {
	Owner   string
	balance float64 // unexported: only this package can touch it
}

// NewAccount is a constructor; it returns a pointer because Account is changed by its methods.
func NewAccount(owner string, initial float64) *Account {
	return &Account{Owner: owner, balance: initial}
}

func (a *Account) Deposit(amount float64) { a.balance += amount }
func (a *Account) Balance() float64       { return a.balance }

func (a *Account) Withdraw(amount float64) error {
	if amount > a.balance {
		return fmt.Errorf("insufficient funds: balance %.2f, requested %.2f", a.balance, amount)
	}
	a.balance -= amount
	return nil
}

type Celsius float64

func (c Celsius) Fahrenheit() float64 { return float64(c)*9/5 + 32 }

// SavingsAccount embeds Account and adds a field and a method of its own.
type SavingsAccount struct {
	Account // embedded: SavingsAccount now has Owner, Deposit, Balance, ...
	Rate    float64
}

func (s *SavingsAccount) AddInterest() { s.Deposit(s.balance * s.Rate) }

// ============================================================================================
// 008 - interfaces
// ============================================================================================

// interfaces is the whole of polymorphism in Go.
func interfaces() {
	fmt.Println("--- 008 interfaces ---")
	interfaceBasics()         // what an interface is and how a type satisfies it
	interfaceValues()         // what an interface holds: copies, pointers (aliasing), nil
	interfacesForDecoupling() // depend on behaviour; swap real code for fakes
	embeddingAndShadowing()   // composing interfaces, embedding, shadowing a method
	interfacesAtRuntime()     // any, type switches, type assertions

	// More: basics/008_interfaces
}

// interfaceBasics shows that satisfying an interface is implicit.
func interfaceBasics() {
	// An interface is a set of METHOD SIGNATURES. Any type that has those methods satisfies it
	// automatically - there is no "implements" keyword. Circle and Rect never mention Shape.
	shapes := []Shape{Circle{R: 1}, Rect{W: 2, H: 3}}
	for _, s := range shapes {
		fmt.Printf("  %-12v area = %.2f\n", s, s.Area())
	}

	// totalArea accepts ANY Shape - including types written years later by someone else.
	fmt.Printf("  total area: %.2f\n", totalArea(shapes))

	// An interface variable can hold any value whose type has the methods.
	var s Shape = Rect{W: 1, H: 1}
	fmt.Printf("  s holds a %T\n", s)
	s = Circle{R: 2}
	fmt.Printf("  now s holds a %T\n", s)

	// Why did %v print "Circle(r=1)" above? Circle has a String() string method, so it satisfies
	// fmt.Stringer - an interface that fmt checks before printing a value. Same mechanism, and
	// again nobody wrote "implements".

	// Keep interfaces SMALL. The most used ones in Go have a single method: io.Reader,
	// io.Writer, fmt.Stringer, error. A one-method interface is named method + "er".
}

// Shape is satisfied by every type that has an Area() float64 method.
type Shape interface {
	Area() float64
}

type Circle struct{ R float64 }
type Rect struct{ W, H float64 }

func (c Circle) Area() float64 { return math.Pi * c.R * c.R }
func (r Rect) Area() float64   { return r.W * r.H }

func (c Circle) String() string { return fmt.Sprintf("Circle(r=%g)", c.R) }
func (r Rect) String() string   { return fmt.Sprintf("Rect(%gx%g)", r.W, r.H) }

func totalArea(shapes []Shape) float64 {
	total := 0.0
	for _, s := range shapes {
		total += s.Area() // the call goes to Circle.Area or Rect.Area, decided at run time
	}
	return total
}

// interfacesForDecoupling shows the main practical use: code depends on WHAT something does,
// not on WHICH concrete type does it.
func interfacesForDecoupling() {
	// OrderService only needs "something that can send a notification". It does not know about
	// e-mail, SMS or tests - so any of them can be plugged in.
	svc := OrderService{notifier: &EmailNotifier{From: "shop@example.com"}}
	svc.PlaceOrder("book")

	// In a test, plug in a fake that just records the calls - no mocking framework needed.
	fake := &FakeNotifier{}
	svc = OrderService{notifier: fake}
	svc.PlaceOrder("pen")
	fmt.Println("  the fake recorded:", fake.Sent)

	// Notify has a POINTER receiver, so only *EmailNotifier satisfies Notifier:
	//	OrderService{notifier: EmailNotifier{}} // ERROR: cannot use EmailNotifier{} (value of struct type EmailNotifier) as Notifier value in struct literal: EmailNotifier does not implement Notifier (method Notify has pointer receiver)

	// The standard library is built the same way. io.Writer has one method -
	// Write(p []byte) (n int, err error) - so one function can write to the terminal, a file, a
	// buffer, a network connection or an HTTP response.
	writeReport(os.Stdout)
	var sb strings.Builder
	writeReport(&sb)
	fmt.Printf("  captured by a strings.Builder: %q\n", sb.String())

	// Rule of thumb: declare an interface where it is USED (here: next to OrderService), with
	// only the methods that code needs. "Accept interfaces, return concrete types."
}

// Notifier is defined by its consumer, OrderService, and lists only what it needs.
type Notifier interface {
	Notify(to, message string) error
}

type OrderService struct {
	notifier Notifier // depends on behaviour, not on a concrete type
}

func (s OrderService) PlaceOrder(item string) {
	// ... save the order somewhere ...
	if err := s.notifier.Notify("customer", "order placed: "+item); err != nil {
		fmt.Println("  could not notify:", err)
	}
}

type EmailNotifier struct{ From string }

func (e *EmailNotifier) Notify(to, message string) error {
	fmt.Printf("  [e-mail from %s to %s] %s\n", e.From, to, message)
	return nil
}

type FakeNotifier struct{ Sent []string }

func (f *FakeNotifier) Notify(to, message string) error {
	f.Sent = append(f.Sent, to+": "+message)
	return nil
}

func writeReport(w io.Writer) {
	fmt.Fprintln(w, "  report: 2 orders, 0 errors") // Fprintln writes to any io.Writer
}

// interfaceValues shows what an interface variable actually holds.
func interfaceValues() {
	// An interface value is a PAIR: (concrete type, value). %T prints the type part.
	// Assigning a concrete value to an interface COPIES it, like every assignment in Go.
	r := Rect{W: 1, H: 1}
	var byValue Shape = r // the interface holds its own copy of r
	r.W = 100             // so changing r afterwards leaves that copy alone:

	fmt.Println("  stored a copy:   ", byValue, "while r is now", r)

	// Storing a POINTER copies only the pointer: the variable and the interface now ALIAS the
	// same struct, and a change through one is visible through the other. (*Rect has all of
	// Rect's methods, so a *Rect is a Shape too.)
	rp := &Rect{W: 1, H: 1}
	var byPointer Shape = rp
	rp.W = 100
	fmt.Println("  stored a pointer:", byPointer, "- the same struct, seen through both")

	// Copying an interface copies the pair. With a pointer inside, both copies still alias one
	// struct - the same story as copying a slice header in section 006.
	other := byPointer
	rp.H = 5
	fmt.Println("  a copy of the interface sees it too:", other)
	// Rule of thumb: store a pointer when the methods must modify the value or it is large -
	// and remember that everyone holding the interface then shares it. Store a value when it
	// is small and only read.

	// The NIL trap: an interface is nil only when BOTH parts are nil. A nil POINTER stored in
	// an interface gives (type *Rect, value nil), and that interface is NOT nil.
	var nothing *Rect
	var s Shape = nothing
	fmt.Printf("  nothing == nil: %t, but s == nil: %t (s holds a %T)\n", nothing == nil, s == nil, s)
	// It bites hardest with errors - see "the typed-nil trap" in section 009.

	// any is a type ALIAS, declared as type any = interface{}: two names for ONE type, so no
	// conversion is ever needed between them. (A declaration without "=", like
	// type Celsius float64 in section 007, creates a NEW, distinct type instead.)
	var x interface{} = 42
	var y any = x
	fmt.Printf("  any and interface{} are the same type: %v (%T)\n", y, y)
}

// embeddingAndShadowing shows how interfaces and structs are built from smaller pieces.
func embeddingAndShadowing() {
	// 1. An interface can EMBED other interfaces and then has all their methods. This is how
	//    the standard library builds io.ReadWriter out of io.Reader and io.Writer.
	var store KeyValueStore = newMemoryStore()
	store.Put("lang", "Go")
	v, _ := store.Get("lang")
	fmt.Println("  KeyValueStore = Getter + Putter:", v)
	var readOnly Getter = store // a bigger interface converts to a smaller one implicitly
	_, found := readOnly.Get("missing")
	fmt.Println("  narrowed to a Getter, a missing key is found:", found)

	// 2. A STRUCT can embed an interface. The struct then satisfies that interface - calls are
	//    forwarded to the embedded value - and it can SHADOW a method with its own version.
	//    This is the decorator pattern: add behaviour around an existing implementation.
	counting := &CountingNotifier{Notifier: &FakeNotifier{}}
	svc := OrderService{notifier: counting} // a *CountingNotifier IS a Notifier
	svc.PlaceOrder("lamp")
	svc.PlaceOrder("desk")
	fmt.Println("  the decorating notifier counted", counting.Count, "notifications")
	// counting.Notify runs CountingNotifier.Notify: the outer method shadows the embedded one,
	// which is still reachable by its full name, counting.Notifier.Notify. Careful: every
	// method you do NOT shadow goes to the embedded value, so it must not be nil.

	// 3. Shadowing is NOT overriding as in Java: there is no virtual dispatch. When a method
	//    of the EMBEDDED type calls another method, it calls its own version, never the outer.
	g := LoudGreeter{Greeter{Name: "Ann"}}
	fmt.Println("  g.Hello():  ", g.Hello())   // the outer method, shadowing Greeter.Hello
	fmt.Println("  g.Welcome():", g.Welcome()) // promoted; inside, it calls Greeter.Hello

	// 4. Shadowing a VARIABLE on purpose: in a type switch, switch x := v.(type) declares a new
	//    x in every case, with that case's type (see describe below). People often reuse the
	//    name - switch v := v.(type) - and there shadowing is the idiom, not a bug (section 001).
}

// Getter and Putter are tiny interfaces; KeyValueStore combines them by embedding.
type Getter interface {
	Get(key string) (string, bool)
}

type Putter interface {
	Put(key, value string)
}

type KeyValueStore interface {
	Getter
	Putter
}

// memoryStore satisfies KeyValueStore, and with it Getter and Putter.
type memoryStore struct{ data map[string]string }

func newMemoryStore() *memoryStore { return &memoryStore{data: make(map[string]string)} }

func (m *memoryStore) Get(key string) (string, bool) {
	v, ok := m.data[key]
	return v, ok
}

func (m *memoryStore) Put(key, value string) { m.data[key] = value }

// CountingNotifier embeds a Notifier and shadows its Notify method.
type CountingNotifier struct {
	Notifier // embedded interface: every method we do not shadow is forwarded to it
	Count    int
}

func (c *CountingNotifier) Notify(to, message string) error {
	c.Count++
	return c.Notifier.Notify(to, message) // call the shadowed method by its full name
}

// Greeter and LoudGreeter show that embedding is not inheritance.
type Greeter struct{ Name string }

func (g Greeter) Hello() string   { return "hello, " + g.Name }
func (g Greeter) Welcome() string { return g.Hello() + " - welcome!" } // always Greeter.Hello

type LoudGreeter struct{ Greeter }

func (l LoudGreeter) Hello() string { return strings.ToUpper(l.Greeter.Hello()) }

// interfacesAtRuntime shows how to get the concrete value back out of an interface.
func interfacesAtRuntime() {
	// any (another name for interface{}) has no methods, so EVERY type satisfies it. That is
	// how fmt.Println accepts anything. The price is type safety - prefer concrete types,
	// small interfaces or generics (010), and use any only when the type really is unknown.
	values := []any{42, "hello", 3.5, Circle{R: 1}, nil}

	// A TYPE SWITCH branches on the concrete type stored in the interface.
	for _, v := range values {
		fmt.Println("  describe:", describe(v))
	}

	// A TYPE ASSERTION x.(T) extracts the concrete value. Use the comma-ok form: the
	// single-result form s.(Rect) PANICS when s holds something else.
	var s Shape = Circle{R: 3}
	if c, ok := s.(Circle); ok {
		fmt.Println("  s is a Circle with radius", c.R)
	}
	if _, ok := s.(Rect); !ok {
		fmt.Println("  s is not a Rect")
	}

	// You can also ask "does this value have a method?" by asserting to another interface.
	if st, ok := s.(fmt.Stringer); ok {
		fmt.Println("  s can describe itself:", st.String())
	}
}

func describe(v any) string {
	switch x := v.(type) { // in each case, x has that case's type
	case int:
		return fmt.Sprintf("int %d, doubled: %d", x, x*2)
	case string:
		return fmt.Sprintf("string %q of length %d", x, len(x))
	case Shape: // a case can be an interface: any type with Area() matches
		return fmt.Sprintf("a Shape with area %.2f", x.Area())
	case nil:
		return "nil - no value at all"
	default:
		return fmt.Sprintf("something else: %T", x)
	}
}

// ============================================================================================
// 009 - errors
// ============================================================================================

// errorHandling covers how Go reports failures and how callers react to them.
func errorHandling() {
	fmt.Println("--- 009 errors ---")
	errorBasics()   // errors are values: return, check, add context
	kindsOfErrors() // plain, sentinel or custom type - and how callers inspect each
	errorPitfalls() // typed nil, handling twice, errors.Join, deferred Close, panic

	// More: basics/009_errors
}

// errorBasics shows the pattern you will write every day.
func errorBasics() {
	// error is a built-in interface with a single method:
	//
	//	type error interface { Error() string }
	//
	// A function that can fail returns an error as its LAST result: nil means success,
	// anything else describes the failure. There are no exceptions and no try/catch - a failure
	// travels back through ordinary return values, so every place that can fail is visible.

	// The pattern: call, check immediately, return early. The happy path stays unindented.
	port, err := parsePort("8080")
	fmt.Println("  parsePort(8080):", port, err)

	// Every layer adds CONTEXT with fmt.Errorf and %w, so the final message reads like a trail
	// from what the program was doing down to the original cause:
	_, err = startServer(map[string]string{"port": "80a"})
	fmt.Println("  startServer:", err)

	// Conventions:
	//   - when err != nil, do not use the other results: they are zero values (or partial)
	//   - messages are lowercase, without a trailing period or an "error:" prefix, because
	//     they get chained into longer messages like the one above
	//   - add context that the inner function does not know (WHICH port, WHICH file)
}

func parsePort(s string) (int, error) {
	p, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("parse port %q: %w", s, err) // %w keeps err inside the new error
	}
	if p < 1 || p > 65535 {
		return 0, fmt.Errorf("parse port: %d is out of range 1-65535", p) // a brand-new error
	}
	return p, nil
}

// startServer reads its configuration. Each step can fail, and each failure returns at once.
func startServer(config map[string]string) (int, error) {
	raw, ok := config["port"]
	if !ok {
		return 0, errors.New("start server: no port configured")
	}
	port, err := parsePort(raw)
	if err != nil {
		return 0, fmt.Errorf("start server: %w", err)
	}
	return port, nil
}

// kindsOfErrors shows the three kinds of errors and how a caller inspects each one.
func kindsOfErrors() {
	// Which kind to return depends on what the CALLER needs to know:
	//
	//   the caller only needs "it failed"       -> errors.New / fmt.Errorf  (an opaque error)
	//   the caller must react to ONE condition  -> a sentinel: var ErrNotFound = errors.New(...)
	//   the caller needs DETAILS (field, code)  -> a custom type: type ValidationError struct{...}
	//
	// Start with opaque errors. A sentinel or an error type that you export becomes part of
	// your API, so add one only when a caller really needs it.

	// SENTINEL: a package-level value. errors.Is walks the whole chain of wrapped errors.
	_, err := loadUser(42)
	fmt.Println("  wrapped sentinel:", err)
	fmt.Println("  errors.Is(err, ErrNotFound):", errors.Is(err, ErrNotFound))
	fmt.Println("  err == ErrNotFound:         ", err == ErrNotFound, "<- == sees only the outer error")

	// CUSTOM TYPE: carries data. errors.As finds the first error of that type in the chain and
	// stores it in your variable - note the &, As needs a pointer to fill in.
	err = register(-5)
	var ve *ValidationError
	if errors.As(err, &ve) {
		fmt.Printf("  errors.As: field=%q problem=%q\n", ve.Field, ve.Problem)
	}
	// Go 1.26 added a generic form that needs no variable declared up front:
	if ve, ok := errors.AsType[*ValidationError](err); ok {
		fmt.Println("  errors.AsType: field", ve.Field)
	}

	// %w versus %v: %w WRAPS, so callers can still find the cause with Is/As. %v only copies
	// the cause's TEXT into the new message - use it to hide a detail callers should not use.
	hidden := fmt.Errorf("load user: %v", ErrNotFound)
	fmt.Printf("  wrapped with %%v: same text, but errors.Is(hidden, ErrNotFound) = %t - the link is gone\n", errors.Is(hidden, ErrNotFound))

	// Never inspect the text, as in strings.Contains(err.Error(), "not found"): messages
	// change, values and types are the stable contract.
}

// ErrNotFound is a sentinel error: a package-level value callers compare against.
var ErrNotFound = errors.New("not found")

func findUser(id int) (string, error) {
	if id != 1 {
		return "", ErrNotFound
	}
	return "Ann", nil
}

func loadUser(id int) (string, error) {
	name, err := findUser(id)
	if err != nil {
		return "", fmt.Errorf("load user %d: %w", id, err)
	}
	return name, nil
}

// ValidationError is a custom error type: it carries data that callers can inspect.
type ValidationError struct {
	Field, Problem string
}

func (e *ValidationError) Error() string { return e.Field + ": " + e.Problem }

func register(age int) error {
	if age < 0 {
		return fmt.Errorf("register: %w", &ValidationError{Field: "age", Problem: "must not be negative"})
	}
	return nil
}

// errorPitfalls shows the mistakes everybody makes once.
func errorPitfalls() {
	// 1. The TYPED-NIL trap. Returning a nil POINTER of a concrete error type gives a NON-nil
	//    error: the interface holds (type *ValidationError, value nil), see section 008.
	if err := validateBuggy(30); err != nil {
		fmt.Printf("  validateBuggy(30) != nil although nothing failed - it holds a %T\n", err)
	}
	if err := validate(30); err == nil {
		fmt.Println("  validate(30) == nil - on success, return a literal nil")
	}

	// 2. Handle an error ONCE: either deal with it (log it, retry, fall back to a default) or
	//    return it - not both. Logging AND returning prints one failure at every level.
	//	if err != nil {
	//		log.Println(err) // handled here...
	//		return err       // ...and again by the caller: the same failure logged twice
	//	}

	// 3. Several errors at once: errors.Join combines them (nil ones are skipped), and Is/As
	//    still see every one. Handy for validation that should report ALL problems.
	err := validateForm("", -1)
	fmt.Println("  errors.Join:", strings.ReplaceAll(err.Error(), "\n", " | "))
	fmt.Println("  errors.Is(err, ErrEmptyName):", errors.Is(err, ErrEmptyName))

	// 4. Ignoring an error must be a visible decision: _ = f.Close() says "I chose to", a bare
	//    f.Close() looks like an accident. For a file you WRITE, Close can be the only signal
	//    that the data never reached the disk - a NAMED result lets a deferred Close report it.
	fmt.Println("  saveReport:", saveReport(failingFile{}))

	// 5. panic is not error handling. Return errors for everything that can go wrong at run
	//    time (bad input, a missing file, the network); panic only for bugs (section 005).
}

func validateBuggy(age int) error {
	var err *ValidationError // a nil pointer of a CONCRETE type - never declare the result like this
	if age < 0 {
		err = &ValidationError{Field: "age", Problem: "must not be negative"}
	}
	return err // becomes (type *ValidationError, value nil): not a nil error!
}

func validate(age int) error {
	if age < 0 {
		return &ValidationError{Field: "age", Problem: "must not be negative"}
	}
	return nil // a literal nil: the interface is really nil
}

// ErrEmptyName is reported by validateForm.
var ErrEmptyName = errors.New("name is empty")

// validateForm collects every problem instead of stopping at the first one.
func validateForm(name string, age int) error {
	var errs []error
	if name == "" {
		errs = append(errs, ErrEmptyName)
	}
	if age < 0 {
		errs = append(errs, &ValidationError{Field: "age", Problem: "must not be negative"})
	}
	return errors.Join(errs...) // nil when there were no problems
}

// failingFile pretends to be a file whose Close fails, e.g. because the disk is full.
type failingFile struct{}

func (failingFile) Write(p []byte) (int, error) { return len(p), nil }
func (failingFile) Close() error                { return errors.New("disk full") }

// saveReport writes a report and closes w. The result is NAMED (err), so the deferred function
// can still change it after the return statement - here, to report a failed Close.
func saveReport(w io.WriteCloser) (err error) {
	defer func() {
		if closeErr := w.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("save report: close: %w", closeErr)
		}
	}()
	_, err = fmt.Fprintln(w, "report") // = not := : assign the named result, do not shadow it
	if err != nil {
		return fmt.Errorf("save report: %w", err)
	}
	return nil
}

// ============================================================================================
// 010 - generics
// ============================================================================================

// generics covers type parameters, constraints and generic types.
func generics() {
	fmt.Println("--- 010 generics ---")
	whyGenerics()          // the problem generics solve, and the syntax
	constraintsExplained() // what a constraint allows you to do with T
	genericTypes()         // Stack[T], generics in the standard library, when not to use them

	// More: basics/010_generics
}

// whyGenerics shows one function written once for many types.
func whyGenerics() {
	// The problem: without generics you write the same function once per type...
	fmt.Println("  SumInts:", SumInts([]int{1, 2, 3}), " SumFloats:", SumFloats([]float64{1.5, 2.5}))

	// ...or take []any and lose type safety. A TYPE PARAMETER writes it once: Sum[T Number]
	// reads "for any type T that satisfies the constraint Number".
	fmt.Println("  Sum(ints):", Sum([]int{1, 2, 3}), " Sum(floats):", Sum([]float64{1.5, 2.5}))

	// T is usually INFERRED from the arguments. You can also write it out:
	fmt.Println("  Sum[int64]:", Sum[int64]([]int64{10, 20}))

	// The compiler checks every call: string is not in Number.
	//	Sum([]string{"a"}) // ERROR: string does not satisfy Number (string missing in ~int | ~int64 | ~float64)

	// Generic helpers work for any element type and stay fully type-checked.
	words := []string{"go", "is", "fun"}
	lengths := Map(words, func(s string) int { return len(s) }) // []string -> []int
	long := Filter(words, func(s string) bool { return len(s) > 2 })
	fmt.Println("  Map:", lengths, " Filter:", long)
}

func SumInts(nums []int) int {
	total := 0
	for _, n := range nums {
		total += n
	}
	return total
}

func SumFloats(nums []float64) float64 {
	total := 0.0
	for _, n := range nums {
		total += n
	}
	return total
}

// Number is a CONSTRAINT: an interface that lists the allowed types (a "type set").
// ~int means "int, or any type whose underlying type is int" - such as type Age int.
type Number interface {
	~int | ~int64 | ~float64
}

// Sum is the generic version of SumInts and SumFloats.
func Sum[T Number](nums []T) T {
	var total T // the zero value of T: 0 for every Number
	for _, n := range nums {
		total += n // allowed because every type in Number supports +
	}
	return total
}

// Map has two type parameters: the input element type T and the output element type U.
func Map[T, U any](items []T, f func(T) U) []U {
	out := make([]U, 0, len(items))
	for _, v := range items {
		out = append(out, f(v))
	}
	return out
}

func Filter[T any](items []T, keep func(T) bool) []T {
	var out []T
	for _, v := range items {
		if keep(v) {
			out = append(out, v)
		}
	}
	return out
}

// constraintsExplained shows that the constraint decides what the code may do with T.
func constraintsExplained() {
	// The constraints you will use most:
	//   any          - every type, but only operations every type has (assign, pass, store)
	//   comparable   - adds == and != (needed for map keys and searching)
	//   cmp.Ordered  - adds < <= > >= (integers, floats, strings)
	//   your own     - a union of types like Number, or an interface with methods
	fmt.Println("  IndexOf:", IndexOf([]string{"a", "b", "c"}, "c"), " Max:", Max(3, 7), Max("apple", "pear"))

	// With only any, == is not allowed - not every type can be compared.
	//	func equal[T any](a, b T) bool { return a == b } // ERROR: invalid operation: a == b (incomparable types in type set)

	// ~ in action: Age is a new type, but its underlying type is int, so it satisfies Number.
	type Age int
	fmt.Println("  Sum of []Age:", Sum([]Age{30, 40}))

	// A constraint can also require METHODS - generics and interfaces working together. Labels
	// accepts []Circle directly; a func taking []fmt.Stringer would force you to copy the
	// []Circle into a new []fmt.Stringer first.
	fmt.Println("  Labels:", Labels([]Circle{{R: 1}, {R: 2}}))
}

// IndexOf needs == on T, so T must be comparable. (slices.Index does the same.)
func IndexOf[T comparable](items []T, target T) int {
	for i, v := range items {
		if v == target {
			return i
		}
	}
	return -1
}

// Max needs > on T, so T must be cmp.Ordered. (The built-in max does the same.)
func Max[T cmp.Ordered](a, b T) T {
	if a > b {
		return a
	}
	return b
}

// Labels accepts a slice of any type that has a String() string method.
func Labels[T fmt.Stringer](items []T) []string {
	return Map(items, func(v T) string { return v.String() })
}

// genericTypes shows a generic container, the standard library, and when to stop.
func genericTypes() {
	// Types can have type parameters too: one Stack implementation for every element type.
	var ints Stack[int] // the zero value is ready to use
	ints.Push(1)
	ints.Push(2)
	top, _ := ints.Pop()
	fmt.Println("  Stack[int]: popped", top, " size now", ints.Size())

	// Go 1.27: a method may declare its OWN type parameters. Transform (below) turns a
	// Stack[T] into a Stack[U]; U is inferred from the function passed in. One limit remains:
	// a generic method cannot satisfy an interface method.
	ints.Push(20)
	labels := ints.Transform(func(n int) string { return "#" + strconv.Itoa(n) })
	fmt.Println("  generic method Transform: Stack[int] -> Stack[string], size", labels.Size())

	var names Stack[string]
	names.Push("Ann")
	//	names.Push(42) // ERROR: cannot use 42 (untyped int constant) as string value in argument to names.Push
	name, _ := names.Pop()
	_, ok := names.Pop() // the stack is empty now
	fmt.Println("  Stack[string]: popped", name, " popping again gives ok =", ok)

	// Mostly you will USE generics rather than write them. The standard library has many:
	nums := []int{5, 2, 8, 1}
	fmt.Println("  slices.Max:", slices.Max(nums), " slices.Contains:", slices.Contains(nums, 8), " slices.Index:", slices.Index(nums, 2))
	prices := map[string]float64{"tea": 4.5, "coffee": 7}
	fmt.Println("  sorted keys:", slices.Sorted(maps.Keys(prices)), " min/max built-ins:", min(3, 1, 2), max(2.5, 1))

	// When NOT to use generics: if the code only calls METHODS on a value, a plain interface
	// is simpler - totalArea in section 008 needs no type parameter. Reach for generics when
	// the same code works for many types AND the caller cares about the concrete type it
	// gets back: containers (Stack), Map/Filter, Sum, Max.
}

// Stack is a generic type; T is fixed when you write Stack[int], Stack[string], ...
type Stack[T any] struct {
	items []T
}

// The receiver names the type parameter again: (s *Stack[T]).
func (s *Stack[T]) Push(v T) { s.items = append(s.items, v) }

// Pop returns the top element and true, or the zero value of T and false when the stack is empty.
func (s *Stack[T]) Pop() (T, bool) {
	var zero T
	if len(s.items) == 0 {
		return zero, false
	}
	top := s.items[len(s.items)-1]
	s.items = s.items[:len(s.items)-1]
	return top, true
}

func (s *Stack[T]) Size() int { return len(s.items) }

// Transform is a GENERIC METHOD (Go 1.27): besides the receiver's T it has its own type
// parameter U. Before 1.27 this had to be a package-level function.
func (s *Stack[T]) Transform[U any](f func(T) U) *Stack[U] {
	out := &Stack[U]{items: make([]U, 0, len(s.items))}
	for _, v := range s.items {
		out.Push(f(v))
	}
	return out
}

// ============================================================================================
// 011 - concurrency
// ============================================================================================

// concurrency covers goroutines, channels, select, shared state, cancellation and errors.
func concurrency() {
	fmt.Println("--- 011 concurrency ---")
	goroutineBasics()      // go, WaitGroup, panics in goroutines, leaks
	channelBasics()        // unbuffered and buffered, direction, close, comma-ok, deadlock
	selectAndTimeouts()    // waiting on several channels, timeouts, non-blocking select
	sharedState()          // Mutex and atomic counters, data races
	cancellation()         // context: deadlines, and how goroutines stop
	errorsFromGoroutines() // getting errors back from concurrent work
	workerPool()           // the most common pattern, putting it all together

	// More: basics/011_concurrency
}

// goroutineBasics shows how to start goroutines and wait for them.
func goroutineBasics() {
	// A GOROUTINE is a function running concurrently: put go in front of a call. Goroutines
	// are cheap (a few KB of stack that grows on demand), so thousands of them are normal.
	//
	// main does NOT wait for them. When main returns the program ends, and every goroutine
	// that is still running is killed. Wait explicitly: with a sync.WaitGroup or a channel.
	var wg sync.WaitGroup
	squares := make([]int, 5)
	for i := range 5 {
		wg.Go(func() { // wg.Go (Go 1.25) = wg.Add(1) + go + defer wg.Done(), in one call
			squares[i] = i * i // each goroutine writes its OWN element: no conflict
		})
		// Since Go 1.22 every iteration has a fresh i, so capturing it is safe. Older code
		// passes it as an argument instead - go func(i int) { ... }(i) - you will still see it.
	}
	wg.Wait() // block until every goroutine started with wg.Go has finished
	fmt.Println("  squares computed in goroutines:", squares)

	// A panic in ANY goroutine crashes the WHOLE program, and a recover in main cannot catch
	// it. A goroutine that might panic needs its own deferred recover.
	wg.Go(func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Println("  a goroutine recovered from its own panic:", r)
			}
		}()
		var m map[string]int
		m["boom"] = 1 // writing to a nil map panics
	})
	wg.Wait()

	// A goroutine LEAK is a goroutine blocked forever, e.g. sending to a channel that nobody
	// reads any more. It is never cleaned up. So before you write go, know how that goroutine
	// will END: its work runs out, a channel it reads is closed, or its context is cancelled.
}

// channelBasics shows how goroutines pass values to each other.
func channelBasics() {
	// CHANNELS pass values between goroutines: "share memory by communicating". On an
	// UNBUFFERED channel a send waits until a receiver takes the value - both sides meet.
	ch := make(chan string)
	go func() { ch <- "ping" }()
	fmt.Println("  received:", <-ch)

	// A BUFFERED channel holds up to N values; a send blocks only while the buffer is full.
	buf := make(chan int, 2)
	buf <- 1
	buf <- 2 // still no receiver needed: the buffer has room for two
	fmt.Println("  buffered:", <-buf, <-buf, " len/cap now:", len(buf), cap(buf))

	// DIRECTION in a signature says who may send and who may receive, and the compiler holds
	// you to it: chan<- int is send-only, <-chan int is receive-only. A two-stage pipeline:
	fmt.Println("  pipeline generate -> squareAll -> collect:", collect(squareAll(generate(1, 2, 3, 4))))

	// CLOSING: the SENDER closes a channel to say "no more values". After that:
	//   - range over the channel ends once the remaining values are received
	//   - a receive returns at once with the zero value; the comma-ok form tells you why
	//   - a send PANICS, and so does a second close - that is why only the sender closes
	results := make(chan int, 1)
	results <- 7
	close(results)
	v, ok := <-results
	fmt.Println("  after close:", v, ok) // 7 true - values already sent are still delivered
	v, ok = <-results
	fmt.Println("  and then:   ", v, ok) // 0 false - closed and empty

	// DEADLOCK: when every goroutine is blocked, the runtime stops the program. Classic case:
	// an unbuffered send in main with no other goroutine to receive it.
	//	ch := make(chan int)
	//	ch <- 1 // fatal error: all goroutines are asleep - deadlock!
}

// generate sends nums one by one and then closes the channel. Callers get a receive-only
// channel, so they cannot send into it or close it by mistake.
func generate(nums ...int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out) // the sender closes, when it is done
		for _, n := range nums {
			out <- n
		}
	}()
	return out
}

// squareAll receives numbers from in and sends their squares; it ends when in is closed.
func squareAll(in <-chan int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for n := range in {
			out <- n * n
		}
	}()
	return out
}

// collect receives everything until the channel is closed.
func collect(in <-chan int) []int {
	var all []int
	for n := range in {
		all = append(all, n)
	}
	return all
}

// selectAndTimeouts shows waiting on more than one thing at a time.
func selectAndTimeouts() {
	// SELECT waits on several channel operations and runs the first one that is ready (a
	// random one if several are ready at once). Together with time.After it is a timeout.
	slow := make(chan string, 1) // buffered, so the late sender can finish instead of leaking
	go func() {
		time.Sleep(100 * time.Millisecond)
		slow <- "result"
	}()
	select {
	case msg := <-slow:
		fmt.Println("  got:", msg)
	case <-time.After(10 * time.Millisecond):
		fmt.Println("  select: timeout - the result took too long")
	}

	// A default case makes select non-blocking: take a value if one is waiting, else move on.
	jobs := make(chan int)
	select {
	case j := <-jobs:
		fmt.Println("  job", j)
	default:
		fmt.Println("  select with default: nothing is waiting, carry on")
	}
}

// sharedState shows how to protect data that several goroutines modify.
func sharedState() {
	// Without protection, count++ from many goroutines is a DATA RACE: updates get lost and
	// the result is unpredictable. go run -race and go test -race find races - use them.
	var (
		wg    sync.WaitGroup
		mu    sync.Mutex
		count int
	)
	for range 100 {
		wg.Go(func() {
			mu.Lock()
			count++ // only one goroutine at a time gets past Lock
			mu.Unlock()
		})
	}
	wg.Wait()
	fmt.Println("  counter protected by a mutex:", count)

	// For a single number, sync/atomic is simpler: atomic.Int64 has Add, Load and Store.
	var hits atomic.Int64
	for range 100 {
		wg.Go(func() { hits.Add(1) })
	}
	wg.Wait()
	fmt.Println("  atomic counter:", hits.Load())

	// Which tool when:
	//   a Mutex  - several fields that change together (Lock, change, Unlock; often with defer)
	//   atomic   - one counter or flag
	//   channels - handing work or results from one goroutine to another
	// Never copy a struct that contains a Mutex - share it through a pointer.
}

// cancellation shows context: deadlines, and telling goroutines to stop.
func cancellation() {
	// CONTEXT carries a deadline or a cancellation signal into goroutines and API calls. By
	// convention it is the first parameter: func Do(ctx context.Context, ...).
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel() // always call cancel: it releases the context's resources
	fmt.Println("  slowOperation:", slowOperation(ctx))

	// Cancellation is also how goroutines END. naturals would count forever, but it stops as
	// soon as the consumer cancels - so it does not leak.
	genCtx, stop := context.WithCancel(context.Background())
	var first []int
	for n := range naturals(genCtx) {
		first = append(first, n)
		if len(first) == 5 {
			break
		}
	}
	stop() // the generator sees genCtx.Done(), returns and closes its channel
	fmt.Println("  first 5 values of an endless generator:", first)
}

// slowOperation pretends to work for a second, but gives up as soon as ctx is done.
func slowOperation(ctx context.Context) error {
	select {
	case <-time.After(time.Second):
		return nil
	case <-ctx.Done():
		return ctx.Err() // context.DeadlineExceeded or context.Canceled
	}
}

// naturals sends 1, 2, 3, ... until ctx is cancelled.
func naturals(ctx context.Context) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for n := 1; ; n++ {
			select {
			case out <- n:
			case <-ctx.Done(): // without this case the goroutine would block on the send forever
				return
			}
		}
	}()
	return out
}

// errorsFromGoroutines shows how to get errors back from concurrent work.
func errorsFromGoroutines() {
	// A goroutine cannot return anything to the code that started it, not even an error. Hand
	// results back yourself - the simplest way is one slot per goroutine, joined at the end.
	inputs := []string{"10", "x", "30", "4y"}
	values := make([]int, len(inputs))
	errs := make([]error, len(inputs))
	var wg sync.WaitGroup
	for i, in := range inputs {
		wg.Go(func() {
			values[i], errs[i] = strconv.Atoi(in) // own slots: no mutex needed
		})
	}
	wg.Wait()
	if err := errors.Join(errs...); err != nil { // Join skips the nil errors
		fmt.Println("  failed:", strings.ReplaceAll(err.Error(), "\n", " | "))
	}
	fmt.Println("  values:", values)

	// golang.org/x/sync/errgroup wraps this pattern: g.Go(func() error { ... }), and g.Wait()
	// returns the first error. errgroup.WithContext also cancels the others when one fails.
}

// workerPool is the most common pattern: N workers read jobs from one channel and send
// results to another.
func workerPool() {
	jobs := make(chan int)
	results := make(chan int)

	var wg sync.WaitGroup
	for range 3 { // 3 workers
		wg.Go(func() {
			for j := range jobs { // each worker takes jobs until jobs is closed
				results <- j * j
			}
		})
	}

	go func() { // producer: send the jobs, then close
		for j := range 6 {
			jobs <- j
		}
		close(jobs)
	}()

	go func() { // close results once every worker has finished
		wg.Wait()
		close(results)
	}()

	var all []int
	for r := range results {
		all = append(all, r)
	}
	slices.Sort(all) // results arrive in any order
	fmt.Println("  worker pool results:", all)
}

// ============================================================================================
// 012 - iterators and collections
// ============================================================================================

// iteratorsAndCollections covers range-over-func iterators and the slices/maps/cmp packages.
func iteratorsAndCollections() {
	fmt.Println("--- 012 iterators and collections ---")

	// An ITERATOR is a function that pushes values into a yield callback, and range can loop
	// over it. The type is iter.Seq[V], which is just func(yield func(V) bool).
	fmt.Print("  countdown:")
	for n := range Countdown(3) {
		fmt.Print(" ", n)
	}
	fmt.Println()

	// yield returns false when the loop stops early (break), and the iterator must then
	// return. So even an infinite sequence is fine: the loop decides when to stop.
	fmt.Print("  fibonacci up to 50:")
	for n := range Fibonacci() {
		if n > 50 {
			break
		}
		fmt.Print(" ", n)
	}
	fmt.Println()
	fmt.Println("  slices.Collect(Countdown(5)):", slices.Collect(Countdown(5)))

	// iter.Seq2[K, V] yields PAIRS, like range over a slice (index, value) or a map (key,
	// value). The standard library returns them from slices.All, maps.All and others.
	for n, line := range Lines("first\nsecond\nthird") {
		fmt.Printf("  line %d: %s\n", n, line)
	}

	// Iterators compose LAZILY: FilterSeq (generic, below) wraps any iterator, and nothing
	// runs until a loop pulls values. That is why filtering an INFINITE sequence works.
	fmt.Print("  even Fibonacci numbers below 1000:")
	for n := range FilterSeq(Fibonacci(), func(n int) bool { return n%2 == 0 }) {
		if n >= 1000 {
			break // break stops FilterSeq, which stops Fibonacci - nothing keeps running
		}
		fmt.Print(" ", n)
	}
	fmt.Println()
	// An iterator may also defer cleanup (say, closing a file): the deferred call runs when
	// the loop ends, including on break.

	// The slices package - use it instead of hand-written loops.
	people := []Person{{"Ann", 31}, {"Bob", 25}, {"Cid", 40}}
	slices.SortFunc(people, func(a, b Person) int {
		return cmp.Compare(a.Age, b.Age) // negative, zero or positive: like a < b, a == b, a > b
	})
	fmt.Println("  sorted by age:", people)
	i := slices.IndexFunc(people, func(p Person) bool { return p.Name == "Cid" })
	fmt.Println("  Cid is at index", i)
	fmt.Println("  slices.Contains:", slices.Contains([]string{"a", "b"}, "b"), " slices.Equal:", slices.Equal([]int{1, 2}, []int{1, 2}))

	// The maps package. maps.Keys returns an ITERATOR, so wrap it in slices.Sorted (or
	// slices.Collect). Sorting the keys is the standard way to get a stable order from a map.
	stock := map[string]int{"pears": 3, "apples": 5, "kiwis": 0}
	fmt.Print("  stock in key order:")
	for _, k := range slices.Sorted(maps.Keys(stock)) {
		fmt.Printf(" %s=%d", k, stock[k])
	}
	fmt.Println()

	// More: basics/012_iterators_and_collections
}

type Person struct {
	Name string
	Age  int
}

// Countdown yields from, from-1, ..., 1.
func Countdown(from int) iter.Seq[int] {
	return func(yield func(int) bool) {
		for i := from; i >= 1; i-- {
			if !yield(i) {
				return // the loop was stopped with break: stop producing
			}
		}
	}
}

// Fibonacci yields 0, 1, 1, 2, 3, 5, ... forever (until the loop breaks).
func Fibonacci() iter.Seq[int] {
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

// Lines yields (line number, line) pairs, numbering from 1.
func Lines(text string) iter.Seq2[int, string] {
	return func(yield func(int, string) bool) {
		for i, line := range strings.Split(text, "\n") {
			if !yield(i+1, line) {
				return
			}
		}
	}
}

// FilterSeq yields only the values of seq for which keep returns true. It is generic, so it
// works with an iterator of any element type.
func FilterSeq[V any](seq iter.Seq[V], keep func(V) bool) iter.Seq[V] {
	return func(yield func(V) bool) {
		for v := range seq {
			if keep(v) && !yield(v) {
				return // the consumer stopped: stop pulling from seq as well
			}
		}
	}
}

// ============================================================================================
// 013 - standard library essentials
// ============================================================================================

// stdlib covers the packages you use every day.
func stdlib() {
	fmt.Println("--- 013 standard library essentials ---")

	// fmt: %v any value, %+v with field names, %T the type, %q quoted, %d integer, %s string,
	// %.2f float with 2 decimals, %6.2f also padded to width 6. Sprintf returns a string,
	// Fprintf writes to an io.Writer.
	p := Product{Name: "Pen", Price: 2.5, Tags: []string{"office"}}
	fmt.Printf("  %%v: %v\n  %%+v: %+v\n  %%T: %T\n", p, p, p)
	fmt.Printf("  %%q: %q  %%d: %d  %%6.2f: [%6.2f]\n", p.Name, 42, p.Price)
	label := fmt.Sprintf("%s costs %.2f", p.Name, p.Price)
	fmt.Println(" ", label)

	// strings: the everyday helpers.
	fmt.Println("  TrimSpace:", strings.TrimSpace("  hi  "), " ToUpper:", strings.ToUpper("go"), " Contains:", strings.Contains("golang", "lang"))
	parts := strings.Split("a,b,c", ",")
	fmt.Println("  Split:", parts, " Join:", strings.Join(parts, " | "))
	key, value, found := strings.Cut("name=Ann", "=") // split once around a separator
	fmt.Println("  Cut:", key, value, found)
	var sb strings.Builder // build strings in a loop with a Builder, not with +=
	for i := range 3 {
		fmt.Fprintf(&sb, "[%d]", i)
	}
	fmt.Println("  Builder:", sb.String())

	// time: a Duration is a typed number of nanoseconds. Layouts are written with the
	// REFERENCE DATE Mon Jan 2 15:04:05 2006 instead of YYYY-MM-DD: 01 is the month, 02 the
	// day, 15 the hour, 04 the minutes, 2006 the year.
	d := 90 * time.Minute
	fmt.Println("  duration:", d, " in hours:", d.Hours())
	start := time.Date(2026, time.March, 14, 9, 30, 0, 0, time.UTC)
	fmt.Println("  formatted:", start.Format("2006-01-02 15:04"), " +48h is a", start.Add(48*time.Hour).Weekday())
	end, err := time.Parse("2006-01-02", "2026-03-20")
	if err == nil {
		fmt.Println("  parsed:", end.Format("Jan 2, 2006"), " time until then:", end.Sub(start))
	}
	// For measurements: start := time.Now() ... elapsed := time.Since(start).

	// encoding/json: struct TAGS choose the JSON names. Only exported (Upper) fields are encoded.
	data, err := json.Marshal(p)
	fmt.Println("  json.Marshal:", string(data), err)
	var decoded Product
	err = json.Unmarshal([]byte(`{"name":"Book","price":12.5}`), &decoded) // pass a POINTER
	fmt.Printf("  json.Unmarshal: %+v %v\n", decoded, err)

	// Files: os.WriteFile / os.ReadFile for small files. For os.Open or os.Create, defer Close.
	dir, err := os.MkdirTemp("", "essentials")
	if err != nil {
		fmt.Println("  cannot create a temp dir:", err)
		return
	}
	defer os.RemoveAll(dir) // clean up when stdlib returns
	path := filepath.Join(dir, "product.json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		fmt.Println("  write failed:", err)
		return
	}
	content, err := os.ReadFile(path)
	fmt.Println("  read back from the file:", string(content), err)

	// Line by line: os.Open, defer Close, and a bufio.Scanner - it works over any io.Reader
	// (a file, a network connection, strings.NewReader). Scan returns false at the end AND on
	// an error, so check sc.Err() after the loop.
	notes := filepath.Join(dir, "notes.txt")
	_ = os.WriteFile(notes, []byte("buy milk\ncall Ann\nlearn Go\n"), 0o644)
	f, err := os.Open(notes)
	if err != nil {
		fmt.Println("  open failed:", err)
		return
	}
	defer f.Close() // a file we only read: ignoring the Close error is fine here
	sc := bufio.NewScanner(f)
	for n := 1; sc.Scan(); n++ {
		fmt.Printf("  notes.txt:%d %s\n", n, sc.Text())
	}
	if err := sc.Err(); err != nil {
		fmt.Println("  read failed:", err)
	}

	// A missing file: test with errors.Is(err, fs.ErrNotExist), never by matching the text.
	_, err = os.ReadFile(filepath.Join(dir, "missing.txt"))
	fmt.Println("  missing file -> errors.Is(err, fs.ErrNotExist):", errors.Is(err, fs.ErrNotExist))

	// log/slog: structured logs - a message plus key-value pairs. (ReplaceAttr removes the
	// time only so that this program prints the same output on every run.)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.Attr{}
			}
			return a
		},
	}))
	logger.Info("order placed", "item", "book", "quantity", 2)

	// More: basics/013_stdlib_essentials
}

type Product struct {
	Name  string   `json:"name"`
	Price float64  `json:"price"`
	Tags  []string `json:"tags,omitempty"` // omitempty: left out of the JSON when empty
}

// ============================================================================================
// 014 - testing
// ============================================================================================

// testingIntro explains where the tests are and how to run them.
func testingIntro() {
	fmt.Println("--- 014 testing ---")

	// Tests live in files ending in _test.go, next to the code and in the same package.
	// go test finds every func TestXxx(t *testing.T) and runs it. There is deliberately no
	// assertion library: an if plus t.Errorf("got %v, want %v", ...) is the idiom.
	//
	//	go test -v ./basics/000_essentials                   # all tests, verbose
	//	go test -run TestReverse -v ./basics/000_essentials  # only tests matching a pattern
	//	go test -bench . -benchmem ./basics/000_essentials   # benchmarks
	//	go test -race ./basics/000_essentials                # with the data race detector
	//	go test -cover ./basics/000_essentials               # coverage percentage
	//
	// The pieces of the testing package you need first (all used in main_test.go):
	//
	//	t.Errorf / t.Fatalf  report a failure and carry on / stop this test right away
	//	t.Run(name, func)    a subtest: one per table row, selectable with -run 'Test/name'
	//	t.Helper()           in a helper function: failures point at the CALLER's line
	//	t.TempDir()          a fresh directory, deleted automatically after the test
	//	t.Cleanup(func)      run something when the test ends (like defer, for the whole test)
	//	t.Parallel()         let this test run at the same time as other parallel tests
	//
	// Test the error paths as carefully as the happy path: check errors with errors.Is/As,
	// exactly like callers do. go test caches results; -count=1 forces a re-run.
	//
	// main_test.go tests Reverse (below), Sum and Stack (010), divide (005), startServer and
	// validateForm (009) and saveReport with a real file (013).
	fmt.Printf("  Reverse(%q) = %q - see main_test.go\n", "żółw", Reverse("żółw"))

	// More: basics/014_testing
}

// Reverse returns s with its characters in reverse order. It reverses runes, not bytes, so
// multi-byte characters such as ż survive - a test in main_test.go checks exactly that.
func Reverse(s string) string {
	r := []rune(s)
	slices.Reverse(r)
	return string(r)
}
