// Command 010_generics is module 010 of the Go course: generics.
//
// Generics arrived in Go 1.18 and grew in every release since; Go 1.27 added the last big piece,
// generic METHODS. Keep one sentence in mind: a constraint is an interface, and an interface now
// describes not just a set of methods but a SET OF TYPES. Everything else follows from that.
//
//	go run ./basics/010_generics
package main

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"iter"
	"maps"
	"math"
	"math/rand/v2"
	"slices"
	"strings"
	"sync"
)

func main() {
	typeParameters()        // [T any], instantiation, zero values, what any permits
	constraintsAsTypeSets() // unions, ~T, methods + types, comparable, cmp.Ordered
	genericFunctions()      // Map/Filter/Reduce, and the stdlib versions to prefer
	genericTypes()          // Stack[T], generic interfaces, options, typed context keys
	typeInference()         // argument, constraint and untyped-constant inference; 1.21 and 1.27
	genericAliases()        // type Set[T comparable] = map[T]struct{} (Go 1.24)
	genericsAndIterators()  // lazy adapters over iter.Seq
	genericMethods()        // methods with their own type parameters (Go 1.27)
	stdlibGenerics()        // slices, maps, cmp, sync.OnceValue, errors.AsType, ...
	costAndLimits()         // GC-shape stenciling, hard limits, when not to use generics
	commonErrors()          // the messages you will meet, and where to read more
}

// typeParameters shows the syntax: a type parameter list in square brackets after the name.
func typeParameters() {
	fmt.Println("--- type parameters ---")

	// Every parameter has a name and a constraint; [T, U any] share one, [K comparable, V any]
	// do not. Instantiation substitutes type arguments and yields an ordinary function -
	// explicitly (identity[int]) or, usually, by inference.
	fmt.Println("  identity[int](42)      =", identity[int](42))
	fmt.Println("  identity(\"inferred\")   =", identity("inferred"))
	s, n := swap(1, "one")
	fmt.Printf("  swap(1, \"one\")         = (%v, %v)\n", s, n)

	// `var zero T` is the only way to get the zero value of an unknown type.
	fmt.Printf("  zero values: int=%v string=%q bool=%v slice=%v\n", zero[int](), zero[string](), zero[bool](), zero[[]int]())

	// A generic type must be instantiated before it can be used as a type.
	p := pair[string, int]{Key: "answer", Value: 42}
	fmt.Println("  pair[string, int]:", p)
	//	var bare pair // ERROR: cannot use generic type pair[K comparable, V any] without instantiation

	// In the body only operations EVERY type of the constraint supports are allowed. any
	// permits nothing but assignment and the zero value - not +, not ==, not <:
	//	func sum[T any](a, b T) T { return a + b } // ERROR: invalid operation: operator + not defined on a (variable of type T constrained by any)

	// Only declarations take type parameters - never struct fields or func types (see
	// costAndLimits). Methods could not declare their own until Go 1.27 (genericMethods).
}

func identity[T any](v T) T { return v }

func swap[T, U any](a T, b U) (U, T) { return b, a }

func zero[T any]() T {
	var zero T
	return zero
}

type pair[K comparable, V any] struct {
	Key   K
	Value V
}

func (p pair[K, V]) String() string { return fmt.Sprintf("%v=%v", p.Key, p.Value) }

