// Command 002a_basic_types is module 002a of the Go course: the scalar types and strings.
//
// Module 002b covers the composite types. The one rule that shapes everything here: Go has NO
// implicit numeric conversion - not between sizes, not signed/unsigned, not int/float. Every mix
// needs an explicit T(v). Verbose, and the reason Go has almost no accidental-truncation bugs.
//
//	go run ./basics/002a_basic_types
package main

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

func main() {
	overview()              // value semantics, explicit and lossy conversions
	integerTypes()          // sizes, literals, silent overflow, division and remainder
	floatsAndComplex()      // float64 default, Inf/NaN, precision, complex numbers
	booleans()              // no truthiness, short-circuit evaluation
	stringsBasics()         // immutable bytes, len in bytes, slicing, Builder, literal forms
	bytesAndRunes()         // byte/rune aliases, range decodes UTF-8, unicode
	conversionsAndStrconv() // T(v) vs parsing/formatting with strconv
}

// overview shows that assignment copies and that every conversion is explicit - and may lose data.
func overview() {
	fmt.Println("--- overview ---")

	// Predeclared types: bool; int8..int64, uint8..uint64, int, uint, uintptr; byte (= uint8),
	// rune (= int32); float32, float64; complex64, complex128; string.

	// Everything is a value: assignment and argument passing copy. A slice, map or channel
	// merely CONTAINS a pointer, so its copy shares the data (module 006).
	a := 10
	b := a
	b = 20
	fmt.Printf("a=%d b=%d - assignment copied, it did not alias\n", a, b)

	// No implicit conversion, not even int32 -> int64, and int is distinct from int64 even on a
	// 64-bit machine.
	var i32 int32 = 5
	var i64 int64 = int64(i32)
	fmt.Printf("int32 %d -> int64 %d (explicit conversion required)\n", i32, i64)
	//	var bad int64 = i32 // ERROR: cannot use i32 (variable of type int32) as int64 value in variable declaration
	var plain int = 5
	//	var also int64 = plain // ERROR: cannot use plain (variable of type int) as int64 value in variable declaration
	_ = plain

	// A conversion of a VARIABLE can lose information silently: it wraps or truncates.
	large := 300
	fmt.Printf("int8(300) via a variable = %d (wrapped, no warning)\n", int8(large))
	pi := 3.9
	fmt.Printf("int(3.9)=%d int(-3.9)=%d (truncated toward zero), math.Round gives %d\n",
		int(pi), int(-pi), int(math.Round(pi)))

	// A conversion of a CONSTANT is checked at compile time:
	//	fmt.Println(int8(300)) // ERROR: constant 300 overflows int8

	// A defined type (type Celsius float64) is a distinct type with the same representation -
	// module 002b shows why that is useful.
}

// integerTypes shows integer sizes, literal forms and the arithmetic edge cases.
func integerTypes() {
	fmt.Println("\n--- integer types ---")

	// Limits are untyped constants in math. int and uint are 64 bits on amd64/arm64 and
	// 32 on 386/arm. Use int by default; a sized type only for formats, protocols or interop.
	fmt.Printf("int8   %d..%d\n", math.MinInt8, math.MaxInt8)
	fmt.Printf("int16  %d..%d\n", math.MinInt16, math.MaxInt16)
	fmt.Printf("int32  %d..%d\n", math.MinInt32, math.MaxInt32)
	fmt.Printf("int64  %d..%d\n", math.MinInt64, math.MaxInt64)
	fmt.Printf("uint8  0..%d\n", math.MaxUint8)
	fmt.Printf("int on this platform is %d bits\n", strconv.IntSize)
	// uintptr holds a pointer's bit pattern for unsafe code - it does NOT keep the object alive.

	// Decimal, binary, octal (0o755 or legacy 0755), hex, with optional _ separators.
	decimal, binary, octal, hexa := 1_000_000, 0b1010, 0o755, 0xFF
	fmt.Printf("1_000_000=%d 0b1010=%d 0o755=%d 0xFF=%d\n", decimal, binary, octal, hexa)

	// Overflow wraps silently - defined two's-complement behaviour, not UB as in C and no
	// panic as in Rust debug builds.
	var maxByte uint8 = math.MaxUint8
	maxByte++
	var minInt8 int8 = math.MinInt8
	minInt8--
	fmt.Printf("uint8 255+1 = %d, int8 -128-1 = %d (wrapped)\n", maxByte, minInt8)

	// There is no checked_add: test before you overflow (or use math/bits, module 003).
	x, y := math.MaxInt64, 1
	if x > math.MaxInt64-y {
		fmt.Println("overflow detected before it happens")
	}

	// Division truncates toward zero; % is a remainder with the sign of the LEFT operand.
	fmt.Printf("7/2=%d -7/2=%d  -7%%3=%d 7%%-3=%d\n", 7/2, -7/2, -7%3, 7%-3)
	euclid := func(a, n int) int { return ((a % n) + n) % n } // the non-negative modulus idiom
	fmt.Printf("euclid(-7, 3) = %d\n", euclid(-7, 3))

	// Integer division by zero panics; floats give Inf/NaN instead (next topic).
	//	fmt.Println(1 / 0)               // ERROR: invalid operation: division by zero
	//	zero := 0; fmt.Println(1 / zero) // panic: runtime error: integer divide by zero

	// Unsigned 0 - 1 wraps to a huge number, so `for i := uint(len(s)-1); i >= 0; i--` never ends.
	var count uint = 0
	count--
	fmt.Printf("uint 0 - 1 = %d\n", count)

	// An untyped limit takes the context's type - if it fits.
	var big uint64 = math.MaxUint64
	fmt.Println("math.MaxUint64 as uint64:", big)
	//	var bad int = math.MaxUint64 // ERROR: cannot use math.MaxUint64 (untyped int constant 18446744073709551615) as int value in variable declaration (overflows)
}

