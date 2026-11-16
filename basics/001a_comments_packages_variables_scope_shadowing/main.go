// Command 001a_comments_packages_variables_scope_shadowing is module 001a of the Go course:
// comments, packages, variables, zero values, scope and shadowing.
//
// Every module of the course is its own small program. Read main() first - it is the table of
// contents - then read the topic functions top to bottom and compare with the output of:
//
//	go run ./basics/001a_comments_packages_variables_scope_shadowing
package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	comments()           // line, block, doc and directive comments
	packagesAndImports() // package clause, visibility by case, import forms
	variables()          // var vs :=, default types, semicolon insertion
	zeroValues()         // every type has a usable zero value
	scopeAndBlocks()     // block scope, init statements, unused-variable rule
	shadowing()          // inner declarations hide outer ones - the err trap
}

// comments shows the two comment forms of Go and the special meaning some comments have.
func comments() {
	fmt.Println("--- comments ---")

	// A line comment runs to the end of the line.
	fmt.Println(greet("Go")) // it may also trail a statement

	/*
	   A general (block) comment may span lines. This is the only block comment in the course:
	   in practice Go code uses // almost everywhere, and editors comment out code with //.
	*/

	// Block comments do NOT nest (unlike Rust): the first */ closes the comment, so this fails:
	//	/* outer /* inner */ still outer? */
	// ERROR: expected statement, found '*'

	// A block comment without a newline behaves like a space, so it can sit mid-expression.
	// Any other comment behaves like a newline - and can therefore end a statement.
	sum := 2 /* first operand */ + 3
	fmt.Println("2 /* comment */ + 3 =", sum)

	// A comment written directly above a top-level declaration (no blank line) is its doc
	// comment - see greet in greet.go. Convention: it starts with the declared name.
	// `go doc` and pkg.go.dev show it. There is no /// or //! as in Rust.
	//	go doc -u ./basics/001a_comments_packages_variables_scope_shadowing greet  (-u: unexported)
	// Since Go 1.19 doc comments understand paragraphs, indented code, # headings, lists and
	// [Name] links; gofmt reformats them.

	// Directive comments have NO space after //: //go:build, //go:embed, //go:generate.
	// `// go:build linux` (with a space) is an ordinary, silently ignored comment.
	fmt.Println("directives: //go:build linux | //go:embed data.txt | //go:generate stringer")
}

// packagesAndImports shows what a package is and how names cross package boundaries.
func packagesAndImports() {
	fmt.Println("\n--- packages and imports ---")

	// Every file starts with a package clause; all files in a directory share the package.
	// `package main` + `func main()` makes an executable - like this module.

	// An imported package is used through its NAME (from its package clause), not its path:
	// "math/rand/v2" binds rand, "gopkg.in/yaml.v3" binds yaml.
	fmt.Println("strings.ToUpper:", strings.ToUpper("packages are namespaces"))
	fmt.Println("os.Args[0]:", os.Args[0])

	// Visibility is decided by the first letter: Upper = exported, lower = package-private.
	// There is no public/private keyword; the rule covers types, funcs, struct fields, methods.
	fmt.Println("strings.ToUpper is exported; greet (lowercase) is private to this package")

	// A package is ONE namespace shared by all its files. greet and Version live in greet.go,
	// yet are used here without any import. Declaring another greet in this file would fail:
	//	func greet(s string) string // ERROR: greet redeclared in this block
	fmt.Println("from greet.go, no import needed:", greet("package"), "version", Version)

	// The four import forms:
	//	import "fmt"         // normal: fmt.Println
	//	import f "fmt"       // alias:  f.Println
	//	import . "fmt"       // dot:    Println - pollutes the file scope, avoid outside tests
	//	import _ "image/png" // blank:  only runs the package's init() (DB drivers, codecs)
	//
	// Unused imports are a compile error, not a warning:
	//	import "bytes" // ERROR: "bytes" imported and not used
	//
	// Import cycles are forbidden (A -> B -> A). Fix by extracting a third package or by
	// inverting the dependency with an interface.
	// A directory named `internal` can only be imported from its parent's subtree.
}

