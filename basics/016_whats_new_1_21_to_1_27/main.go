// Command 016_whats_new_1_21_to_1_27 is module 016 of the Go course: what changed from Go 1.21
// to Go 1.27.
//
// Thanks to the compatibility promise nothing you learned stops working - but much of what you
// SHOULD write has changed since 1.20. One function per release; language changes matter most,
// library additions mostly replace things you used to hand-write. Every claim runs below.
//
//	go run ./basics/016_whats_new_1_21_to_1_27
package main

import (
	"cmp"
	"errors"
	"fmt"
	"iter"
	"log/slog"
	"maps"
	"math/rand/v2"
	"os"
	"reflect"
	"runtime"
	"slices"
	"strings"
	"sync"
	"time"
	"unique"
	"uuid"
	"weak"
)

func main() {
	go121()             // min/max/clear, slices/maps/cmp, slog, sync.OnceValue
	go122()             // per-iteration loop variables, range over int, math/rand/v2
	go123()             // range over func, iter, unique, timer fixes
	go124()             // generic aliases, os.Root, weak, b.Loop, omitzero, Swiss maps
	go125()             // WaitGroup.Go, synctest, reflect.TypeAssert, cgroup GOMAXPROCS
	go126()             // errors.AsType, reflect iterators, slog.NewMultiHandler
	go127()             // generic methods, promoted literal keys, json/v2, uuid, CutLast
	cumulativePicture() // the whole delta as "stop writing X, write Y"
}

// go121 shows Go 1.21: new builtins and the collection packages.
func go121() {
	fmt.Println("--- Go 1.21 ---")

	// Language: min, max (variadic, any ordered type, strings too) and clear are builtins.
	// Package initialisation order became precisely specified; inference got stronger (a
	// generic function assigned to a func-typed variable or passed as an argument, module 010).
	fmt.Printf("  min(3, 1, 2)=%d max(3, 1, 2)=%d min(\"pear\", \"apple\")=%q\n",
		min(3, 1, 2), max(3, 1, 2), min("pear", "apple"))
	nums := []int{1, 2, 3}
	m := map[string]int{"a": 1, "b": 2}
	clear(nums) // zeroes the elements, keeps the length
	clear(m)    // deletes every entry
	fmt.Printf("  clear: slice -> %v, map -> %v\n", nums, m)

	// slices, maps and cmp graduated from golang.org/x/exp (module 012).
	s := []int{3, 1, 2}
	slices.Sort(s) // no more sort.Slice and its index closure; cmp.Ordered replaces x/exp/constraints
	fmt.Println("  slices.Sort:", s)

	// sync.OnceFunc/OnceValue/OnceValues: lazy init with no sync.Once + package variable.
	config := sync.OnceValue(func() string { return "loaded once, lazily" })
	fmt.Printf("  sync.OnceValue: %q / %q\n", config(), config())

	// errors.ErrUnsupported: a standard sentinel to wrap (module 009).
	err := fmt.Errorf("compress on this platform: %w", errors.ErrUnsupported)
	fmt.Println("  errors.Is(err, errors.ErrUnsupported):", errors.Is(err, errors.ErrUnsupported))

	// Also: log/slog structured logging (module 013); context.WithoutCancel, AfterFunc,
	// WithDeadlineCause. Toolchain: PGO is GA (a default.pgo next to main gives 2-7% for free),
	// and the go.mod `toolchain` directive makes the go command fetch a newer Go itself.
}

