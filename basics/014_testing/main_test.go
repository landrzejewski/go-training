package main

// The runnable half of module 014: every technique main.go describes is exercised here.
//
//	go test -v ./basics/014_testing
//	go test -run 'TestTableDriven/utf8' ./basics/014_testing
//	go test -bench . -benchmem ./basics/014_testing
//	go test -race ./basics/014_testing
//	go test -fuzz FuzzReverse ./basics/014_testing
//	go test -cover ./basics/014_testing

import (
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"
	"unicode/utf8"
)

// TestReverse is the simplest possible test.
func TestReverse(t *testing.T) {
	got := reverse("hello")
	want := "olleh"
	if got != want {
		// The conventional message names the call, then what we got, then what we wanted.
		t.Errorf("reverse(%q) = %q, want %q", "hello", got, want)
	}
}

// assertEqual is an assertion helper.
func assertEqual[T comparable](t *testing.T, got, want T, what string) {
	t.Helper() // always first: failures are then reported at the CALLER's line, not here
	if got != want {
		t.Errorf("%s = %v, want %v", what, got, want)
	}
}

// TestHelperAndCleanup shows t.Helper, t.Cleanup, t.TempDir and t.Setenv.
func TestHelperAndCleanup(t *testing.T) {
	// t.Cleanup runs after the test and all its subtests finish, in LIFO order. Prefer it to
	// defer: it works when registered from a helper, and it still runs after t.Fatal.
	// This one is registered FIRST, so it runs LAST and sees both appends.
	var order []string
	t.Cleanup(func() {
		want := []string{"registered last, runs first", "registered first, runs last"}
		if !slices.Equal(order, want) {
			t.Errorf("cleanup order = %v, want %v", order, want)
		}
	})
	t.Cleanup(func() { order = append(order, "registered first, runs last") })
	t.Cleanup(func() { order = append(order, "registered last, runs first") })

	assertEqual(t, reverse("ab"), "ba", `reverse("ab")`)
	assertEqual(t, isEven(4), true, "isEven(4)")

	// t.TempDir is removed automatically when the test ends.
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "scratch.txt"), []byte("x"), 0o600); err != nil {
		t.Fatalf("WriteFile in t.TempDir(): %v", err)
	}

	// t.Setenv restores the previous value afterwards - and deliberately makes the test
	// non-parallel, since the environment is process-wide.
	t.Setenv("COURSE_EXAMPLE", "set for this test only")
	assertEqual(t, os.Getenv("COURSE_EXAMPLE"), "set for this test only", "os.Getenv")
}

// TestSkipping shows the conventional guard for a slow test.
func TestSkipping(t *testing.T) {
	if testing.Short() {
		t.Skip("skipped under -short")
	}
	t.Log("this line is shown only with -v, or when the test fails")
}

