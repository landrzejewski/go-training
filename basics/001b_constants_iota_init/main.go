// Command 001b_constants_iota_init is module 001b of the Go course: constants, iota, package-level
// variables and init functions.
//
// It continues module 001a. Constants live at compile time and follow their own precision rules;
// package-level variables and init functions run before main, in an order worth knowing.
//
//	go run ./basics/001b_constants_iota_init
package main

import (
	"fmt"
	"math"
	"time"
)

func main() {
	constants()        // typed vs untyped, arbitrary precision, what cannot be constant
	iotaAndEnums()     // iota, the enum idiom, bit flags, when iota is the wrong tool
	packageVariables() // package-level vars initialise in dependency order
	initFunctions()    // init(), the startup order, blank imports
}

// constants shows typed and untyped constants and why untyped ones adapt to their context.
func constants() {
	fmt.Println("--- constants ---")

	fmt.Printf("currentYear=%d taxRate=%.2f secondsPerDay=%d greetingLen=%d\n",
		currentYear, taxRate, secondsPerDay, greetingLen)

	// An UNTYPED constant is an ideal number that gets a type only where one is needed,
	// so the same constant becomes three types with no conversion.
	var asInt int = currentYear
	var asInt64 int64 = currentYear
	var asFloat float64 = currentYear
	fmt.Printf("one untyped constant, three types: %T %T %T\n", asInt, asInt64, asFloat)

	// math.Pi is untyped, so it fits both float widths without a cast.
	var pi32 float32 = math.Pi
	var pi64 float64 = math.Pi
	fmt.Printf("math.Pi as float32=%.7f as float64=%.15f\n", pi32, pi64)

	// With no context (:=, an any parameter) an untyped constant takes its DEFAULT type:
	// int, rune, float64, complex128, string or bool.
	fmt.Printf("default types: %T %T %T %T\n", currentYear, 0.5, 'x', "s")

	// A TYPED constant obeys the usual no-implicit-conversion rule. Prefer untyped constants
	// unless you want that constraint.
	//	var n int = taxRate // ERROR: cannot use taxRate (constant 0.23 of type float64) as int value in variable declaration

	// Untyped constants have at least 256 bits of precision, so intermediate results do not
	// overflow; a constant only has to fit when it lands in a typed destination.
	const backDown = huge >> 190
	fmt.Println("1<<200 >> 190 =", backDown)
	//	var overflow int64 = huge // ERROR: cannot use huge (untyped int constant 1606938044258990275541962092341162602522202993782792835301376) as int64 value in variable declaration (overflows)

	// This is why durations compose naturally: the untyped 60 becomes a time.Duration...
	timeout := 60 * time.Second
	fmt.Println("60 * time.Second =", timeout)
	// ...while an int VARIABLE already has a type:
	//	n := 60
	//	bad := n * time.Second // ERROR: invalid operation: n * time.Second (mismatched types int and time.Duration)

	// Only booleans, numbers, runes and strings can be constant. No constant slices, maps,
	// structs or pointers - those need var, which anyone in the package can then modify.
	//	const list = []int{1, 2, 3}    // ERROR: []int{…} (value of type []int) is not constant
	//	const table = map[string]int{} // ERROR: map[string]int{} (value of type map[string]int) is not constant
	//	const now = time.Now()         // ERROR: time.Now() (value of struct type time.Time) is not constant

	// Inside a const block an entry with no expression repeats the previous expression -
	// the rule that makes iota work (next topic).
	const (
		a = 10
		b // also 10
		c // also 10
	)
	fmt.Println("omitted expressions repeat the previous one:", a, b, c)

	// const works at package level and inside functions, and an unused constant is not an
	// error - unlike an unused variable.
	const unusedIsFine = "no compile error here"
}

const (
	currentYear           = 2025         // untyped
	taxRate       float64 = 0.23         // typed: bound to float64
	secondsPerDay         = 60 * 60 * 24 // evaluated at compile time
	huge                  = 1 << 200     // fits no runtime integer type - fine while untyped
	greetingLen           = len("hello") // len of a constant string is itself constant
)

// iotaAndEnums shows the enum idiom: a defined type, a const block and iota.
func iotaAndEnums() {
	fmt.Println("\n--- iota and enums ---")

	// Go has no enum type. weekday has a String method, so %v prints the name.
	fmt.Printf("sunday=%v(%d) wednesday=%v(%d) saturday=%v(%d)\n",
		sunday, sunday, wednesday, wednesday, saturday, saturday)

	// The set is not closed: any int converts to a weekday.
	fmt.Println("an enum is just an int, so this is legal:", weekday(42))

	// The size constants are untyped, so through Printf's ...any they take their default type int.
	fmt.Printf("kb=%d mb=%d gb=%d tb=%d\n", kb, mb, gb, tb)

	// Bit flags: combine with |, test with &, clear with &^ (AND NOT, module 003).
	perms := permRead | permWrite
	fmt.Printf("perms=%03b readable=%t writable=%t executable=%t\n",
		perms, perms&permRead != 0, perms&permWrite != 0, perms&permExecute != 0)
	perms &^= permWrite
	fmt.Printf("after clearing write: %03b writable=%t\n", perms, perms&permWrite != 0)

	// iota is the index of the current LINE (ConstSpec) in the block, starting at 0 and reset
	// in every const block. It counts lines, not identifiers.
	const (
		zero         = iota            // 0
		one                            // 1
		_                              // 2 - skipped, but the line still advances iota
		three                          // 3
		fourA, fourB = iota, iota * 10 // one line, so both see 4: 4 and 40
		fiveA, fiveB                   // the whole expression repeats: 5 and 50
	)
	fmt.Println("iota counts lines:", zero, one, three, fourA, fourB, fiveA, fiveB)

	// iota is only right for consecutive values whose numbers are arbitrary. Meaningful,
	// non-consecutive values (HTTP codes, errnos, wire tags) are written out literally:
	// `100 + iota` would produce 100/101/102, which are not the codes you meant.
	fmt.Printf("status codes: %d=%v %d=%v %d=%v\n",
		statusOK, statusOK, statusNoContent, statusNoContent, statusNotFound, statusNotFound)

	// iota also bakes ordinals into your API: inserting a constant in the middle silently
	// renumbers everything after it and breaks any stored or transmitted value.
}