// constraintsAsTypeSets shows interfaces that list types, not only methods.
func constraintsAsTypeSets() {
	fmt.Println("\n--- constraints as type sets ---")

	// A union (|) of type elements puts those types in the set; see number below.
	fmt.Println("  sum([]int{1, 2, 3})       =", sum([]int{1, 2, 3}))
	fmt.Println("  sum([]float64{1.5, 2.5})  =", sum([]float64{1.5, 2.5}))

	// ~int means "any type whose UNDERLYING type is int". Without ~ only int itself matches and
	// every defined type in the caller's code is silently rejected - almost always write ~.
	mine := []myInt{1, 2, 3}
	fmt.Println("  sum([]myInt{1, 2, 3})     =", sum(mine), "<- thanks to ~int")
	//	fmt.Println(sumExact(mine)) // ERROR: myInt does not satisfy exactInt (possibly missing ~ for int in exactInt)
	fmt.Println("  sumExact([]int{1, 2, 3})  =", sumExact([]int{1, 2, 3}))

	// Write such type sets yourself. golang.org/x/exp/constraints (Integer, Float, Ordered) is
	// experimental, outside the Go 1 compatibility promise and an extra dependency - and its
	// Ordered is just the standard cmp.Ordered (Go 1.21): integers, floats and strings, for <.
	fmt.Printf("  minOf via cmp.Ordered: %v / %q\n", minOf(5, 3), minOf("b", "a"))

	// A constraint may require methods AND type elements at once.
	fmt.Println("  describe(myInt(21))       =", describe(myInt(21)))

	// Structural operations need every type in the set to support them: S ~[]E may be
	// indexed and ranged over, a mixed set may not:
	//	func f[T []int | map[string]int](t T) { for range t {} } // ERROR: cannot range over t (variable of type T constrained by []int | map[string]int): []int and map[string]int have different underlying types
	type idList []string
	v, ok := first(idList{"first", "second"})
	fmt.Printf("  first(idList{...})        = %q ok=%t\n", v, ok)

	// An interface with type elements is a constraint ONLY, never a variable type.
	//	var x number // ERROR: cannot use type number outside a type constraint: interface contains type constraints

	// comparable = the types supporting == and !=; slices, maps and funcs are rejected at
	// compile time. But an interface type like any satisfies comparable (Go 1.20) - "satisfies
	// but does not implement" - so == on it can still panic at run time.
	fmt.Println("  contains([]string{\"a\", \"b\"}, \"b\") =", contains([]string{"a", "b"}, "b"))
	//	contains([][]int{{1}}, []int{1}) // ERROR: []int does not satisfy comparable
}

type number interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 |
		~float32 | ~float64
}

type exactInt interface{ int } // no ~: exactly int

type printableNumber interface {
	number
	String() string
}

type myInt int

func (m myInt) String() string { return fmt.Sprintf("myInt(%d)", int(m)) }

func sum[T number](values []T) T {
	var total T
	for _, v := range values {
		total += v
	}
	return total
}

func sumExact[T exactInt](values []T) T {
	var total T
	for _, v := range values {
		total += v
	}
	return total
}

func describe[T printableNumber](v T) string { return fmt.Sprintf("%s (doubled: %v)", v.String(), v*2) }

func first[S ~[]E, E any](s S) (E, bool) {
	if len(s) == 0 {
		var zero E
		return zero, false
	}
	return s[0], true
}

func contains[T comparable](s []T, v T) bool {
	for _, e := range s {
		if e == v {
			return true
		}
	}
	return false
}

// genericFunctions shows type-safe code without duplication - and what the stdlib already has.
func genericFunctions() {
	fmt.Println("\n--- generic functions ---")

	// Before 1.18 the choices were code generation, copy-paste per type, or any + assertions
	// (run-time errors, allocations). Use the WEAKEST constraint the body compiles with:
	// comparable for ==, cmp.Ordered for <, a type set only for arithmetic.
	nums := []int{1, 2, 3, 4, 5}
	fmt.Println("  mapSlice to int:   ", mapSlice(nums, func(n int) int { return n * 2 }))
	fmt.Println("  mapSlice to string:", mapSlice(nums, func(n int) string { return fmt.Sprintf("#%d", n) }), "<- U inferred")
	fmt.Println("  filterSlice even:  ", filterSlice(nums, func(n int) bool { return n%2 == 0 }))
	fmt.Println("  reduce to int:     ", reduce(nums, 0, func(acc, n int) int { return acc + n }))
	fmt.Printf("  reduce to string:   %q\n", reduce(nums, "", func(acc string, n int) string { return acc + fmt.Sprint(n) }))

	// The standard library has no Map/Filter/Reduce on purpose - a for loop is clearer, and
	// iterators (module 012) cover composition. But reach for what IS there before writing it:
	fmt.Printf("  slices.Contains=%t slices.Index=%d slices.Max=%d min/max builtins=%d/%d\n",
		slices.Contains(nums, 3), slices.Index(nums, 3), slices.Max(nums), min(3, 1, 2), max(3, 1, 2))
	sorted := slices.Clone(nums)
	slices.SortFunc(sorted, func(a, b int) int { return cmp.Compare(b, a) })
	fmt.Println("  slices.SortFunc descending:", sorted)

	// maps.Keys/Values return an iter.Seq, not a slice - wrap them in slices.Sorted or Collect.
	m := map[string]int{"b": 2, "a": 1, "c": 3}
	fmt.Println("  slices.Sorted(maps.Keys(m)):", slices.Sorted(maps.Keys(m)))

	// A generic function can be passed to another, but its type arguments must be fixed by
	// then (inferred or explicit) - see typeInference.
}