// variables shows the two ways to declare a variable and how types are inferred.
func variables() {
	fmt.Println("\n--- variables ---")

	var explicit int = 42 // name, type, value
	var inferred = 42     // type inferred from the value
	var zeroed int        // no value: the zero value (see zeroValues)
	short := 42           // short declaration - the idiomatic choice inside a function
	fmt.Printf("explicit=%d inferred=%d zeroed=%d short=%d\n", explicit, inferred, zeroed, short)

	// Name before type (var x int) reads left-to-right even for complex types:
	//	var f func(int) (string, error)

	// Grouped declaration.
	var (
		name    = "Go"
		version = 1.27
		stable  = true
	)
	fmt.Printf("name=%s version=%v stable=%t\n", name, version, stable)

	// Several at once, and a swap without a temporary: the right side is evaluated first.
	a, b := 1, 2
	a, b = b, a
	fmt.Println("after swap: a =", a, "b =", b)

	// Inference uses the DEFAULT type of the literal, not the "smallest fitting" one.
	i := 5      // int (not int8, not int64)
	f := 5.0    // float64 (never float32)
	r := 'G'    // rune = int32 - prints as the number 71
	s := "Go"   // string
	t := true   // bool
	c := 1 + 2i // complex128
	fmt.Printf("%T %T %T %T %T %T\n", i, f, r, s, t, c)
	fmt.Printf("r := 'G' holds %d; use %%c to see %c\n", r, r)

	// Any other type must be asked for explicitly.
	var big int64 = 5
	small := float32(5)
	fmt.Printf("var big int64 -> %T, float32(5) -> %T\n", big, small)

	// := needs a value to infer from:
	//	x := int // ERROR: int (type) is not an expression
	//	p := nil // ERROR: use of untyped nil in assignment
	// and is illegal at package level, where only var/const/type/func may appear:
	//	count := 0 // ERROR: syntax error: non-declaration statement outside function body

	// Prefer := inside functions. Use var when you want the zero value (var buf bytes.Buffer),
	// need an explicit type (var x float32 = 5), or declare at package level.

	// Semicolons are inserted automatically at the end of a line ending in an identifier,
	// literal, `) ] }`, return, break, ... That is why `{` must stay on the same line:
	//	if a == 2
	//	{          // ERROR: syntax error: unexpected newline, expected { after if clause
	fmt.Println("braces open on the same line - semicolon insertion depends on it")
}

// zeroValues shows that Go has no uninitialised memory: every variable starts at its zero value.
func zeroValues() {
	fmt.Println("\n--- zero values ---")

	var (
		i  int
		f  float64
		b  bool
		s  string
		p  *int
		l  []int
		m  map[string]int
		a  [3]int
		c  chan int
		fn func()
		e  error // an interface
	)
	fmt.Printf("int      %d\n", i)
	fmt.Printf("float64  %v\n", f)
	fmt.Printf("bool     %t\n", b)
	fmt.Printf("string   %q  <- empty, never nil\n", s)
	fmt.Printf("*int     %v\n", p)
	fmt.Printf("[]int    %v (nil? %t, len %d)\n", l, l == nil, len(l))
	fmt.Printf("map      %v (nil? %t, len %d)\n", m, m == nil, len(m))
	fmt.Printf("[3]int   %v  <- an array is never nil\n", a)
	fmt.Printf("chan     %v\n", c)
	fmt.Printf("func     nil? %t\n", fn == nil)
	fmt.Printf("error    %v\n", e)

	// A struct zeroes every field, recursively.
	var z zeroDemo
	fmt.Printf("struct   %+v\n", z)

	// The "useful zero value": a well-designed type works with no constructor.
	// bytes.Buffer, sync.Mutex, sync.WaitGroup and strings.Builder all do.
	var sb strings.Builder
	sb.WriteString("a zero strings.Builder is ready to use")
	fmt.Println(sb.String())

	// A nil slice supports len, cap, range and append.
	var nilSlice []int
	nilSlice = append(nilSlice, 1, 2, 3)
	fmt.Println("append to a nil slice:", nilSlice)

	// A nil map can be READ (zero value, ok=false)...
	var nilMap map[string]int
	value, ok := nilMap["missing"]
	fmt.Printf("read from a nil map: value=%d ok=%t\n", value, ok)
	// ...but writing panics - create it with make or a literal first:
	//	nilMap["key"] = 1 // panic: assignment to entry in nil map

	// The price: a plain field cannot tell "unset" from "set to zero". When that matters use
	// a pointer (*int, nil = unset) or an extra bool.
}