// weekday is a DEFINED type (no =), so it can carry methods. An alias (type weekday = int)
// could not have a String method and would print as a bare number.
type weekday int

const (
	sunday    weekday = iota // 0
	monday                   // 1 - the expression repeats, iota advances
	tuesday                  // 2
	wednesday                // 3
	thursday                 // 4
	friday                   // 5
	saturday                 // 6
)

// String makes weekday a fmt.Stringer. The stringer tool can generate it for you via
// a `//go:generate stringer -type=weekday` directive.
func (d weekday) String() string {
	names := [...]string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}
	if d < 0 || int(d) >= len(names) {
		return fmt.Sprintf("weekday(%d)", int(d))
	}
	return names[d]
}

// Byte sizes: the whole expression repeats, so the shift grows with iota. Left UNTYPED on
// purpose - a defined type would force a conversion at every use.
const (
	_  = iota             // skip 0; the line still advances iota
	kb = 1 << (10 * iota) // 1 << 10
	mb                    // 1 << 20
	gb                    // 1 << 30
	tb                    // 1 << 40
)

// permission is a set of bit flags: each constant is one distinct bit.
type permission uint8

const (
	permRead    permission = 1 << iota // 1
	permWrite                          // 2
	permExecute                        // 4
)

// status has meaningful values, so they are written out rather than derived from iota.
type status int

const (
	statusOK        status = 200
	statusNoContent status = 204
	statusNotFound  status = 404
)

func (s status) String() string {
	switch s {
	case statusOK:
		return "OK"
	case statusNoContent:
		return "No Content"
	case statusNotFound:
		return "Not Found"
	default:
		return fmt.Sprintf("Unknown(%d)", int(s))
	}
}

// packageVariables shows that package-level variables initialise in dependency order.
func packageVariables() {
	fmt.Println("\n--- package-level variables ---")

	// derived is declared BEFORE base, yet holds the right value: initialisation follows the
	// dependency graph, not the text. Within one level, declaration order decides.
	fmt.Printf("base=%d derived=%d (declared first, initialised second)\n", base, derived)

	// Unlike a constant, a package var may be initialised by any function call.
	fmt.Println("started was set by time.Now() before main ran:", !started.IsZero())

	// A dependency cycle is rejected at compile time (followed by "x refers to y" etc.):
	//	var x = y + 1 // ERROR: initialization cycle for x
	//	var y = x + 1

	// Package vars are global mutable state: visible to every file, exempt from the unused-
	// variable rule and unsynchronised across goroutines. Prefer passing values explicitly;
	// keep package vars for process-wide things such as sentinel errors (module 009).
}

// Order of declaration does not matter at package level.
var (
	derived  = base * 2      // depends on base, so it is initialised after it
	base     = computeBase() // depends on a function call
	started  = time.Now()    // a non-constant initialiser: only var can do this
	traceLog []string        // zero value: a nil slice
)

func computeBase() int {
	traceLog = append(traceLog, "computeBase() ran")
	return 21
}

// initFunctions shows init(), which runs after all package vars and before main.
func initFunctions() {
	fmt.Println("\n--- init functions ---")

	fmt.Println("the lookup table was built by init():", lookupTable)
	fmt.Println("full initialisation trace:", traceLog)

	// Startup order: imported packages (recursively, each exactly once), then this package's
	// vars in dependency order, then its init()s in declaration order (across files in the order
	// go build presents them - sorted by file name), then main.main.
	fmt.Println("order: imports -> package vars -> init()s -> main")

	// init takes no parameters, returns nothing and cannot be called or referenced:
	//	init() // ERROR: undefined: init

	// Legitimate uses: registering with a registry (database/sql drivers, image decoders),
	// validating package state, precomputing tables. A blank import exists for this - it runs
	// the package's init and nothing else, as examples/rest.go does:
	//	import _ "github.com/lib/pq" // pq's init calls sql.Register("postgres", ...)
	fmt.Println(`import _ "github.com/lib/pq" exists only to run the driver's init()`)

	// Use init sparingly: it runs implicitly, cannot return an error (its only failure mode is
	// a panic before main starts) and always runs in tests. Explicit setup from main is clearer.
}

var lookupTable map[string]int

// Precomputing a table is a legitimate init: it cannot be a constant and must exist before
// anything in the package uses it.
func init() {
	lookupTable = map[string]int{"one": 1, "two": 2, "three": 3}
	traceLog = append(traceLog, "first init() ran")
}

// A package may have many init functions - they are not in the namespace, so unlike every other
// name they can repeat. Within a file they run top to bottom.
func init() {
	traceLog = append(traceLog, "second init() ran")
}
