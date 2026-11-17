package solutions

// Solutions for exercises 14.1-14.3. Here the tests themselves are the solutions.
//
//	go test -v -run 'TestIsLeapYear|TestParseAge|TestJoin' ./basics/000_essentials/solutions
//	go test -run '^$' -bench Join -benchmem ./basics/000_essentials/solutions

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

// ---- Exercise 14.1: a table-driven test --------------------------------------------------------

func TestIsLeapYear(t *testing.T) {
	tests := []struct {
		name string
		year int
		want bool
	}{
		{"divisible by 4", 2024, true},
		{"not divisible by 4", 2023, false},
		{"century", 1900, false},
		{"divisible by 400", 2000, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) { // each case is a named subtest: -run 'TestIsLeapYear/century'
			if got := IsLeapYear(tt.year); got != tt.want {
				t.Errorf("IsLeapYear(%d) = %t, want %t", tt.year, got, tt.want)
			}
		})
	}
}

// ---- Exercise 14.2: testing the error path -----------------------------------------------------

func TestParseAge(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		got, err := ParseAge("42")
		if err != nil {
			t.Fatalf("ParseAge(%q) returned error %v", "42", err) // Fatal: checking got makes no sense now
		}
		if got != 42 {
			t.Errorf("ParseAge(%q) = %d, want 42", "42", got)
		}
	})

	t.Run("out of range", func(t *testing.T) {
		for _, in := range []string{"-1", "151"} {
			_, err := ParseAge(in)
			if !errors.Is(err, ErrAgeOutOfRange) { // Error, not Fatal: report every bad input
				t.Errorf("ParseAge(%q) error = %v, want ErrAgeOutOfRange", in, err)
			}
		}
	})

	t.Run("not a number", func(t *testing.T) {
		_, err := ParseAge("abc")
		if err == nil {
			t.Fatalf("ParseAge(%q) returned no error", "abc")
		}
		if errors.Is(err, ErrAgeOutOfRange) {
			t.Errorf("ParseAge(%q) error = %v, must not be ErrAgeOutOfRange", "abc", err)
		}
	})
}

// ---- Exercise 14.3: benchmarks (and checking both versions agree) ------------------------------

// TestJoin compares both implementations with strings.Join from the standard library.
func TestJoin(t *testing.T) {
	for _, parts := range [][]string{nil, {"a"}, {"a", "b", "c"}} {
		want := strings.Join(parts, ", ")
		if got := JoinPlus(parts, ", "); got != want {
			t.Errorf("JoinPlus(%q) = %q, want %q", parts, got, want)
		}
		if got := JoinBuilder(parts, ", "); got != want {
			t.Errorf("JoinBuilder(%q) = %q, want %q", parts, got, want)
		}
	}
}

var benchParts = strings.Fields(strings.Repeat("gopher ", 1000)) // 1000 words

func BenchmarkJoinPlus(b *testing.B) {
	for b.Loop() {
		JoinPlus(benchParts, ",")
	}
}

func BenchmarkJoinBuilder(b *testing.B) {
	for b.Loop() {
		JoinBuilder(benchParts, ",")
	}
}

func ExampleJoinBuilder() {
	fmt.Println(JoinBuilder([]string{"a", "b", "c"}, ", "))
	// Output: a, b, c
}