// go122 shows Go 1.22: the loop-variable fix - the most consequential change in a decade.
func go122() {
	fmt.Println("\n--- Go 1.22 ---")

	// Loop variables are now PER ITERATION, so closures and goroutines see their own value.
	// Before, all shared one variable and saw the last value - the most common Go bug ever,
	// and the reason for `i := i` in so many loop bodies. Gated on the go.mod `go` line: a
	// module saying go 1.21 or older keeps the old behaviour under any toolchain.
	var closures []func() int
	for i := 0; i < 3; i++ { // the three-clause form - range over int never had the old rule
		closures = append(closures, func() int { return i })
	}
	got := make([]int, 0, 3)
	for _, c := range closures {
		got = append(got, c())
	}
	fmt.Println("  closures over the loop variable:", got, "(before 1.22: [3 3 3])")

	// range over an integer replaces most three-clause loops.
	fmt.Print("  for i := range 5:")
	for i := range 5 {
		fmt.Print(" ", i)
	}
	fmt.Print(" | for range 3:")
	for range 3 {
		fmt.Print(" *")
	}
	fmt.Println()

	// math/rand/v2, the first v2 package in the stdlib: PCG and ChaCha8, no global lock,
	// Seed and Read removed, generic rand.N. (r.N below is the (*Rand).N METHOD from 1.27.)
	r := rand.New(rand.NewPCG(42, 1024)) // seeded, so the output is reproducible
	fmt.Printf("  math/rand/v2: IntN(100)=%d Float64()=%.4f N(int64(1000))=%d\n", r.IntN(100), r.Float64(), r.N(int64(1000)))

	fmt.Printf("  slices.Concat: %v, cmp.Or(\"\", \"fallback\"): %q\n",
		slices.Concat([]int{1}, []int{2, 3}), cmp.Or("", "fallback"))

	// net/http patterns with methods and wildcards - mux.HandleFunc("GET /items/{id}", h) and
	// r.PathValue("id") - removed the main reason for a third-party router (module 018).
}

// go123 shows Go 1.23: iterators.
func go123() {
	fmt.Println("\n--- Go 1.23 ---")

	// range over a FUNCTION func(yield func(V) bool): any sequence - a tree walk, a file's
	// lines, a cursor, an infinite series - works with a plain for loop. Stop writing
	// ForEach(callback) or a goroutine + channel just to produce values (module 012).
	fmt.Print("  range over count(5):")
	for v := range count(5) {
		fmt.Print(" ", v)
	}
	fmt.Println()

	// iter.Seq/Seq2/Pull, and iterator functions everywhere: slices.All, Values, Backward,
	// Collect, Sorted, SortedFunc, Chunk, Repeat; maps.Keys, Values, All, Collect, Insert.
	s := []string{"c", "a", "b"}
	fmt.Println("  slices.Sorted(slices.Values(s)):", slices.Sorted(slices.Values(s)))
	fmt.Print("  slices.Backward(s):")
	for _, v := range slices.Backward(s) {
		fmt.Print(" ", v)
	}
	fmt.Println()
	// maps.Keys returns an ITERATOR - unlike the x/exp version it replaced, which caught many.
	m := map[string]int{"b": 2, "a": 1}
	fmt.Printf("  maps.Keys is a %T; wrap it: %v\n", maps.Keys(m), slices.Sorted(maps.Keys(m)))

	// unique.Make interns: equal values share one allocation and compare by pointer.
	h1, h2 := unique.Make("a repeated string value"), unique.Make("a repeated string value")
	fmt.Println("  unique.Make: h1 == h2 is", h1 == h2)

	// Timers: an unreferenced, unfired Timer/Ticker is now collected and its channel is
	// unbuffered - fixing the time.After leak in hot loops (asynctimerchan=1 reverted it,
	// a knob Go 1.27 removed). Toolchain: opt-in telemetry, `go telemetry on`.
}

func count(n int) iter.Seq[int] {
	return func(yield func(int) bool) {
		for i := range n {
			if !yield(i) {
				return
			}
		}
	}
}

