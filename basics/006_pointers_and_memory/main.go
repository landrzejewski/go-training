// Command 006_pointers_and_memory is module 006 of the Go course: pointers, value semantics,
// escape analysis, aliasing and unsafe.
//
// Almost every "why did my change not stick?" question in Go traces back to one sentence:
// Go has NO reference types - every assignment and every argument pass copies a value. What
// varies is what the value is: an int, a three-word slice header, a pointer to a hash table.
//
//	go run ./basics/006_pointers_and_memory
package main

import (
	"fmt"
	"runtime"
	"structs"
	"unsafe"
)

func main() {
	pointers()        // & and *, nil, no arithmetic, addressability, new vs &T{}
	valueSemantics()  // what a copy actually copies; the append discontinuity
	pointerOrValue()  // choosing a parameter type
	escapeAnalysis()  // stack vs heap, -gcflags=-m, allocation counts, GC
	aliasingHazards() // four aliasing bugs and their fixes
	unsafeAndLayout() // Sizeof/Offsetof, padding, structs.HostLayout, zero-copy strings
}

// pointers shows taking addresses, dereferencing and what Go does NOT allow with pointers.
func pointers() {
	fmt.Println("--- pointers ---")

	// &x takes the address, *p dereferences; *T is "pointer to T".
	x := 42
	p := &x
	fmt.Printf("x=%d  p=%p  *p=%d\n", x, p, *p)
	*p = 99
	fmt.Printf("after *p = 99: x=%d\n", x)

	// Pointers compare by address with == and != only.
	q := &x
	fmt.Printf("p == q: %t (same address)\n", p == q)

	// The zero value is nil, and dereferencing nil panics.
	var nilPtr *int
	fmt.Printf("var p *int -> %v, isNil=%t\n", nilPtr, nilPtr == nil)
	//	fmt.Println(*nilPtr) // panic: runtime error: invalid memory address or nil pointer dereference

	// No pointer arithmetic and no ordering. That is what lets Go have a precise GC and stay
	// memory-safe without a borrow checker (unsafe can do it - see unsafeAndLayout).
	//	p++                // ERROR: invalid operation: p++ (non-numeric type *int)
	//	p = p + 1          // ERROR: invalid operation: p + 1 (mismatched types *int and untyped int)
	//	fmt.Println(p < q) // ERROR: invalid operation: p < q (operator < not defined on pointer)

	// Returning &local is safe and idiomatic: escape analysis moves local to the heap.
	// In C it would dangle.
	fmt.Println("returned &local:", *makeLocal())

	// new(T) allocates a zeroed T; &T{...} does the same and can set fields - prefer it.
	viaNew := new(node)
	viaLiteral := &node{Value: 7}
	fmt.Printf("new(node) = %+v, &node{Value: 7} = %+v\n", *viaNew, *viaLiteral)

	// Field access and method calls dereference automatically: p.Field means (*p).Field,
	// which is why an explicit * is rare in Go.
	viaLiteral.Value = 8
	fmt.Printf("p.Value = %d == (*p).Value = %d\n", viaLiteral.Value, (*viaLiteral).Value)

	// Addressable: variables, slice elements, *p, fields of addressable structs.
	// NOT addressable: map entries, string bytes, constants, call results.
	s := []int{1, 2, 3}
	fmt.Printf("a slice element is addressable: &s[0] = %p\n", &s[0])
	m := map[string]int{"a": 1}
	_ = m
	//	fmt.Println(&m["a"])      // ERROR: invalid operation: cannot take address of m["a"] (map index expression of type int)
	//	fmt.Println(&"text")       // ERROR: invalid operation: cannot take address of "text" (untyped string constant)
	//	fmt.Println(&makeLocal()) // ERROR: invalid operation: cannot take address of makeLocal() (value of type *int)

	// The classic reason to want pointers at all: linked structures.
	list := &node{Value: 1, Next: &node{Value: 2, Next: &node{Value: 3}}}
	fmt.Print("linked list: ")
	for n := list; n != nil; n = n.Next {
		fmt.Print(n.Value, " ")
	}
	fmt.Println()
}

type node struct {
	Value int
	Next  *node
}

func makeLocal() *int {
	local := 7
	return &local // escape analysis promotes local to the heap
}

