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

type testInput struct {
	name string
	value int
	expected bool
}

func TestIsEvenParametrized(t *testing.T) { 
	/*var parameters = []struct {
		name string
		value int
		expected bool
	} {
		{ "Should be even", 4, true },
		{ "Should be not even", 9, false },
	}*/
	var parameters = []testInput {
		{ "4 should be even", 4, true },
		{ "9 should be not even", 9, false },
	}
	for _, entry := range parameters {
		t.Run(entry.name, func(t *testing.T) {
			result := IsEven(entry.value)
			if !result {
				t.Errorf("%v but result is %v", entry.name, result)
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