type zeroDemo struct {
	Number  int
	Text    string
	Flag    bool
	Pointer *int
	Slice   []string
	Map     map[string]int
	Nested  struct{ Inner float64 }
}

// scopeAndBlocks shows lexical block scope, init statements and the unused-variable rule.
func scopeAndBlocks() {
	fmt.Println("\n--- scope and blocks ---")

	// A name is visible from its declaration to the end of the innermost enclosing { }.
	outer := "function block"
	{
		inner := "nested block"
		fmt.Println("inside the block both are visible:", outer, "/", inner)
	}
	//	fmt.Println(inner) // ERROR: undefined: inner

	// Package-level names are visible in any order; a local does not exist before its line.
	//	fmt.Println(later) // ERROR: undefined: later
	later := "declared after the line above"
	fmt.Println(later)

	// if/for/switch may start with an init statement. Its variables live in the whole
	// if/else chain and vanish afterwards - the standard way to keep err from leaking.
	if n, err := fmt.Sscanf("7", "%d", new(int)); err != nil {
		fmt.Println("scan failed:", err)
	} else {
		fmt.Println("init statement: scanned", n, "item(s); n and err exist only here")
	}
	//	fmt.Println(n) // ERROR: undefined: n

	switch size := len(outer); {
	case size > 10:
		fmt.Println("switch init statement: length", size)
	default:
		fmt.Println("switch init statement: short")
	}

	// An unused LOCAL variable is a compile error (package vars and parameters are exempt):
	//	unused := 1 // ERROR: declared and not used: unused
	// The blank identifier _ discards a value and can never be read.
	computed := 1 + 1
	_ = computed
	_, _ = 1, 2
	//	fmt.Println(_) // ERROR: cannot use _ as value
	fmt.Println("`_ = computed` silences the unused-variable error")

	// := needs at least ONE new name on the left; the others are plain assignments,
	// provided they were declared in the SAME block (otherwise you get shadowing).
	first, shared := 1, 2
	second, shared := 3, 4 // second is new, shared is only assigned
	fmt.Println("redeclaration:", first, second, shared)
	//	first, second := 5, 6 // ERROR: no new variables on left side of :=
}

// shadowing shows how an inner declaration hides an outer name, and the classic bug it causes.
func shadowing() {
	fmt.Println("\n--- shadowing ---")

	// An inner block may declare a name that hides an outer one; the outer one is untouched.
	x := "outer"
	{
		x := "inner"
		fmt.Println("inside the block, x =", x)
	}
	fmt.Println("after the block, x =", x)

	// Unlike Rust, shadowing in the SAME block is not allowed:
	//	x := "again" // ERROR: no new variables on left side of :=

	// The classic bug: := inside a block creates a NEW err, the outer one stays nil.
	var err error
	if true {
		_, err := fmt.Println("(this call sets only the INNER err)")
		_ = err
	}
	fmt.Println("outer err after the block:", err, "- the error was silently lost")

	// The fix: assign with = so the outer variable is written.
	var n int
	if true {
		n, err = fmt.Println("(this call sets the OUTER err)")
	}
	fmt.Println("outer n and err now:", n, err)

	// Predeclared names (len, cap, new, make, close, min, max, error, any, int, ...) are not
	// keywords - they live in the universe block and can be shadowed too.
	{
		len := "this string shadows the builtin len"
		fmt.Println(len)
		//	len("abc") // ERROR: invalid operation: cannot call len (variable of type string): string is not a function
	}
	fmt.Println("builtin len works again here:", len("abc"))

	// Shadowing a builtin at package level hides it in EVERY file of the package - so never
	// name a package-level helper close or len.
	// go vet does not report shadowing by default; the separate `shadow` analyzer does.
}