// floatsAndComplex shows IEEE-754 behaviour and the built-in complex numbers.
func floatsAndComplex() {
	fmt.Println("\n--- floating-point and complex numbers ---")

	// float64 is the default and the right choice unless you need to halve memory.
	var f32 float32 = 1.0 / 3.0
	var f64 float64 = 1.0 / 3.0
	fmt.Printf("float32 1/3 = %.10f (~7 digits), float64 1/3 = %.17f (~15 digits)\n", f32, f64)

	// Float division by zero does not panic: it yields +Inf, -Inf or NaN.
	zero := 0.0
	posInf, negInf, nan := 1.0/zero, -1.0/zero, zero/zero
	fmt.Printf("1/0=%v -1/0=%v 0/0=%v  IsInf=%t IsNaN=%t\n",
		posInf, negInf, nan, math.IsInf(posInf, 1), math.IsNaN(nan))

	// NaN equals nothing, not even itself, so floats are only partially ordered (cmp.Compare
	// defines a total order for sorting). A NaN map key can never be looked up again.
	fmt.Printf("nan == nan is %t\n", nan == nan)

	// Binary floats cannot represent most decimal fractions. Compare with a tolerance, and for
	// money use integer minor units (examples/monetary_amount.go). Variables are needed here:
	// the CONSTANT expression 0.1 + 0.2 is computed exactly at compile time and equals 0.3.
	tenth, fifth := 0.1, 0.2
	sum := tenth + fifth
	const epsilon = 1e-9
	fmt.Printf("0.1+0.2 = %.20f, == 0.3 is %t, within epsilon is %t\n",
		sum, sum == 0.3, math.Abs(sum-0.3) < epsilon)

	// No implicit int -> float promotion.
	count := 3
	price := 1.5
	fmt.Printf("float64(count) * price = %.2f\n", float64(count)*price)
	//	fmt.Println(count * price) // ERROR: invalid operation: count * price (mismatched types int and float64)

	// complex64/complex128 are built in: literals like 3+4i, complex(), real(), imag().
	c := 3 + 4i
	fmt.Printf("c=%v real=%.0f imag=%.0f abs=%.0f, (1-1i)^2 = %v\n",
		c, real(c), imag(c), math.Hypot(real(c), imag(c)), complex(1, -1)*complex(1, -1))
}

// booleans shows that bool is not a number and that && and || short-circuit.
func booleans() {
	fmt.Println("\n--- booleans ---")

	// true and false are predeclared constants, not keywords (they can be shadowed).
	// The zero value false usually reads right: var found bool means "not found yet".
	var yes, no = true, false
	fmt.Printf("yes=%t no=%t !yes=%t\n", yes, no, !yes)

	// No truthiness and no conversion to or from numbers - write `n != 0` explicitly.
	//	if 1 { }             // ERROR: non-boolean condition in if statement
	//	var n int = int(yes) // ERROR: cannot convert yes (variable of type bool) to type int
	toInt := func(b bool) int { // the idiom when you really need 0/1
		if b {
			return 1
		}
		return 0
	}
	fmt.Println("toInt(true) =", toInt(true), "toInt(false) =", toInt(false))

	// && and || short-circuit: the right operand runs only if the left does not decide.
	calls := 0
	sideEffect := func() bool { calls++; return true }
	_ = false && sideEffect()
	_ = true || sideEffect()
	fmt.Printf("after two short-circuited expressions, sideEffect ran %d times\n", calls)

	// That is what makes the nil guard safe. && binds tighter than ||, and both looser than
	// comparisons: a == b && c > d || e is ((a == b) && (c > d)) || e (module 003).
	var p *point
	if p != nil && p.X > 0 {
		fmt.Println("unreachable")
	}
	fmt.Println("`p != nil && p.X > 0` never dereferences a nil p")
}

