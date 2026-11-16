// Command 002b_composite_types is module 002b of the Go course: arrays, slices, maps, structs,
// defined types and aliases.
//
// Two of the composite types are pure values (arrays, structs) and two contain a pointer to shared
// data (slices, maps) - almost every surprise in this module comes from that distinction.
//
//	go run ./basics/002b_composite_types
package main

import (
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"
	"unsafe"
)

func main() {
	arrays()                 // fixed length in the type, value semantics, comparable
	sliceHeader()            // pointer+len+cap, sharing, three-index slices, nil vs empty
	appendCopyClear()        // append returns a header, growth, copy bounded by len, clear
	mapsBasics()             // nil maps, comma-ok, random order, non-addressable entries
	structsBasics()          // literals, tags, Go 1.27 promoted-field keys, empty struct
	definedTypesAndAliases() // type T U vs type T = U, generic aliases (Go 1.24)
}

// arrays shows that an array's length is part of its type and that arrays are values.
func arrays() {
	fmt.Println("--- arrays ---")

	// [3]int and [4]int are different types; the length must be a compile-time constant
	// and len(a) == cap(a) is itself a constant.
	var fixed [3]int
	fmt.Printf("var [3]int: %v len=%d cap=%d type=%T\n", fixed, len(fixed), cap(fixed), fixed)

	explicit := [3]string{"a", "b", "c"}
	inferred := [...]string{"a", "b", "c", "d"} // [...] infers the length: 4
	fmt.Printf("explicit %T=%v  inferred %T=%v\n", explicit, explicit, inferred, inferred)

	// Indexed literal: only the named positions are set; the largest index can imply the length.
	sparse := [100]int{1, 10: 3, 99: 100}
	implied := [...]int{99: 1}
	fmt.Printf("sparse: [0]=%d [1]=%d [10]=%d [99]=%d; [...]int{99: 1} has len %d\n",
		sparse[0], sparse[1], sparse[10], sparse[99], len(implied))

	// An array is a VALUE: assignment and passing copy every element (no decay to a pointer as
	// in C). That is why APIs pass slices instead.
	original := [3]int{1, 2, 3}
	copied := original
	copied[0] = 999
	mutateArray(original)
	fmt.Printf("original=%v copied=%v - copied on assignment and on the call\n", original, copied)

	// Arrays are comparable when their elements are - so, unlike slices, they can be map keys.
	fmt.Printf("[3]int{1,2,3} == original is %t\n", original == [3]int{1, 2, 3})
	//	fmt.Println(original == [4]int{1, 2, 3, 0}) // ERROR: invalid operation: original == [4]int{…} (mismatched types [3]int and [4]int)
	seen := map[[3]int]string{{1, 2, 3}: "the first triple"}
	fmt.Println("an array as a map key:", seen[[3]int{1, 2, 3}])

	// Multidimensional arrays are arrays of arrays, laid out contiguously.
	grid := [2][3]int{{1, 2, 3}, {4, 5, 6}}
	fmt.Println("grid:", grid)

	// A constant out-of-range index is a compile error; a variable one panics at runtime.
	//	fmt.Println(original[5]) // ERROR: invalid argument: index 5 out of bounds [0:3]

	// Use arrays when the size is part of the meaning: a [16]byte UUID, a lookup table
	// indexed by an enum. Otherwise use a slice.
}

func mutateArray(a [3]int) { a[0] = -1 } // works on a copy

