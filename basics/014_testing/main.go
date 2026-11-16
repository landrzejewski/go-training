// Command 014_testing is module 014 of the Go course: testing with go test.
//
// go test compiles every _test.go file into the package and runs every TestXxx: no runner, no
// annotations and, by design, no assertion library - an if plus a t.Errorf saying what you got
// and wanted reads better. The tests are in main_test.go; this program shows units and commands.
//
//	go run ./basics/014_testing
//	go test -v ./basics/014_testing
package main

import (
	"errors"
	"fmt"
	"io"
	"strings"
)

func main() {
	testBasics()         // _test.go files, the four function shapes, Error vs Fatal
	tableDriven()        // table-driven tests, subtests, t.Parallel
	benchmarks()         // b.Loop, -benchmem, benchstat
	fuzzing()            // properties, the seed corpus, testdata/fuzz
	examples()           // Example functions verified against // Output:
	synctestBubbles()    // testing/synctest: a fake clock, Wait and Sleep
	coverageAndDoubles() // coverage, test doubles without a framework, useful flags
}

// testBasics shows what go test looks for (see TestReverse, TestHelperAndCleanup, TestSkipping).
func testBasics() {
	fmt.Println("--- the basics of go test ---")

	// A file ending in _test.go is compiled only by go test, never into the shipped binary.
	// Four function shapes are recognised; the name after the prefix must start upper-case
	// (Testfoo is NOT a test):
	//	func TestXxx(t *testing.T)       a test
	//	func BenchmarkXxx(b *testing.B)  a benchmark
	//	func FuzzXxx(f *testing.F)       a fuzz target
	//	func ExampleXxx()                a documented, verified example
	//
	// Reporting:
	//	t.Error/Errorf  mark failed, keep going
	//	t.Fatal/Fatalf  mark failed, stop this test now (runtime.Goexit - never from another goroutine)
	//	t.Log/Logf      shown only with -v, or on failure
	//	t.Skip/Skipf    skip, usually behind testing.Short()
	//
	// TestMain(m *testing.M) takes over the package's whole run for one-time setup and teardown;
	// it must call m.Run() (since Go 1.15 simply returning is enough - no os.Exit needed).
	fmt.Printf("reverse(%q) = %q, isEven(4) = %t\n", "Gęś", reverse("Gęś"), isEven(4))
	kv, _ := parseKV("a=1, b=2")
	fmt.Printf("parseKV(\"a=1, b=2\") = %v\n", kv)
	fmt.Println("run: go test -v ./basics/014_testing")
}

// reverse is the unit under test for most of this module.
func reverse(s string) string {
	r := []rune(s)
	for i, j := 0, len(r)-1; i < j; i, j = i+1, j-1 {
		r[i], r[j] = r[j], r[i]
	}
	return string(r)
}

// isEven mirrors examples/condition.go, this repo's original testing example.
func isEven(n int) bool { return n%2 == 0 }

var errEmpty = errors.New("empty input")

// parseKV parses "k=v, k=v" and is the unit for the table-driven and fuzz tests.
func parseKV(s string) (map[string]string, error) {
	if strings.TrimSpace(s) == "" {
		return nil, errEmpty
	}
	out := make(map[string]string)
	for pair := range strings.SplitSeq(s, ",") {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		k, v, found := strings.Cut(pair, "=")
		if !found {
			return nil, fmt.Errorf("parseKV: %q is not a key=value pair", pair)
		}
		out[strings.TrimSpace(k)] = strings.TrimSpace(v)
	}
	return out, nil
}

// tableDriven points at the dominant Go testing idiom (see TestTableDriven*, TestParallelSubtests).
func tableDriven() {
	fmt.Println("\n--- table-driven tests and subtests ---")

	// A slice of anonymous structs holds the cases; one loop runs each in its own t.Run
	// subtest, which gets a name in the output, independent pass/fail, -run filtering by path,
	// and its own scope for t.Parallel and t.Cleanup. Spaces in names become _ and duplicates get
	// #01, so keep names short and identifier-like. Keep cases independent of each other.
	fmt.Println("run one case: go test -run 'TestTableDriven/utf8' ./basics/014_testing")

	// -race is close to mandatory for anything concurrent; -count=2 on top also catches state
	// leaking from one run into the next.
	fmt.Println("run with the race detector: go test -race -count=2 ./basics/014_testing")
}

// benchmarks shows how to run benchmarks (see BenchmarkReverse and friends).
func benchmarks() {
	fmt.Println("\n--- benchmarks ---")

	// go test re-runs a benchmark with a growing iteration count until the timing is stable,
	// then reports ns/op. -benchmem adds B/op and allocs/op - usually more actionable than time,
	// because allocations are what you can actually reduce.
	fmt.Println("go test -run XXX -bench . -benchmem ./basics/014_testing")

	// A single run is noise: run -count=10 and compare two runs with benchstat
	// (golang.org/x/perf/cmd/benchstat), which says whether a difference is significant.
	fmt.Println("go test -run XXX -bench . -count=10 ./basics/014_testing > new.txt; benchstat old.txt new.txt")

	// Classic mistakes: timing the setup, letting the compiler delete the work (b.Loop fixes
	// both) and measuring on a laptop whose thermal governor throttles the CPU.
}

