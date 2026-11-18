// Package monitor is the reference solution for Project 5 in basic_exercises.md: it checks
// a list of services concurrently, retries failures with a growing pause, classifies every
// service as up, degraded or down, and reports the results as a text table and as JSON.
//
// The package knows nothing about HTTP or databases. A service is anything that implements
// Checker, so real checks and test fakes plug in the same way.
package monitor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"
	"time"
)

// ---- Status ------------------------------------------------------------------------------------

// Status is the health of one service.
type Status int

const (
	Up       Status = iota // the first attempt succeeded quickly
	Degraded               // it worked, but only after a retry or too slowly
	Down                   // every attempt failed
)

// String implements fmt.Stringer.
func (s Status) String() string {
	switch s {
	case Up:
		return "up"
	case Degraded:
		return "degraded"
	case Down:
		return "down"
	default:
		return fmt.Sprintf("Status(%d)", int(s))
	}
}

// MarshalText implements encoding.TextMarshaler. encoding/json uses it, so a Status appears in
// JSON as "down" instead of 2.
func (s Status) MarshalText() ([]byte, error) {
	return []byte(s.String()), nil
}

// ---- Checkers ----------------------------------------------------------------------------------

// Checker is one service to monitor. Check must return promptly once ctx is done.
type Checker interface {
	Name() string
	Check(ctx context.Context) error
}

// CheckFunc turns a name and a function into a Checker.
type CheckFunc struct {
	ServiceName string
	Fn          func(ctx context.Context) error
}

func (c CheckFunc) Name() string                    { return c.ServiceName }
func (c CheckFunc) Check(ctx context.Context) error { return c.Fn(ctx) }

// ---- Configuration -----------------------------------------------------------------------------

// Config controls a Run. The zero value is usable: every zero field gets a default.
type Config struct {
	Workers       int           // checks running at the same time (default 4)
	Timeout       time.Duration // limit for ONE attempt (default 1s)
	Attempts      int           // attempts per service, including the first (default 3)
	Backoff       time.Duration // pause before the 2nd attempt, doubled after each failure (default 100ms)
	SlowThreshold time.Duration // a success slower than this counts as Degraded (default 500ms)
	Logger        *slog.Logger  // nil: no logging
}

// withDefaults returns a copy of c with every zero field filled in. c is a value receiver, so
// the caller's Config is not changed.
func (c Config) withDefaults() Config {
	if c.Workers <= 0 {
		c.Workers = 4
	}
	if c.Timeout <= 0 {
		c.Timeout = time.Second
	}
	if c.Attempts <= 0 {
		c.Attempts = 3
	}
	if c.Backoff <= 0 {
		c.Backoff = 100 * time.Millisecond
	}
	if c.SlowThreshold <= 0 {
		c.SlowThreshold = 500 * time.Millisecond
	}
	if c.Logger == nil {
		c.Logger = slog.New(slog.DiscardHandler)
	}
	return c
}

// ---- Retry -------------------------------------------------------------------------------------

// Retry calls f until it succeeds, the attempts run out, or ctx is done. It pauses for backoff
// before the second attempt, then for twice as long before each further one. It returns the
// value and error of the last attempt, plus the number of attempts made.
//
// It is generic so it can retry any operation that returns a value: here that value is the
// latency of a check, elsewhere it could be an HTTP response or a database row.
func Retry[T any](ctx context.Context, attempts int, backoff time.Duration, f func(ctx context.Context) (T, error)) (T, int, error) {
	var zero T
	if attempts < 1 {
		return zero, 0, errors.New("retry: attempts must be at least 1")
	}
	for n := 1; ; n++ {
		v, err := f(ctx)
		if err == nil || n == attempts {
			return v, n, err
		}
		select {
		case <-ctx.Done():
			return v, n, fmt.Errorf("%w (retry stopped: %w)", err, ctx.Err()) // two %w: both stay checkable
		case <-time.After(backoff):
		}
		backoff *= 2
	}
}

// ---- Running the checks ------------------------------------------------------------------------

// Result is the outcome for one service.
type Result struct {
	Name     string        `json:"name"`
	Status   Status        `json:"status"` // written as "up" / "degraded" / "down" thanks to MarshalText
	Attempts int           `json:"attempts"`
	Latency  time.Duration `json:"latency_ns"` // a Duration is an int64 count of nanoseconds
	Error    string        `json:"error,omitempty"`
}

// Run checks every service and returns one Result per checker, in the same order. Cancelling
// ctx makes the remaining checks fail fast.
func Run(ctx context.Context, checkers []Checker, cfg Config) []Result {
	cfg = cfg.withDefaults()
	results := make([]Result, len(checkers))
	jobs := make(chan int)

	var wg sync.WaitGroup
	for range cfg.Workers {
		wg.Go(func() {
			for i := range jobs {
				results[i] = check(ctx, checkers[i], cfg) // own slot: no mutex needed
			}
		})
	}
	for i := range checkers {
		jobs <- i
	}
	close(jobs)
	wg.Wait()
	return results
}

// check runs all attempts for one service and classifies the outcome.
func check(ctx context.Context, c Checker, cfg Config) Result {
	latency, attempts, err := Retry(ctx, cfg.Attempts, cfg.Backoff, func(ctx context.Context) (time.Duration, error) {
		return checkOnce(ctx, c, cfg.Timeout)
	})

	r := Result{Name: c.Name(), Attempts: attempts, Latency: latency}
	switch {
	case err != nil:
		r.Status = Down
		r.Error = err.Error()
	case attempts > 1 || latency > cfg.SlowThreshold:
		r.Status = Degraded
	default:
		r.Status = Up
	}

	cfg.Logger.Info("checked", "service", r.Name, "status", r.Status, "attempts", r.Attempts, "latency", r.Latency)
	return r
}

// checkOnce runs a single attempt with its own deadline and measures how long it took.
func checkOnce(ctx context.Context, c Checker, timeout time.Duration) (time.Duration, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	start := time.Now()
	err := c.Check(ctx)
	return time.Since(start), err
}

// ---- Reporting ---------------------------------------------------------------------------------

// Summary counts the results per status.
type Summary struct {
	Up, Degraded, Down int
}

// Summarize counts the results.
func Summarize(results []Result) Summary {
	var s Summary
	for _, r := range results {
		switch r.Status {
		case Up:
			s.Up++
		case Degraded:
			s.Degraded++
		case Down:
			s.Down++
		}
	}
	return s
}

// Healthy reports whether no service is down.
func (s Summary) Healthy() bool { return s.Down == 0 }

// String implements fmt.Stringer: "3 services: 1 up, 1 degraded, 1 down".
func (s Summary) String() string {
	return fmt.Sprintf("%d services: %d up, %d degraded, %d down", s.Up+s.Degraded+s.Down, s.Up, s.Degraded, s.Down)
}

// WriteTable writes one aligned line per result, e.g.
//
//	database   down        3   50ms  context deadline exceeded
func WriteTable(w io.Writer, results []Result) error {
	for _, r := range results {
		line := fmt.Sprintf("%-10s %-9s %3d %6v  %s", r.Name, r.Status, r.Attempts, r.Latency.Round(time.Millisecond), r.Error)
		if _, err := fmt.Fprintln(w, strings.TrimRight(line, " ")); err != nil { // no trailing spaces without an error
			return err
		}
	}
	return nil
}

// WriteJSON writes the results as an indented JSON array.
func WriteJSON(w io.Writer, results []Result) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(results)
}