// valueSemantics shows that every copy is a copy - of a value that may itself contain a pointer.
func valueSemantics() {
	fmt.Println("\n--- value semantics ---")

	// What a copy copies, and whether it then shares data:
	//	int, float   the number                no
	//	array [N]T   all N elements            no
	//	struct       every field, recursively  no (unless a field is one of the below)
	//	string       pointer + len             yes, but the bytes are immutable - invisible
	//	slice        pointer + len + cap       YES - the elements
	//	map, chan    one pointer               YES - the whole table / channel
	//	func         one pointer (+ closure)   YES
	//	pointer      an address                YES - by definition
	//	interface    type word + data word     depends on what it holds
	type pt struct{ X, Y int }
	a := pt{1, 2}
	b := a
	b.X = 99
	arr1 := [3]int{1, 2, 3}
	arr2 := arr1
	arr2[0] = 99
	sl1 := []int{1, 2, 3}
	sl2 := sl1
	sl2[0] = 99
	fmt.Printf("struct: a=%v b=%v | array: %v %v | slice: %v %v (SHARED)\n", a, b, arr1, arr2, sl1, sl2)

	// A struct with slice and map fields is "half shared".
	orig := config{Name: "original", Tags: []string{"a", "b"}, Meta: map[string]string{"k": "v"}, Count: 1}
	copied := orig
	copied.Name = "copy"          // independent
	copied.Count = 2              // independent
	copied.Tags[0] = "MODIFIED"   // reaches orig.Tags too
	copied.Meta["k"] = "MODIFIED" // and orig.Meta
	fmt.Printf("orig:   %+v\ncopied: %+v\n", orig, copied)

	// append is the discontinuity: s[0] = x always reaches the shared array, but append writes
	// in place only when there is spare capacity. Same function, two outcomes:
	noRoom := []int{1, 2, 3} // len 3, cap 3
	appendInPlace(noRoom)
	fmt.Printf("no spare capacity, append reallocated: caller sees %v\n", noRoom)
	room := make([]int, 3, 10)
	copy(room, []int{1, 2, 3})
	appendInPlace(room)
	fmt.Printf("spare capacity, append wrote in place: backing array %v, yet len is still %d\n",
		room[:4], len(room))

	// So: to let a function grow a slice, return it (or, rarely, take a *[]T).
	// A map argument is always shared - there is no way to pass one by value.
	viaPointer := []int{1, 2, 3}
	appendViaPointer(&viaPointer)
	fmt.Printf("returning the slice: %v; via *[]T: %v\n", appendCorrectly([]int{1, 2, 3}), viaPointer)
}

type config struct {
	Name  string
	Tags  []string          // a header: shared on copy
	Meta  map[string]string // a pointer: shared on copy
	Count int               // a number: copied
}

func appendInPlace(s []int)         { s = append(s, 99); _ = s }
func appendCorrectly(s []int) []int { return append(s, 99) }
func appendViaPointer(s *[]int)     { *s = append(*s, 99) }

// pointerOrValue shows how to choose between T and *T: semantics first, performance second.
func pointerOrValue() {
	fmt.Println("\n--- pointer or value? ---")

	// Use a POINTER when the function must modify the caller's value...
	c := counter{}
	c.incByValue()
	c.incByValue()
	fmt.Printf("after two incByValue():   n=%d (each modified a copy)\n", c.n)
	c.incByPointer() // shorthand for (&c).incByPointer() - legal because c is addressable
	c.incByPointer()
	fmt.Printf("after two incByPointer(): n=%d\n", c.n)

	// ...or when the type is genuinely large. "Large" means hundreds of bytes: a 3-field struct
	// is cheaper to copy than to chase through a pointer.
	fmt.Printf("sizeof(small)=%d sizeof(*small)=%d sizeof(large)=%d bytes\n",
		unsafe.Sizeof(small{}), unsafe.Sizeof((*small)(nil)), unsafe.Sizeof(large{}))

	// ...or when the type must not be copied: anything holding a sync.Mutex, sync.WaitGroup,
	// strings.Builder or atomic.Int64. go vet's copylocks check catches it - always heed it:
	//	type guarded struct{ mu sync.Mutex }
	//	func lockAndUse(g guarded) {} // go vet: lockAndUse passes lock by value: training.pl/go/basics/006_pointers_and_memory.guarded contains sync.Mutex

	// ...or when nil must mean "absent", distinct from the zero value.
	fmt.Println("optional value via a pointer:", describe(nil), "|", describe(intPtr(0)))

	// Use a VALUE for small types, for immutability at the call site (a value parameter cannot
	// surprise the caller) and for natural values: time.Time, netip.Addr, coordinates, IDs.

	// Never take *[]T, *map or *chan just "to avoid a copy" - the copy is already a header.
	// Consistency beats both rules: if one method needs a pointer receiver, give all of them one
	// and pass the type as a pointer throughout (module 007).
}

