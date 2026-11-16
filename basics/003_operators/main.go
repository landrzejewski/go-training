// Command 003_operators is module 003 of the Go course: operators.
//
// Go's operator set is deliberately small. There is no operator overloading, so + always means
// what the language says for that type. There is also no ternary ?:, no ** for powers, no <=>
// and no null-coalescing operator.
//
//	go run ./basics/003_operators
package main

import (
	"fmt"
	"math"
	"math/bits"
	"slices"
)

func main() {
	arithmetic() // + - * / %, no **, no promotion, ++ is a statement
	comparison() // equality vs ordering, comparability, no chaining
	logical()    // && || !, short-circuit, no XOR, no ternary
	assignment() // compound forms, tuple assignment, addressability
	bitwise()    // & | ^ &^, shifts, math/bits
	precedence() // five levels, left-associative, where Go and C differ
}

// arithmetic shows the arithmetic operators and their integer-specific rules.
func arithmetic() {
	fmt.Println("--- arithmetic operators ---")

	a, b := 20, 7
	fmt.Printf("%d+%d=%d  %d-%d=%d  %d*%d=%d  %d/%d=%d  %d%%%d=%d  -a=%d\n",
		a, b, a+b, a, b, a-b, a, b, a*b, a, b, a/b, a, b, a%b, -a)

	// + also concatenates strings - the only built-in "overload".
	fmt.Println(`"Go" + "lang" =`, "Go"+"lang")

	// Integer division truncates toward zero; % keeps the LEFT operand's sign. Division by an
	// integer zero panics; floats give ±Inf/NaN (module 002a).
	fmt.Printf("-7/2=%d (not -4)  -7%%3=%d  7%%-3=%d\n", -7/2, -7%3, 7%-3)

	// % is for integers only (unlike Rust) - math.Mod is the float version.
	//	fmt.Println(5.5 % 2.0) // ERROR: invalid operation: operator % not defined on 5.5 (untyped float constant)
	fmt.Println("math.Mod(5.5, 2.0) =", math.Mod(5.5, 2.0))

	// ^ is XOR, not a power. There is no power operator: math.Pow (float64) or a loop.
	fmt.Printf("2 ^ 10 = %d  <- XOR, not 1024!  math.Pow(2, 10) = %v  intPow(2, 10) = %d\n",
		2^10, math.Pow(2, 10), intPow(2, 10))

	// Both operands must have the same type - no int/float promotion. Untyped constants adapt.
	var i int = 5
	var f float64 = 2.0
	//	fmt.Println(i * f) // ERROR: invalid operation: i * f (mismatched types int and float64)
	fmt.Printf("float64(i)*f = %v; i*2 = %d and f*2 = %v (the untyped 2 adapts)\n", float64(i)*f, i*2, f*2)

	// Overflow wraps silently. math/bits returns the carry/overflow explicitly.
	var m8 uint8 = 255
	sum, carry := bits.Add64(math.MaxUint64, 1, 0)
	hi, lo := bits.Mul64(math.MaxUint64, 2)
	fmt.Printf("uint8 255+1 = %d; bits.Add64(MaxUint64, 1, 0) = sum %d carry %d; Mul64 128-bit: hi=%d lo=%d\n",
		m8+1, sum, carry, hi, lo)

	// ++ and -- are STATEMENTS, postfix only - never inside an expression.
	counter := 0
	counter++
	counter--
	counter++
	fmt.Println("counter after ++ -- ++ =", counter)
	//	x := counter++ // ERROR: syntax error: unexpected ++ at end of statement
	//	++counter      // ERROR: syntax error: unexpected ++, expected }
}

// intPow computes base**exp exactly, in integers.
func intPow(base, exp int) int {
	result := 1
	for range exp {
		result *= base
	}
	return result
}

