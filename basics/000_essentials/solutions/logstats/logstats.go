// Package logstats is the reference solution for Project 3 in basic_exercises.md: it parses
// log files, collects statistics, analyses several files concurrently and writes a JSON report.
//
// A log line looks like this (the message may contain spaces):
//
//	2026-03-14 09:30:00 ERROR disk full
package logstats

import (
	"bufio"
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"iter"
	"maps"
	"slices"
	"strings"
	"sync"
	"time"
)

// ---- Levels ------------------------------------------------------------------------------------

// Level is the severity of a log entry.
type Level int

const (
	Debug Level = iota
	Info
	Warn
	Error
)

// levelNames is indexed by Level, so the order must match the constants above.
var levelNames = [...]string{"DEBUG", "INFO", "WARN", "ERROR"}

// String implements fmt.Stringer.
func (l Level) String() string {
	if l < 0 || int(l) >= len(levelNames) {
		return fmt.Sprintf("Level(%d)", int(l))
	}
	return levelNames[l]
}

// ParseLevel turns "ERROR" (in any case) into Error.
func ParseLevel(s string) (Level, error) {
	for i, name := range levelNames {
		if strings.EqualFold(s, name) {
			return Level(i), nil
		}
	}
	return 0, fmt.Errorf("unknown level %q", s)
}

// ---- Parsing -----------------------------------------------------------------------------------

// Entry is one parsed log line.
type Entry struct {
	Time    time.Time
	Level   Level
	Message string
}

// layout is the reference date written the way the log writes dates.
const layout = "2006-01-02 15:04:05"

// ParseLine parses a single log line.
func ParseLine(line string) (Entry, error) {
	parts := strings.SplitN(strings.TrimSpace(line), " ", 4) // date, time, level, message
	if len(parts) < 4 {
		return Entry{}, errors.New("want: date time level message")
	}
	t, err := time.Parse(layout, parts[0]+" "+parts[1])
	if err != nil {
		return Entry{}, err
	}
	level, err := ParseLevel(parts[2])
	if err != nil {
		return Entry{}, err
	}
	return Entry{Time: t, Level: level, Message: strings.TrimSpace(parts[3])}, nil
}

// ParseError says which line of which source could not be parsed, and why.
type ParseError struct {
	Source string
	Line   int
	Err    error
}

func (e *ParseError) Error() string { return fmt.Sprintf("%s:%d: %v", e.Source, e.Line, e.Err) }

// Unwrap lets errors.Is and errors.As look at the cause, for example a *time.ParseError.
func (e *ParseError) Unwrap() error { return e.Err }

// Entries yields the entries of r one by one, numbering lines from 1 and skipping blank
// lines. A line that cannot be parsed is yielded as a *ParseError, so the caller decides
// whether to stop or carry on. A read error is yielded once, at the end.
func Entries(source string, r io.Reader) iter.Seq2[Entry, error] {
	return func(yield func(Entry, error) bool) {
		sc := bufio.NewScanner(r)
		for line := 1; sc.Scan(); line++ {
			text := sc.Text()
			if strings.TrimSpace(text) == "" {
				continue
			}
			e, err := ParseLine(text)
			if err != nil {
				err = &ParseError{Source: source, Line: line, Err: err}
			}
			if !yield(e, err) {
				return
			}
		}
		if err := sc.Err(); err != nil { // Scan returns false on EOF AND on errors - check which
			yield(Entry{}, fmt.Errorf("%s: read: %w", source, err))
		}
	}
}

// ---- Statistics --------------------------------------------------------------------------------

// Stats collects counts over any number of entries. Create one with NewStats.
type Stats struct {
	Total    int
	Invalid  int            // lines that could not be parsed
	ByLevel  map[Level]int  // entries per level
	PerHour  [24]int        // entries per hour of the day: an array, because the size is fixed
	Messages map[string]int // ERROR messages and how often each occurred
	First    time.Time      // the earliest entry (zero if there are none)
	Last     time.Time      // the latest entry
}

// NewStats returns empty statistics with their maps ready to use.
func NewStats() *Stats {
	return &Stats{ByLevel: make(map[Level]int), Messages: make(map[string]int)}
}

// Add counts one entry.
func (s *Stats) Add(e Entry) {
	s.Total++
	s.ByLevel[e.Level]++
	s.PerHour[e.Time.Hour()]++
	if e.Level == Error {
		s.Messages[e.Message]++
	}
	if s.First.IsZero() || e.Time.Before(s.First) {
		s.First = e.Time
	}
	if e.Time.After(s.Last) {
		s.Last = e.Time
	}
}

