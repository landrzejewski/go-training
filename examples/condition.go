package examples

// IsEven reports whether value is divisible by two. It exists mainly as the
// subject of the unit-test and benchmark examples in condition_test.go.
func IsEven(value int) bool {
	return value%2 == 0
}