func mapSlice[T, U any](s []T, f func(T) U) []U {
	out := make([]U, len(s))
	for i, v := range s {
		out[i] = f(v)
	}
	return out
}

func filterSlice[T any](s []T, keep func(T) bool) []T {
	var out []T
	for _, v := range s {
		if keep(v) {
			out = append(out, v)
		}
	}
	return out
}

func reduce[T, A any](s []T, initial A, f func(A, T) A) A {
	acc := initial
	for _, v := range s {
		acc = f(acc, v)
	}
	return acc
}

// genericTypes shows types with type parameters, and two idioms they enable.
func genericTypes() {
	fmt.Println("\n--- generic types ---")

	// stack[int] is an ordinary type once instantiated. The receiver repeats the parameters,
	// func (s *stack[T]) Push(v T), but declared no new ones before Go 1.27.
	ints := &stack[int]{}
	ints.Push(1)
	ints.Push(2)
	v, _ := ints.Pop()
	fmt.Printf("  stack[int]:     Pop()=%v (%T) size=%d\n", v, v, ints.Size())
	// The usual "useful zero value" rule: popping an empty stack gives T's zero value and false.
	z, ok := (&stack[float64]{}).Pop()
	fmt.Printf("  empty stack:    Pop()=%v ok=%t\n", z, ok)
	// The same stack[T] lives in examples/common/stack.go - exercise 2 in notes.md.

	// Generic interfaces work the same way.
	var c container[string] = &sliceContainer[string]{}
	c.Add("x")
	c.Add("y")
	fmt.Printf("  container[string]: Get(1)=%q Len()=%d\n", c.Get(1), c.Len())

	memo := newCache[string, []int]()
	memo.Set("primes", []int{2, 3, 5})
	got, _ := memo.Get("primes")
	fmt.Printf("  cache[string, []int]: %v (%T) - no assertion anywhere\n", got, got)

	// Recursion is fine with the SAME argument; a new argument per level would need infinitely
	// many instantiations:
	//	type deep[T any] struct{ next *deep[[]T] } // ERROR: instantiation cycle: T instantiated as []T
	head := &node[string]{Value: "a", Next: &node[string]{Value: "b"}}
	fmt.Print("  node[string] list:")
	for n := head; n != nil; n = n.Next {
		fmt.Print(" ", n.Value)
	}
	fmt.Println()

	// Generic functional options (module 005 has the concrete version): one option[T] type
	// serves every configurable struct, and T is inferred at the call site.
	srv := build(func(e *endpoint) { e.Host = "example.com" }, withPort(8443))
	fmt.Printf("  generic functional options: %+v\n", *srv)

	// Typed context keys (module 011 asks for an unexported key type): the key's type
	// parameter fixes the VALUE type, so retrieval can no longer assert the wrong type.
	ctx := withValue(context.Background(), userKey, "ada")
	ctx = withValue(ctx, retriesKey, 3)
	user, userOK := valueOf(ctx, userKey)          // (string, bool)
	retries, retriesOK := valueOf(ctx, retriesKey) // (int, bool)
	fmt.Printf("  typed context keys: user=%q(%t) retries=%d(%t)\n", user, userOK, retries, retriesOK)
}

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

