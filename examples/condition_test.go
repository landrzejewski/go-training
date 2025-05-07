package examples

import (
	"testing"
)

func TestIsEven(t *testing.T) {
	result := IsEven(4)
	if !result {
		t.Errorf("Expected 'true' is %v", result)
	}
}

func TestIsEvenParametrized(t *testing.T) {
	tests := []struct {
		name     string
		input    int
		expected bool
	}{
		{"Even number", 2, true},
		{"Odd number", 3, false},
		{"Zero", 0, true},
		{"Negative even number", -4, true},
		{"Negative odd number", -5, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsEven(tt.input)
			if result != tt.expected {
				t.Errorf("IsEven(%d) = %v; want %v", tt.input, result, tt.expected)
			}
		})
	}
}

// go test training.pl/go/examples --test.Short()
// go test training.pl/go/examples -cover
func TestParaller(t *testing.T) {  
	// defer cleanup()

	t.Cleanup(cleanup)

	if testing.Short() {
		t.Skip("Skipping paraller tests")
	}
	t.Run("Test 3 in Parallel", func(t *testing.T) {
		t.Parallel()
		result := IsEven(3)
		if result {
			t.Errorf("Result was incorrect, got: %v, want: %v.", result, false)
		}
	})
	t.Run("Test 4 in Parallel", func(t *testing.T) {
		t.Parallel()
		result := IsEven(4)
		if !result {
			t.Errorf("Result was incorrect, got: %v, want: %v.", result, true)
		}
	})
}

func cleanup() {
}

func BenchmarkIsEven(b *testing.B) {
	for range b.N {
		IsEven(4)
	}
}