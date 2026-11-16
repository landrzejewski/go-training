// Command 007_structs_and_methods is module 007 of the Go course: structs and methods.
//
// Go has no classes and no inheritance. It has structs for data, methods on any type defined in
// the package, embedding for reuse and interfaces for substitutability (module 008) - and keeps
// those two apart on purpose: no super, no virtual dispatch, no fragile base class.
//
//	go run ./basics/007_structs_and_methods
package main

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"sync"
)

func main() {
	methods()                   // receivers on structs, numbers, slices, maps and funcs
	receivers()                 // value vs pointer receivers, method sets, nil receivers
	constructors()              // NewXxx, (T, error), usable zero values
	embedding()                 // promotion, shadowing, no virtual dispatch, JSON flattening
	tagsAndReflection()         // struct tags, encoding/json options, reflect
	comparabilityAndAnonymous() // == on structs, deliberately uncomparable, anonymous structs
}

// methods shows that a method is a function with a receiver, on any type defined in this package.
func methods() {
	fmt.Println("--- methods ---")

	// func (r rect) Area() float64: the receiver sits before the name. Call it a one- or
	// two-letter abbreviation of the type, the same in every method - never this or self.
	r := rect{Width: 3, Height: 4}
	fmt.Printf("rect %v: area=%g perimeter=%g\n", r, r.Area(), r.Perimeter())

	// The base type may be ANY defined type, not only a struct.
	fmt.Println("celsius(-5).IsFreezing() =", celsius(-5).IsFreezing()) // numeric type
	fmt.Println("intList{1, 2, 3}.Sum() =", intList{1, 2, 3}.Sum())     // slice type
	set := stringSet{}                                                  // map type
	set.Add("go")
	fmt.Printf("stringSet: Has(\"go\")=%t Has(\"rust\")=%t\n", set.Has("go"), set.Has("rust"))

	// A method on a FUNCTION type - the trick behind http.HandlerFunc.
	isEmpty := validator(func(s string) bool { return s == "" })
	notEmpty := isEmpty.Negate()
	fmt.Printf("func type with a method: isEmpty(\"\")=%t notEmpty(\"\")=%t\n", isEmpty(""), notEmpty(""))

	// The base type must be declared in THIS package; define your own type from it first
	// (like celsius here, or Celsius in module 002b):
	//	func (i int) Double() int { return i * 2 }      // ERROR: cannot define new methods on non-local type int
	//	func (t time.Time) Tomorrow() time.Time { ... } // ERROR: cannot define new methods on non-local type time.Time
	//	func (p *int) Zero() { *p = 0 }                 // ERROR: cannot define new methods on non-local type int
	// *T receivers are fine, but the base type itself may not be a pointer or interface type:
	//	type rectPtr *rect
	//	func (p rectPtr) M() // ERROR: invalid receiver type rectPtr (pointer or interface type)

	// Methods belong to their type, so two types may each have a String method - the only
	// place Go looks like overloading.
	fmt.Println("rect.String and celsius.String coexist:", r, "/", celsius(21.5))

	// A method expression T.M is the plain function, receiver first; a method value x.M has
	// the receiver bound (module 005).
	area := rect.Area
	fmt.Println("method expression rect.Area(r) =", area(r))
}

type rect struct {
	Width, Height float64
}

func (r rect) Area() float64      { return r.Width * r.Height }
func (r rect) Perimeter() float64 { return 2 * (r.Width + r.Height) }
func (r rect) String() string     { return fmt.Sprintf("%gx%g", r.Width, r.Height) }

type celsius float64

func (c celsius) IsFreezing() bool { return c <= 0 }
func (c celsius) String() string   { return fmt.Sprintf("%.1f°C", float64(c)) }

type intList []int

func (l intList) Sum() (total int) {
	for _, v := range l {
		total += v
	}
	return total
}

type stringSet map[string]struct{}

func (s stringSet) Add(v string) { s[v] = struct{}{} }

func (s stringSet) Has(v string) bool {
	_, ok := s[v]
	return ok
}