type point struct{ X, Y int }

// stringsBasics shows that a string is an immutable sequence of bytes.
func stringsBasics() {
	fmt.Println("\n--- strings ---")

	// A string is a read-only (pointer, length) header over bytes - not nil-terminated, and its
	// zero value is "", never nil. len counts BYTES and indexing yields a byte.
	s := "Hello, Gophers"
	fmt.Printf("s=%q len=%d; s[0] = %d as a number, %c as a character, type %T\n",
		s, len(s), s[0], s[0], s[0])

	polish := "Zażółć gęślą jaźń"
	fmt.Printf("%q: len=%d bytes but %d runes\n", polish, len(polish), utf8.RuneCountInString(polish))
	fmt.Printf("polish[1]=%d is ASCII 'a'; polish[2]=%d is the FIRST byte of 'ż', not a character\n",
		polish[1], polish[2])

	// Literals are UTF-8, but a string built from arbitrary bytes (file, socket) may hold anything.
	invalid := string([]byte{0xff, 0xfe, 0x41})
	fmt.Printf("arbitrary bytes as a string: valid UTF-8? %t\n", utf8.ValidString(invalid))

	// Immutable: to modify, convert to []byte or []rune (each conversion copies) and back.
	//	s[0] = 'h' // ERROR: cannot assign to s[0] (neither addressable nor a map index expression)
	modified := []byte(s)
	modified[0] = 'h'
	fmt.Printf("via []byte: %q (original still %q)\n", string(modified), s)

	// Slicing is a byte range, O(1), and shares memory. Cutting a multi-byte rune in half gives
	// invalid UTF-8 - silently, unlike Rust, which panics.
	fmt.Printf("s[7:14]=%q; polish[:2]=%q is fine, polish[:3]=%q is broken\n", s[7:14], polish[:2], polish[:3])

	// s += x in a loop is quadratic (a new string each time); strings.Builder is linear.
	var sb strings.Builder
	for i := range 3 {
		fmt.Fprintf(&sb, "part%d ", i) // Builder is an io.Writer
	}
	fmt.Printf("strings.Builder result: %q\n", strings.TrimSpace(sb.String()))

	// Comparison is byte-wise lexicographic - not locale-aware (golang.org/x/text/collate is).
	fmt.Printf("\"apple\" < \"banana\" = %t, \"Z\" < \"a\" = %t (uppercase sorts first)\n",
		"apple" < "banana", "Z" < "a")

	// Interpreted "..." processes escapes and is one line; raw `...` has no escapes and may span
	// lines - ideal for regexps, Windows paths, JSON fixtures, struct tags. It cannot hold a `.
	interpreted := "line1\nline2\ttabbed \u00e9 \x41"
	raw := `no escapes here: \n stays literal, C:\Users\lukas`
	multiline := `raw strings
can span
several lines`
	fmt.Printf("interpreted: %q\nraw:         %s\nraw multiline has %d lines\n",
		interpreted, raw, strings.Count(multiline, "\n")+1)
}