// go124 shows Go 1.24: generic aliases, os.Root, weak pointers and friends.
func go124() {
	fmt.Println("\n--- Go 1.24 ---")

	// Language: an alias may take type parameters (module 010).
	tags := set[string]{"a": {}, "b": {}}
	fmt.Println("  generic alias type set[T comparable] = map[T]struct{}:", len(tags), "entries")

	// os.Root confines every operation to a directory; traversal is refused by the OS, not by
	// string checks. Use it for any path from outside - instead of filepath.Clean + prefix.
	dir, _ := os.MkdirTemp("", "whatsnew-*")
	defer os.RemoveAll(dir)
	_ = os.WriteFile(dir+"/ok.txt", []byte("safe"), 0o600)
	if root, err := os.OpenRoot(dir); err == nil {
		defer root.Close()
		_, inErr := root.Open("ok.txt")
		_, outErr := root.Open("../../../etc/passwd")
		fmt.Printf("  os.Root: inside=%v escape=%v\n", inErr, outErr)
	}

	// weak.Pointer[T]: a reference the GC may clear once nothing else holds the value - for
	// caches that must not retain their contents (unique uses it internally).
	value := &payload{Name: "collectable"}
	wp := weak.Make(value)
	fmt.Println("  weak.Pointer: Value() != nil while still reachable:", wp.Value() != nil)
	runtime.KeepAlive(value)

	// strings.SplitSeq, FieldsSeq, Lines...: iterators, no slice allocated.
	fmt.Print("  strings.SplitSeq:")
	for part := range strings.SplitSeq("a,b,c", ",") {
		fmt.Printf(" %q", part)
	}
	fmt.Println()

	// Also: `for b.Loop()` replaces `for range b.N` - setup is excluded from timing and the
	// compiler can no longer delete the body (module 014); json:",omitzero" omits a zero
	// struct or time.Time, which omitempty never did (module 007); crypto/mlkem;
	// runtime.AddCleanup, a better SetFinalizer. Runtime: Swiss-table maps, up to ~30% faster
	// on some operations, less memory, no API change. Toolchain: the go.mod `tool` directive
	// (go get -tool, go tool <name>) replaces the tools.go build-tag hack.
}

type set[T comparable] = map[T]struct{}

type payload struct{ Name string }

// go125 shows Go 1.25: WaitGroup.Go, typed reflection, container-aware GOMAXPROCS.
func go125() {
	fmt.Println("\n--- Go 1.25 ---")

	// WaitGroup.Go(f) = Add(1) + go + defer Done(). The "Add inside the goroutine" race can no
	// longer be written.
	var wg sync.WaitGroup
	results := make([]int, 4)
	for i := range 4 {
		wg.Go(func() { results[i] = i * i })
	}
	wg.Wait()
	fmt.Println("  sync.WaitGroup.Go:", results)

	// reflect.TypeAssert[T]: a typed, allocation-free assertion from a reflect.Value.
	v := reflect.ValueOf(any("typed assertion"))
	if s, ok := reflect.TypeAssert[string](v); ok {
		fmt.Printf("  reflect.TypeAssert[string]: %q\n", s)
	}
	_, ok := reflect.TypeAssert[int](v)
	fmt.Println("  reflect.TypeAssert[int]: ok =", ok)

	// GOMAXPROCS now honours the cgroup CPU limit and follows it when it changes - a pod
	// limited to 2 CPUs on a 64-core node no longer runs 64 Ps. Stop tuning it by hand.
	fmt.Printf("  GOMAXPROCS=%d on a %d-CPU machine\n", runtime.GOMAXPROCS(0), runtime.NumCPU())

	// testing/synctest: a fake clock and deterministic scheduling - an hour-long timeout test
	// runs in microseconds and cannot flake; no more time.Sleep in tests (module 014).
	// Also: os.Root gained ReadFile, WriteFile, MkdirAll, RemoveAll, Rename, Symlink, Chmod...;
	// slog.GroupAttrs; trace.FlightRecorder; http.CrossOriginProtection; T.Attr and T.Output.
	// The core-type notion left the spec (no code change). The Green Tea GC arrived as
	// GOEXPERIMENT=greenteagc (10-40% less GC overhead on many small objects), default in 1.26.
	// `go doc -http` serves documentation locally.
}

