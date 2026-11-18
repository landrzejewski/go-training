package logstats

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"testing/iotest"
	"time"
)

const sampleLog = `2026-03-14 09:30:00 INFO service started
2026-03-14 09:31:10 WARN cache miss rate high
2026-03-14 10:02:00 ERROR disk full

2026-03-14 10:05:00 ERROR disk full
this line is broken
2026-03-14 11:00:00 ERROR connection refused
2026-03-14 25:00:00 INFO bad hour
2026-03-14 11:30:00 DEBUG tick
`

func TestParseLine(t *testing.T) {
	tests := []struct {
		name    string
		line    string
		want    Entry
		wantErr bool
	}{
		{
			name: "valid",
			line: "2026-03-14 09:30:00 ERROR disk full",
			want: Entry{Time: time.Date(2026, 3, 14, 9, 30, 0, 0, time.UTC), Level: Error, Message: "disk full"},
		},
		{
			name: "level in lower case",
			line: "2026-03-14 09:30:00 warn slow",
			want: Entry{Time: time.Date(2026, 3, 14, 9, 30, 0, 0, time.UTC), Level: Warn, Message: "slow"},
		},
		{name: "too few fields", line: "2026-03-14 09:30:00 ERROR", wantErr: true},
		{name: "bad time", line: "2026-03-14 9:30 ERROR x", wantErr: true},
		{name: "unknown level", line: "2026-03-14 09:30:00 FATAL x", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseLine(tt.line)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseLine(%q) error = %v, wantErr %t", tt.line, err, tt.wantErr)
			}
			if !tt.wantErr && (!got.Time.Equal(tt.want.Time) || got.Level != tt.want.Level || got.Message != tt.want.Message) {
				t.Errorf("ParseLine(%q) = %+v, want %+v", tt.line, got, tt.want)
			}
		})
	}
}

func TestLevel(t *testing.T) {
	if got := fmt.Sprint(Debug, Info, Warn, Error, Level(9)); got != "DEBUG INFO WARN ERROR Level(9)" {
		t.Errorf("String = %q", got)
	}
	for _, l := range []Level{Debug, Info, Warn, Error} {
		if parsed, err := ParseLevel(l.String()); err != nil || parsed != l {
			t.Errorf("ParseLevel(%q) = %v, %v", l, parsed, err)
		}
	}
}

func TestAnalyzePartialResults(t *testing.T) {
	s, err := Analyze("app.log", strings.NewReader(sampleLog))
	if s.Total != 6 || s.Invalid != 2 {
		t.Errorf("Total = %d, Invalid = %d; want 6 and 2", s.Total, s.Invalid)
	}
	if s.ByLevel[Error] != 3 || s.Messages["disk full"] != 2 {
		t.Errorf("ByLevel = %v, Messages = %v", s.ByLevel, s.Messages)
	}
	if s.PerHour[10] != 2 || s.PerHour[11] != 2 {
		t.Errorf("PerHour = %v", s.PerHour)
	}

	// The error describes BOTH bad lines; errors.As finds the first one.
	pe, ok := errors.AsType[*ParseError](err)
	if !ok {
		t.Fatalf("error = %v, want a *ParseError inside", err)
	}
	if pe.Source != "app.log" || pe.Line != 6 {
		t.Errorf("first ParseError at %s:%d, want app.log:6", pe.Source, pe.Line)
	}
	if !strings.Contains(err.Error(), "app.log:8:") {
		t.Errorf("error %q does not mention line 8", err)
	}
	var timeErr *time.ParseError // Unwrap lets errors.As reach the cause
	if !errors.As(err, &timeErr) {
		t.Error("the bad-hour line should carry a *time.ParseError")
	}
}

func TestEntriesStopsOnBreak(t *testing.T) {
	n := 0
	for range Entries("app.log", strings.NewReader(sampleLog)) {
		n++
		if n == 2 {
			break
		}
	}
	if n != 2 {
		t.Errorf("got %d entries, want 2", n)
	}
}