// Merge adds the counts of other to s.
func (s *Stats) Merge(other *Stats) {
	s.Total += other.Total
	s.Invalid += other.Invalid
	for level, n := range other.ByLevel {
		s.ByLevel[level] += n
	}
	for h, n := range other.PerHour {
		s.PerHour[h] += n
	}
	for msg, n := range other.Messages {
		s.Messages[msg] += n
	}
	if !other.First.IsZero() && (s.First.IsZero() || other.First.Before(s.First)) {
		s.First = other.First
	}
	if other.Last.After(s.Last) {
		s.Last = other.Last
	}
}

// Analyze reads one source. Invalid lines are counted and skipped, so the statistics are still
// useful (a partial result); the parse errors come back joined into one error. Use errors.As
// to get the first *ParseError.
func Analyze(source string, r io.Reader) (*Stats, error) {
	s := NewStats()
	var errs []error
	for e, err := range Entries(source, r) {
		if err != nil {
			// errors.AsType (Go 1.26) is the generic form of errors.As: no pointer variable needed.
			if _, isParseErr := errors.AsType[*ParseError](err); isParseErr {
				s.Invalid++
			}
			errs = append(errs, err)
			continue
		}
		s.Add(e)
	}
	return s, errors.Join(errs...) // nil when errs is empty
}

// Source is one named input, for example an open file.
type Source struct {
	Name   string
	Reader io.Reader
}

// AnalyzeAll analyses the sources with a pool of workers and merges the statistics. The
// errors are joined in the order of the sources, whichever worker finished first.
func AnalyzeAll(sources []Source, workers int) (*Stats, error) {
	type result struct {
		stats *Stats
		err   error
	}
	results := make([]result, len(sources)) // one slot per source: no mutex, order kept
	jobs := make(chan int)

	var wg sync.WaitGroup
	for range max(workers, 1) {
		wg.Go(func() {
			for i := range jobs {
				s, err := Analyze(sources[i].Name, sources[i].Reader)
				results[i] = result{stats: s, err: err}
			}
		})
	}
	for i := range sources {
		jobs <- i
	}
	close(jobs) // the workers' range loops end
	wg.Wait()

	total := NewStats()
	var errs []error
	for _, r := range results {
		total.Merge(r.stats)
		errs = append(errs, r.err)
	}
	return total, errors.Join(errs...) // Join skips nil errors
}

// ---- Report ------------------------------------------------------------------------------------

// MessageCount is a message with the number of times it occurred.
type MessageCount struct {
	Message string `json:"message"`
	Count   int    `json:"count"`
}

// Report is the JSON-friendly summary of Stats.
type Report struct {
	Total     int            `json:"total"`
	Invalid   int            `json:"invalid"`
	ByLevel   map[string]int `json:"by_level"` // JSON keys are strings: "ERROR", not 3
	PerHour   map[string]int `json:"per_hour"` // only the hours that had entries: "09": 12
	TopErrors []MessageCount `json:"top_errors"`
	First     time.Time      `json:"first"`
	Last      time.Time      `json:"last"`
}

// Report builds a Report with the topN most frequent ERROR messages.
func (s *Stats) Report(topN int) Report {
	r := Report{
		Total:     s.Total,
		Invalid:   s.Invalid,
		ByLevel:   make(map[string]int, len(s.ByLevel)),
		PerHour:   make(map[string]int),
		TopErrors: []MessageCount{}, // not nil, so JSON shows [] instead of null
		First:     s.First,
		Last:      s.Last,
	}
	for level, n := range s.ByLevel {
		r.ByLevel[level.String()] = n
	}
	for hour, n := range s.PerHour {
		if n > 0 {
			r.PerHour[fmt.Sprintf("%02d", hour)] = n // zero-padded, so the keys sort by time
		}
	}
	for _, msg := range TopN(s.Messages, topN) {
		r.TopErrors = append(r.TopErrors, MessageCount{Message: msg, Count: s.Messages[msg]})
	}
	return r
}

// WriteJSON writes the report as indented JSON. (encoding/json writes map keys sorted.)
func (r Report) WriteJSON(w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(r)
}

// TopN returns the n keys with the highest counts, highest first; ties are broken by key so
// the result is always the same. It is generic: it works for map[string]int, map[int]int, ...
func TopN[K cmp.Ordered](counts map[K]int, n int) []K {
	keys := slices.Collect(maps.Keys(counts))
	slices.SortFunc(keys, func(a, b K) int {
		return cmp.Or(
			cmp.Compare(counts[b], counts[a]), // higher count first
			cmp.Compare(a, b),                 // then by key
		)
	})
	return keys[:min(max(n, 0), len(keys))]
}