// go126 shows Go 1.26: errors.AsType, reflect iterators, slog.NewMultiHandler.
func go126() {
	fmt.Println("\n--- Go 1.26 ---")

	// errors.AsType[T]: no pre-declared target, no & - instead of var t *E; errors.As(err, &t).
	wrapped := fmt.Errorf("outer: %w", &parseError{Line: 7, Column: 2})
	if pe, ok := errors.AsType[*parseError](wrapped); ok {
		fmt.Printf("  errors.AsType[*parseError]: line=%d column=%d\n", pe.Line, pe.Column)
	}

	// reflect iterators - Type.Fields, Methods, Ins, Outs; Value.Fields, Methods - replace
	// `for i := 0; i < t.NumField(); i++`.
	fmt.Print("  reflect Type.Fields():")
	for f := range reflect.TypeFor[point]().Fields() {
		fmt.Print(" ", f.Name)
	}
	fmt.Print(" | Type.Methods():")
	for meth := range reflect.TypeFor[*account]().Methods() {
		fmt.Print(" ", meth.Name)
	}
	fmt.Println()

	// slog.NewMultiHandler fans one record out to several handlers (text to the console,
	// JSON to a file) with no third-party package.
	var buf strings.Builder
	noTime := func(_ []string, a slog.Attr) slog.Attr {
		if a.Key == slog.TimeKey {
			return slog.Attr{}
		}
		return a
	}
	logger := slog.New(slog.NewMultiHandler(slog.NewTextHandler(&buf, &slog.HandlerOptions{ReplaceAttr: noTime})))
	logger.Info("one record", slog.String("to", "many handlers"))
	fmt.Print("  slog.NewMultiHandler: ", buf.String())

	// Also: bytes.Buffer.Peek (module 013); testing.TB.ArtifactDir() for files CI collects;
	// context-aware net.Dialer.DialTCP/DialUDP/...; http.ClientConn; netip.Prefix.Compare;
	// crypto/hpke; GODEBUG=tracebacklabels (default flipped in 1.27).
}

type parseError struct{ Line, Column int }

func (e *parseError) Error() string { return fmt.Sprintf("parse error at %d:%d", e.Line, e.Column) }

type point struct {
	X, Y int
	name string
}

type account struct{ Balance int }

func (a *account) Deposit(n int)  { a.Balance += n }
func (a *account) Withdraw(n int) { a.Balance -= n }

// go127 shows Go 1.27: the biggest language change since generics themselves.
func go127() {
	fmt.Println("\n--- Go 1.27 ---")

	// Generic METHODS: a method may declare its own type parameters next to the receiver's
	// (module 010). Before, the workaround was slices.SortFunc(s, f) instead of s.SortFunc(f).
	// A generic method still cannot satisfy an interface (one signature vs infinitely many).
	// With `go 1.26` in go.mod:
	//	func (l list[E]) Apply[F any](...) // ERROR: generic method requires go1.27 or later (-lang was set to go1.26; check go.mod)
	l := list[int]{1, 2, 3}
	labels := l.Apply(func(n int) string { return fmt.Sprintf("<%d>", n) })
	fmt.Printf("  generic method list[int].Apply -> %v (%T)\n", labels, labels)
	r := rand.New(rand.NewPCG(7, 7)) // the stdlib adopted it at once: (*Rand).N
	fmt.Printf("  (*rand.Rand).N: r.N(100)=%d r.N(int64(100))=%d\n", r.N(100), r.N(int64(100)))

	// The spec now says function type inference applies in ALL assignment contexts - struct
	// fields, map values, returns... (the compiler already accepted these with -lang=go1.21).
	table := map[string]func(int, int) int{"min": minOf}
	type holder struct{ Pick func(string, string) string }
	h := holder{Pick: minOf}
	fmt.Printf("  inferred into a map value: %d, a struct field: %q\n", table["min"](4, 9), h.Pick("b", "a"))

	// A struct-literal key may be any field selector, including a PROMOTED field. With
	// `go 1.26` in go.mod:
	//	line{Name: "diagonal"} // ERROR: use of promoted field object.Name in struct literal of type line requires go1.27 or later (-lang was set to go1.26; check go.mod)
	ln := line{Name: "diagonal", Length: 5} // before: line{object: object{Name: "diagonal"}, Length: 5}
	fmt.Printf("  promoted-field literal key: %+v\n", ln)

	// strings.CutLast / bytes.CutLast split at the LAST separator - no LastIndex + two slices.
	dir, base, _ := strings.CutLast("/usr/local/bin/go", "/")
	stem, ext, _ := strings.CutLast("archive.tar.gz", ".")
	fmt.Printf("  strings.CutLast: (%q, %q) and (%q, %q)\n", dir, base, stem, ext)

	// uuid: RFC 9562 in the stdlib - New, NewV4, NewV7, Parse, MustParse, Compare. V7 is
	// time-ordered: use it for database keys instead of github.com/google/uuid v4.
	fmt.Println("  uuid.NewV7():", uuid.NewV7())

	// encoding/json/v2 + encoding/json/jsontext are stable: syntax (jsontext) split from
	// semantics, options as arguments, case-SENSITIVE matching, duplicate names rejected, real
	// streaming, faster, and no guessing (a raw time.Duration is an error). encoding/json is
	// unchanged and not deprecated (module 013).
	// Also: maphash.Hasher[T]/ComparableHasher[T]; synctest.Sleep; httptest.NewTestServer(t, h)
	// registers its own cleanup; url.URL.Clone, Values.Clone; big.Int.Divide with a rounding
	// mode; database/sql column scanning; Unicode 17; crypto/mldsa. Removed GODEBUGs:
	// gotypesalias, tlsrsakex, asynctimerchan, ... - the escape hatches are temporary.
}