// TestTableDriven is the dominant Go idiom: a table of cases and one loop of subtests.
func TestTableDriven(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", ""},
		{"single", "a", "a"},
		{"ascii", "hello", "olleh"},
		{"utf8", "Gęś", "śęG"},
		{"emoji", "ab🙂", "🙂ba"},
		{"palindrome", "kajak", "kajak"},
	}
	for _, tt := range tests {
		// t.Run makes a subtest with its own *testing.T: -run 'TestTableDriven/utf8' selects it.
		t.Run(tt.name, func(t *testing.T) {
			if got := reverse(tt.in); got != tt.want {
				t.Errorf("reverse(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// TestTableDrivenWithErrors adds expected errors to the table.
func TestTableDrivenWithErrors(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    map[string]string
		wantErr bool  // expect some error
		errIs   error // if non-nil, the error must match this sentinel
	}{
		{"single", "a=1", map[string]string{"a": "1"}, false, nil},
		{"several", "a=1, b=2", map[string]string{"a": "1", "b": "2"}, false, nil},
		{"spaces trimmed", " a = 1 ", map[string]string{"a": "1"}, false, nil}, // runs as spaces_trimmed
		{"empty", "  ", nil, true, errEmpty},
		{"no equals", "a=1,oops", nil, true, nil}, // any error is fine here
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseKV(tt.in)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parseKV(%q) succeeded, want an error", tt.in)
				}
				// Compare errors with errors.Is, never with == or by message (module 009).
				if tt.errIs != nil && !errors.Is(err, tt.errIs) {
					t.Fatalf("parseKV(%q) error = %v, want %v", tt.in, err, tt.errIs)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseKV(%q) unexpected error: %v", tt.in, err)
			}
			// maps.Equal beats reflect.DeepEqual: typed, faster and clearer.
			if !maps.Equal(got, tt.want) {
				t.Errorf("parseKV(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

// TestParallelSubtests shows t.Parallel and why the parent uses t.Cleanup, not defer.
func TestParallelSubtests(t *testing.T) {
	// Parallel subtests run only AFTER the parent's body has returned, so a defer here would
	// run before them. A cleanup registered on the parent waits for every child.
	t.Cleanup(func() { t.Log("parent cleanup runs after every parallel child has finished") })

	for _, in := range []string{"alpha", "beta", "gamma"} {
		// Since Go 1.22 the loop variable is per-iteration, so the old `in := in` is not needed.
		// In pre-1.22 code its absence was a real bug: every parallel subtest saw the last case.
		t.Run(in, func(t *testing.T) {
			t.Parallel() // may run concurrently with the other parallel subtests of this parent
			if got := reverse(reverse(in)); got != in {
				t.Errorf("round trip of %q = %q", in, got)
			}
		})
	}
}

// BenchmarkReverse shows the b.Loop form (Go 1.24) - the one to use.
func BenchmarkReverse(b *testing.B) {
	// Compared with the old `for range b.N`: setup before the loop is excluded from the timing
	// automatically (no b.ResetTimer), and the compiler cannot optimise the body away - a
	// benchmark that silently measured nothing was a classic b.N mistake.
	input := strings.Repeat("gopher ", 20)
	for b.Loop() {
		reverse(input)
	}
}

// BenchmarkReverseBySize uses sub-benchmarks (b.Run) to sweep the input size.
func BenchmarkReverseBySize(b *testing.B) {
	for _, size := range []int{8, 128, 2048} {
		b.Run(fmt.Sprintf("size=%d", size), func(b *testing.B) {
			input := strings.Repeat("a", size)
			b.ReportAllocs() // like -benchmem, for this benchmark only; b.ReportMetric adds custom units
			for b.Loop() {
				reverse(input)
			}
		})
	}
}

func BenchmarkParseKV(b *testing.B) {
	input := "host=localhost, port=8080, tls=true, timeout=30s"
	b.ReportAllocs()
	for b.Loop() {
		if _, err := parseKV(input); err != nil {
			b.Fatal(err)
		}
	}
}

// The two ways of building a string, so -benchmem shows why Builder wins (module 002a).
func BenchmarkConcatPlus(b *testing.B) {
	for b.Loop() {
		s := ""
		for i := range 100 {
			s += string(rune('a' + i%26))
		}
		_ = s
	}
}

func BenchmarkConcatBuilder(b *testing.B) {
	for b.Loop() {
		var sb strings.Builder
		for i := range 100 {
			sb.WriteByte(byte('a' + i%26))
		}
		_ = sb.String()
	}
}

// FuzzReverse checks a PROPERTY - reversing twice is the identity - rather than an expected value.
func FuzzReverse(f *testing.F) {
	// f.Add supplies the seed corpus - all that plain go test runs.
	for _, seed := range []string{"", "a", "hello", "Gęś", "🙂🙃", "kajak"} {
		f.Add(seed)
	}
	// The parameters after *testing.T are the generated inputs. Supported types: the numeric
	// types, bool, string, []byte and rune.
	f.Fuzz(func(t *testing.T, in string) {
		once := reverse(in)
		twice := reverse(once)
		// The round trip holds only for valid UTF-8: []rune(s) turns every invalid byte into
		// U+FFFD. The fuzzer finds such inputs within seconds.
		if !utf8.ValidString(in) {
			t.Skip("invalid UTF-8 input: the []rune conversion is lossy by design")
		}
		if twice != in {
			t.Errorf("reverse(reverse(%q)) = %q, want %q", in, twice, in)
		}
		// A second invariant: reversing preserves the rune count.
		if a, b := utf8.RuneCountInString(in), utf8.RuneCountInString(once); a != b {
			t.Errorf("rune count changed: %d -> %d", a, b)
		}
	})
}

// FuzzParseKV checks the weakest useful property: it must never panic.
func FuzzParseKV(f *testing.F) {
	f.Add("a=1")
	f.Add("a=1,b=2")
	f.Add("=")
	f.Add(",,,")
	f.Fuzz(func(t *testing.T, in string) {
		got, err := parseKV(in) // may return an error, must never panic
		if err == nil && got == nil {
			t.Errorf("parseKV(%q) returned a nil map and a nil error", in)
		}
	})
}

// The name decides where go doc shows an example: Example, ExampleFoo, ExampleT, ExampleT_M,
// plus a lower-case suffix for a variant (ExampleFoo_withOptions). go vet checks that the name
// refers to a real identifier, so for an UNEXPORTED unit use the package-level form
// Example_suffix - as these do. Without an // Output: comment an example is compiled but NOT
// run: handy when it needs the network, and easy to do by accident.
func Example_reverse() {
	fmt.Println(reverse("hello"))
	fmt.Println(reverse("Gęś"))
	// Output:
	// olleh
	// śęG
}

func Example_parseKV() {
	kv, err := parseKV("host=localhost, port=8080")
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(kv["host"], kv["port"])
	// Output: localhost 8080
}

// Unordered output compares the lines as a SET - which is what map iteration requires.
func Example_parseKVUnordered() {
	kv, _ := parseKV("a=1, b=2, c=3")
	for k, v := range kv {
		fmt.Printf("%s=%s\n", k, v)
	}
	// Unordered output:
	// b=2
	// a=1
	// c=3
}

// TestSynctestFakeClock sleeps for an hour, instantly.
func TestSynctestFakeClock(t *testing.T) {
	realStart := time.Now()
	// Inside the bubble time.Now, Sleep, After, timers and tickers use a fake clock that starts
	// at a fixed instant and advances instantly whenever every bubbled goroutine is blocked.
	synctest.Test(t, func(t *testing.T) {
		fakeStart := time.Now()
		done := make(chan struct{})
		go func() {
			time.Sleep(time.Hour)
			close(done)
		}()
		<-done
		if elapsed := time.Since(fakeStart); elapsed != time.Hour {
			t.Errorf("fake elapsed = %v, want exactly 1h", elapsed)
		}
	})
	if realElapsed := time.Since(realStart); realElapsed > time.Second {
		t.Errorf("the real test took %v; it should be instant", realElapsed)
	}
}

// TestSynctestWait replaces the `time.Sleep(10ms) // let the goroutine catch up` anti-pattern.
func TestSynctestWait(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		results := make([]int, 3)
		for i := range 3 {
			go func() { results[i] = i * 10 }() // distinct indexes, so no race
		}
		// Wait blocks until every OTHER goroutine in the bubble has exited or is durably
		// blocked - no sleeping, no polling, no flakiness.
		synctest.Wait()
		if want := []int{0, 10, 20}; !slices.Equal(results, want) {
			t.Errorf("results = %v, want %v", results, want)
		}
	})
}

// TestSynctestSleepAdvancesTheClock shows the distinction that trips people up: Wait returns as
// soon as the others are durably blocked - and a sleeping goroutine IS durably blocked - so Wait
// does not move the clock. synctest.Sleep (Go 1.27) advances the clock and then waits.
func TestSynctestSleepAdvancesTheClock(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var mu sync.Mutex
		var finished []time.Duration
		for _, d := range []time.Duration{1 * time.Second, 2 * time.Second, 3 * time.Second} {
			go func() {
				time.Sleep(d)
				mu.Lock()
				finished = append(finished, d)
				mu.Unlock()
			}()
		}
		finishedCount := func() int { mu.Lock(); defer mu.Unlock(); return len(finished) }

		synctest.Wait() // all three are asleep, i.e. durably blocked: nothing has finished
		if n := finishedCount(); n != 0 {
			t.Errorf("after Wait(), %d goroutines had finished, want 0 - Wait does not move the clock", n)
		}

		synctest.Sleep(2500 * time.Millisecond) // the 1s and 2s sleepers wake
		if n := finishedCount(); n != 2 {
			t.Errorf("after Sleep(2.5s), %d goroutines had finished, want 2", n)
		}

		// synctest.Test waits for every bubbled goroutine to exit before returning; one that can
		// never finish deadlocks the bubble and FAILS the test - a goroutine leak becomes a test
		// failure. This last Sleep is illustrative: Test would let the 3s sleeper finish anyway.
		synctest.Sleep(time.Second)
	})
}

// TestSynctestTimeout is the shape synctest makes easy: asserting on a real timeout, instantly.
func TestSynctestTimeout(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const timeout = 5 * time.Minute
		slow := make(chan int)
		go func() {
			time.Sleep(10 * time.Minute) // never arrives within the budget
			slow <- 1
		}()

		start := time.Now()
		select {
		case <-slow:
			t.Error("the slow operation should not have completed within the timeout")
		case <-time.After(timeout):
			if elapsed := time.Since(start); elapsed != timeout {
				t.Errorf("timed out after %v, want exactly %v", elapsed, timeout)
			}
		}

		synctest.Sleep(10 * time.Minute) // let the slow goroutine send, so it can exit
		<-slow
	})
}

// fakeStore is the whole mocking framework you need: a struct with the method the code uses.
type fakeStore struct {
	data  map[string]string
	err   error
	calls []string // what was asked for, so the test can assert on the interaction
}

func (f *fakeStore) Get(key string) (string, error) {
	f.calls = append(f.calls, key)
	if f.err != nil {
		return "", f.err
	}
	v, ok := f.data[key]
	if !ok {
		return "", fmt.Errorf("fakeStore: %q: %w", key, errNotFound)
	}
	return v, nil
}

// TestWithATestDouble tests greet against a fake store.
func TestWithATestDouble(t *testing.T) {
	fake := &fakeStore{data: map[string]string{"user": "Ada"}}
	assertEqual(t, greet(fake, "user"), "Hello, Ada!", `greet(fake, "user")`)
	assertEqual(t, greet(fake, "absent"), "Hello, stranger!", `greet(fake, "absent")`)

	// Assert on the interaction as well as the result.
	if len(fake.calls) != 2 {
		t.Errorf("fake.calls = %v, want 2 calls", fake.calls)
	}

	// A failing store, with no framework in sight.
	broken := &fakeStore{err: errors.New("connection refused")}
	assertEqual(t, greet(broken, "user"), "Hello, stranger!", "greet(broken)")
}

// TestReaderDouble substitutes strings.NewReader for a file: no filesystem, no mocking library.
func TestReaderDouble(t *testing.T) {
	assertEqual(t, countLines(strings.NewReader("a\nb\nc")), 3, "countLines")
	assertEqual(t, countLines(strings.NewReader("")), 0, "countLines(empty)")
}

// TestHTTPTestServer tests an HTTP handler through a real server and without one.
func TestHTTPTestServer(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/fail" {
			http.Error(w, "nope", http.StatusInternalServerError)
			return
		}
		fmt.Fprintf(w, "hello from %s", r.URL.Path)
	})

	// httptest.NewTestServer (Go 1.27) registers its own t.Cleanup, so there is no
	// `defer srv.Close()` to forget. It returns an UNSTARTED server: call Start (or StartTLS)
	// yourself - unlike httptest.NewServer, which starts immediately.
	srv := httptest.NewTestServer(t, handler)
	srv.Start()

	resp, err := http.Get(srv.URL + "/greeting")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body) // drain, so the keep-alive connection can be reused
	assertEqual(t, resp.StatusCode, http.StatusOK, "status")

	// ResponseRecorder tests a handler with no server at all.
	rec := httptest.NewRecorder()
	handler(rec, httptest.NewRequest(http.MethodGet, "/fail", nil))
	assertEqual(t, rec.Code, http.StatusInternalServerError, "recorder status")
	assertEqual(t, strings.TrimSpace(rec.Body.String()), "nope", "recorder body")
}

// TestInternalTestPackage explains which names a test can see, and attaches metadata.
func TestInternalTestPackage(t *testing.T) {
	// This file is package main - the internal test package - so it reaches unexported names
	// such as errEmpty. In a library, a file declaring package foo_test (the external test
	// package) sees only the exported API: a useful discipline that tests what callers can
	// reach and breaks import cycles. A command cannot be imported, so here only the internal
	// form is useful.
	if !errors.Is(fmt.Errorf("wrapped: %w", errEmpty), errEmpty) {
		t.Fatal("errEmpty should match through wrapping")
	}

	// t.Attr (Go 1.25) attaches structured metadata to the result, for CI to pick up.
	t.Attr("module", "014")
	t.Attr("topic", "testing")
}