// fuzzing shows the property FuzzReverse checks (see FuzzReverse and FuzzParseKV).
func fuzzing() {
	fmt.Println("\n--- fuzzing ---")

	// Fuzzing (Go 1.18) checks a PROPERTY, not an expected value:
	//	round trip    Decode(Encode(x)) == x  <- reverse(reverse(s)) == s below
	//	differential  the fast version agrees with an obviously-correct slow one
	//	invariant     the output is always sorted / always valid UTF-8
	//	no panic      the weakest property, and it still finds plenty of bugs
	for _, s := range []string{"", "abc", "Gęś", "🙂🙃"} {
		twice := reverse(reverse(s))
		fmt.Printf("reverse twice %-8q -> %-8q holds=%t\n", s, twice, twice == s)
	}

	// Plain go test runs only the seed corpus, as ordinary test cases. -fuzz generates new
	// inputs until you stop it or it finds a failure; only one target at a time, and not with a
	// multi-package pattern. A failing input is saved under testdata/fuzz/FuzzXxx/ and replayed
	// by every later go test - COMMIT it, it is a regression test the fuzzer wrote for you.
	fmt.Println("go test -run XXX -fuzz FuzzReverse -fuzztime 10s ./basics/014_testing")
}

// examples shows how Example functions are run (see Example_reverse and Example_parseKV).
func examples() {
	fmt.Println("\n--- examples as documentation ---")

	// An Example is documentation AND a test: go doc and pkg.go.dev show it next to the
	// identifier it names, and go test runs it and compares stdout with its // Output: comment,
	// so the documentation cannot go stale.
	fmt.Println(reverse("hello"), "<- what Example_reverse prints and verifies")
	fmt.Println("go test -v -run Example ./basics/014_testing")
}

// synctestBubbles introduces testing/synctest (Go 1.25; see the TestSynctest* functions).
func synctestBubbles() {
	fmt.Println("\n--- testing/synctest ---")

	// Concurrent tests that sleep are slow, and tests that depend on scheduling are flaky.
	// synctest.Test runs a function in a BUBBLE with a fake clock that jumps forward instantly
	// whenever every goroutine in the bubble is blocked - a one-hour timeout test takes
	// microseconds and measures exactly one hour. It replaces every
	//	time.Sleep(100 * time.Millisecond) // give it time to finish
	// the biggest single source of flaky Go tests.
	//
	// Caveats: only goroutines started INSIDE the bubble belong to it, and real I/O (network,
	// files) blocks for real. Use it for concurrent logic, not for integration tests.
	fmt.Println("go test -v -run Synctest ./basics/014_testing")
}

// coverageAndDoubles shows coverage and test doubles (see TestWithATestDouble, TestReaderDouble,
// TestHTTPTestServer).
func coverageAndDoubles() {
	fmt.Println("\n--- coverage, test doubles and the toolchain ---")

	// Coverage is a diagnostic, not a target: it shows what is definitely untested, nothing about
	// whether covered lines are tested well. Chasing 100% produces tests that assert nothing.
	// (-covermode=atomic is selected automatically under -race.)
	fmt.Println("go test -cover ./basics/014_testing")
	fmt.Println("go test -coverprofile=cover.out ./basics/014_testing && go tool cover -html=cover.out")

	// Go needs no mocking framework, because interfaces are satisfied implicitly (module 008).
	// Take io.Reader instead of *os.File and strings.NewReader is the fake.
	fmt.Println("countLines over a string, no file needed:", countLines(strings.NewReader("a\nb\nc")))

	// Take a small interface defined where it is used, and a test writes a struct with the one
	// method it needs (fakeStore in main_test.go). Embedding the interface in a fake lets it
	// override only the methods a test exercises (module 008).
	fmt.Println(greet(mapStore{"user": "Ada"}, "user"), greet(mapStore{}, "user"))

	// Also worth knowing:
	//	net/http/httptest   a real test server and a ResponseRecorder
	//	testing/fstest      MapFS, an in-memory fs.FS
	//	go-cmp              github.com/google/go-cmp/cmp: a readable diff instead of DeepEqual's
	//	                    bare false - the one testing dependency worth taking
	//	t.ArtifactDir()     Go 1.26: a directory for files (logs, profiles) CI should collect
	//	go test -json       machine-readable output
	//	-shuffle=on         randomise test order to catch order dependencies
	//	-count=1            bypass the test cache
}

// countLines takes the smallest interface it needs, which is what makes it testable.
func countLines(r io.Reader) int {
	data, err := io.ReadAll(r)
	if err != nil || len(data) == 0 {
		return 0
	}
	return strings.Count(string(data), "\n") + 1
}

var errNotFound = errors.New("not found")

// store is the small, consumer-side interface greet depends on.
type store interface {
	Get(key string) (string, error)
}

// greet is the unit under test for the test double: it depends on store, not on a database.
func greet(s store, key string) string {
	name, err := s.Get(key)
	if err != nil {
		return "Hello, stranger!"
	}
	return "Hello, " + name + "!"
}

// mapStore is the "production" store; the tests use their own fake.
type mapStore map[string]string

func (m mapStore) Get(key string) (string, error) {
	if v, ok := m[key]; ok {
		return v, nil
	}
	return "", fmt.Errorf("mapStore: %q: %w", key, errNotFound)
}
