// Command 008_interfaces is module 008 of the Go course: interfaces.
//
// Interfaces are the whole of Go's polymorphism, and satisfaction is implicit: a type implements
// an interface just by having its methods - it never says so and need not know the interface
// exists. So interfaces belong to the CONSUMER: declare one where it is used, listing only what
// you need. This inverts the dependency direction of Java or C# and is why Go has so few "IFoo"s.
//
//	go run ./basics/008_interfaces
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"sort"
	"strings"
)

func main() {
	implicitSatisfaction() // structural typing, the pointer-receiver trap, small interfaces
	interfaceValues()      // (type, value) pairs, the typed-nil trap, comparison
	typeAssertions()       // x.(T), comma-ok, optional interfaces, type switches
	composition()          // embedding interfaces, test doubles, accept interfaces
	emptyInterface()       // any, unknown JSON, any vs generics
	stdlibInterfaces()     // Stringer, TextMarshaler, io.Writer, sort.Interface
}

// implicitSatisfaction shows that a type satisfies an interface just by having its methods.
func implicitSatisfaction() {
	fmt.Println("--- implicit satisfaction ---")

	// No implements keyword: circle, square and *growing all fit shape without mentioning it.
	shapes := []shape{circle{R: 1}, square{Side: 2}, &growing{}}
	for _, s := range shapes {
		fmt.Printf("  %-13T Area()=%.4f\n", s, s.Area())
	}

	// Structural, so it works for types you do not own: *strings.Reader and *bytes.Buffer
	// satisfy io.Reader with no coordination (and *os.File would satisfy your own Closer).
	for _, r := range []io.Reader{strings.NewReader("from a strings.Reader"), bytes.NewBufferString("from a bytes.Buffer")} {
		data, _ := io.ReadAll(r)
		fmt.Printf("  io.Reader: %q\n", data)
	}

	// The method-set rule of module 007 is THE stumbling block: a pointer-receiver method
	// belongs to *T only, so a plain T does not satisfy the interface.
	//	var s shape = growing{} // ERROR: cannot use growing{} (value of struct type growing) as shape value in variable declaration: growing does not implement shape (method Area has pointer receiver)
	var ok shape = &growing{}
	fmt.Printf("  growing needs a POINTER to satisfy shape: %.0f\n", ok.Area())

	// Keep interfaces small - "the bigger the interface, the weaker the abstraction". The most
	// used ones have ONE method: io.Reader, io.Writer, io.Closer, fmt.Stringer, error
	// (sort.Interface has three and counts as large). A one-method interface is named
	// method + "-er": Reader, Writer, Stringer, Formatter.
}

// shape is a tiny consumer-side interface; nothing declares that it implements it.
type shape interface {
	Area() float64
}

type named interface {
	Name() string
}

type circle struct{ R float64 }
type square struct{ Side float64 }

func (c circle) Area() float64 { return math.Pi * c.R * c.R }
func (c circle) Name() string  { return "circle" }
func (s square) Area() float64 { return s.Side * s.Side }
func (s square) Name() string  { return "square" }

type growing struct{ n float64 }

func (g *growing) Area() float64 { g.n++; return g.n } // pointer receiver: only *growing is a shape

// Compile-time assertions cost nothing and break the build if a type drifts. Use them when
// nothing else in the package would catch the regression.
var (
	_ shape = circle{}
	_ shape = (*growing)(nil) // the POINTER, because Area has a pointer receiver
)