// comparison shows == != < <= > >= and which types support them.
func comparison() {
	fmt.Println("\n--- comparison operators ---")

	// The result is an untyped bool.
	a, b := 10, 20
	fmt.Printf("%d==%d:%t  !=:%t  <:%t  <=:%t  >:%t  >=:%t\n",
		a, b, a == b, a != b, a < b, a <= b, a > b, a >= b)

	// Strings order byte-wise - not by locale or Unicode collation.
	fmt.Printf("\"apple\" < \"banana\": %t  \"Z\" < \"a\": %t  \"ą\" > \"z\": %t (not Polish order)\n",
		"apple" < "banana", "Z" < "a", "ą" > "z")

	// Equality works on every COMPARABLE type; ordering only on integers, floats and strings.
	// Booleans and pointers have == but no <.
	x, y := true, false
	p, q := &a, &b
	fmt.Printf("true != false: %t; pointers compare ADDRESSES: p == q %t, p == &a %t\n",
		x != y, p == q, p == &a)
	//	fmt.Println(x < y) // ERROR: invalid operation: x < y (operator < not defined on bool)
	//	fmt.Println(p < q) // ERROR: invalid operation: p < q (operator < not defined on pointer)

	// Comparisons do not chain: 1 < 2 yields a bool, and bool < int is undefined.
	//	fmt.Println(1 < 2 < 3) // ERROR: invalid operation: 1 < 2 < 3 (mismatched types untyped bool and untyped int)
	fmt.Println("a < b && b < 30:", a < b && b < 30)

	// Different types never compare, not even two integer types.
	var i32 int32 = 10
	//	fmt.Println(i32 == a) // ERROR: invalid operation: i32 == a (mismatched types int32 and int)
	fmt.Println("int(i32) == a:", int(i32) == a)

	// NaN compares false against everything, so floats are only partially ordered;
	// cmp.Compare gives a total order (NaN first) for sorting.
	nan := math.NaN()
	fmt.Printf("NaN == NaN: %t, NaN < 1: %t, NaN > 1: %t\n", nan == nan, nan < 1, nan > 1)

	// Slices, maps and functions compare only against nil. Use slices.Equal / maps.Equal;
	// reflect.DeepEqual is slow and treats nil and empty slices as different.
	s1, s2 := []int{1, 2}, []int{1, 2}
	//	fmt.Println(s1 == s2) // ERROR: invalid operation: s1 == s2 (slice can only be compared to nil)
	fmt.Println("slices.Equal(s1, s2):", slices.Equal(s1, s2))

	// An interface is comparable at compile time but PANICS at runtime when its dynamic
	// values are not.
	var i1, i2 any = []int{1}, []int{1}
	fmt.Println("comparing two interfaces holding slices:", safeCompare(i1, i2))
}

// safeCompare compares two interfaces and recovers if the comparison panics.
func safeCompare(a, b any) (result string) {
	defer func() {
		if r := recover(); r != nil {
			result = fmt.Sprintf("recovered from %v", r)
		}
	}()
	return fmt.Sprintf("%t", a == b)
}

// logical shows && || ! and the guarantees around them.
func logical() {
	fmt.Println("\n--- logical operators ---")

	// They work on bool only - there is no truthiness (module 002a).
	t, f := true, false
	fmt.Printf("t&&f=%t  t||f=%t  !t=%t\n", t && f, t || f, !t)

	// && and || short-circuit. This is a language guarantee, not an optimisation.
	evaluated := 0
	track := func(v bool) bool { evaluated++; return v }
	_ = f && track(true)
	_ = t || track(true)
	fmt.Printf("false && g() and true || g(): g ran %d times\n", evaluated)
	_ = t && track(true)
	fmt.Printf("true && g(): g ran %d time\n", evaluated)

	// So guards like this are safe: s[0] is never evaluated on an empty slice.
	var s []int
	if len(s) > 0 && s[0] == 1 {
		fmt.Println("unreachable")
	}

	// Precedence: ! then && then ||, all looser than comparisons - no parentheses needed here.
	a, b, c, d := 1, 1, 2, 3
	fmt.Printf("a==b && c<d || false = %t, parsed as ((a==b) && (c<d)) || false\n", a == b && c < d || false)

	// There is no boolean XOR operator: != is XOR for bools.
	fmt.Printf("XOR via !=: true!=false=%t true!=true=%t\n", t != f, t != t)

	// There is no ternary: `v := a > c ? a : c` is an if/else. Since Go 1.21, min and max cover
	// the most common case.
	value := c
	if a > c {
		value = a
	}
	fmt.Printf("if/else gives %d, max(a, c) gives %d\n", value, max(a, c))
}