// bytesAndRunes shows the difference between indexing (bytes) and ranging (runes) over text.
func bytesAndRunes() {
	fmt.Println("\n--- bytes and runes ---")

	// byte = uint8 and rune = int32 are true ALIASES: the same types, the names document intent.
	var b byte = 65
	var u8 uint8 = b
	var r rune = 'A' // rune literals use single quotes and print as numbers unless %c/%q
	var i32 int32 = r
	fmt.Printf("byte=%d uint8=%d rune=%d int32=%d, %%c gives %c\n", b, u8, r, i32, r)

	// range over a string DECODES UTF-8: i is the byte offset (it jumps), r the rune.
	s := "Gęś"
	fmt.Printf("%q: len=%d bytes, %d runes\n", s, len(s), utf8.RuneCountInString(s))
	for i, r := range s {
		fmt.Printf("  range: i=%d rune=%q U+%04X width=%d\n", i, r, r, utf8.RuneLen(r))
	}
	// Indexing does NOT decode - it yields raw bytes.
	for i := range len(s) {
		fmt.Printf("  index: s[%d]=%d (0x%02X)\n", i, s[i], s[i])
	}

	// Invalid bytes decode to utf8.RuneError (U+FFFD) and advance by one.
	for i, r := range string([]byte{0x47, 0xff, 0x53}) {
		fmt.Printf("  broken: i=%d rune=%q isRuneError=%t\n", i, r, r == utf8.RuneError)
	}

	// []rune(s) copies, but gives random access by code point.
	runes := []rune(s)
	fmt.Printf("runes[1]=%q len=%d, reversed by runes: %q\n", runes[1], len(runes), reverse(s))

	// string(int) is a RUNE conversion: string(rune(65)) is "A", strconv.Itoa(65) is "65".
	//	string(i) // go vet: conversion from int to string yields a string of one rune, not a string of digits
	fmt.Printf("string(rune(65))=%q strconv.Itoa(65)=%q\n", string(rune(65)), strconv.Itoa(65))

	// unicode classifies runes; unicode/utf8 handles the encoding.
	for _, r := range []rune{'G', 'ę', '7', ' ', '!'} {
		fmt.Printf("  %q letter=%t digit=%t space=%t upper=%q\n",
			r, unicode.IsLetter(r), unicode.IsDigit(r), unicode.IsSpace(r), unicode.ToUpper(r))
	}

	// A rune is a code point, not a user-perceived character: é may be 'e' + a combining accent,
	// and neither len nor RuneCountInString counts "characters" (see golang.org/x/text).
	combining := "e\u0301"
	fmt.Printf("%q: %d bytes, %d runes, 1 perceived character\n",
		combining, len(combining), utf8.RuneCountInString(combining))
}

// reverse reverses a string by runes, so multi-byte characters survive intact.
func reverse(s string) string {
	r := []rune(s)
	for i, j := 0, len(r)-1; i < j; i, j = i+1, j-1 {
		r[i], r[j] = r[j], r[i]
	}
	return string(r)
}

// conversionsAndStrconv contrasts conversion T(v), which cannot fail, with parsing, which can.
func conversionsAndStrconv() {
	fmt.Println("\n--- conversions and strconv ---")

	// Conversion reinterprets a value as another type. It never returns an error - it may truncate.
	i := 42
	fmt.Printf("float64(%d)=%v int8(%d)=%d byte(%d)=%d\n", i, float64(i), i, int8(i), i, byte(i))

	// Parsing text CAN fail, so strconv returns an error. For single values it beats fmt.Sscan:
	// faster, with typed errors.
	if n, err := strconv.Atoi("123"); err == nil {
		fmt.Println(`strconv.Atoi("123") =`, n)
	}
	if _, err := strconv.Atoi("12x"); err != nil {
		fmt.Println(`strconv.Atoi("12x") failed:`, err)
		// The error is a *strconv.NumError wrapping ErrSyntax or ErrRange (errors.As: module 009).
		var numErr *strconv.NumError
		if errors.As(err, &numErr) {
			fmt.Printf("  typed error: func=%s input=%q reason=%v\n", numErr.Func, numErr.Num, numErr.Err)
		}
	}
	if _, err := strconv.ParseInt("999999999999999999999", 10, 64); err != nil {
		fmt.Println("out of range:", err)
	}

	// ParseInt(s, base, bitSize): base 0 infers 0x/0b/0o/0 prefixes; bitSize is the width the
	// result must fit, but it always returns an int64 - convert afterwards.
	for _, in := range []string{"0xFF", "0b1010", "0o755", "255"} {
		v, err := strconv.ParseInt(in, 0, 64)
		fmt.Printf("  ParseInt(%q, 0, 64) = %d (err=%v)\n", in, v, err)
	}

	// Formatting. FormatFloat: 'f' plain, 'e' scientific, 'g' shortest; prec -1 = the fewest
	// digits that round-trip exactly.
	fmt.Printf("Itoa(255)=%q FormatInt(255,2)=%q FormatInt(255,16)=%q\n",
		strconv.Itoa(255), strconv.FormatInt(255, 2), strconv.FormatInt(255, 16))
	fmt.Printf("FormatFloat(1/3,'f',4)=%q  prec -1=%q\n",
		strconv.FormatFloat(1.0/3.0, 'f', 4, 64), strconv.FormatFloat(1.0/3.0, 'f', -1, 64))
	fmt.Printf("Quote(%s)=%s\n", `"a\tb"`, strconv.Quote("a\tb"))

	fmt.Print(`ParseBool accepts "1", "true", "T": `)
	for _, in := range []string{"1", "true", "T"} {
		v, _ := strconv.ParseBool(in)
		fmt.Printf("%t ", v)
	}
	fmt.Println()
}