// interfaceValues shows what an interface value is made of, and the typed-nil trap.
func interfaceValues() {
	fmt.Println("\n--- interface values and the nil trap ---")

	// An interface value is two words: (dynamic type, dynamic value). %v prints the value,
	// %T the type - the two fastest debugging tools here. It is nil only if BOTH words are nil.
	var i any
	fmt.Printf("  var i any     -> value=%v type=%T isNil=%t\n", i, i, i == nil)
	i = 42
	fmt.Printf("  i = 42        -> value=%v type=%T isNil=%t\n", i, i, i == nil)
	i = circle{R: 1}
	fmt.Printf("  i = circle{1} -> value=%v type=%T isNil=%t\n", i, i, i == nil)

	// The most notorious gotcha in Go: a nil POINTER stored in an interface gives (*T, nil),
	// which is NOT a nil interface.
	var p *myError
	var asError error = p
	fmt.Printf("  p == nil: %t, but error(p) == nil: %t - it holds (%T, nil)\n", p == nil, asError == nil, asError)

	// It almost always shows up as a function returning a concrete nil error pointer.
	if err := buggyReturn(false); err != nil {
		fmt.Println("  buggyReturn(false) reports an error although nothing failed:", err)
	}
	if err := correctReturn(false); err == nil {
		fmt.Println("  correctReturn(false) returns a real nil - return a literal nil on success")
	}
	// Calling a method through it does not panic by itself: the method runs with a nil
	// receiver (myError.Error copes); only a dereference inside would panic (module 007).

	// Interfaces compare equal when dynamic type AND value are equal.
	var a, b any = 1, 1
	var c, d any = 1, int64(1)
	fmt.Println("  any(1) == any(1):       ", a == b)
	fmt.Println("  any(1) == any(int64(1)):", c == d, "<- different dynamic types")
	// An uncomparable dynamic type (slice, map, func) compiles but panics at run time.
	fmt.Println("  any([]int{1}) == any([]int{1}):", safeCompare([]int{1}, []int{1}))
}

type myError struct{ Msg string }

func (e *myError) Error() string {
	if e == nil {
		return "<nil *myError>" // a nil receiver is fine as long as we do not dereference it
	}
	return e.Msg
}

func buggyReturn(fail bool) error {
	var e *myError // nil pointer of a CONCRETE type - never declare the result like this
	if fail {
		e = &myError{Msg: "real failure"}
	}
	return e // a nil *myError converted to error is a NON-nil interface
}

func correctReturn(fail bool) error {
	if fail {
		return &myError{Msg: "real failure"}
	}
	return nil // a literal nil interface
}

// safeCompare reports a == b, or the panic that comparing them caused.
func safeCompare(a, b any) (result string) {
	defer func() {
		if r := recover(); r != nil {
			result = fmt.Sprintf("recovered from %v", r)
		}
	}()
	return fmt.Sprintf("%t", a == b)
}

// typeAssertions shows how to get the concrete value back out of an interface.
func typeAssertions() {
	fmt.Println("\n--- type assertions and type switches ---")

	var x any = "a string"

	// Comma-ok never panics: on failure ok is false and the value is T's zero value.
	// Use it unless a failure genuinely means the program is broken.
	if s, ok := x.(string); ok {
		fmt.Printf("  comma-ok succeeded: %q\n", s)
	}
	if n, ok := x.(int); !ok {
		fmt.Printf("  comma-ok failed safely: n=%d ok=%t\n", n, ok)
	}
	// The single-value form panics on a mismatch.
	fmt.Println("  x.(int):", catchPanic(func() { _ = x.(int) }))

	// Asserting to an INTERFACE asks whether the value also has those methods - the idiomatic
	// way to discover optional behaviour: if c, ok := r.(io.Closer); ok { c.Close() }.
	// io.Copy does it to find a fast path (io.ReaderFrom, io.WriterTo).
	for _, w := range []io.Writer{&bytes.Buffer{}, &strings.Builder{}} {
		_, ok := w.(io.ReaderFrom)
		fmt.Printf("  %-16T implements io.ReaderFrom: %t\n", w, ok)
	}

	// A type switch handles several types at once; see describe.
	for _, v := range []any{42, "text", 3.14, true, []int{1}, circle{R: 2}, nil} {
		fmt.Printf("  %-6s -> %s\n", fmt.Sprint(v), describe(v))
	}

	// Switching over CONCRETE types is a smell - a method would do that dispatch for you.
	// Switching over interfaces to discover capabilities is idiomatic.
	// Go 1.25 added reflect.TypeAssert[T](v), an allocation-free assertion on a reflect.Value -
	// for reflective code only.
}

func describe(v any) string {
	switch t := v.(type) { // in each case t has that case's type
	case nil: // matches a nil interface
		return "a nil interface"
	case int:
		return fmt.Sprintf("an int, doubled %d", t*2)
	case string:
		return fmt.Sprintf("a string of length %d", len(t))
	case float64, float32:
		// Several types in one case: t keeps the interface type any.
		//	_ = t * 2 // ERROR: invalid operation: t * 2 (mismatched types any and untyped int)
		return "a float (t is still any here)"
	case shape: // a case can be an INTERFACE
		return fmt.Sprintf("something with an Area(): %.2f", t.Area())
	default:
		return fmt.Sprintf("something else (%T)", t)
	}
}