type validator func(string) bool

func (v validator) Negate() validator {
	return func(s string) bool { return !v(s) }
}

// receivers shows value vs pointer receivers and the method-set rule that follows from them.
func receivers() {
	fmt.Println("\n--- value and pointer receivers, method sets ---")

	// A value receiver gets a COPY; a pointer receiver gets the address, so changes stick.
	acc := account{Owner: "Ada", Balance: 100}
	acc.DepositByValue(50)
	fmt.Println("after DepositByValue(50):  ", acc.Describe(), "(only the copy was credited)")
	acc.DepositByPointer(50)
	fmt.Println("after DepositByPointer(50):", acc.Describe())

	// Go takes the address or dereferences for you: acc.DepositByPointer(50) is
	// (&acc).DepositByPointer(50). That needs an ADDRESSABLE value - a map entry or a
	// function's result is not one:
	accounts := map[string]account{"ada": {Owner: "Ada"}}
	//	accounts["ada"].DepositByPointer(1) // ERROR: cannot call pointer method DepositByPointer on account
	_ = accounts

	// The method set rule - the part that actually bites:
	//	T  has the methods declared with receiver T
	//	*T has the methods declared with receiver T AND *T
	// So when a method has a pointer receiver, only *T implements an interface that needs it:
	// "... does not implement I (method M has pointer receiver)" (module 008).
	fmt.Println("methods of  account:", methodNames(reflect.TypeOf(account{})))
	fmt.Println("methods of *account:", methodNames(reflect.TypeOf(&account{})))

	// Choosing: be consistent - if one method needs a pointer receiver, give them all one.
	// Pointer: it mutates, the struct is large, it holds a sync.Mutex, or nil is meaningful.
	// Value: small immutable types (time.Time, netip.Addr), and String() of a small type.

	// A nil pointer receiver is legal - the method runs with t == nil, only dereferencing it
	// panics. Some types rely on it on purpose.
	var empty *tree
	fmt.Println("Height() on a nil *tree:", empty.Height())
	t := &tree{Value: 1, Left: &tree{Value: 2}, Right: &tree{Value: 3, Right: &tree{Value: 4}}}
	fmt.Printf("real tree: Height()=%d Sum()=%d\n", t.Height(), t.Sum())

	// A type holding a sync.Mutex must never be copied, so all its methods take a pointer.
	c := &safeCounter{}
	var wg sync.WaitGroup
	for range 100 {
		wg.Go(c.Inc) // Go 1.25: WaitGroup.Go replaces Add(1) + go func() { defer Done() }()
	}
	wg.Wait()
	fmt.Println("safeCounter after 100 concurrent Inc():", c.Get())
	// A copy compiles, but go vet's copylocks check reports it:
	//	copied := *c // go vet: assignment copies lock value to copied: <pkg>.safeCounter contains sync.Mutex
}

type account struct {
	Owner   string
	Balance int
}

func (a account) DepositByValue(n int)    { a.Balance += n }
func (a *account) DepositByPointer(n int) { a.Balance += n }
func (a account) Describe() string        { return fmt.Sprintf("%s: %d", a.Owner, a.Balance) }

// methodNames lists the exported methods in t's method set.
func methodNames(t reflect.Type) []string {
	names := make([]string, 0, t.NumMethod())
	for i := range t.NumMethod() {
		names = append(names, t.Method(i).Name)
	}
	return names
}

type tree struct {
	Value       int
	Left, Right *tree
}

func (t *tree) Height() int {
	if t == nil { // a nil receiver is perfectly legal
		return 0
	}
	return 1 + max(t.Left.Height(), t.Right.Height())
}

func (t *tree) Sum() int {
	if t == nil {
		return 0
	}
	return t.Value + t.Left.Sum() + t.Right.Sum()
}

type safeCounter struct {
	mu sync.Mutex
	n  int
}

func (c *safeCounter) Inc() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.n++
}

func (c *safeCounter) Get() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.n
}