// sliceHeader shows the three-word slice header and how slices of one array alias each other.
func sliceHeader() {
	fmt.Println("\n--- slices: the header, len and cap ---")

	// A slice is a header (pointer to a backing array, len, cap). Copying it copies the pointer,
	// so both copies see the same elements (module 006).
	s := []int{1, 2, 3, 4, 5}
	made := make([]int, 3, 10) // make(T, len, cap)
	fmt.Printf("s=%v len=%d cap=%d; make([]int, 3, 10)=%v len=%d cap=%d\n",
		s, len(s), cap(s), made, len(made), cap(made))

	// s[low:high] has len high-low but cap = cap(s)-low: capacity runs to the end of the
	// BACKING ARRAY, not to high. Bounds are 0 <= low <= high <= cap(s) - cap, not len.
	backing := make([]int, 300)
	window := backing[2:5]
	extended := window[:cap(window)] // re-extend past len: the elements were never gone
	fmt.Printf("backing[2:5]: len=%d cap=%d; window[:cap(window)] has len %d\n",
		len(window), cap(window), len(extended))
	// So a small slice of a huge array keeps the whole array alive - slices.Clone breaks that.

	// Slices of the same array alias each other.
	base := []int{0, 1, 2, 3, 4}
	left, right := base[:3], base[2:]
	left[2] = 99 // base[2], which is also right[0]
	fmt.Printf("base=%v left=%v right=%v - one write, three views\n", base, left, right)
	independent := slices.Clone(left)
	independent[0] = -1
	fmt.Printf("after slices.Clone and a write: base=%v clone=%v\n", base, independent)

	// The three-index form s[low:high:max] sets cap = max-low, so append must reallocate
	// instead of growing into a neighbour's memory.
	full := []int{1, 2, 3, 4, 5}
	loose := full[0:2]   // cap 5
	tight := full[0:2:2] // cap 2
	loose = append(loose, 99)
	fmt.Printf("append to full[0:2] clobbered the original: full=%v\n", full)
	tight = append(tight, 77)
	fmt.Printf("append to full[0:2:2] reallocated: full=%v tight=%v\n", full, tight)

	// nil vs empty: both have len 0 and support range and append; test emptiness with
	// len(s) == 0. They differ only where nil-ness is inspected: s == nil, reflect.DeepEqual
	// and encoding/json. Prefer the nil slice as the zero value.
	var nilSlice []int
	emptySlice := []int{}
	nilJSON, _ := json.Marshal(nilSlice)
	emptyJSON, _ := json.Marshal(emptySlice)
	fmt.Printf("nil: len=%d isNil=%t json=%s | empty: len=%d isNil=%t json=%s\n",
		len(nilSlice), nilSlice == nil, nilJSON, len(emptySlice), emptySlice == nil, emptyJSON)

	// Slices are not comparable (so not map keys either) - only against nil.
	//	fmt.Println(nilSlice == emptySlice) // ERROR: invalid operation: nilSlice == emptySlice (slice can only be compared to nil)
	fmt.Println("slices.Equal(nilSlice, emptySlice) =", slices.Equal(nilSlice, emptySlice))
}

// appendCopyClear shows the builtins that grow, copy and reset slices.
func appendCopyClear() {
	fmt.Println("\n--- append, copy, clear and growth ---")

	// append returns a NEW header - always use the result. With spare capacity it writes in
	// place; without, it allocates a bigger array. So append sometimes shares and sometimes not.
	s := []int{1, 2, 3}
	s = append(s, 4, 5)
	s = append(s, []int{6, 7}...) // spread another slice
	fmt.Printf("after appends: %v len=%d cap=%d\n", s, len(s), cap(s))
	//	append(s, 6) // ERROR: append(s, 6) (value of type []int) is not used
	fmt.Printf("append([]byte(\"Go\"), \"lang\"...) = %q\n", append([]byte("Go"), "lang"...))

	// Growth is amortised (about 2x for small slices, tapering to ~1.25x). The factor is an
	// implementation detail that has changed between releases - never rely on it.
	var grow []int
	prev := -1
	fmt.Print("cap after each reallocation while appending 10 ints:")
	for i := range 10 {
		grow = append(grow, i)
		if cap(grow) != prev {
			fmt.Print(" ", cap(grow))
			prev = cap(grow)
		}
	}
	fmt.Println()
	// If you know the final size, preallocate and skip regrowth entirely.
	sized := make([]int, 0, 10)
	for i := range 10 {
		sized = append(sized, i)
	}
	fmt.Printf("make([]int, 0, 10) then 10 appends: cap stayed %d\n", cap(sized))

	// copy(dst, src) copies min(len(dst), len(src)) - bounded by LENGTH, not capacity -
	// and handles overlap.
	dst := make([]int, 3)
	n := copy(dst, []int{1, 2, 3, 4, 5})
	fmt.Printf("copy into len-3 dst: %d copied -> %v\n", n, dst)
	empty := make([]int, 0, 10)
	n = copy(empty, []int{1, 2, 3})
	fmt.Printf("copy into len-0 cap-10 dst: %d copied -> %v (a classic mistake)\n", n, empty)

	// Prefer slices.Delete/Insert/Concat (Go 1.21) over hand-written append tricks.
	letters := []string{"a", "b", "c", "d"}
	letters = slices.Delete(letters, 1, 2)
	fmt.Printf("slices.Delete(s, 1, 2) = %v\n", letters)
	letters = slices.Insert(letters, 1, "B")
	fmt.Printf("slices.Insert(s, 1, \"B\") = %v, Concat = %v\n", letters, slices.Concat(letters, []string{"e", "f"}))
	// The manual delete leaves the old tail element in the array - with a []*T that pointer
	// keeps its object alive. slices.Delete zeroes the vacated slots.
	manual := []string{"a", "b", "c", "d"}
	manual = append(manual[:1], manual[2:]...)
	fmt.Printf("append(s[:1], s[2:]...) = %v\n", manual)

	// clear (Go 1.21) zeroes every element of a slice, keeping its length; on a map it deletes
	// every key. Use it to drop references for the GC.
	nums := []int{1, 2, 3}
	clear(nums)
	fmt.Printf("clear on a slice: %v len=%d\n", nums, len(nums))
}