// assignment shows compound assignment, tuple assignment and what may appear on the left.
func assignment() {
	fmt.Println("\n--- assignment and compound assignment ---")

	// Every binary arithmetic and bitwise operator has an op= form.
	x := 10
	x += 5
	x -= 3
	x *= 2
	x /= 4
	x %= 4
	fmt.Println("10 += 5, -= 3, *= 2, /= 4, %= 4 ->", x)

	v := 0b1100
	v &= 0b1010
	v |= 0b0001
	v ^= 0b1111
	v <<= 2
	v &^= 0b100000
	fmt.Printf("0b1100 &= 1010, |= 0001, ^= 1111, <<= 2, &^= 100000 -> %06b\n", v)

	// Assignment is a STATEMENT, not an expression, so the C typo `if (x = 0)` cannot exist.
	//	a := (b = 5) // ERROR: syntax error: unexpected =, expected )
	//	if x = 5 {}  // ERROR: syntax error: cannot use assignment x = 5 as value

	// Tuple assignment evaluates the WHOLE right side (and the left-side index expressions)
	// first: a swap needs no temporary, and s[i] below uses the OLD i.
	a, b := 1, 2
	a, b = b, a
	s := []int{0, 10, 20}
	i := 1
	i, s[i] = 0, 99
	fmt.Printf("swap: a=%d b=%d; i, s[i] = 0, 99 -> i=%d s=%v\n", a, b, i, s)

	// x op= y evaluates x only ONCE - it matters when x is m[f()] or s[f()].
	calls := 0
	index := func() int { calls++; return 0 }
	arr := []int{5}
	arr[index()] += 10
	fmt.Printf("arr[index()] += 10 called index() %d time; arr=%v\n", calls, arr)

	// _ discards a value on the left of any assignment.
	_, second := 1, 2
	fmt.Println("discarded the first value, kept", second)

	// The left side must be addressable (a variable, *p, a slice element, a field of an
	// addressable struct) or a map index. A field of a map entry and a string byte are neither.
	type box struct{ N int }
	m := map[string]box{"a": {N: 1}}
	str := "hello"
	//	m["a"].N = 2 // ERROR: cannot assign to struct field m["a"].N in map
	//	str[0] = 'H' // ERROR: cannot assign to str[0] (neither addressable nor a map index expression)
	_, _ = m, str
}

// bitwise shows the bitwise and shift operators, including Go's own &^.
func bitwise() {
	fmt.Println("\n--- bitwise and shift operators ---")

	var a, b uint8 = 0b1100, 0b1010
	fmt.Printf("a=%04b b=%04b  a&b=%04b  a|b=%04b  a^b=%04b\n", a, b, a&b, a|b, a^b)

	// &^ (AND NOT) clears in a every bit set in b - exactly a & ^b, as one operator.
	fmt.Printf("a&^b = %04b, equal to a&(^b): %t\n", a&^b, a&^b == a&(^b))
	// Unary ^ is NOT (Go has no ~): all ones for ^0 unsigned, -x-1 for signed.
	fmt.Printf("^a = %08b\n", ^a)

	// The idiomatic way to clear a flag.
	const (
		flagRead  = 1 << 0
		flagWrite = 1 << 1
		flagExec  = 1 << 2
	)
	perms := flagRead | flagWrite | flagExec
	fmt.Printf("perms=%03b, perms &^ flagWrite = %03b\n", perms, perms&^flagWrite)

	var u uint8 = 0b0000_0011
	fmt.Printf("%08b << 2 = %08b, >> 1 = %08b\n", u, u<<2, u>>1)

	// Shifting by more than the width is DEFINED (unlike C): the result is 0, or -1 for a
	// negative signed value. (A variable count keeps go vet from flagging the constant form.)
	overshift := uint(100)
	fmt.Printf("uint8(3) << 100 = %d, int8(-1) >> 100 = %d\n", u<<overshift, int8(-1)>>overshift)

	// >> is arithmetic (sign-extending) on signed types, logical (zero-filling) on unsigned.
	var signed int8 = -8
	var unsigned uint8 = 0b1111_1000
	fmt.Printf("int8(-8) >> 1 = %d; uint8 %08b >> 1 = %08b\n", signed>>1, unsigned, unsigned>>1)

	// The count may be signed (since Go 1.13) but must not be negative.
	//	fmt.Println(1 << -1)                 // ERROR: invalid operation: negative shift count -1 (untyped int constant)
	//	shift := -1; fmt.Println(1 << shift) // panic: runtime error: negative shift amount

	// Constant shifts use arbitrary precision - what makes 1 << (10 * iota) work (module 001b).
	const huge = 1 << 62
	fmt.Println("const 1 << 62 =", huge)

	// math/bits has what the operators lack.
	var n uint = 0b1011_0110
	fmt.Printf("n=%08b OnesCount=%d LeadingZeros8=%d TrailingZeros=%d Len=%d\n",
		n, bits.OnesCount(n), bits.LeadingZeros8(uint8(n)), bits.TrailingZeros(n), bits.Len(n))
	fmt.Printf("RotateLeft8(n, 2)=%08b Reverse8(n)=%08b\n", bits.RotateLeft8(uint8(n), 2), bits.Reverse8(uint8(n)))
}