func (s *stack[T]) Size() int { return len(s.items) }

type container[T any] interface {
	Add(T)
	Get(i int) T
	Len() int
}

type sliceContainer[T any] struct{ items []T }

func (c *sliceContainer[T]) Add(v T)     { c.items = append(c.items, v) }
func (c *sliceContainer[T]) Get(i int) T { return c.items[i] }
func (c *sliceContainer[T]) Len() int    { return len(c.items) }

type cache[K comparable, V any] struct {
	mu    sync.RWMutex
	items map[K]V
}

func newCache[K comparable, V any]() *cache[K, V] { return &cache[K, V]{items: make(map[K]V)} }

func (c *cache[K, V]) Set(k K, v V) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[k] = v
}

func (c *cache[K, V]) Get(k K) (V, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, ok := c.items[k]
	return v, ok
}

type node[T any] struct {
	Value T
	Next  *node[T]
}

type option[T any] func(*T)

type endpoint struct {
	Host string
	Port int
}

func withPort(p int) option[endpoint] { return func(e *endpoint) { e.Port = p } }

func build[T any](opts ...option[T]) *T {
	var t T
	for _, opt := range opts {
		opt(&t)
	}
	return &t
}

// ctxKey ties a context key to its value type; T is deliberately unused in the struct.
type ctxKey[T any] struct{ name string }

var (
	userKey    = ctxKey[string]{"user"}
	retriesKey = ctxKey[int]{"retries"}
)

func withValue[T any](ctx context.Context, key ctxKey[T], v T) context.Context {
	return context.WithValue(ctx, key, v)
}

func valueOf[T any](ctx context.Context, key ctxKey[T]) (T, bool) {
	v, ok := ctx.Value(key).(T)
	return v, ok
}

// typeInference shows how the compiler finds type arguments, and where it gives up.
func typeInference() {
	fmt.Println("\n--- type inference ---")

	// 1. Function argument inference: the argument types give T.
	fmt.Printf("  minOf(5, 3)          = %v, minOf(\"b\", \"a\") = %q\n", minOf(5, 3), minOf("b", "a"))

	// 2. Constraint type inference: with [S ~[]E, E any], S = names gives E = string although
	// no argument mentions E (go.dev/blog/deconstructing-type-parameters explains the shape).
	type names []string
	l, _ := last(names{"first", "last"})
	fmt.Printf("  last(names{...})     = %q\n", l)

	// 3. Untyped constants are reconciled with EACH OTHER first, then with typed arguments.
	fmt.Printf("  minOf(1, 2.0)        = %v (%T) <- untyped int and float reconcile\n", minOf(1, 2.0), minOf(1, 2.0))
	//	minOf(1, "a") // ERROR: in call to minOf, mismatched types untyped int and untyped string (cannot infer T)

	// A TYPED argument fixes T; constants must then be representable in it, and two typed
	// arguments of different types never unify - convert one, or instantiate.
	var typedInt int = 1
	fmt.Printf("  minOf(typedInt, 2.0) = %v (%T) <- 2.0 is representable as int\n", minOf(typedInt, 2.0), minOf(typedInt, 2.0))
	//	minOf(typedInt, 2.5) // ERROR: cannot use 2.5 (untyped float constant) as int value in argument to minOf (truncated)
	//	var typedFloat float64 = 2
	//	minOf(typedInt, typedFloat) // ERROR: in call to minOf, type float64 of typedFloat does not match inferred type int for T

	// Inference never looks at the assignment target of a CALL, so a parameter used only in
	// the result must be given explicitly:
	//	var x []string = zero() // ERROR: in call to zero, cannot infer T (declared at ...)
	fmt.Println("  zero[[]string]()     =", zero[[]string](), "(nil)")

	// Go 1.21: a generic FUNCTION VALUE is inferred from its target type - a func-typed
	// variable, a parameter it is passed to, a result it is returned as.
	var minInt func(int, int) int = minOf
	fmt.Println("  var f func(int, int) int = minOf ->", minInt(4, 7))
	fmt.Println("  combineWith(minOf, 9, 2)        ->", combineWith(minOf, 9, 2))
	fmt.Println("  return minOf                    ->", pickMin()(3, 8))

	// The Go 1.27 spec states it as one rule: inference applies in EVERY assignment context -
	// struct fields, map and slice elements, channel sends, conversions. (The gc compiler
	// already accepted these with -lang=go1.21.)
	type holder struct{ Reduce func(string, string) string }
	h := holder{Reduce: minOf}
	table := map[string]func(int) int{"identity": identity}
	fmt.Printf("  struct field %q, map value %v\n", h.Reduce("b", "a"), table["identity"](5))

	// You may always instantiate explicitly - when inference fails, or when it reads clearer.
	fmt.Println("  apply with an inferred closure:", apply(func(n int) int { return n * 3 }, 7))
}