// catchPanic runs f and returns whatever it panicked with.
func catchPanic(f func()) (recovered any) {
	defer func() {
		if r := recover(); r != nil {
			recovered = r
		}
	}()
	f()
	return "did not panic"
}

// composition shows interfaces built from interfaces, and interfaces embedded in structs.
func composition() {
	fmt.Println("\n--- interface embedding and composition ---")

	// Embedding unions method sets: io.ReadWriteCloser is literally interface{ Reader; Writer;
	// Closer }. Overlapping methods are fine (Go 1.14+) if the signatures are identical.
	var rw readWriter = &memory{}
	rw.Write("a")
	rw.Write("b")
	fmt.Printf("  readWriter: %q\n", rw.Read())
	var justReader reader = rw // a composed value narrows to either half
	fmt.Printf("  narrowed to reader: %q\n", justReader.Read())

	var ns namedShape = circle{R: 1}
	fmt.Printf("  namedShape: %s with area %.4f\n", ns.Name(), ns.Area())

	// Since Go 1.18 an interface may also hold type elements. Such an interface is a generic
	// constraint only (module 010), never a value type:
	//	type number interface{ ~int | ~float64 }
	//	var x number // ERROR: cannot use type number outside a type constraint: interface contains type constraints

	// Embedding an interface in a STRUCT: the struct satisfies it by forwarding, and overrides
	// single methods - a test double for a big interface. Sharp tool: every method you did
	// NOT override panics with a nil dereference while the embedded interface is nil.
	var w io.Writer = failingWriter{}
	_, err := w.Write([]byte("anything"))
	fmt.Println("  test double overriding one method:", err)

	// "Accept interfaces, return structs": take the smallest interface you need. A function
	// that reads takes io.Reader, not *os.File - so a test passes strings.NewReader, no mocks.
	fmt.Println("  countLines over a strings.Reader:", countLines(strings.NewReader("a\nb\nc")))
	fmt.Println("  countLines over a bytes.Buffer:  ", countLines(bytes.NewBufferString("x\ny")))
}

type reader interface{ Read() string }
type writer interface{ Write(string) }

type readWriter interface {
	reader
	writer
}

type namedShape interface {
	shape
	named
}

type memory struct{ data []string }

func (m *memory) Read() string   { return strings.Join(m.data, ",") }
func (m *memory) Write(s string) { m.data = append(m.data, s) }

type failingWriter struct {
	io.Writer // embedded interface, deliberately nil
}

func (f failingWriter) Write(p []byte) (int, error) {
	return 0, errors.New("simulated write failure")
}

func countLines(r io.Reader) int {
	data, err := io.ReadAll(r)
	if err != nil || len(data) == 0 {
		return 0
	}
	return strings.Count(string(data), "\n") + 1
}

// emptyInterface shows any, the interface every type satisfies, and why generics often beat it.
func emptyInterface() {
	fmt.Println("\n--- the empty interface and any ---")

	// any is a predeclared ALIAS for interface{} (Go 1.18) - the same type, a better name.
	var a any = 1
	var b interface{} = a // no conversion needed
	fmt.Printf("  any and interface{} are one type: %T holds %v\n", b, b)

	// It says nothing, so it supports NO operations until you assert it back. It is not a
	// supertype - it is erasure.
	//	fmt.Println(a + 1) // ERROR: invalid operation: a + 1 (mismatched types any and untyped int)
	fmt.Println("  a.(int) + 1 =", a.(int)+1)

	// Storing a non-pointer value in an any usually allocates (the data word must point
	// somewhere) - why Println(x) allocates and hot loops avoid any. Legitimate uses: fmt's
	// ...any, unknown JSON, context.WithValue, pre-generics containers.

	// Unknown JSON decodes into a documented set: bool, float64 (EVERY number), string, nil,
	// []any, map[string]any. Forgetting that 36 is a float64 is a classic bug.
	var decoded any
	_ = json.Unmarshal([]byte(`{"name":"Ada","age":36,"tags":["x"],"active":true,"note":null}`), &decoded)
	obj := decoded.(map[string]any)
	for _, k := range []string{"name", "age", "tags", "active", "note"} {
		fmt.Printf("    %-7s -> %-14T %v\n", k, obj[k], obj[k])
	}

	// A []any container loses the element type: every read needs an assertion, a wrong one
	// panics at run time. A generic stack[T] (module 010) keeps it - no boxing, no assertion.
	anyStack := []any{1, "two", 3.0}
	fmt.Printf("  []any: %v - read back with s[0].(int)\n", anyStack)
	typed := &stack[int]{}
	typed.Push(1)
	typed.Push(2)
	v, _ := typed.Pop()
	fmt.Printf("  stack[int].Pop() = %d (%T), no assertion\n", v, v)
}