// precedence shows Go's five binary precedence levels.
func precedence() {
	fmt.Println("\n--- precedence and associativity ---")

	// Five levels (C has fifteen), all binary operators left-associative:
	//	5 (highest)  *  /  %  <<  >>  &  &^
	//	4            +  -  |  ^
	//	3            ==  !=  <  <=  >  >=
	//	2            &&
	//	1 (lowest)   ||
	// Unary operators (+ - ! ^ * & <-) bind tighter than any binary one.
	fmt.Printf("2 + 3*4    = %d  (* before +)\n", 2+3*4)
	fmt.Printf("6 & 3 + 1  = %d   (& is level 5: (6&3) + 1)\n", 6&3+1)

	// The C trap that does NOT exist in Go: & binds tighter than ==.
	fmt.Printf("6 & 3 == 2 = %t (Go: (6&3) == 2; C: 6 & (3==2))\n", 6&3 == 2)
	// + and | share level 4, left to right - here C agrees.
	fmt.Printf("3 + 5 | 6  = %d  ((3+5) | 6)\n", 3+5|6)

	// Where Go and C really differ: << is tighter than + in Go, looser in C; ^ is level 4 in Go,
	// much lower in C.
	fmt.Printf("1 + 2<<3   = %d  (Go: 1 + (2<<3); C gives 24)\n", 1+2<<3)
	fmt.Printf("3 ^ 5 + 1  = %d   (Go: (3^5) + 1; C gives 5)\n", 3^5+1)

	// Comparisons sit between arithmetic and &&/||.
	a, b, c, d := 1, 2, 3, 4
	fmt.Printf("a+1 == b && c < d = %t (no parentheses needed)\n", a+1 == b && c < d)

	yes, no, alsoNo := true, false, false
	fmt.Printf("yes || no && alsoNo = %t (&& first); (yes || no) && alsoNo = %t\n",
		yes || no && alsoNo, (yes || no) && alsoNo)

	// Left associativity.
	fmt.Printf("100 / 10 / 2 = %d, 10 - 3 - 2 = %d\n", 100/10/2, 10-3-2)

	// Unary operators first: *p + 1 dereferences, then adds. (*, & and <- are binary or unary
	// depending on context.)
	p := &a
	fmt.Printf("*p + 1 = %d, -a * 2 = %d, !(a == b) = %t\n", *p+1, -a*2, !(a == b))

	// Know the table for READING code; when WRITING, parenthesise anything a reader would have
	// to look up. gofmt shows the grouping with spacing: it writes 1+x*3, not 1 + x * 3.
	x := 2
	fmt.Println("1+x*3 =", 1+x*3)
}