// constructors shows the NewXxx convention for types whose zero value is not enough.
func constructors() {
	fmt.Println("\n--- constructors ---")

	// Go has no constructors: a struct literal builds a value and the zero value should work.
	// When setup IS needed (a map, an invariant, an unexported field) write a plain function
	// New (list.New) or NewThing (bytes.NewBuffer), returning *T when the type has pointer
	// methods or must not be copied, T for a small value type.
	reg := newRegistry("services")
	reg.Set("http", 80)
	reg.Set("https", 443)
	fmt.Printf("newRegistry: name=%s len=%d\n", reg.name, reg.Len())
	// Without it the map field is nil:
	//	var broken registry
	//	broken.Set("x", 1) // panic: assignment to entry in nil map

	// Construction that can fail returns (T, error) - never a half-built value. A Must variant
	// is only for inputs that are constants in practice (module 005).
	if r, err := newRect(3, 4); err == nil {
		fmt.Println("newRect(3, 4) =", r)
	}
	if _, err := newRect(-1, 4); err != nil {
		fmt.Println("newRect(-1, 4) ->", err)
	}

	// Many optional parameters: functional options (module 005) or a config struct - not
	// NewFoo, NewFooWithBar, NewFooWithBarAndBaz.

	// Best of all, design the type so `var x T` simply works - an unusable zero value is a cost.
	var sb strings.Builder
	var mu sync.Mutex
	sb.WriteString("no constructor needed")
	mu.Lock()
	mu.Unlock()
	fmt.Printf("zero values that just work: strings.Builder(%q), sync.Mutex, bytes.Buffer\n", sb.String())
}

type registry struct {
	entries map[string]int // unexported and must be non-nil: a constructor is warranted
	name    string
}

func newRegistry(name string) *registry {
	return &registry{entries: make(map[string]int), name: name}
}

func (r *registry) Set(k string, v int) { r.entries[k] = v }
func (r *registry) Len() int            { return len(r.entries) }

func newRect(w, h float64) (rect, error) {
	if w <= 0 || h <= 0 {
		return rect{}, fmt.Errorf("newRect: dimensions must be positive, got %gx%g", w, h)
	}
	return rect{Width: w, Height: h}, nil
}

// embedding shows embedded fields: promotion, shadowing, and why it is not inheritance.
func embedding() {
	fmt.Println("\n--- embedding, promotion and shadowing ---")

	// An embedded field has a type and no name; its field name is the type's name, and its
	// fields and methods are PROMOTED - usable unqualified on the outer struct.
	c := client{logger: logger{Prefix: "CLIENT"}, address: address{City: "Warsaw"}, Name: "Ada"}
	fmt.Printf("promoted field:   c.Prefix=%q c.City=%q\n", c.Prefix, c.City)
	fmt.Printf("promoted method:  c.Log(...)=%q\n", c.Log("connected"))
	fmt.Printf("qualified access: c.logger.Prefix=%q\n", c.logger.Prefix)

	// Shadowing: an outer field or method of the same name wins; the inner one stays reachable.
	fmt.Printf("c.Show()         = %q <- the outer method wins\n", c.Show())
	fmt.Printf("c.address.Show() = %q <- the shadowed one\n", c.address.Show())

	// NOT inheritance: promotion only spells outer.Inner.M() as outer.M(). There is no virtual
	// dispatch - a promoted method that calls another method calls its OWN type's version.
	// Coming from Java or C++, this is the single most important difference.
	loud := loudLogger{logger{Prefix: "app"}}
	fmt.Printf("loud.Log(\"hi\")      = %q <- the outer method\n", loud.Log("hi"))
	fmt.Printf("loud.LogTwice(\"hi\") = %q <- logger.LogTwice calls logger.Log\n", loud.LogTwice("hi"))

	// The same name at the same depth compiles, but USING it unqualified is an error.
	// A shallower name wins over a deeper one, with no error.
	tw := twoShows{address: address{City: "Gdansk"}, client: c}
	//	tw.Show() // ERROR: ambiguous selector tw.Show
	fmt.Println("qualified:", tw.address.Show(), "/", tw.client.Show())
	fmt.Println("tw.City at depth 1 beats tw.client.address.City at depth 2:", tw.City)

	// Embedding a pointer lets the inner value be shared or nil; a value keeps one allocation.
	p := pointerClient{logger: &logger{Prefix: "PTR"}}
	fmt.Printf("pointer embed: p.Log(...)=%q\n", p.Log("x"))

	// Embedding an INTERFACE in a struct makes the struct satisfy it, forwarding to the stored
	// value, and lets you override single methods - the usual way to write test doubles. Any
	// method you did not override panics while the embedded interface is nil.
	// Interfaces embed interfaces as well: interface{ io.Reader; io.Writer } (module 008).

	// encoding/json FLATTENS promoted fields - often wanted, occasionally a nasty surprise.
	encoded, _ := json.Marshal(wire{address: address{City: "Krakow"}, ID: 7})
	fmt.Printf("json.Marshal flattens promoted fields: %s\n", encoded)
}