func TestEntriesReportsReadErrors(t *testing.T) {
	errDisk := errors.New("disk on fire")
	// One good line, then a reader that fails: like a file on a dying disk.
	r := io.MultiReader(strings.NewReader("2026-03-14 09:30:00 INFO first\n"), iotest.ErrReader(errDisk))
	s, err := Analyze("broken.log", r)
	if !errors.Is(err, errDisk) {
		t.Errorf("error = %v, want it to wrap errDisk", err)
	}
	if _, ok := errors.AsType[*ParseError](err); ok || s.Invalid != 0 {
		t.Errorf("a read error is not a parse error (Invalid = %d)", s.Invalid)
	}
	if s.Total != 1 {
		t.Errorf("Total = %d, want 1: the line before the failure still counts", s.Total)
	}
}

func TestAnalyzeAllMergesInOrder(t *testing.T) {
	sources := []Source{
		{"a.log", strings.NewReader("2026-03-14 08:00:00 ERROR disk full\nbroken\n")},
		{"b.log", strings.NewReader("2026-03-13 23:59:59 INFO early\n2026-03-14 08:30:00 ERROR timeout\n")},
		{"c.log", strings.NewReader("also broken\n2026-03-15 07:00:00 ERROR disk full\n")},
	}
	s, err := AnalyzeAll(sources, 2)
	if s.Total != 4 || s.Invalid != 2 || s.Messages["disk full"] != 2 {
		t.Errorf("merged stats: Total=%d Invalid=%d Messages=%v", s.Total, s.Invalid, s.Messages)
	}
	if want := time.Date(2026, 3, 13, 23, 59, 59, 0, time.UTC); !s.First.Equal(want) {
		t.Errorf("First = %v, want %v", s.First, want)
	}
	if want := "a.log:2: want: date time level message\nc.log:1: want: date time level message"; err == nil || err.Error() != want {
		t.Errorf("error =\n%v\nwant\n%v", err, want)
	}
}

func TestTopN(t *testing.T) {
	counts := map[string]int{"b": 2, "a": 2, "c": 5, "d": 1}
	if got := fmt.Sprint(TopN(counts, 3)); got != "[c a b]" {
		t.Errorf("TopN(3) = %s, want [c a b]", got)
	}
	if got := len(TopN(counts, 10)); got != 4 {
		t.Errorf("len(TopN(10)) = %d, want 4", got)
	}
	if got := TopN(map[int]int{7: 1, 3: 9}, 1); got[0] != 3 { // works for int keys too
		t.Errorf("TopN(int keys) = %v", got)
	}
}

func TestReportJSONRoundTrip(t *testing.T) {
	s, _ := Analyze("app.log", strings.NewReader(sampleLog))
	var buf bytes.Buffer
	if err := s.Report(2).WriteJSON(&buf); err != nil {
		t.Fatalf("WriteJSON: %v", err)
	}
	var decoded Report
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("the report is not valid JSON: %v", err)
	}
	if decoded.ByLevel["ERROR"] != 3 || len(decoded.TopErrors) != 2 || decoded.TopErrors[0].Message != "disk full" {
		t.Errorf("decoded report = %+v", decoded)
	}
}

func ExampleReport_WriteJSON() {
	log := "2026-03-14 09:30:00 INFO started\n" +
		"2026-03-14 10:00:00 ERROR disk full\n" +
		"2026-03-14 10:15:00 ERROR disk full\n" +
		"2026-03-14 10:20:00 ERROR timeout\n"
	s, err := Analyze("app.log", strings.NewReader(log))
	if err != nil {
		fmt.Println(err)
		return
	}
	if err := s.Report(1).WriteJSON(os.Stdout); err != nil {
		fmt.Println(err)
	}
	// Output:
	// {
	//   "total": 4,
	//   "invalid": 0,
	//   "by_level": {
	//     "ERROR": 3,
	//     "INFO": 1
	//   },
	//   "per_hour": {
	//     "09": 1,
	//     "10": 3
	//   },
	//   "top_errors": [
	//     {
	//       "message": "disk full",
	//       "count": 2
	//     }
	//   ],
	//   "first": "2026-03-14T09:30:00Z",
	//   "last": "2026-03-14T10:20:00Z"
	// }
}