func minOf[T cmp.Ordered](a, b T) T {
	if a < b {
		return a
	}
	return b
}

func last[S ~[]E, E any](s S) (E, bool) {
	if len(s) == 0 {
		var zero E
		return zero, false
	}
	return s[len(s)-1], true
}

func apply[T any](f func(T) T, v T) T { return f(v) }

func combineWith(f func(int, int) int, a, b int) int { return f(a, b) }

func pickMin() func(int, int) int { return minOf } // T inferred from the result type

// genericAliases shows aliases with type parameters (Go 1.24).
func genericAliases() {
	fmt.Println("\n--- generic type aliases (Go 1.24) ---")

	// Before 1.24 `type set[T comparable] = map[T]struct{}` was a syntax error. An alias is
	// still the SAME type as its target: assignable both ways, no conversion.
	s := set[string]{"a": {}, "b": {}}
	var plain map[string]struct{} = s
	var back set[string] = plain
	fmt.Println("  set[string] and map[string]struct{} are one type: len =", len(back))

	var upper transformer[string, string] = strings.ToUpper
	fmt.Printf("  transformer[string, string] = strings.ToUpper: %q\n", upper("alias"))
	fmt.Println("  stringMap[int]:", stringMap[int]{"one": 1})

	// An alias of a generic struct carries the target's methods - it IS the target.
	r := result[int](success(42))
	bad := failure[int](errors.New("failed"))
	fmt.Printf("  result[int] = outcome[int]: value=%v ok=%t | failure: ok=%t err=%v\n", r.Value, r.IsOK(), bad.IsOK(), bad.Err)

	// But an alias cannot declare methods of its own - for that, define a type (no =):
	//	func (s set[T]) Add(v T) {} // ERROR: cannot define new methods on generic alias type set[T comparable]
	// Uses: moving a generic type between packages without breaking the old name, and short
	// names for long instantiations.
}

type (
	set[T comparable]     = map[T]struct{}
	result[T any]         = outcome[T]
	stringMap[V any]      = map[string]V
	transformer[T, U any] = func(T) U
)

type outcome[T any] struct {
	Value T
	Err   error
}

func (o outcome[T]) IsOK() bool { return o.Err == nil }

func success[T any](v T) outcome[T] { return outcome[T]{Value: v} }

func failure[T any](err error) outcome[T] { return outcome[T]{Err: err} }

// genericsAndIterators shows generic adapters over iter.Seq, where Map/Filter finally pay off.
func genericsAndIterators() {
	fmt.Println("\n--- generics and iterators ---")

	// iter.Seq[V] is just the generic func type func(yield func(V) bool) (Go 1.23). Over a
	// slice Map/Filter beat nothing; over a Seq they compose LAZILY, with no intermediate
	// slices, and stop as soon as the consumer does. Module 012 covers iterators in full.
	pipeline := takeSeq(
		mapSeq(
			filterSeq(naturals(), func(n int) bool { return n%3 == 0 }),
			func(n int) string { return fmt.Sprintf("<%d>", n) },
		), 5)
	fmt.Print("  first 5 multiples of 3 from an INFINITE source:")
	for s := range pipeline {
		fmt.Print(" ", s)
	}
	fmt.Println()

	// Bridges: slices.Values/All/Collect/Sorted, maps.Keys/Values/All/Collect.
	nums := []int{4, 1, 3, 2}
	evens := slices.Collect(filterSeq(slices.Values(nums), func(n int) bool { return n%2 == 0 }))
	fmt.Println("  slices.Values -> filterSeq -> slices.Collect:", evens)
	fmt.Println("  slices.Sorted(slices.Values(nums)):          ", slices.Sorted(slices.Values(nums)))
}