type logger struct{ Prefix string }

func (l logger) Log(msg string) string { return l.Prefix + ": " + msg }

// LogTwice calls l.Log - and that is ALWAYS logger's Log, never an outer "override".
func (l logger) LogTwice(msg string) string { return l.Log(msg) + " / " + l.Log(msg) }

type address struct{ City string }

func (a address) Show() string { return "address in " + a.City }

type client struct {
	logger  // Prefix, Log and LogTwice are promoted
	address // City and Show are promoted
	Name    string
}

// Show shadows the promoted address.Show.
func (c client) Show() string { return "client " + c.Name }

type loudLogger struct{ logger }

func (l loudLogger) Log(msg string) string { return strings.ToUpper(l.logger.Log(msg)) }

type twoShows struct {
	address // Show at depth 1
	client  // Show at depth 1 too -> ambiguous
}

type pointerClient struct{ *logger }

type wire struct {
	address
	ID int `json:"id"`
}

// tagsAndReflection shows struct tags and how reflection-based libraries read them.
func tagsAndReflection() {
	fmt.Println("\n--- struct tags and reflection ---")

	// A tag is a string after a field - opaque to the compiler, read by libraries through
	// reflection. Convention: space-separated key:"value" pairs in a raw string. A typo such as
	// json:"nane" compiles silently; go vet's structtag check catches malformed tags.
	p := product{ID: 1, Name: "Gopher plush", Price: 29.99, Internal: "not serialised", secret: "hidden"}
	encoded, _ := json.MarshalIndent(p, "  ", "  ")
	fmt.Printf("  %s\n", encoded)
	fmt.Println("  price is a STRING (,string), tags is absent (omitempty), Internal is excluded (-),")
	fmt.Println("  and secret is invisible: reflection only sees exported fields, no tag changes that")

	// Reading the tags yourself - exactly what encoding/json does.
	t := reflect.TypeOf(p)
	for f := range t.Fields() { // Go 1.26: Type.Fields returns an iter.Seq[StructField]
		fmt.Printf("    %-8s exported=%-5t json=%-16q db=%-12q validate=%q\n",
			f.Name, f.IsExported(), f.Tag.Get("json"), f.Tag.Get("db"), f.Tag.Get("validate"))
	}

	// Lookup tells "absent" from "present but empty"; Get returns "" for both.
	nameField, _ := t.FieldByName("Name")
	if _, ok := nameField.Tag.Lookup("yaml"); !ok {
		fmt.Println("  Tag.Lookup(\"yaml\"): absent")
	}

	// omitempty skips false, 0, nil and empty strings/slices/maps - but never an empty STRUCT.
	// Go 1.24 omitzero skips the type's zero value, fixing that (and a zero time.Time).
	type inner struct{ A int }
	type demo struct {
		EmptyStruct  inner `json:"emptyStruct,omitempty"` // NOT omitted
		ZeroStruct   inner `json:"zeroStruct,omitzero"`   // omitted
		EmptySlice   []int `json:"emptySlice,omitempty"`  // omitted
		ExplicitZero int   `json:"explicitZero"`          // present
	}
	out, _ := json.Marshal(demo{})
	fmt.Printf("  omitempty vs omitzero: %s\n", out)

	// reflect cannot set unexported fields either.
	v := reflect.ValueOf(&p).Elem()
	fmt.Printf("  CanSet(Name)=%t CanSet(secret)=%t\n", v.FieldByName("Name").CanSet(), v.FieldByName("secret").CanSet())
	v.FieldByName("Name").SetString("renamed through reflection")
	fmt.Printf("  after a reflective set: %q\n", p.Name)

	// Reflection is powerful, slow, and moves errors from compile time to run time. Keep it for
	// serialisation, ORMs, validators and test helpers; prefer generics (module 010) elsewhere.
}

