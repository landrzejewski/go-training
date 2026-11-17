package monitor

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"
)

// ---- Fake checkers: each one simulates a kind of service ---------------------------------------

var errRefused = errors.New("connection refused")

func healthy(name string) Checker {
	return CheckFunc{ServiceName: name, Fn: func(context.Context) error { return nil }}
}

func broken(name string) Checker {
	return CheckFunc{ServiceName: name, Fn: func(context.Context) error { return errRefused }}
}

// hanging never answers; it only gives up when its context is done, as a good Checker must.
func hanging(name string) Checker {
	return CheckFunc{ServiceName: name, Fn: func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	}}
}

// slow answers correctly, but only after d (or earlier if ctx is done).
func slow(name string, d time.Duration) Checker {
	return CheckFunc{ServiceName: name, Fn: func(ctx context.Context) error {
		select {
		case <-time.After(d):
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}}
}

// flaky fails the first `failures` times, then works. It is a pointer so the counter survives
// between calls. Only one goroutine checks a given service, so the counter needs no mutex.
type flaky struct {
	name     string
	failures int
	calls    int
}

func (f *flaky) Name() string { return f.name }
func (f *flaky) Check(context.Context) error {
	f.calls++
	if f.calls <= f.failures {
		return errRefused
	}
	return nil
}

// testConfig keeps the tests fast: short timeouts, tiny backoff.
func testConfig() Config {
	return Config{Workers: 3, Timeout: 50 * time.Millisecond, Attempts: 3, Backoff: time.Millisecond, SlowThreshold: 500 * time.Millisecond}
}

// ---- Tests -------------------------------------------------------------------------------------

func TestRetry(t *testing.T) {
	ctx := context.Background()

	t.Run("succeeds on the third attempt", func(t *testing.T) {
		calls := 0
		v, n, err := Retry(ctx, 5, time.Millisecond, func(context.Context) (string, error) {
			calls++
			if calls < 3 {
				return "", errRefused
			}
			return "ok", nil
		})
		if v != "ok" || n != 3 || err != nil {
			t.Errorf("Retry = %q, %d, %v; want ok, 3, nil", v, n, err)
		}
	})

	t.Run("gives up after all attempts", func(t *testing.T) {
		_, n, err := Retry(ctx, 3, time.Millisecond, func(context.Context) (int, error) { return 0, errRefused })
		if n != 3 || !errors.Is(err, errRefused) {
			t.Errorf("Retry = %d, %v; want 3 attempts and errRefused", n, err)
		}
	})

	t.Run("stops when the context is cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(ctx)
		cancel()
		_, n, err := Retry(ctx, 10, time.Hour, func(context.Context) (int, error) { return 0, errRefused })
		if n != 1 || !errors.Is(err, context.Canceled) || !errors.Is(err, errRefused) {
			t.Errorf("Retry = %d, %v; want 1 attempt and both errors", n, err)
		}
	})

	t.Run("rejects zero attempts", func(t *testing.T) {
		if _, _, err := Retry(ctx, 0, time.Millisecond, func(context.Context) (int, error) { return 1, nil }); err == nil {
			t.Error("want an error for 0 attempts")
		}
	})
}

func TestRunClassifies(t *testing.T) {
	checkers := []Checker{
		healthy("api"),
		&flaky{name: "queue", failures: 2},
		broken("database"),
		hanging("legacy"),
	}
	results := Run(context.Background(), checkers, testConfig())

	want := []struct {
		name     string
		status   Status
		attempts int
	}{
		{"api", Up, 1},
		{"queue", Degraded, 3},
		{"database", Down, 3},
		{"legacy", Down, 3},
	}
	if len(results) != len(want) {
		t.Fatalf("got %d results, want %d", len(results), len(want))
	}
	for i, w := range want {
		r := results[i]
		if r.Name != w.name || r.Status != w.status || r.Attempts != w.attempts {
			t.Errorf("results[%d] = %s/%v/%d, want %s/%v/%d", i, r.Name, r.Status, r.Attempts, w.name, w.status, w.attempts)
		}
	}
	if !strings.Contains(results[3].Error, "deadline exceeded") {
		t.Errorf("legacy error = %q, want a timeout", results[3].Error)
	}
	if s := Summarize(results); s.String() != "4 services: 1 up, 1 degraded, 2 down" || s.Healthy() {
		t.Errorf("summary = %v, healthy = %t", s, s.Healthy())
	}
}