type counter struct{ n int }

func (c counter) incByValue()    { c.n++ }
func (c *counter) incByPointer() { c.n++ }

type small struct{ X, Y, Z int }
type large struct{ Data [4096]byte }

func intPtr(n int) *int { return &n }

func describe(p *int) string {
	if p == nil {
		return "absent"
	}
	return fmt.Sprintf("present with value %d (not the same as absent, even at zero)", *p)
}

// escapeAnalysis shows that the compiler, not you, decides between stack and heap.
func escapeAnalysis() {
	fmt.Println("\n--- stack, heap and escape analysis ---")

	// A value whose lifetime the compiler cannot bound to the frame ESCAPES to the heap;
	// otherwise it stays on the (free) stack. Common reasons to escape: returning &local, storing
	// a pointer in a global/slice/map/channel, assigning to an interface (so fmt.Println(x)
	// allocates), a closure outliving the frame, a size unknown at compile time (make([]T, n)).
	// See the decisions with -m (and -m -m for the reasons):
	//	go build -gcflags='-m' ./basics/006_pointers_and_memory
	// prints lines such as these (NN = line number):
	//	basics/006_pointers_and_memory/main.go:NN:2: moved to heap: local                  <- makeLocal
	//	basics/006_pointers_and_memory/main.go:NN:14: "returned &local:" escapes to heap   <- boxed into any
	escapes := makeLocal()
	stays := staysOnStack()
	fmt.Printf("escaping local -> %d, non-escaping local -> %d\n", *escapes, stays)

	// What matters is the allocation COUNT. Preallocating removes the regrowth allocations - and
	// a constant-size make that does not escape may even live on the stack (0 heap allocations).
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	preallocated := make([]int, 0, 1000)
	for i := range 1000 {
		preallocated = append(preallocated, i)
	}
	runtime.ReadMemStats(&after)
	preallocCount := after.Mallocs - before.Mallocs

	runtime.ReadMemStats(&before)
	var regrown []int
	for i := range 1000 {
		regrown = append(regrown, i)
	}
	runtime.ReadMemStats(&after)
	fmt.Printf("appending 1000 ints: preallocated ~%d allocations, from nil ~%d\n",
		preallocCount, after.Mallocs-before.Mallocs)
	// Other tools: sync.Pool for buffers, no boxing into any in hot paths - and measure first
	// with go test -bench . -benchmem (module 014).

	// Goroutine stacks start at 2 KB and grow by copying (to 1 GB on 64-bit, 250 MB on 32-bit),
	// so deep recursion usually works. Because stacks MOVE, a pointer's numeric address is not
	// stable - never keep one as an integer.
	fmt.Printf("GOMAXPROCS=%d NumGoroutine=%d NumCPU=%d\n",
		runtime.GOMAXPROCS(0), runtime.NumGoroutine(), runtime.NumCPU())

	// The GC is concurrent, tri-colour mark-and-sweep - neither generational nor compacting,
	// tuned for low pauses. GOGC sets the growth target; GOMEMLIMIT (Go 1.19) a soft memory
	// ceiling, which together make Go usable in containers. The Green Tea collector (better
	// scanning locality) was opt-in in Go 1.25 and is the default since Go 1.26.
}

func staysOnStack() int {
	local := 7
	p := &local // the address never leaves this frame, so local stays on the stack
	return *p
}