func mapSeq[T, U any](seq iter.Seq[T], f func(T) U) iter.Seq[U] {
	return func(yield func(U) bool) {
		for v := range seq {
			if !yield(f(v)) {
				return
			}
		}
	}
}

func filterSeq[T any](seq iter.Seq[T], keep func(T) bool) iter.Seq[T] {
	return func(yield func(T) bool) {
		for v := range seq {
			if keep(v) && !yield(v) {
				return
			}
		}
	}
}

func takeSeq[T any](seq iter.Seq[T], n int) iter.Seq[T] {
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
				return // stop before pulling element n+1 from the source
			}
		}
	}
}

func naturals() iter.Seq[int] {
	return func(yield func(int) bool) {
		for i := 1; ; i++ {
			if !yield(i) {
				return
			}
		}
	}
}

// genericMethods shows methods that declare type parameters of their own (Go 1.27).
func genericMethods() {
	fmt.Println("\n--- generic methods (Go 1.27) ---")

	// Before 1.27 a method could only use the receiver's parameters, so the workaround was a
	// package-level function: slices.SortFunc(s, f), not s.SortFunc(f). Now list[E].Apply may
	// declare its own F - the spec's own example. With `go 1.26` in go.mod you still get:
	//	func (l list[E]) Apply[F any](...) // ERROR: generic method requires go1.27 or later (-lang was set to go1.26; check go.mod)
	nums := list[int]{1, 2, 3}
	labels := nums.Apply(func(n int) string { return fmt.Sprintf("n%d", n) })
	widths := nums.Apply(func(n int) float64 { return float64(n) * 1.5 })
	fmt.Printf("  Apply -> %v (%T), %v (%T)\n", labels, labels, widths, widths)

	// Chaining generic methods - impossible before 1.27.
	joined := nums.
		Apply(func(n int) string { return strings.Repeat("*", n) }).
		Fold("", func(acc, s string) string { return acc + s + "|" })
	fmt.Printf("  chained Apply then Fold: %q\n", joined)

	// Explicit instantiation; a method VALUE must be instantiated (or inferred from its target).
	fmt.Println("  nums.Apply[bool](odd):", nums.Apply[bool](func(n int) bool { return n%2 == 1 }))
	//	asValue := nums.Apply // ERROR: cannot use generic function nums.Apply without instantiation
	asValue := nums.Apply[string]
	fmt.Println("  method value nums.Apply[string]:", asValue(func(n int) string { return fmt.Sprint(n * 10) }))

	// The standard library adopted it at once: (*rand.Rand).N, formerly only rand.N.
	r := rand.New(rand.NewPCG(1, 2))
	fmt.Printf("  (*rand.Rand).N: N(100)=%d N(int64(5))=%d\n", r.N(100), r.N(int64(5)))

	// The restriction that remains: a generic method cannot satisfy an interface - it has
	// infinitely many signatures, an interface method has one. Keep methods that must satisfy
	// an interface non-generic:
	//	type applier interface{ Apply[F any](func(int) F) list[F] } // ERROR: interface method must have no type parameters
	var ln lener = nums // Len is not generic, so this works
	fmt.Println("  the non-generic Len() satisfies an interface:", ln.Len())

	// Use it where a transformation truly belongs to the receiver's type; do not rush to turn
	// package-level generic functions into methods - slices.SortFunc reads perfectly well.
}

type list[E any] []E