// stack is a minimal generic stack; module 010 explains it.
type stack[T any] struct{ items []T }

func (s *stack[T]) Push(v T) { s.items = append(s.items, v) }

func (s *stack[T]) Pop() (T, bool) {
	var zero T
	if len(s.items) == 0 {
		return zero, false
	}
	v := s.items[len(s.items)-1]
	s.items = s.items[:len(s.items)-1]
	return v, true
}

// stdlibInterfaces shows the standard interfaces that plug a type into the ecosystem.
func stdlibInterfaces() {
	fmt.Println("\n--- standard library interfaces worth knowing ---")

	// fmt.Stringer - String() string - is used by %v, %s and Println.
	t := temperature(21.5)
	fmt.Printf("  fmt.Stringer: %v / %s\n", t, t)

	// encoding.TextMarshaler/TextUnmarshaler: one pair serves JSON values, JSON map KEYS
	// (they must encode to strings), XML and YAML - usually better than json.Marshaler.
	value, _ := json.Marshal(map[string]temperature{"outside": t})
	key, _ := json.Marshal(map[temperature]string{t: "outside"})
	fmt.Printf("  TextMarshaler as a JSON value: %s, as a map key: %s\n", value, key)
	var parsed temperature
	_ = parsed.UnmarshalText([]byte("18.0°C"))
	fmt.Println("  TextUnmarshaler round-trip:", parsed)

	// io.Reader/io.Writer - the most important pair: implement them and io.Copy, bufio,
	// the encoding packages, net/http, compression and hashing work with your type.
	var buf bytes.Buffer
	fmt.Fprint(upperWriter{w: &buf}, "wrapping an io.Writer is trivial")
	fmt.Printf("  io.Writer decorator: %q\n", buf.String())
	var dst bytes.Buffer
	n, _ := io.Copy(&dst, strings.NewReader("io.Copy joins any Reader to any Writer"))
	fmt.Printf("  io.Copy moved %d bytes: %q\n", n, dst.String())
	// io.Closer, io.Seeker, io.ReaderFrom, io.WriterTo are optional, found by assertion.

	// sort.Interface (Len, Less, Swap) still fits things that are not plain slices; for
	// ordinary slices prefer slices.SortFunc (Go 1.21, module 012).
	shapes := byArea{square{Side: 3}, circle{R: 1}, square{Side: 1}}
	sort.Sort(shapes)
	fmt.Print("  sort.Interface, ascending by area:")
	for _, s := range shapes {
		fmt.Printf(" %.2f", s.Area())
	}
	fmt.Println()

	// Also: error (module 009), json.Marshaler/Unmarshaler, fmt.Formatter and fmt.GoStringer
	// (full control of %v and %#v, rarely needed). context.Context is accepted, not implemented.
}

type temperature float64

// String converts to float64 first: formatting t itself with %v would call String again forever.
func (t temperature) String() string { return fmt.Sprintf("%.1f°C", float64(t)) }

func (t temperature) MarshalText() ([]byte, error) { return []byte(t.String()), nil }

func (t *temperature) UnmarshalText(b []byte) error {
	var f float64
	if _, err := fmt.Sscanf(string(b), "%f°C", &f); err != nil {
		return fmt.Errorf("temperature: parsing %q: %w", b, err)
	}
	*t = temperature(f)
	return nil
}

// upperWriter is an io.Writer decorator that upper-cases everything passing through.
type upperWriter struct{ w io.Writer }

func (u upperWriter) Write(p []byte) (int, error) {
	if _, err := u.w.Write([]byte(strings.ToUpper(string(p)))); err != nil {
		return 0, err
	}
	// The io.Writer contract: report how much of p was consumed, not how many bytes came out
	// (ToUpper can change the length: "ß" -> "SS").
	return len(p), nil
}

type byArea []shape

func (s byArea) Len() int           { return len(s) }
func (s byArea) Less(i, j int) bool { return s[i].Area() < s[j].Area() }
func (s byArea) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
