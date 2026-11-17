package main

// The test half of the essentials file: one example of each kind of test function.
//
//	go test -v ./basics/000_essentials
//	go test -run 'TestReverse/unicode' -v ./basics/000_essentials
//	go test -bench . -benchmem ./basics/000_essentials
//	go test -race -cover ./basics/000_essentials

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

// TestReverse is a TABLE-DRIVEN test: a slice of cases plus one loop. t.Run gives every case
// its own name, so a failure says which case broke and -run can select a single one.
func TestReverse(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{"empty", "", ""},
		{"ascii", "hello", "olleh"},
		{"unicode", "żółw", "włóż"}, // would fail if Reverse swapped bytes instead of runes
		{"palindrome", "kajak", "kajak"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel() // the cases are independent, so they may run at the same time
			if got := Reverse(tt.in); got != tt.want {
				t.Errorf("Reverse(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// TestStack checks the generic Stack. t.Fatal stops the test at once; t.Error records the
// failure and carries on, so one run can report several problems.
func TestStack(t *testing.T) {
	var s Stack[string]
	if _, ok := s.Pop(); ok {
		t.Fatal("Pop on an empty stack returned ok = true")
	}
	s.Push("a")
	s.Push("b")
	if got, _ := s.Pop(); got != "b" {
		t.Errorf("Pop() = %q, want %q (last in, first out)", got, "b")
	}
	if got := s.Size(); got != 1 {
		t.Errorf("Size() = %d, want 1", got)
	}
}

// TestSum runs the same generic function with two type arguments.
func TestSum(t *testing.T) {
	if got := Sum([]int{1, 2, 3}); got != 6 {
		t.Errorf("Sum(ints) = %d, want 6", got)
	}
	if got := Sum([]float64{0.5, 0.25}); got != 0.75 {
		t.Errorf("Sum(floats) = %g, want 0.75", got)
	}
}

// TestDivideByZero tests the error path - the case that is easiest to forget.
func TestDivideByZero(t *testing.T) {
	if _, err := divide(1, 0); err == nil {
		t.Error("divide(1, 0) returned no error")
	}
}

// TestStartServer checks the ERROR paths the way a caller would: with errors.Is, not by
// comparing message text. The table holds the error each case must wrap (nil = success).
func TestStartServer(t *testing.T) {
	tests := []struct {
		name    string
		config  map[string]string
		want    int
		wantErr error
	}{
		{"valid", map[string]string{"port": "8080"}, 8080, nil},
		{"not a number", map[string]string{"port": "80a"}, 0, strconv.ErrSyntax}, // wrapped twice
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := startServer(tt.config)
			assertErrorIs(t, err, tt.wantErr)
			if got != tt.want {
				t.Errorf("startServer() = %d, want %d", got, tt.want)
			}
		})
	}

	// Some errors are plain: there is nothing to errors.Is against, so only check err != nil.
	if _, err := startServer(map[string]string{}); err == nil {
		t.Error("startServer without a port returned no error")
	}
}

// assertErrorIs is a test HELPER. Because of t.Helper, a failure is reported at the line in
// the test that called it, not at the line inside this function.
func assertErrorIs(t *testing.T, err, target error) {
	t.Helper()
	if target == nil {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		return
	}
	if !errors.Is(err, target) {
		t.Fatalf("error = %v, want it to wrap %v", err, target)
	}
}

// TestValidateForm checks that errors.Join keeps every problem visible to errors.Is/As.
func TestValidateForm(t *testing.T) {
	err := validateForm("", -1)
	assertErrorIs(t, err, ErrEmptyName)
	if _, ok := errors.AsType[*ValidationError](err); !ok {
		t.Errorf("error = %v, want a *ValidationError inside", err)
	}
	assertErrorIs(t, validateForm("Ann", 30), nil) // valid input: really nil, no typed-nil trap
}

// TestSaveReportToFile writes into a real file. t.TempDir gives a fresh directory that is
// removed after the test; t.Cleanup registers work to run when the test ends.
func TestSaveReportToFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "report.txt")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := saveReport(f); err != nil { // saveReport closes f itself
		t.Fatalf("saveReport: %v", err)
	}

	r, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { r.Close() }) // runs after the test, even if it fails half-way
	data, err := io.ReadAll(r)
	if err != nil || string(data) != "report\n" {
		t.Errorf("file content = %q, %v; want %q", data, err, "report\n")
	}

	// And the failure path, with a fake whose Close fails.
	if err := saveReport(failingFile{}); err == nil {
		t.Error("saveReport ignored the Close error")
	}
}

// BenchmarkReverse measures speed: go test -bench . -benchmem
// b.Loop runs the body as many times as needed for a stable measurement.
func BenchmarkReverse(b *testing.B) {
	for b.Loop() {
		Reverse("hello, wörld")
	}
}

// ExampleReverse is documentation AND a test: go test compares what it prints with the
// Output comment, and go doc shows it next to Reverse.
func ExampleReverse() {
	fmt.Println(Reverse("Gopher"))
	// Output: rehpoG
}