type list[E any] []E

func (l list[E]) Apply[F any](f func(E) F) list[F] {
	out := make(list[F], len(l))
	for i, v := range l {
		out[i] = f(v)
	}
	return out
}

func minOf[T cmp.Ordered](a, b T) T {
	if a < b {
		return a
	}
	return b
}

type object struct {
	Name  string
	Color string
}

type line struct {
	object // Name and Color are promoted
	Length float64
}

// cumulativePicture prints the whole delta since 1.20 as a set of habits.
func cumulativePicture() {
	fmt.Println("\n--- the cumulative picture ---")

	changes := []struct{ old, new, since string }{
		{"your own Min/Max helper", "min, max (builtins)", "1.21"},
		{"sort.Slice", "slices.Sort / slices.SortFunc", "1.21"},
		{"x/exp/slices, x/exp/maps", "slices, maps", "1.21"},
		{"x/exp/constraints.Ordered", "cmp.Ordered", "1.21"},
		{"sync.Once + a package var", "sync.OnceValue", "1.21"},
		{"i := i at the top of a loop", "nothing - it was fixed", "1.22"},
		{"for i := 0; i < n; i++", "for i := range n", "1.22"},
		{"math/rand", "math/rand/v2", "1.22"},
		{"a callback-based ForEach", "an iter.Seq", "1.23"},
		{"filepath.Clean + prefix check", "os.Root", "1.24"},
		{"for range b.N", "for b.Loop()", "1.24"},
		{"a tools.go file", "the `tool` directive", "1.24"},
		{"time.Sleep in a test", "testing/synctest", "1.25"},
		{"wg.Add(1); go func(){...}()", "wg.Go(f)", "1.25"},
		{"var t *E; errors.As(err, &t)", "errors.AsType[*E](err)", "1.26"},
		{"for i := 0; i < t.NumField()", "range t.Fields()", "1.26"},
		{"LastIndex + two slices", "strings.CutLast", "1.27"},
		{"github.com/google/uuid", "uuid", "1.27"},
	}
	fmt.Printf("  %-31s -> %-30s %s\n", "STOP WRITING", "USE INSTEAD", "SINCE")
	for _, c := range changes {
		fmt.Printf("  %-31s -> %-30s %s\n", c.old, c.new, c.since)
	}

	// Start considering: generic methods where a transformation belongs on the type (1.27),
	// json/v2 for speed, strictness or streaming (1.27), uuid.NewV7 for keys (1.27), slog
	// instead of log for collected output (1.21), iterators for non-slice sequences (1.23),
	// unique for many equal copies (1.23), PGO (1.21).

	// Go got larger, but idiomatic Go got SHORTER. Two changes needed no code at all: the
	// loop-variable fix (1.22) made existing code correct; Swiss maps (1.24) and Green Tea GC
	// (default 1.26) made it faster.
	fmt.Printf("\n  compiled and run by %s at %s\n", runtime.Version(), time.Now().Format(time.RFC3339))
}