// mapsBasics shows map[K]V: a pointer to a runtime hash table.
func mapsBasics() {
	fmt.Println("\n--- maps ---")

	// Copying a map shares the table - it is never deep-copied by assignment.
	ages := map[string]int{"Ada": 36, "Alan": 41, "Grace": 45}
	counts := make(map[string]int)
	counts["seen"]++ // a missing key reads as 0, so ++ works on a fresh key
	fmt.Printf("ages=%v counts=%v\n", ages, counts)

	// The nil map is readable (zero values) but writing to it panics.
	var nilMap map[string]int
	fmt.Printf("nil map: len=%d, reading gives %d\n", len(nilMap), nilMap["anything"])
	//	nilMap["k"] = 1 // panic: assignment to entry in nil map

	// A missing key yields the zero value; comma-ok tells "missing" from "present but zero".
	ages["Zero"] = 0
	for _, key := range []string{"Ada", "Zero", "Nobody"} {
		v, ok := ages[key]
		fmt.Printf("  ages[%q] = %d, present=%t\n", key, v, ok)
	}

	delete(ages, "Zero")
	delete(ages, "Not there") // a no-op, not an error; clear(m) removes every key
	fmt.Printf("after delete: %v\n", ages)

	// Iteration order is randomised ON PURPOSE, anew on every range, so nobody can depend on it.
	fmt.Println("three ranges over the same map:")
	for range 3 {
		var order []string
		for k := range ages {
			order = append(order, k)
		}
		fmt.Printf("  %v\n", order)
	}
	// For a deterministic order, sort the keys.
	for _, k := range slices.Sorted(maps.Keys(ages)) {
		fmt.Printf("  sorted: %s=%d\n", k, ages[k])
	}
	cloned := maps.Clone(ages)
	fmt.Printf("maps.Clone equal to the original? %t\n", maps.Equal(ages, cloned))

	// A map entry is not addressable: no &m[k], no m[k].Field = v.
	// Read-modify-write the whole value, or store pointers.
	type counter struct{ N int }
	structs := map[string]counter{"a": {N: 1}}
	//	structs["a"].N++ // ERROR: cannot assign to struct field structs["a"].N in map
	entry := structs["a"]
	entry.N++
	structs["a"] = entry
	pointers := map[string]*counter{"a": {N: 1}}
	pointers["a"].N++ // fine: we modify the pointee, not the entry
	fmt.Printf("read-modify-write: %v; via pointer: N=%d\n", structs, pointers["a"].N)

	// The key type must be comparable: no slices, maps or funcs. A float key is legal but NaN
	// can never be found again; an interface key panics at runtime on an uncomparable value.
	//	bad := map[[]int]string{} // ERROR: invalid map key type []int
	byPoint := map[point]string{{X: 1, Y: 2}: "a comparable struct works as a key"}
	fmt.Println(byPoint[point{X: 1, Y: 2}])

	// Maps are not safe for concurrent use: a concurrent write is detected and aborts with
	// "fatal error: concurrent map writes" - not a recoverable panic. Use a mutex or sync.Map
	// (module 011). Since Go 1.24 maps are Swiss tables: faster, same API.
}

type point struct{ X, Y int }

// structsBasics shows struct literals, field visibility, tags and comparability.
func structsBasics() {
	fmt.Println("\n--- structs ---")

	// A struct is a value: assignment copies every field. No inheritance - composition and
	// embedding instead (module 007).
	// Prefer KEYED literals: omitted fields are zero and the code survives new fields.
	u := user{ID: 1, Name: "Ada", password: "secret"}
	fmt.Printf("%+v\n", u)
	// A positional literal must list every field in order and breaks when the struct changes
	// (go vet warns for structs from other packages).
	full := user{2, "Alan", "alan@example.com", "", "hunter2"}
	fmt.Printf("positional: %+v\n", full)
	//	partial := user{3, "Grace"} // ERROR: too few values in struct literal of type user

	// encoding/json uses reflection, so it cannot see unexported fields at all.
	encoded, _ := json.Marshal(u)
	fmt.Printf("json.Marshal: %s (password is absent because it is unexported)\n", encoded)

	// Go 1.27: a literal key may be any field selector, including a promoted field.
	promoted := line{Name: "diagonal", Length: 5}
	classic := line{object: object{Name: "diagonal"}, Length: 5} // the pre-1.27 spelling
	fmt.Printf("Go 1.27 promoted-field key: %+v\nbefore 1.27: %+v\n", promoted, classic)
	// The selector may not go through an embedded POINTER - there is no struct behind nil:
	//	type ref struct{ *object }
	//	_ = ref{Name: "x"} // ERROR: invalid implicit pointer indirection to reach Name

	a := object{Name: "original"}
	b := a
	b.Name = "copy"
	fmt.Printf("a=%v b=%v - assignment copied every field\n", a, b)

	// Structs compare field by field - only if every field is comparable. (An interface field
	// holding an uncomparable value panics at runtime instead.)
	fmt.Printf("a == object{Name: \"original\"} is %t\n", a == object{Name: "original"})
	type withSlice struct{ S []int }
	//	fmt.Println(withSlice{} == withSlice{}) // ERROR: invalid operation: withSlice{} == withSlice{} (struct containing []int cannot be compared)

	// struct{} takes zero bytes: map[string]struct{} is the idiomatic set, chan struct{} a signal.
	members := map[string]struct{}{"a": {}, "b": {}}
	_, inSet := members["a"]
	fmt.Printf("set contains \"a\": %t; struct{} occupies %d bytes\n", inSet, unsafe.Sizeof(struct{}{}))

	// Anonymous structs suit table-driven tests and one-off JSON payloads.
	cases := []struct {
		in   string
		want int
	}{{"", 0}, {"abc", 3}}
	for _, c := range cases {
		fmt.Printf("  case len(%q) == %d: %t\n", c.in, c.want, len(c.in) == c.want)
	}

	// Field order affects size through alignment padding (module 006 measures it). Optimise it
	// only for millions of instances or a hot cache line.
}