// Apply declares F in addition to the receiver's E.
func (l list[E]) Apply[F any](f func(E) F) list[F] {
	out := make(list[F], len(l))
	for i, v := range l {
		out[i] = f(v)
	}
	return out
}

func (l list[E]) Fold[A any](initial A, f func(A, E) A) A {
	acc := initial
	for _, v := range l {
		acc = f(acc, v)
	}
	return acc
}

func (l list[E]) Len() int { return len(l) }

type lener interface{ Len() int }

// stdlibGenerics tours where the standard library itself uses generics.
func stdlibGenerics() {
	fmt.Println("\n--- generics in the standard library ---")

	// sync.OnceValue / OnceValues / OnceFunc (1.21): lazy initialisation without sync.Once
	// boilerplate or a package-level variable.
	expensive := sync.OnceValue(func() string {
		fmt.Println("    (the expensive computation runs exactly once)")
		return "computed"
	})
	fmt.Printf("  sync.OnceValue: %q then %q\n", expensive(), expensive())
	v, err := sync.OnceValues(func() (int, error) { return 42, nil })()
	fmt.Println("  sync.OnceValues:", v, err)

	// cmp (1.21): Ordered, Compare, Less; cmp.Or (1.22) returns the first non-zero argument.
	fmt.Printf("  cmp.Or(\"\", \"\", \"fallback\") = %q\n", cmp.Or("", "", "fallback"))
	fmt.Printf("  cmp.Compare(1, 2)=%d; NaN-safe, NaN sorts first: cmp.Compare(NaN, 1)=%d\n",
		cmp.Compare(1, 2), cmp.Compare(math.NaN(), 1))

	// slices (1.21): Sort, SortFunc, Contains, Index, Max, Min, Reverse, Clone, Equal, Compact,
	// Insert, Delete, BinarySearch, and the 1.23 bridges All, Values, Collect, Sorted, Backward.
	// maps (1.21): Clone, Equal, EqualFunc, Copy, DeleteFunc; 1.23: Keys, Values, All, Collect, Insert.
	fmt.Println("  slices.Compact(slices.Sorted(...)):", slices.Compact(slices.Sorted(slices.Values([]int{3, 1, 2, 2}))))
	idx, found := slices.BinarySearch([]int{1, 2, 3}, 2)
	fmt.Printf("  slices.BinarySearch: index=%d found=%t\n", idx, found)

	// errors.AsType[T] (1.26) - the generic errors.As (module 009).
	wrapped := fmt.Errorf("outer: %w", &parseError{Line: 3})
	if pe, ok := errors.AsType[*parseError](wrapped); ok {
		fmt.Println("  errors.AsType[*parseError]: line", pe.Line)
	}

	// Also: iter.Seq/Seq2/Pull (1.23); unique.Make[T] interning (1.23); weak.Pointer[T] (1.24);
	// reflect.TypeAssert[T] (1.25); maphash.Hasher[T]/ComparableHasher[T] and the generic
	// (*rand.Rand).N method (1.27). sync.Map is a notable NON-adopter: still any-based, for
	// compatibility. Deliberately absent: Optional[T], Result[T], Set[T], Map/Filter.
}

type parseError struct{ Line int }

func (e *parseError) Error() string { return fmt.Sprintf("parse error at line %d", e.Line) }