type product struct {
	ID       int      `json:"id" db:"product_id" validate:"required"`
	Name     string   `json:"name" db:"name" validate:"required,min=3"`
	Price    float64  `json:"price,string" db:"price"` // ,string: encode the number as a JSON string
	Tags     []string `json:"tags,omitempty" db:"-"`
	Internal string   `json:"-"` // never serialised (json:"-," would mean a field named "-")
	secret   string
}

// comparabilityAndAnonymous shows when == works on structs, and inline anonymous struct types.
func comparabilityAndAnonymous() {
	fmt.Println("\n--- comparability, copying and anonymous structs ---")

	// A struct is comparable if and only if every field is (module 002b). Comparison is field
	// by field, unexported fields included - which is why time.Time asks you to use Equal.
	type allComparable struct {
		A int
		B string
		C [2]int
	}
	x := allComparable{1, "a", [2]int{1, 2}}
	y := allComparable{1, "a", [2]int{1, 2}}
	fmt.Println("all fields comparable: x == y is", x == y)

	// One slice, map or func field makes the whole struct uncomparable, at compile time.
	type withSlice struct {
		A int
		S []int
	}
	//	fmt.Println(withSlice{} == withSlice{}) // ERROR: invalid operation: withSlice{} == withSlice{} (struct containing []int cannot be compared)
	_ = withSlice{}

	// An interface field compiles, but == panics at run time if the dynamic values are not
	// comparable.
	type withAny struct{ V any }
	fmt.Println("interface field holding ints:  ", safeCompare(withAny{V: 1}, withAny{V: 1}))
	fmt.Println("interface field holding slices:", safeCompare(withAny{V: []int{1}}, withAny{V: []int{1}}))

	// A zero-size uncomparable blank field makes a struct uncomparable ON PURPOSE, so callers
	// cannot depend on == and you stay free to add a slice field later.
	type futureProof struct {
		_ [0]func()
		A int
	}
	//	fmt.Println(futureProof{A: 1} == futureProof{A: 1}) // ERROR: invalid operation: futureProof{…} == futureProof{…} (struct containing [0]func() cannot be compared)
	_ = futureProof{A: 1}

	// Assignment copies every field, but slice and map fields still share their data
	// (module 006). There is no deep copy in the standard library - write a Clone method.

	// An anonymous struct declares its type inline. Two of them are identical when field names,
	// types AND tags match.
	config := struct {
		Host string
		Port int
	}{Host: "localhost", Port: 8080}
	fmt.Printf("anonymous struct: %+v (%T)\n", config, config)

	// Its natural home: table-driven tests (module 014)...
	cases := []struct {
		name string
		in   rect
		want float64
	}{
		{"unit", rect{1, 1}, 1},
		{"3x4", rect{3, 4}, 12},
	}
	for _, c := range cases {
		fmt.Printf("  case %-4s Area()=%g want=%g pass=%t\n", c.name, c.in.Area(), c.want, c.in.Area() == c.want)
	}

	// ...and one-off JSON payloads, without a named type cluttering the package.
	payload, _ := json.Marshal(struct {
		Status string `json:"status"`
		Count  int    `json:"count"`
	}{Status: "ok", Count: 2})
	fmt.Printf("  one-off JSON payload: %s\n", payload)
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