func TestSlowServiceIsDegraded(t *testing.T) {
	cfg := testConfig()
	cfg.SlowThreshold = 5 * time.Millisecond
	results := Run(context.Background(), []Checker{slow("reports", 20*time.Millisecond)}, cfg)
	if r := results[0]; r.Status != Degraded || r.Attempts != 1 {
		t.Errorf("result = %+v, want Degraded after 1 attempt", r)
	}
}

func TestRunStopsWhenCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancelled before we even start

	cfg := testConfig()
	cfg.Timeout = time.Hour // without the cancellation this test would hang for hours
	start := time.Now()
	results := Run(ctx, []Checker{hanging("a"), hanging("b")}, cfg)
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Errorf("Run took %v after cancellation", elapsed)
	}
	for _, r := range results {
		if r.Status != Down {
			t.Errorf("%s: status %v, want down", r.Name, r.Status)
		}
	}
}

func TestDefaults(t *testing.T) {
	cfg := Config{}.withDefaults()
	if cfg.Workers != 4 || cfg.Attempts != 3 || cfg.Timeout != time.Second || cfg.Logger == nil {
		t.Errorf("defaults = %+v", cfg)
	}
}

func TestWriteJSON(t *testing.T) {
	var buf bytes.Buffer
	results := []Result{{Name: "db", Status: Down, Attempts: 3, Error: "boom"}, {Name: "api", Status: Up, Attempts: 1}}
	if err := WriteJSON(&buf, results); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), `"status": "down"`) {
		t.Errorf("JSON does not contain the status as text:\n%s", buf.String())
	}
	if strings.Contains(buf.String(), `"error": ""`) {
		t.Errorf("omitempty should drop the empty error:\n%s", buf.String())
	}
	var decoded []map[string]any
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil || len(decoded) != 2 {
		t.Errorf("invalid JSON: %v", err)
	}
}

func TestWriteTable(t *testing.T) {
	results := []Result{
		{Name: "api", Status: Up, Attempts: 1, Latency: 12*time.Millisecond + 300*time.Microsecond},
		{Name: "database", Status: Down, Attempts: 3, Latency: 150 * time.Millisecond, Error: "timeout"},
	}
	var buf bytes.Buffer
	if err := WriteTable(&buf, results); err != nil {
		t.Fatal(err)
	}
	want := "api        up          1   12ms\n" +
		"database   down        3  150ms  timeout\n"
	if buf.String() != want {
		t.Errorf("table =\n%s\nwant\n%s", buf.String(), want)
	}
}

func TestLogging(t *testing.T) {
	var logs bytes.Buffer
	cfg := testConfig()
	cfg.Logger = slog.New(slog.NewTextHandler(&logs, nil))
	Run(context.Background(), []Checker{healthy("api")}, cfg)
	if !strings.Contains(logs.String(), "service=api status=up attempts=1") {
		t.Errorf("log = %q", logs.String())
	}
}

func ExampleRun() {
	checkers := []Checker{
		CheckFunc{ServiceName: "api", Fn: func(context.Context) error { return nil }},
		CheckFunc{ServiceName: "database", Fn: func(context.Context) error { return errors.New("connection refused") }},
	}
	cfg := Config{Attempts: 2, Backoff: time.Millisecond}
	results := Run(context.Background(), checkers, cfg)
	for _, r := range results {
		line := fmt.Sprintf("%-9s %-5v attempts=%d", r.Name, r.Status, r.Attempts)
		if r.Error != "" {
			line += " " + r.Error
		}
		fmt.Println(line)
	}
	fmt.Println(Summarize(results))
	// Output:
	// api       up    attempts=1
	// database  down  attempts=2 connection refused
	// 2 services: 1 up, 0 degraded, 1 down
}