// costAndLimits shows how generics are compiled, their hard limits, and when to avoid them.
func costAndLimits() {
	fmt.Println("\n--- cost, limits and when not to use generics ---")

	// GC-shape stenciling with dictionaries - neither C++'s copy per type nor Java's boxing.
	// Types with the same GC shape (size + pointer layout) share one compiled copy, and a hidden
	// dictionary carries the type details: all pointer-shaped types (*T, map, chan, func) share
	// one; a slice, an interface, int, float64 and each struct shape get their own.
	// Result: modest binaries; sometimes slower than concrete code (the dictionary indirection
	// can block inlining) but usually faster than any + assertions. Measure before assuming:
	// go test -bench . -benchmem (module 014).
	nums := []int{5, 2, 8, 1}
	fmt.Printf("  concrete:  maxInt(%v) = %d\n", nums, maxInt(nums))
	fmt.Printf("  generic:   slices.Max(%v) = %d\n", nums, slices.Max(nums))
	fmt.Printf("  any-based: maxAny(...) = %v <- boxes, asserts, fails at RUN time:\n", maxAny([]any{5, 2, 8, 1}))
	fmt.Println("    maxAny([]any{1, \"two\"}):", catchPanic(func() { maxAny([]any{1, "two"}) }))

	// Hard limits: no type parameters on struct fields or func types,
	//	type bad[T any] struct{ f func[U any](U) U } // ERROR: syntax error: function type must have no type parameters
	// a generic method cannot satisfy an interface; no specialisation (no faster version for
	// one T); no "any type with +" except by listing the types; no variadic type parameters,
	// no higher-kinded types; comparable is compile-time only (an any inside can still panic).

	// When NOT to use them (go.dev/blog/when-generics): behaviour that differs per type is an
	// INTERFACE; one call site is concrete code; field names and tags are reflection; a data
	// structure used with one type stays concrete. Write the concrete code first and let the
	// duplication tell you when a type parameter earns its place.
	for _, s := range []shape{circle{R: 1}, square{Side: 2}} {
		fmt.Printf("  interface - each type computes it differently: %-11T Area()=%.4f\n", s, s.Area())
	}
	fmt.Println("  type parameter - identical logic for every T: contains =",
		contains([]int{1, 2}, 2), contains([]string{"a"}, "b"))
}

func maxInt(s []int) int {
	best := s[0]
	for _, v := range s[1:] {
		best = max(best, v)
	}
	return best
}

// maxAny is the pre-generics version: it boxes every element and fails at run time on bad input.
func maxAny(s []any) any {
	best := s[0].(int)
	for _, v := range s[1:] {
		best = max(best, v.(int))
	}
	return best
}

type shape interface{ Area() float64 }

type circle struct{ R float64 }
type square struct{ Side float64 }

func (c circle) Area() float64 { return math.Pi * c.R * c.R }
func (s square) Area() float64 { return s.Side * s.Side }

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

// commonErrors summarises the compiler messages met above, each verified with Go 1.27.
func commonErrors() {
	fmt.Println("\n--- common errors ---")

	for _, e := range []struct{ message, fix string }{
		{"operator + not defined on a (... constrained by any)", "constrain to a type set with +"},
		{"X does not satisfy Y (possibly missing ~ for int in Y)", "add ~ to the type element"},
		{"cannot use type N outside a type constraint", "it is a constraint, not a type"},
		{"in call to F, cannot infer T", "instantiate explicitly: F[int](...)"},
		{"mismatched types untyped int and untyped string", "make the literals one kind"},
		{"type float64 of f does not match inferred type int", "convert one, or instantiate"},
		{"cannot use generic type S[T any] without instantiation", "write S[int]"},
		{"cannot use generic function x.M without instantiation", "write x.M[int]"},
		{"interface method must have no type parameters", "keep the method non-generic"},
		{"function type must have no type parameters", "only declarations may have them"},
		{"[]int does not satisfy comparable", "a comparable key, or a compare func"},
		{"cannot define new methods on generic alias type", "a definition (no =), not an alias"},
	} {
		fmt.Printf("  %-55s -> %s\n", e.message, e.fix)
	}

	// Read more:
	//	https://go.dev/ref/spec                            Type parameters, Type constraints, Instantiations
	//	https://go.dev/blog/intro-generics                 the tutorial
	//	https://go.dev/blog/when-generics                  when NOT to use them
	//	https://go.dev/blog/deconstructing-type-parameters why [S ~[]E, E any]
	//	https://go.dev/blog/range-functions                the iterator design
	//	https://go.dev/doc/go1.27                          generic methods, inference everywhere
	// In this course: module 012 (iter, slices, maps, cmp), module 016 (when each feature
	// landed), examples/common/stack.go and math.go (generics used in earnest).
}
