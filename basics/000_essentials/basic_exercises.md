# Basic exercises

Three exercises for each section of [`main.go`](main.go) (the essentials of modules 001–014), 45 in total.
After them come [five larger projects](#projects) that use everything at once.
Within a section the exercises get a little harder: the first practises one idea, and the third combines several.
They use only what the section and the ones before it have shown. When an exercise needs something new, the *Hint* names it.

Reference solutions are in the [`solutions`](solutions) package. Each section has one solution file, plus a `_test.go` file with `Example` functions that show the expected output.

## How to work

1. Read the section in `main.go` and run it: `go run ./basics/000_essentials`.
2. Create your own package, for example `basics/000_essentials/mywork/`, with `package mywork` at the top of each file.
   Write the functions and types **with exactly the signatures given below**.
3. To check your work, copy the matching test file (e.g. `solutions/ex08_interfaces_test.go`) into your folder.
   Change its first line to `package mywork` and run:

   ```
   go test ./basics/000_essentials/mywork
   ```

   Each `Example` function compares what it prints with its `// Output:` comment.
   A few tests use code from an earlier exercise (11.3 and 12.3 use `WordCount` from 2b.2, for example), so solve the exercises in order.
4. Only then compare your code with the solution, and run all the solutions with `go test ./basics/000_essentials/solutions`.

Some exercises have more than one good solution. If your tests pass and the code is easy to read, the solution is fine.

| Section | Topic | Solution |
|---|---|---|
| [001](#001---variables-constants-scope) | variables, constants, scope | [`ex01_basics.go`](solutions/ex01_basics.go) |
| [002a](#002a---basic-types) | basic types | [`ex02a_basic_types.go`](solutions/ex02a_basic_types.go) |
| [002b](#002b---arrays-slices-maps-structs) | arrays, slices, maps, structs | [`ex02b_composite_types.go`](solutions/ex02b_composite_types.go) |
| [003](#003---operators) | operators | [`ex03_operators.go`](solutions/ex03_operators.go) |
| [004](#004---control-flow) | control flow | [`ex04_control_flow.go`](solutions/ex04_control_flow.go) |
| [005](#005---functions) | functions | [`ex05_functions.go`](solutions/ex05_functions.go) |
| [006](#006---pointers) | pointers | [`ex06_pointers.go`](solutions/ex06_pointers.go) |
| [007](#007---structs-and-methods) | structs and methods | [`ex07_structs_and_methods.go`](solutions/ex07_structs_and_methods.go) |
| [008](#008---interfaces) | interfaces | [`ex08_interfaces.go`](solutions/ex08_interfaces.go) |
| [009](#009---errors) | errors | [`ex09_errors.go`](solutions/ex09_errors.go) |
| [010](#010---generics) | generics | [`ex10_generics.go`](solutions/ex10_generics.go) |
| [011](#011---concurrency) | concurrency | [`ex11_concurrency.go`](solutions/ex11_concurrency.go) |
| [012](#012---iterators-and-collections) | iterators and collections | [`ex12_iterators_and_collections.go`](solutions/ex12_iterators_and_collections.go) |
| [013](#013---standard-library-essentials) | standard library | [`ex13_stdlib.go`](solutions/ex13_stdlib.go) |
| [014](#014---testing) | testing | [`ex14_testing_test.go`](solutions/ex14_testing_test.go) |
| [Projects](#projects) | 1 library · 2 cache · 3 log analyser · 4 search engine · 5 health monitor | [`library`](solutions/library) · [`cache`](solutions/cache) · [`logstats`](solutions/logstats) · [`search`](solutions/search) · [`monitor`](solutions/monitor) |

---

## 001 - Variables, constants, scope

Section: `basics()` in `main.go` · Full modules: [001a](../001a_comments_packages_variables_scope_shadowing), [001b](../001b_constants_iota_init)

### 1.1 Seconds into days, hours, minutes

Declare three constants: `SecondsPerMinute`, `SecondsPerHour` and `SecondsPerDay`. Build each one from the previous constant (`SecondsPerHour = 60 * SecondsPerMinute`), so there are no magic numbers like 3600.
Then write a function that splits a number of seconds into whole days, hours, minutes and the seconds that remain.

```go
func SplitDuration(total int) (days, hours, minutes, seconds int)
```

```
SplitDuration(90061)  -> 1 1 1 1
SplitDuration(59)     -> 0 0 0 59
```

*Hint:* `/` gives the whole units and `%` gives what is left over.

### 1.2 Traffic light enum

Define the type `TrafficLight` and the constants `Red`, `Yellow` and `Green` using `iota`. Then write:

```go
func NextLight(l TrafficLight) TrafficLight // Red -> Yellow -> Green -> Red
func LightName(l TrafficLight) string       // "red", "yellow", "green", or "unknown"
```

```
NextLight(Green)             -> Red
LightName(Yellow)            -> "yellow"
LightName(TrafficLight(7))   -> "unknown"
```

*Hint:* the constants are numbers (0, 1, 2), so `NextLight` can be computed instead of listing every case. An extra unexported constant at the end of the `iota` block can hold the number of lights.

### 1.3 Find the shadowing bug

The function below should add up numbers given as strings, and return **0 and the error** for the first value that is not a number.
It compiles, but `SumStrings([]string{"1", "x", "3"})` returns `1 <nil>`. Explain why the error gets lost, then fix the function.

```go
func SumStrings(values []string) (int, error) {
	total := 0
	var err error
	for _, v := range values {
		n, err := strconv.Atoi(v)
		if err != nil {
			break
		}
		total += n
	}
	return total, err
}
```

```
SumStrings([]string{"1", "2", "3"}) -> 6 <nil>
SumStrings([]string{"1", "x", "3"}) -> 0 strconv.Atoi: parsing "x": invalid syntax
```

*Hint:* look at what `:=` does inside the loop's block.

---

## 002a - Basic types

Section: `basicTypes()` · Full module: [002a](../002a_basic_types)

### 2a.1 Explicit conversions

```go
func CelsiusToFahrenheit(c float64) float64 // F = C*9/5 + 32
func Average3(a, b, c int) float64          // the exact average, with the fraction
```

```
CelsiusToFahrenheit(100) -> 212
Average3(1, 2, 2)        -> 1.6666666666666667 (not 1!)
```

*Hint:* where exactly you put `float64(...)` matters.

### 2a.2 Bytes, letters, digits

Return the length of `s` in **bytes**, and count how many **letters** and **digits** it contains. The string may contain Polish characters.

```go
func CharStats(s string) (bytes, letters, digits int)
```

```
CharStats("Łódź 2026!") -> 13 4 4
CharStats("Go")         -> 2 2 0
```

*Hint:* `range` over a string gives runes, and the `unicode` package has `IsLetter` and `IsDigit`.

### 2a.3 Adding numbers written as strings

Add two integers given as strings and return the sum **as a string**. If either input is not an integer, return `""` and the parsing error.

```go
func AddStrings(a, b string) (string, error)
```

```
AddStrings("40", "2") -> "42" <nil>
AddStrings("4o", "2") -> "" strconv.Atoi: parsing "4o": invalid syntax
```

*Hint:* `strconv.Atoi` and `strconv.Itoa`. Note that `string(42)` does not give `"42"`.

---

## 002b - Arrays, slices, maps, structs

Section: `compositeTypes()` · Full module: [002b](../002b_composite_types)

### 2b.1 Remove without side effects

Return a **new** slice without the element at index `i`. The input slice must not change. If the index is out of range, return an unchanged copy.

```go
func RemoveAt(nums []int, i int) []int
```

```
nums := []int{10, 20, 30, 40}
RemoveAt(nums, 1) -> [10 30 40]
RemoveAt(nums, 9) -> [10 20 30 40]
nums              -> still [10 20 30 40]
```

*Hint:* `append(nums[:i], nums[i+1:]...)` looks right but breaks the last rule. Why? Remember that slices share memory.

### 2b.2 Word count

Count how often each word appears. Words are separated by whitespace and compared case-insensitively. Punctuation simply counts as part of a word.

```go
func WordCount(text string) map[string]int
```

```
WordCount("The cat and the hat") -> map[and:1 cat:1 hat:1 the:2]
```

*Hint:* `strings.Fields` and `strings.ToLower`. Reading a missing key gives 0, so `counts[w]++` just works.

### 2b.3 Group students by grade

```go
type Student struct {
	Name  string
	Grade int
}

func GroupByGrade(students []Student) map[int][]string
```

For each grade, return the names of the students who got it, in input order.

```
GroupByGrade([]Student{{"Ann", 4}, {"Bob", 5}, {"Cid", 3}, {"Eve", 4}})
  -> map[3:[Cid] 4:[Ann Eve] 5:[Bob]]
```

---

## 003 - Operators

Section: `operators()` · Full module: [003](../003_operators)

### 3.1 Leap year

A year is a leap year if it is divisible by 4. Centuries are the exception: they must be divisible by 400. Write it as a single boolean expression.

```go
func IsLeapYear(year int) bool
```

```
2024 -> true    2023 -> false    1900 -> false    2000 -> true
```

### 3.2 Percentage

Return `part` as a percentage of `total`. When `total` is 0, return 0 instead of dividing by zero.

```go
func Percentage(part, total int) float64
```

```
Percentage(1, 3) -> 33.333...
Percentage(1, 4) -> 25
Percentage(5, 0) -> 0
```

*Hint:* `1 / 3` in Go is `0`.

### 3.3 Digit sum

Return the sum and the number of decimal digits of `n`. Treat a negative number as its absolute value, and count `0` as having one digit. Do not convert the number to a string.

```go
func DigitSum(n int) (sum, count int)
```

```
DigitSum(1234) -> 10 4
DigitSum(-505) -> 10 3
DigitSum(0)    -> 0 1
```

*Hint:* `n % 10` is the last digit, and `n /= 10` removes it.

---

## 004 - Control flow

Section: `controlFlow()` · Full module: [004](../004_control_flow)

### 4.1 FizzBuzz

Return the numbers 1..n as strings. Replace multiples of 3 with `"Fizz"`, multiples of 5 with `"Buzz"`, and multiples of both with `"FizzBuzz"`. For n ≤ 0, return an empty slice.

```go
func FizzBuzz(n int) []string
```

```
FizzBuzz(15) -> [1 2 Fizz 4 Buzz Fizz 7 8 Fizz Buzz 11 Fizz 13 14 FizzBuzz]
```

*Hint:* a `switch` with no value. Mind the order of the cases.

### 4.2 Grades

Turn points from 0 to 100 into a letter: 90+ `"A"`, 80+ `"B"`, 70+ `"C"`, 60+ `"D"`, and anything lower `"F"`. Points outside 0–100 give `"invalid"`.

```go
func Grade(points int) string
```

```
95 -> A    80 -> B    61 -> D    12 -> F    101 -> invalid
```

### 4.3 Primes

Return every prime number up to and including `limit`, in increasing order.

```go
func Primes(limit int) []int
```

```
Primes(30) -> [2 3 5 7 11 13 17 19 23 29]
Primes(1)  -> []
```

*Hint:* use two nested loops. As soon as you find a divisor, `break` out of the inner loop. It is enough to test divisors `d` with `d*d <= n`.

---

## 005 - Functions

Section: `functions()` · Full module: [005](../005_functions)

### 5.1 Maximum of any number of ints

Return the largest argument. Write the loop yourself and do not use the built-in `max`. The signature makes `MaxOf()` with no arguments a compile error, so there is no empty case to handle.

```go
func MaxOf(first int, rest ...int) int
```

```
MaxOf(3)                     -> 3
MaxOf(3, 9, 2)               -> 9
nums := []int{4, 8, 1}
MaxOf(nums[0], nums[1:]...)  -> 8
```

### 5.2 ID generator

Return a function that produces the IDs `"prefix-1"`, `"prefix-2"` and so on. Two generators must count independently.

```go
func NewIDGenerator(prefix string) func() string
```

```
userID := NewIDGenerator("user")
orderID := NewIDGenerator("order")
userID(), userID(), orderID(), userID() -> user-1 user-2 order-1 user-3
```

*Hint:* a closure. Where does the counter live?

### 5.3 Calculator with a dispatch table

Apply `op` (`"+"`, `"-"`, `"*"` or `"/"`) to `a` and `b`. Keep the operations in a `map[string]func(a, b float64) float64` instead of a long `if` chain.
Return an error for an unknown operator and for division by zero.

```go
func Calculate(a, b float64, op string) (float64, error)
```

```
Calculate(6, 3, "*") -> 18 <nil>
Calculate(1, 0, "/") -> 0 division by zero
Calculate(2, 2, "^") -> 0 unknown operator "^"
```

---

## 006 - Pointers

Section: `pointers()` · Full module: [006](../006_pointers_and_memory)

### 6.1 Swap through pointers

```go
func SwapInts(a, b *int)
```

```
a, b := 1, 2
SwapInts(&a, &b)  // now a == 2, b == 1
```

### 6.2 Players: copies vs. the real thing

```go
type Player struct {
	Name  string
	Score int
}

func AddBonus(players []Player, bonus int)               // adds bonus to every player's score
func FindPlayer(players []Player, name string) *Player   // nil when there is no such player
```

A change made through the pointer that `FindPlayer` returns must be visible in the slice.

```
players := []Player{{"Ann", 10}, {"Bob", 7}}
AddBonus(players, 5)                    // [{Ann 15} {Bob 12}]
FindPlayer(players, "Bob").Score = 100  // [{Ann 15} {Bob 100}]
FindPlayer(players, "Zoe")              // nil
```

*Hint:* `for _, p := range players { p.Score += bonus }` compiles but changes nothing. Why? The same reason matters for `FindPlayer`: return `&players[i]`, not the address of a loop copy.

### 6.3 A linked list

```go
type Node struct {
	Value int
	Next  *Node
}

func Prepend(head *Node, v int) *Node // a new list with v in front; head may be nil
func Values(head *Node) []int         // the values from head to tail
```

A `nil` `*Node` is the empty list.

```
var list *Node
Values(list) -> []
list = Prepend(list, 0); list = Prepend(list, 1); list = Prepend(list, 2)
Values(list) -> [2 1 0]
```

---

## 007 - Structs and methods

Section: `structsAndMethods()` · Full module: [007](../007_structs_and_methods)

### 7.1 Rectangle

```go
type Rectangle struct {
	Width, Height float64
}

func (r Rectangle) Area() float64
func (r Rectangle) Perimeter() float64
func (r *Rectangle) Scale(factor float64) // multiplies both sides
func (r Rectangle) String() string        // "3x4" - fmt will use it automatically
```

```
r := Rectangle{Width: 3, Height: 4}
fmt.Println(r, r.Area(), r.Perimeter()) -> 3x4 12 14
r.Scale(2)
fmt.Println(r, r.Area(), r.Perimeter()) -> 6x8 48 28
```

Explain why `Scale` is the only method with a pointer receiver.

### 7.2 Shopping cart

```go
type CartItem struct {
	Price    float64
	Quantity int
}

type ShoppingCart struct {
	items map[string]CartItem
}

func NewCart() *ShoppingCart
func (c *ShoppingCart) Add(name string, price float64, quantity int) // adding again increases the quantity
func (c *ShoppingCart) Remove(name string) bool                      // false if it was not there
func (c *ShoppingCart) Total() float64
```

```
cart := NewCart()
cart.Add("apple", 0.5, 4)
cart.Add("bread", 3.2, 1)
cart.Add("apple", 0.5, 2)
cart.Total()          -> 6.20
cart.Remove("bread")  -> true
cart.Remove("milk")   -> false
cart.Total()          -> 3.00
```

*Hint:* why is `NewCart` needed at all? (What happens when you write to a nil map?) Also, a map entry cannot be changed in place with `c.items[name].Quantity += ...`. Read the entry, change it, then write it back.

### 7.3 Embedding

```go
type Vehicle struct {
	Brand  string
	Wheels int
}

func (v Vehicle) Describe() string // "Fiat with 4 wheels"

type Car struct {
	Vehicle // embedded
	Doors int
}

func (c Car) Describe() string // "Fiat with 4 wheels and 5 doors" - reuse Vehicle.Describe
```

```
car := Car{Vehicle: Vehicle{Brand: "Fiat", Wheels: 4}, Doors: 5}
car.Brand               -> Fiat   (promoted field)
car.Describe()          -> Fiat with 4 wheels and 5 doors
car.Vehicle.Describe()  -> Fiat with 4 wheels
```

---

## 008 - Interfaces

Section: `interfaces()` · Full module: [008](../008_interfaces)

### 8.1 Animals that speak

```go
type Speaker interface {
	Sound() string
}

type Dog struct{}
type Cat struct{}
type Cow struct{}

func Chorus(speakers []Speaker) string // the sounds separated by spaces
```

`Dog`, `Cat` and `Cow` return `"Woof"`, `"Meow"` and `"Moo"`. None of them may mention `Speaker`.

```
Chorus([]Speaker{Dog{}, Cat{}, Cow{}}) -> "Woof Meow Moo"
```

Finally, add a fourth animal in a **different file** without touching `Chorus`. Notice what that shows about interfaces.

### 8.2 A clock you can control

`Greeting` returns `"Good morning"` before 12:00, `"Good afternoon"` before 18:00, and `"Good evening"` otherwise.
If it called `time.Now()` directly, you could not test it at 9 am and at 9 pm. So it takes a small interface instead:

```go
type Clock interface {
	Now() time.Time
}

type RealClock struct{}             // Now() returns time.Now()
type FixedClock struct{ T time.Time } // Now() always returns T

func Greeting(c Clock) string
```

```
at9 := FixedClock{T: time.Date(2026, time.January, 1, 9, 0, 0, 0, time.UTC)}
Greeting(at9)          -> Good morning
Greeting(RealClock{})  -> depends on when you run it
```

*Hint:* `var _ Clock = RealClock{}` is a one-line compile-time check that a type satisfies an interface.

### 8.3 Sizes of anything

Add up the sizes of the items:
- a `string` counts its bytes
- a `[]int` counts its length
- an `int` counts as its own value
- any type with a `Size() int` method counts what `Size()` returns
- everything else (including `nil`) counts as 0

```go
type Sizer interface {
	Size() int
}

type File struct {
	Name  string
	Bytes int
}

func (f File) Size() int

func TotalSize(items []any) int
```

```
TotalSize([]any{"abc", []int{1, 2}, 10, File{Name: "a.txt", Bytes: 100}, 3.5, nil}) -> 115
```

*Hint:* a type switch. A `case` can name an interface (`case Sizer:`), and then it matches every type that has the method.

---

## 009 - Errors

Section: `errorHandling()` · Full module: [009](../009_errors)

### 9.1 Parse an age

Parse `s` as an age between 0 and 150.
- Not a number: return the `strconv` error, **wrapped** with context.
- Out of range: return an error that wraps the sentinel `ErrAgeOutOfRange`.

```go
var ErrAgeOutOfRange = errors.New("age out of range")

func ParseAge(s string) (int, error)
```

```
ParseAge("42")  -> 42 <nil>
ParseAge("abc") -> parse age "abc": strconv.Atoi: parsing "abc": invalid syntax
ParseAge("200") -> parse age 200: age out of range
errors.Is(err, ErrAgeOutOfRange) -> true for "200"
errors.Is(err, strconv.ErrSyntax) -> true for "abc"
```

### 9.2 A custom error type

```go
type InsufficientFundsError struct {
	Balance, Amount float64
}

func (e *InsufficientFundsError) Error() string    // "insufficient funds: balance 100.00, requested 130.00"
func (e *InsufficientFundsError) Missing() float64 // Amount - Balance

func Withdraw(balance, amount float64) (float64, error)
```

`Withdraw` returns the new balance. A non-positive amount is a plain error. An amount larger than the balance returns `*InsufficientFundsError`, wrapped as `"withdraw: %w"`.
The caller must use `errors.As` to read `Missing()`. A type assertion will not work, because the error is wrapped.

```
Withdraw(100, 30)   -> 70 <nil>
Withdraw(100, 130)  -> withdraw: insufficient funds: balance 100.00, requested 130.00
                       errors.As(...) -> true, Missing() == 30
Withdraw(100, -5)   -> withdraw: amount must be positive   (errors.As -> false)
```

### 9.3 Errors through two layers

```go
var ErrMissingSetting = errors.New("missing setting")

func ReadSetting(settings map[string]string, key string) (int, error)
func Port(settings map[string]string) (int, error) // the "port" setting, must be 1-65535
```

`ReadSetting` wraps `ErrMissingSetting` for a missing key, or the `strconv` error for a value that is not a number.
`Port` calls `ReadSetting`, wraps whatever error it gets with `"config: %w"`, and checks the range.
The caller should be able to say "no port configured? use 80" with `errors.Is`, even though the sentinel is two levels down.

```
Port(map[string]string{"port": "8080"})  -> 8080 <nil>
Port(map[string]string{})                -> config: setting "port": missing setting
Port(map[string]string{"port": "http"})  -> config: setting "port": strconv.Atoi: parsing "http": invalid syntax
Port(map[string]string{"port": "70000"}) -> config: port 70000 out of range 1-65535
```

---

## 010 - Generics

Section: `generics()` · Full module: [010](../010_generics)

### 10.1 The largest element of any slice

Write the function once so that it works for `[]int`, `[]float64`, `[]string` and so on.
For an empty slice, return the zero value of `T` and an error.

```go
func Largest[T cmp.Ordered](items []T) (T, error)
```

```
Largest([]int{3, 9, 2})                    -> 9 <nil>
Largest([]string{"pear", "apple", "plum"}) -> plum <nil>
Largest([]float64{})                       -> 0 largest: empty slice
```

*Hint:* `var zero T`. Then answer this: why would `[T any]` not compile here?

### 10.2 Group by any key

This is the general version of `GroupByGrade` from 2b.3. Put every item into the group chosen by `key`, keeping input order within each group.

```go
func GroupBy[T any, K comparable](items []T, key func(T) K) map[K][]T
```

```
GroupBy([]string{"apple", "avocado", "banana", "blueberry", "cherry"},
        func(s string) string { return s[:1] })
  -> map[a:[apple avocado] b:[banana blueberry] c:[cherry]]

GroupBy([]int{1, 2, 3, 4, 5}, func(n int) bool { return n%2 == 0 })
  -> map[false:[1 3 5] true:[2 4]]
```

Why does `T` only need `any`, while `K` must be `comparable`?

### 10.3 A generic Set

```go
type Set[T comparable] struct {
	items map[T]struct{}
}

func NewSet[T comparable](items ...T) *Set[T]
func (s *Set[T]) Add(v T)
func (s *Set[T]) Has(v T) bool
func (s *Set[T]) Len() int
func (s *Set[T]) Union(other *Set[T]) *Set[T] // a NEW set; s and other do not change
func (s *Set[T]) Items() []T                  // in any order
```

Bonus: make the zero value work too, so that `var s Set[string]; s.Add("x")` does not panic.

```
a := NewSet(1, 2, 3, 2)          // T inferred as int
a.Len(), a.Has(2), a.Has(4)      -> 3 true false
a.Union(NewSet(3, 4)).Items()    -> 1 2 3 4 (in some order)
NewSet[string]()                 // no arguments, so T must be written out
```

*Hint:* `struct{}` takes no memory, which makes `map[T]struct{}` the usual way to write a set in Go.

---

## 011 - Concurrency

Section: `concurrency()` · Full module: [011](../011_concurrency)

Run your tests with `go test -race`. The race detector tells you if two goroutines touch the same data without synchronisation.

### 11.1 Parallel sum

Split `nums` into `parts` chunks, sum each chunk in its own goroutine, then add up the partial sums. Treat `parts < 1` as 1.

```go
func ParallelSum(nums []int, parts int) int
```

```
nums = 1..100
ParallelSum(nums, 4)  -> 5050
ParallelSum(nums, 7)  -> 5050
ParallelSum(nil, 3)   -> 0
```

*Hint:* `sync.WaitGroup` with `wg.Go(...)`. Give each goroutine its own slot in a `[]int` for its result: no two goroutines write to the same place, so you need no mutex.

### 11.2 Timeout

Run `f` in a goroutine. Return its result, or `ErrTimeout` if it takes longer than `d`.

```go
var ErrTimeout = errors.New("timeout")

func RunWithTimeout(f func() int, d time.Duration) (int, error)
```

```
RunWithTimeout(func() int { return 42 }, time.Second)                         -> 42 <nil>
RunWithTimeout(func() int { time.Sleep(time.Second); return 1 }, 10*time.Millisecond) -> 0 timeout
```

*Hint:* `select` with `time.After(d)`. Should the result channel be buffered? Think about what happens to the slow goroutine when nobody receives its result any more.

### 11.3 Concurrent word count

Count words across all texts, like `WordCount` from 2b.2, using one goroutine per text. Merge everything into one result map.

```go
func CountWordsConcurrently(texts []string) map[string]int
```

```
CountWordsConcurrently([]string{"go is fun", "Go is fast", "fun fun fun"})
  -> map[fast:1 fun:4 go:2 is:2]
```

*Hint:* a `sync.Mutex` around the shared map. Do the counting **outside** the lock, and lock only to merge.

---

## 012 - Iterators and collections

Section: `iteratorsAndCollections()` · Full module: [012](../012_iterators_and_collections)

### 12.1 Range iterator

Yield `start`, `start+step`, `start+2*step`, … while the value is below `end`, or above `end` for a negative step. This works like `range()` in Python. A step of 0 yields nothing.
The iterator must stop as soon as the loop that uses it calls `break`.

```go
func Range(start, end, step int) iter.Seq[int]
```

```
for i := range Range(0, 10, 3)   -> 0 3 6 9
for i := range Range(5, 0, -2)   -> 5 3 1
slices.Collect(Range(1, 4, 1))   -> [1 2 3]
```

### 12.2 Sort people

Sort in place by age, youngest first. People of the same age are sorted by name.

```go
type Person struct {
	Name string
	Age  int
}

func SortPeople(people []Person)
```

```
[{Cid 31} {Bob 25} {Ann 31} {Eve 19}] -> [{Eve 19} {Bob 25} {Ann 31} {Cid 31}]
```

*Hint:* `slices.SortFunc` and `cmp.Compare`. Compare the ages first, and only when they are equal compare the names.

### 12.3 Most frequent words

Return the `n` most frequent words, most frequent first, with ties in alphabetical order. If `n` is larger than the number of words, return all of them.

```go
func TopWords(counts map[string]int, n int) []string
```

```
counts := WordCount("the cat and the hat and the bat")
TopWords(counts, 2)   -> [the and]
TopWords(counts, 4)   -> [the and bat cat]
TopWords(counts, 100) -> [the and bat cat hat]
```

*Hint:* `maps.Keys` returns an iterator, and `slices.Collect` turns it into a slice that you can sort.

---

## 013 - Standard library essentials

Section: `stdlib()` · Full module: [013](../013_stdlib_essentials)

### 13.1 Parse key=value pairs

Parse `"key=value"` pairs separated by `;`. Trim the spaces around keys and values. Skip parts that have no `=` or have an empty key.

```go
func ParseKeyValues(s string) map[string]string
```

```
ParseKeyValues(" name = Ann; age=31;; city=Warsaw ; broken; =x")
  -> map[age:31 city:Warsaw name:Ann]
```

*Hint:* `strings.Split`, `strings.Cut`, `strings.TrimSpace`.

### 13.2 Save and load books as JSON

```go
type Book struct {
	Title  string   `json:"title"`
	Author string   `json:"author"`
	Year   int      `json:"year"`
	Tags   []string `json:"tags,omitempty"`
}

func SaveBooks(path string, books []Book) error // indented JSON
func LoadBooks(path string) ([]Book, error)
```

Wrap errors with context (`"save books: %w"`). For a missing file, the error from `LoadBooks` must still satisfy `errors.Is(err, fs.ErrNotExist)`.

```
SaveBooks(path, []Book{{Title: "Dune", Author: "Herbert", Year: 1965}})
file content:
[
  {
    "title": "Dune",
    "author": "Herbert",
    "year": 1965
  }
]
```

*Hint:* `json.MarshalIndent`, `os.WriteFile`, `os.ReadFile`, `json.Unmarshal`. In tests and examples, write to a temporary directory (`t.TempDir()` or `os.MkdirTemp`).

### 13.3 Parse a log line

```go
type LogEntry struct {
	Time    time.Time
	Level   string
	Message string
}

func ParseLogLine(line string) (LogEntry, error)
```

The line format is `date time LEVEL message`. The message may contain spaces.

```
ParseLogLine("2026-03-14 09:30:00 ERROR disk full")
  -> Level "ERROR", Message "disk full", Time 2026-03-14 09:30:00 UTC (a Saturday)
ParseLogLine("2026-13-01 10:00:00 INFO hello")
  -> error: parsing time "2026-13-01 10:00:00": month out of range
ParseLogLine("hello")
  -> error
```

*Hint:* `strings.SplitN(line, " ", 4)`, plus `time.Parse` with a layout written using the reference date `2006-01-02 15:04:05`.

---

## 014 - Testing

Section: `testingIntro()` · Full module: [014](../014_testing)

These exercises are about writing **tests**. Put them in a `_test.go` file next to your code and run them with `go test -v`.

### 14.1 Table-driven test

Write `TestIsLeapYear` for your `IsLeapYear` from 3.1. Use a table of cases (name, year, expected result) with at least 2024, 2023, 1900 and 2000, and run each case as a subtest with `t.Run`.
Make a failure message say what was received and what was expected: `IsLeapYear(1900) = true, want false`.

Check that you can run a single case: `go test -run 'TestIsLeapYear/century' -v`.
Then break `IsLeapYear` on purpose and watch the test fail.

### 14.2 Testing the error path

Write `TestParseAge` for `ParseAge` from 9.1, with three subtests:
- **valid**: `"42"` gives 42 and no error. If there is an error, stop at once with `t.Fatalf`, because checking the value makes no sense then.
- **out of range**: `"-1"` and `"151"` both give an error for which `errors.Is(err, ErrAgeOutOfRange)` is true. Use `t.Errorf` so that every bad input is reported.
- **not a number**: `"abc"` gives an error that is *not* `ErrAgeOutOfRange`.

### 14.3 Benchmark: `+=` vs `strings.Builder`

Write two functions that join strings, then measure which is faster.

```go
func JoinPlus(parts []string, sep string) string    // result += ...
func JoinBuilder(parts []string, sep string) string // strings.Builder
```

1. Write `TestJoin`, which checks that both functions give the same result as `strings.Join` for `nil`, one element and three elements.
2. Write `BenchmarkJoinPlus` and `BenchmarkJoinBuilder`, each joining 1000 words, using `for b.Loop() { ... }`.
3. Write `ExampleJoinBuilder` with an `// Output:` comment.
4. Run `go test -run '^$' -bench Join -benchmem` and compare `ns/op`, `B/op` and `allocs/op`. Can you explain the difference?

---

## Projects

Five larger tasks. Each one is a small but realistic package that needs almost everything from sections 001–014 at the same time.
Expect one to three hours per project.

How the projects differ from the exercises above:
- The requirements say **what** the package must do, and the internals are up to you. Only the exported API is fixed, so the reference tests can run against your code.
- **Writing tests is part of every project.** Each project lists the tests to write. Run them with `go test -race`.
- Each reference solution is a package of its own (`solutions/library`, `solutions/cache`, …), with its tests in the same folder.
  To check your version, give your package the same name (e.g. `package library`), copy the solution's `_test.go` next to your code, and run `go test -race .`.
  The tests bring their own fakes (clocks, services), so they only need your exported API. Project 5 is the one exception, noted in its API.

| Project | Builds on |
|---|---|
| [1 - Library](#project-1---library) | structs, maps, methods, the `Clock` interface, errors, `time`, JSON files, an iterator, a mutex |
| [2 - Generic cache with expiry](#project-2---generic-cache-with-expiry) | generic types, a fake clock, a `Stringer`, closures, `iter.Seq2`, a mutex |
| [3 - Log analyser](#project-3---log-analyser) | an enum, parsing, a custom error with `Unwrap`, `errors.Join`, `io.Reader`, generics, a worker pool, JSON |
| [4 - Mini search engine](#project-4---mini-search-engine) | runes and `unicode`, maps of maps, a filter-chain interface, a generic `Set`, `errors.Join`, a worker pool, `RWMutex`, lazy iterators |
| [5 - Service health monitor](#project-5---service-health-monitor) | interfaces, `context`, a generic `Retry`, `select`, a worker pool, `MarshalText`, `slog` |

### Project 1 - Library

A small library lends books to members. It must enforce the lending rules, charge late fees, list overdue loans, and survive a restart by saving its state to a JSON file.
Several librarians use it at the same time, so it must be safe for concurrent use.

**Practises:** constants (001), structs and maps (002b), methods with pointer receivers and a constructor (007), a `Clock` interface as in 8.2 (008), sentinel and custom errors (009), a small generic helper (010), a mutex (011), an `iter.Seq` (012), `time` arithmetic and JSON files (013), tests (014).

**Requirements**
1. A book has an ID, title, author and year. The ID and the title are required, and IDs are unique.
2. A member may borrow a book that exists and is not on loan. A loan lasts `LoanPeriod` (14 days).
3. A member can have at most `MaxLoans` (3) books at a time. Going over that returns a wrapped `*LoanLimitError` that says who the member is and what the limit is.
4. On return, the late fee is `FeePerDay` (50 cents) for every **full** day after the due date. Returning a book that is not on loan is an error.
5. `Search` finds books whose title or author contains the query, ignoring case, sorted by title.
6. `Overdue` yields the overdue loans, the most overdue first. Calling other `Library` methods inside the loop must not deadlock.
7. `Save` writes everything as indented JSON, sorted, so the file does not change if the data does not. `Load` reads it back and rejects loans of unknown books.
8. Every method is safe to call from many goroutines at once.

**API**

```go
const (
	LoanPeriod = 14 * 24 * time.Hour
	MaxLoans   = 3
	FeePerDay  = 50 // cents
)

var (
	ErrInvalidBook   = errors.New("invalid book")
	ErrDuplicateBook = errors.New("duplicate book")
	ErrBookNotFound  = errors.New("book not found")
	ErrAlreadyOnLoan = errors.New("book already on loan")
	ErrNotOnLoan     = errors.New("book not on loan")
)

type LoanLimitError struct {
	MemberID string
	Limit    int
}

func (e *LoanLimitError) Error() string

type Clock interface {
	Now() time.Time
}

type RealClock struct{}

func (RealClock) Now() time.Time

type Book struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Author string `json:"author"`
	Year   int    `json:"year"`
}

type Loan struct {
	BookID   string    `json:"book_id"`
	MemberID string    `json:"member_id"`
	Borrowed time.Time `json:"borrowed"`
	Due      time.Time `json:"due"`
}

type Library struct{ /* your fields */ }

func New(clock Clock) *Library
func (l *Library) AddBook(b Book) error
func (l *Library) Borrow(memberID, bookID string) (Loan, error)
func (l *Library) Return(bookID string) (int, error) // the late fee in cents
func (l *Library) Search(query string) []Book
func (l *Library) Overdue() iter.Seq[Loan]
func (l *Library) Save(path string) error
func Load(path string, clock Clock) (*Library, error)
func LateFee(due, returned time.Time) int
```

**Example**

```
lib := library.New(clock)                     // a fake clock set to 2026-03-01 10:00
lib.AddBook(library.Book{ID: "b1", Title: "Dune", Author: "Herbert", Year: 1965})
loan, _ := lib.Borrow("ann", "b1")            // loan.Due is 2026-03-15
_, err := lib.Borrow("bob", "b1")             // borrow "b1": book already on loan
clock.Advance(library.LoanPeriod + 3*24*time.Hour)
for loan := range lib.Overdue() { ... }       // b1, borrowed by ann
fee, _ := lib.Return("b1")                    // 150 = 3 full days late
```

**Tests to write**
- A table-driven test of `LateFee`: early, exactly on time, 23 hours late, one day late, ten and a half days late.
- Every error of `AddBook` and `Borrow`, checked with `errors.Is`. For the loan limit, use `errors.As` and check `MemberID` and `Limit`.
- Borrow and return with a fake clock that you move forward. 2 days and 5 hours late costs 2 × `FeePerDay`.
- `Overdue` yields in the right order, and calling `Return` inside the loop works.
- `Save`, then `Load` from `t.TempDir()`: the books and the loans come back.
- 50 goroutines borrow the same book at once, and exactly one succeeds. Run it with `-race`.

**Hints**
- In `Borrow`, keep the check ("is it on loan?") and the update under **one** lock. Lock–check–unlock followed by lock–write lets two goroutines pass the check.
- In `Overdue`, copy the overdue loans while holding the lock, unlock, then yield. Yielding while holding the lock deadlocks as soon as the loop calls `Return`, because `sync.Mutex` cannot be locked twice by the same goroutine.
- `Duration / Duration` is a plain number: `int(late / (24 * time.Hour))` is the number of full days.
- Map order is random, so sort before saving. One generic `sortedValues[K cmp.Ordered, V any](m map[K]V) []V` serves both maps.

**Going further** (optional, no reference solution): reservations (a queue of members waiting for a book), a member's loan history, renewing a loan once.

**Solution:** [`solutions/library`](solutions/library)

### Project 2 - Generic cache with expiry

An in-memory cache in front of something slow, such as a database or an exchange-rate API.
Entries expire after a fixed time-to-live (TTL), the cache never holds more than `capacity` entries, and it counts hits and misses.
It works for any key and value type, and it is safe for concurrent use.

**Practises:** maps and zero values (002b), closures as loaders (005), a `Stringer` (007/008), a `Clock` interface (008), errors (009), generic types (010), a mutex (011), `iter.Seq2` (012), tests with a fake clock (014).

**Requirements**
1. `New` rejects a TTL ≤ 0 (`ErrInvalidTTL`) and a capacity < 1 (`ErrInvalidCapacity`).
2. `Set` stores a value for one TTL. Setting an existing key replaces the value and restarts its TTL. An entry is alive while `now < time it was set + TTL`.
3. `Get` returns `(value, true)` for a live entry and `(zero value, false)` for a missing or expired one. An expired entry is removed when `Get` finds it.
4. When a **new** key arrives and the cache is full, drop the expired entries first. If it is still full, evict the entry that expires soonest and count one eviction. Overwriting an existing key never evicts.
5. `Len` counts only live entries. `All` yields only live entries, and its loop body may use the cache.
6. `GetOrLoad` returns the cached value. On a miss it calls `load`, stores the result and returns it. A load error is returned wrapped, and nothing is stored. `load` must run **without** the lock held.
7. `Stats` counts hits, misses and evictions, and `Stats.String()` gives `hits=3 misses=1 evictions=0 hit-rate=75%`.

**API**

```go
var (
	ErrInvalidTTL      = errors.New("ttl must be positive")
	ErrInvalidCapacity = errors.New("capacity must be at least 1")
)

type Clock interface {
	Now() time.Time
}

type RealClock struct{}

func (RealClock) Now() time.Time

type Stats struct {
	Hits, Misses, Evictions int
}

func (s Stats) HitRate() float64
func (s Stats) String() string

type Cache[K comparable, V any] struct{ /* your fields */ }

func New[K comparable, V any](ttl time.Duration, capacity int, clock Clock) (*Cache[K, V], error)
func (c *Cache[K, V]) Set(key K, value V)
func (c *Cache[K, V]) Get(key K) (V, bool)
func (c *Cache[K, V]) Delete(key K)
func (c *Cache[K, V]) Len() int
func (c *Cache[K, V]) Stats() Stats
func (c *Cache[K, V]) All() iter.Seq2[K, V]
func (c *Cache[K, V]) GetOrLoad(key K, load func(K) (V, error)) (V, error)
```

**Example**

```
prices, _ := cache.New[string, float64](10*time.Minute, 100, clock)
prices.Set("BTC", 61000)
prices.Get("BTC")                       // 61000 true
clock.Advance(11 * time.Minute)
prices.Get("BTC")                       // 0 false - expired
prices.GetOrLoad("ETH", fetchPrice)     // a miss: calls fetchPrice, stores the result
prices.Stats()                          // hits=1 misses=2 evictions=0 hit-rate=33%
```

**Tests to write**
- `New` rejects bad arguments (table-driven, `errors.Is`).
- An entry expires at exactly its TTL. Move a fake clock with an `Advance` method instead of calling `time.Sleep`.
- The eviction order and the `Evictions` counter. Also check that expired entries make room without counting as evictions.
- `GetOrLoad` calls `load` only once for a repeated key, wraps load errors (`errors.Is`), and stores nothing after an error.
- `All` skips expired entries and stops at `break`.
- 20 goroutines call `Set`, `Get`, `GetOrLoad` and `All` on one cache under `-race`. Your fake clock needs a mutex too!

**Hints**
- `New` has no parameter of type `K` or `V`, so Go cannot infer them. Callers must write `New[string, int](...)`.
- Store a small generic struct per entry: `entry[V any]` with `value V` and `expires time.Time`.
- Go allows deleting from a map while ranging over it.
- In `All`, copy the live entries into a new map under the lock, then yield from the copy.

**Going further** (optional): true LRU (least recently used) eviction with `container/list`; a background goroutine that removes expired entries every minute and stops via a `context`; concurrent `GetOrLoad` calls for the same key that share one load (see `golang.org/x/sync/singleflight`).

**Solution:** [`solutions/cache`](solutions/cache)

### Project 3 - Log analyser

Read log files with lines like `2026-03-14 09:30:00 ERROR disk full`. Count the entries per level and per hour, find the most frequent error messages, analyse many files in parallel, and produce a JSON report.
Real logs contain garbage, so a bad line must not stop the analysis.

**Practises:** an enum with `iota` and `String` (001/007), arrays and maps (002b), interfaces from the standard library (`io.Reader`, 008), a custom error type with `Unwrap` and `errors.Join` (009), a generic `TopN` (010), a worker pool (011), `iter.Seq2` (012), strings, `time.Parse` and JSON (013, building on exercise 13.3), tests with fake readers (014).

**Requirements**
1. `Level` has the values `Debug`, `Info`, `Warn` and `Error`. `String()` gives `"ERROR"` and so on, and `ParseLevel` accepts any letter case.
2. `ParseLine` parses `date time LEVEL message`. The message may contain spaces.
3. `Entries` yields one `(Entry, error)` per non-blank line. A bad line yields a `*ParseError` with the source name, the line number (from 1) and the cause, and the loop may go on. If the reader fails, that error is yielded once, at the end.
4. `Stats` counts the entries: total, invalid lines, per level, per hour of the day (`[24]int`), ERROR messages, and the first and last timestamps. `Merge` adds in another `Stats`.
5. `Analyze` returns **partial results**: statistics for the valid lines, plus all parse errors joined into one error with `errors.Join`.
6. `AnalyzeAll` runs `Analyze` on all sources using `workers` goroutines and merges the results. The errors come out in the order of the sources, whichever worker finished first.
7. `Report(topN)` builds a JSON-ready report, and `WriteJSON` writes it indented. In the report:
   - levels appear as names (`"ERROR": 3`)
   - only the hours that had entries appear, as two-digit strings (`"09": 1`)
   - `top_errors` is `[]`, never `null`
8. `TopN` returns the keys with the highest counts. Ties are broken by key, so the result is always the same.

**API**

```go
type Level int

const (
	Debug Level = iota
	Info
	Warn
	Error
)

func (l Level) String() string
func ParseLevel(s string) (Level, error)

type Entry struct {
	Time    time.Time
	Level   Level
	Message string
}

func ParseLine(line string) (Entry, error)

type ParseError struct {
	Source string
	Line   int
	Err    error
}

func (e *ParseError) Error() string // app.log:6: unknown level "FATAL"
func (e *ParseError) Unwrap() error

func Entries(source string, r io.Reader) iter.Seq2[Entry, error]

type Stats struct {
	Total    int
	Invalid  int
	ByLevel  map[Level]int
	PerHour  [24]int
	Messages map[string]int // ERROR messages -> count
	First    time.Time
	Last     time.Time
}

func NewStats() *Stats
func (s *Stats) Add(e Entry)
func (s *Stats) Merge(other *Stats)
func Analyze(source string, r io.Reader) (*Stats, error)

type Source struct {
	Name   string
	Reader io.Reader
}

func AnalyzeAll(sources []Source, workers int) (*Stats, error)

type MessageCount struct {
	Message string `json:"message"`
	Count   int    `json:"count"`
}

type Report struct {
	Total     int            `json:"total"`
	Invalid   int            `json:"invalid"`
	ByLevel   map[string]int `json:"by_level"`
	PerHour   map[string]int `json:"per_hour"`
	TopErrors []MessageCount `json:"top_errors"`
	First     time.Time      `json:"first"`
	Last      time.Time      `json:"last"`
}

func (s *Stats) Report(topN int) Report
func (r Report) WriteJSON(w io.Writer) error
func TopN[K cmp.Ordered](counts map[K]int, n int) []K
```

**Example**

```
2026-03-14 09:30:00 INFO started            s, err := logstats.Analyze("app.log", file)
2026-03-14 10:00:00 ERROR disk full         s.Report(1).WriteJSON(os.Stdout)
2026-03-14 10:15:00 ERROR disk full
2026-03-14 10:20:00 ERROR timeout           {
                                              "total": 4,
                                              "invalid": 0,
                                              "by_level": { "ERROR": 3, "INFO": 1 },
                                              "per_hour": { "09": 1, "10": 3 },
                                              "top_errors": [ { "message": "disk full", "count": 2 } ],
                                              "first": "2026-03-14T09:30:00Z",
                                              "last": "2026-03-14T10:20:00Z"
                                            }
```

**Tests to write**
- `ParseLine` as a table: valid, lower-case level, too few fields, bad time, unknown level.
- `Analyze` on a sample with two bad lines. Check that:
  - the counts are right
  - `errors.As` finds a `*ParseError` with the right line number
  - `errors.As` also reaches the `*time.ParseError` inside it, thanks to `Unwrap`
- A reader that fails half-way: `io.MultiReader(strings.NewReader("..."), iotest.ErrReader(errDisk))`. The error must wrap `errDisk`, and the line before the failure still counts.
- `AnalyzeAll` with 3 sources and 2 workers: the counts are merged, `First` is the earliest time, and the errors are in source order. Run with `-race`.
- `TopN` with ties, with `n` larger than the map, and with `int` keys.
- The report survives a JSON round trip (`json.Unmarshal` back into a `Report`).

**Hints**
- `strings.SplitN(line, " ", 4)` keeps the whole message, spaces included, as the 4th part.
- `bufio.Scanner.Scan` returns false both at the end of the input and on an error, so always check `sc.Err()` afterwards.
- Add `Unwrap() error` to `*ParseError`, so `errors.Is` and `errors.As` can see the cause.
- In `AnalyzeAll`, give every source its own slot in a results slice. Then no mutex is needed, and the order is kept for free.
- `cmp.Or(cmp.Compare(...), cmp.Compare(...))` is a compact two-key comparison: the first non-zero result wins.

**Going further** (optional): a command that takes file names as arguments (`os.Open` + `defer f.Close()`); a `-since` time filter; a per-hour bar chart printed in the terminal.

**Solution:** [`solutions/logstats`](solutions/logstats)

### Project 4 - Mini search engine

A small full-text search engine, the core of what sits behind every search box.
It splits documents into words, cleans them up with a chain of filters, builds an **inverted index** (word → the documents that contain it), and answers queries like `go concurrency -java` with results ranked by relevance.
It indexes many documents in parallel, can save and reload itself as JSON, and must stay correct while one goroutine adds documents and others search.

**Practises:** strings, runes and `unicode` (002a/013), maps of maps (002b), closures (005), a pointer to a slice (006), methods on a function type (007), a `TokenFilter` interface that accepts your own types (008), a custom error with `Unwrap` and `errors.Join` (009), a generic `Set[T]` (010), a worker pool and `sync.RWMutex` (011), lazy `iter.Seq`/`iter.Seq2` (012), JSON over `io.Reader`/`io.Writer` (013), tests (014).

**Requirements**
1. A word is a run of letters and digits (Unicode-aware, so `zażółć` is one word), and everything else separates words. `Tokens` is **lazy**: it yields one token at a time and stops when the loop breaks.
2. An `Analyzer` runs its filters in order, and the first filter that drops a token ends the chain. Provide three filters:
   - `Lowercase`
   - `MinLength(n)`, which counts characters, not bytes
   - `StopWords(...)`

   Any type with a `Filter` method can be used as a filter, for example your own simple stemmer.
3. `DefaultAnalyzer` is `Lowercase`, then `MinLength(2)`, then the stop words `an and are for in is of on or the to with`. `NewIndex(nil)` uses it.
4. `Add` rejects a blank ID (`ErrEmptyID`) and an ID already in use (`ErrDuplicateID`). Both are wrapped in a `*DocError` that carries the ID.
5. Weights: a word in the **title counts 2**, a word in the body counts 1. For every term, the index keeps the documents that contain it and their weighted counts.
6. `AddAll` analyses the documents on `workers` goroutines, then inserts them in input order. Every bad document is reported in one joined error (`errors.Join`), and the good ones are added anyway.
7. `Search`:
   - query words go through the same analyzer as the documents, so case and stop words work the same way
   - a hit contains **all** normal terms and **none** of the `-terms`
   - score = the sum of the weighted counts of the normal terms
   - results are ordered by score (highest first), then by ID
   - no hits is an empty slice and a nil error
   - a query with no normal term left (empty, only stop words, only `-terms`) returns a wrapped `ErrEmptyQuery`
8. `Terms` yields every term with the number of documents containing it, in alphabetical order.
9. `Save` writes the **documents**, not the index, as JSON sorted by ID. `Load` reads them and rebuilds the index.
10. An `Index` is safe for concurrent use: many searches at once, while `Add` gets exclusive access.

**API**

```go
var (
	ErrEmptyID     = errors.New("document id is empty")
	ErrDuplicateID = errors.New("duplicate document id")
	ErrEmptyQuery  = errors.New("query has no search terms")
)

type DocError struct {
	ID  string
	Err error
}

func (e *DocError) Error() string // document "d1": duplicate document id
func (e *DocError) Unwrap() error

type TokenFilter interface {
	Filter(token string) (result string, keep bool)
}

type FilterFunc func(token string) (string, bool)

func (f FilterFunc) Filter(token string) (string, bool)

var Lowercase TokenFilter

func MinLength(n int) TokenFilter
func StopWords(words ...string) TokenFilter

type Analyzer struct{ /* your fields */ }

func NewAnalyzer(filters ...TokenFilter) *Analyzer
func DefaultAnalyzer() *Analyzer
func (a *Analyzer) Tokens(text string) iter.Seq[string]

type Set[T comparable] struct{ /* your fields */ }

func NewSet[T comparable](items ...T) *Set[T]
func (s *Set[T]) Add(v T)
func (s *Set[T]) Has(v T) bool
func (s *Set[T]) Len() int
func (s *Set[T]) Intersect(other *Set[T]) *Set[T]  // a NEW set
func (s *Set[T]) Difference(other *Set[T]) *Set[T] // in s but not in other, a NEW set
func (s *Set[T]) All() iter.Seq[T]

type Document struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Body  string `json:"body"`
}

type Result struct {
	ID    string
	Title string
	Score int
}

type Index struct{ /* your fields */ }

func NewIndex(a *Analyzer) *Index // nil means DefaultAnalyzer()
func (ix *Index) Add(doc Document) error
func (ix *Index) AddAll(docs []Document, workers int) error
func (ix *Index) Search(query string) ([]Result, error)
func (ix *Index) Len() int
func (ix *Index) Terms() iter.Seq2[string, int]
func (ix *Index) Save(w io.Writer) error
func Load(r io.Reader, a *Analyzer, workers int) (*Index, error)
```

**Example**

```
ix := search.NewIndex(nil)
ix.AddAll([]search.Document{
	{ID: "go-intro", Title: "Learning Go", Body: "Go is simple. Goroutines make concurrency easy."},
	{ID: "java-intro", Title: "Learning Java", Body: "Java has threads for concurrency."},
	{ID: "go-generics", Title: "Go generics", Body: "Type parameters in Go."},
}, 2)

ix.Search("go")                 // go-generics(3) go-intro(3)   - title 2 + body 1; a tie, so by ID
ix.Search("concurrency")        // go-intro(1) java-intro(1)
ix.Search("concurrency -java")  // go-intro(1)
ix.Search("the")                // error: search "the": query has no search terms
```

**Tests to write**
- `Tokens` as a table:
  - punctuation (`"Hello, World!"`)
  - `"Go's"` (the `s` is too short)
  - stop words
  - Polish letters in upper case (`"ZAŻÓŁĆ gęślą jaźń"`)
  - numbers (`"Go 1.27"`)
  - a text with no words at all
- `Tokens` stops after `break`, even on a very long text.
- A filter of **your own struct type** (e.g. a `pluralStripper` that removes a trailing `s`): after it, `goroutine` finds documents that say `goroutines`.
- `Search` as a table: a single term, AND, case-insensitivity, exclusion, stop words in the query, no hits, an unknown term. Check the exact order, including ties broken by ID.
- The scores themselves: a title hit (2) ranks above a body hit (1).
- Every way to get `ErrEmptyQuery`: `""`, spaces, only stop words, only `-go`.
- `Add` and `AddAll` errors:
  - `errors.AsType[*DocError]` (or `errors.As`) gives the ID
  - `errors.Is` finds both sentinels in a joined error
  - the good documents of the batch are still added
- `Save` into a `bytes.Buffer`, then `Load`: the same query gives the same results. Also test that invalid JSON and duplicate IDs in the file are errors.
- 50 goroutines add documents while 50 others search and range over `Terms`. Run it with `-race`.

**Hints**
- `strings.FieldsFuncSeq(text, isSeparator)` (Go 1.24) yields the words one by one, which is exactly the lazy iterator you need. `unicode.IsLetter` and `unicode.IsDigit` know Polish letters.
- The index is a `map[string]map[string]int` (term → document ID → count). Create the inner map on first use. Reading from a missing inner map is fine, because reading a nil map is allowed.
- AND is an intersection and NOT is a difference. With a generic `Set[T]` (as in 10.3), `Search` stays short.
- Analyse the document **before** taking the lock, since that is the slow part, and lock only to insert. With an `RWMutex`, `Search` takes `RLock` and `Add` takes `Lock`.
- In the query parser, a pointer to the slice you are filling (`target := &include`, then `*target = append(*target, tok)`) avoids writing the loop twice.
- Build the results with `make([]Result, 0, n)`, so that "no hits" is `[]` and not `nil`. That difference shows in JSON.

**Going further** (optional, no reference solution): prefix search (`conc*`); phrase search (`"type parameters"`, which needs word positions in the index); a snippet of the text around each hit; a command that indexes every `.txt` file in a directory (`filepath.WalkDir`) and answers queries typed on `os.Stdin`.

**Solution:** [`solutions/search`](solutions/search)

### Project 5 - Service health monitor

Check a list of services (APIs, databases, queues) concurrently. Every attempt has a timeout, and a failure is retried after a pause that doubles each time.
Each service ends up as **up**, **degraded** or **down**, and the results are reported as a text table and as JSON.
The package does not know how to check anything itself. A service is anything that implements `Checker`, which makes the package easy to test with fakes.

**Practises:** an enum with `String` (001/007), a `Config` struct with defaults (007), interfaces and an adapter struct (008), errors that wrap two errors (009), a generic `Retry` (010), `context` timeouts and cancellation, `select` and a worker pool (011), `fmt` widths, `MarshalText` for JSON, and `slog` (013), tests with fake services (014).

**Requirements**
1. A `Checker` has `Name()` and `Check(ctx) error`, and must return soon after `ctx` is done. `CheckFunc` turns a name and a function into a `Checker`.
2. Every zero field of `Config` gets a default: 4 workers, a 1 s timeout per attempt, 3 attempts, a 100 ms backoff, a 500 ms slow threshold, and no logging.
3. `Retry` runs `f` up to `attempts` times. Before the 2nd attempt it waits `backoff`, before the 3rd `2 × backoff`, and so on. If `ctx` is done during a wait, it stops at once and returns an error that wraps **both** the last error and `ctx.Err()`.
4. Classification:
   - **down** if every attempt failed
   - **degraded** if it succeeded but needed more than one attempt, or took longer than `SlowThreshold`
   - **up** otherwise
5. `Run` checks all services with `Workers` goroutines and returns the results in the order of the checkers. A cancelled `ctx` makes it finish quickly.
6. In JSON, a `Status` appears as `"up"`, `"degraded"` or `"down"`, and an empty error is left out. `WriteTable` writes aligned columns with no trailing spaces.
7. Every check is logged with `slog`: the service, status, attempts and latency.

**API**

```go
type Status int

const (
	Up Status = iota
	Degraded
	Down
)

func (s Status) String() string
func (s Status) MarshalText() ([]byte, error)

type Checker interface {
	Name() string
	Check(ctx context.Context) error
}

type CheckFunc struct {
	ServiceName string
	Fn          func(ctx context.Context) error
}

func (c CheckFunc) Name() string
func (c CheckFunc) Check(ctx context.Context) error

type Config struct {
	Workers       int
	Timeout       time.Duration
	Attempts      int
	Backoff       time.Duration
	SlowThreshold time.Duration
	Logger        *slog.Logger
}

func (c Config) withDefaults() Config // unexported, but the reference tests call it

type Result struct {
	Name     string        `json:"name"`
	Status   Status        `json:"status"`
	Attempts int           `json:"attempts"`
	Latency  time.Duration `json:"latency_ns"`
	Error    string        `json:"error,omitempty"`
}

func Retry[T any](ctx context.Context, attempts int, backoff time.Duration, f func(ctx context.Context) (T, error)) (T, int, error)
func Run(ctx context.Context, checkers []Checker, cfg Config) []Result

type Summary struct {
	Up, Degraded, Down int
}

func Summarize(results []Result) Summary
func (s Summary) Healthy() bool
func (s Summary) String() string // 4 services: 1 up, 1 degraded, 2 down
func WriteTable(w io.Writer, results []Result) error
func WriteJSON(w io.Writer, results []Result) error
```

**Example**

```
checkers := []monitor.Checker{
	monitor.CheckFunc{ServiceName: "api", Fn: pingAPI},
	monitor.CheckFunc{ServiceName: "database", Fn: pingDB},
}
results := monitor.Run(ctx, checkers, monitor.Config{Attempts: 2})
monitor.WriteTable(os.Stdout, results)
	api        up          1    3ms
	database   down        2     1s  context deadline exceeded
fmt.Println(monitor.Summarize(results))
	2 services: 1 up, 0 degraded, 1 down
```

**Tests to write**
- Fake checkers:
  - one that is always healthy
  - one that always fails
  - a *hanging* one that waits for `ctx.Done()`
  - a *slow* one that answers after a delay
  - a *flaky* one that fails N times, then works (a pointer type with a call counter)
- `Retry`:
  - succeeds on the 3rd attempt
  - gives up after all attempts
  - stops at once on a cancelled context, and `errors.Is` finds both `context.Canceled` and the last error
  - rejects `attempts < 1`
- `Run` classifies the services and keeps the input order:
  - healthy → up after 1 attempt
  - flaky (2 failures) → degraded after 3
  - failing → down after 3
  - hanging → down with "deadline exceeded"
- A slow service with a small `SlowThreshold` is degraded after 1 attempt.
- A cancelled `ctx` with a 1-hour timeout still returns within a second.
- The output:
  - the JSON contains `"status": "down"` and no empty `"error"`
  - the table has the exact column layout
  - the log contains `service=api status=up attempts=1` (write it with `slog.NewTextHandler` into a `bytes.Buffer`)
- Keep the tests fast (50 ms timeout, 1 ms backoff) and run them with `-race`.

**Hints**
- Give every **attempt** its own `context.WithTimeout(ctx, cfg.Timeout)`, with `defer cancel()` inside the function that makes the attempt.
- In `Retry`, wait with `select { case <-ctx.Done(): …; case <-time.After(backoff): }` rather than `time.Sleep`, so that cancellation is noticed during the pause.
- `fmt.Errorf("%w (retry stopped: %w)", err, ctx.Err())`: since Go 1.20 one error can wrap several.
- `slog.New(slog.DiscardHandler)` is a logger that throws everything away, a good default for a nil `Logger`.
- Implementing `MarshalText` is enough for `encoding/json` to write a type as a JSON string.

**Going further** (optional): a real HTTP checker with `net/http` and `http.NewRequestWithContext`; running the monitor every 30 s with a `time.Ticker` and printing only status changes; adding random jitter to the backoff.

**Solution:** [`solutions/monitor`](solutions/monitor)