// aliasingHazards shows the four aliasing bugs behind most surprises in real Go code.
func aliasingHazards() {
	fmt.Println("\n--- aliasing hazards ---")

	// 1. A sub-slice keeps the whole backing array alive. Copy what you keep.
	huge := make([]byte, 1<<20)
	leaky := huge[:10]
	safe := append([]byte(nil), huge[:10]...) // or slices.Clone
	fmt.Printf("1. huge[:10]: len=%d cap=%d (1 MB retained); copy: len=%d cap=%d\n",
		len(leaky), cap(leaky), len(safe), cap(safe))

	// 2. Overlapping slices: append on the shorter one overwrites the longer one. The
	// three-index form s[a:b:b] caps the capacity so append must reallocate.
	base := []int{1, 2, 3, 4, 5}
	front := base[0:2]
	front = append(front, 99)
	fmt.Printf("2. append through base[0:2] overwrote the parent: base=%v\n", base)
	base = []int{1, 2, 3, 4, 5}
	capped := append(base[0:2:2], 99)
	fmt.Printf("   with base[0:2:2]: base=%v capped=%v\n", base, capped)

	// 3. A struct copy shares its slice and map fields. Go has no built-in deep copy
	// (slices.Clone and maps.Clone go one level deep) - write one and keep it next to the type.
	original := record{ID: 1, Tags: []string{"a", "b"}}
	shallow := original
	shallow.Tags[0] = "CHANGED"
	fmt.Printf("3. shallow copy: original=%v shallow=%v\n", original, shallow)
	original = record{ID: 1, Tags: []string{"a", "b"}}
	deep := original.clone()
	deep.Tags[0] = "CHANGED"
	fmt.Printf("   deep copy:    original=%v deep=%v\n", original, deep)

	// 4. A []*T keeps every pointee alive. Deleting by re-slicing leaves the old tail pointer
	// in the backing array; slices.Delete zeroes the vacated slots so the GC can collect them.
}

type record struct {
	ID   int
	Tags []string
}

// clone is the hand-written deep copy Go does not give you.
func (r record) clone() record {
	out := r                                    // copies ID and the Tags HEADER
	out.Tags = append([]string(nil), r.Tags...) // now give it its own array
	return out
}

// unsafeAndLayout shows memory layout and the unsafe escape hatch.
func unsafeAndLayout() {
	fmt.Println("\n--- unsafe and struct layout ---")

	// Sizeof, Alignof and Offsetof are compile-time constants and safe to use. The rest of
	// unsafe leaves the type system and the Go 1 compatibility promise.
	fmt.Printf("Sizeof: int=%d string=%d []int=%d map=%d any=%d\n",
		unsafe.Sizeof(int(0)), unsafe.Sizeof(""), unsafe.Sizeof([]int(nil)),
		unsafe.Sizeof(map[int]int(nil)), unsafe.Sizeof(any(nil)))
	// A string is 2 words, a slice 3, a map 1, an interface 2 - exactly the valueSemantics table.

	// Fields are laid out in declaration order with alignment padding between them; ordering
	// largest-first can shrink a struct (the fieldalignment analyzer reports savings).
	fmt.Printf("padded{bool,int64,bool}=%d bytes, packed{int64,bool,bool}=%d bytes\n",
		unsafe.Sizeof(padded{}), unsafe.Sizeof(packed{}))
	fmt.Printf("offsets in padded: A=%d B=%d C=%d; Alignof(int64)=%d\n",
		unsafe.Offsetof(padded{}.A), unsafe.Offsetof(padded{}.B), unsafe.Offsetof(padded{}.C),
		unsafe.Alignof(int64(0)))

	// structs.HostLayout (Go 1.23) is a zero-size marker: this struct must use the platform's
	// C layout. It documents intent for cgo/syscall structs and costs nothing.
	fmt.Printf("cCompatible with a HostLayout marker = %d bytes\n", unsafe.Sizeof(cCompatible{}))

	// unsafe.Pointer converts between any *T; unsafe.Add/Slice/String/SliceData/StringData are
	// the sanctioned pointer arithmetic. The one pattern worth knowing: zero-copy string/[]byte
	// in a profiled hot path - valid only while nothing mutates the bytes.
	b := []byte("zero-copy")
	s := unsafe.String(unsafe.SliceData(b), len(b))
	back := unsafe.Slice(unsafe.StringData(s), len(s))
	fmt.Printf("unsafe.String over []byte: %q, unsafe.Slice back: %q (no allocation)\n", s, back)

	// uintptr is NOT a pointer: a plain integer the GC does not track, and stacks move. An
	// object referenced only by a uintptr can be collected or moved. Only the conversions listed
	// in the unsafe.Pointer docs are valid. In application code, essentially never use unsafe -
	// its homes are cgo, syscall wrappers and profiled hot paths.
}

// padded wastes space: each bool is followed by padding to align the next field.
type padded struct {
	A bool  // 1 byte + 7 padding
	B int64 // 8
	C bool  // 1 byte + 7 padding
}

// packed has the same fields, largest first.
type packed struct {
	B int64 // 8
	A bool  // 1
	C bool  // 1 + 6 padding
}

// cCompatible must match the platform C ABI.
type cCompatible struct {
	_     structs.HostLayout
	Flags uint32
	Count uint32
}