type object struct {
	Name  string
	Color string
}

// line embeds object, so Name and Color are promoted into it.
type line struct {
	object
	Length float64
}

// user has struct tags: raw strings read at runtime by reflection. A typo in a tag is silent to
// the compiler - go vet's structtag check catches it.
type user struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Email    string `json:"email,omitempty"`
	Nickname string `json:"nickname,omitzero"` // Go 1.24: omit when the field is the zero value
	password string // unexported: invisible to encoding/json
}

// definedTypesAndAliases contrasts `type T U` (a new type) with `type T = U` (a second name).
func definedTypesAndAliases() {
	fmt.Println("\n--- defined types, aliases and generic aliases ---")

	// A DEFINED type is distinct from its underlying type and can carry methods - a domain type
	// out of a primitive. It keeps the underlying operators but not the original's methods.
	body := celsius(36.6)
	fmt.Printf("body=%v (%T) -> %.2f (%T)\n", body, body, float64(body.toFahrenheit()), body.toFahrenheit())

	// So the compiler catches unit mix-ups (untyped constants still adapt: body + 5.0 is fine).
	//	var f fahrenheit = body // ERROR: cannot use body (variable of float64 type celsius) as fahrenheit value in variable declaration
	//	bad := body + f         // ERROR: invalid operation: body + f (mismatched types celsius and fahrenheit)

	// Conversion to and from the underlying type is always allowed and free at runtime.
	asFloat := float64(body)
	fmt.Printf("float64(body) = %v (%T)\n", asFloat, asFloat)

	p := filePath("/usr/local/bin/go")
	fmt.Printf("filePath(%q).base() = %q\n", string(p), p.base())

	// An ALIAS is the same type under another name: interchangeable, and %T prints the target.
	// Aliases exist for gradual refactoring across packages, not for readability.
	var t text = "an alias is interchangeable with its target"
	var s string = t
	var temp temperature = 21.5
	var plain float64 = temp
	fmt.Printf("%s (%%T prints %T); temperature and float64: %v %v\n", s, t, temp, plain)

	// An alias cannot carry its own methods:
	//	func (t text) shout() string { return t } // ERROR: cannot define new methods on non-local type text

	// Go 1.24: aliases may have type parameters - and are still the very same type.
	seen := set[string]{"a": {}, "b": {}}
	_, ok := seen["a"]
	var asMap map[string]struct{} = seen
	fmt.Printf("set[string] = %v, contains \"a\"=%t, assignable to map[string]struct{}: len=%d\n",
		seen, ok, len(asMap))

	// Rule of thumb: use a defined type whenever a bare string or int carries meaning the
	// compiler could check (IDs, currency codes, paths, units); use an alias only to refactor.
}

// Defined types: distinct, and able to carry methods.
type (
	celsius    float64
	fahrenheit float64
	filePath   string
)

// Aliases: other names for existing types - no methods, no distinction.
type (
	text        = string
	temperature = float64
)

// set is a generic alias (Go 1.24).
type set[T comparable] = map[T]struct{}

func (c celsius) String() string { return fmt.Sprintf("%.1f°C", float64(c)) }

func (c celsius) toFahrenheit() fahrenheit { return fahrenheit(c*9/5 + 32) }

func (p filePath) base() string {
	if i := strings.LastIndex(string(p), "/"); i >= 0 {
		return string(p)[i+1:]
	}
	return string(p)
}
