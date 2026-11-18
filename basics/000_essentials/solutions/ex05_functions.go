package solutions

import (
	"errors"
	"fmt"
	"strconv"
)

// ---- Exercise 5.1: variadic functions ----------------------------------------------------------

// MaxOf returns the largest of its arguments. Making the first argument a separate parameter
// means the compiler rejects MaxOf() with no arguments - there is no empty case to handle.
func MaxOf(first int, rest ...int) int {
	largest := first
	for _, v := range rest { // rest is a []int
		if v > largest {
			largest = v
		}
	}
	return largest
}

// ---- Exercise 5.2: closures --------------------------------------------------------------------

// NewIDGenerator returns a function that produces "prefix-1", "prefix-2", ... Each generator
// has its own counter, captured by the closure.
func NewIDGenerator(prefix string) func() string {
	n := 0
	return func() string {
		n++ // n survives between calls because the closure keeps it alive
		return prefix + "-" + strconv.Itoa(n)
	}
}

// ---- Exercise 5.3: functions as values ---------------------------------------------------------

// operations is a dispatch table: operator -> the function that implements it.
var operations = map[string]func(a, b float64) float64{
	"+": func(a, b float64) float64 { return a + b },
	"-": func(a, b float64) float64 { return a - b },
	"*": func(a, b float64) float64 { return a * b },
	"/": func(a, b float64) float64 { return a / b },
}

// Calculate applies op ("+", "-", "*" or "/") to a and b. It returns an error for an
// unknown operator and for division by zero.
func Calculate(a, b float64, op string) (float64, error) {
	f, ok := operations[op]
	if !ok {
		return 0, fmt.Errorf("unknown operator %q", op)
	}
	if op == "/" && b == 0 {
		return 0, errors.New("division by zero")
	}
	return f(a, b), nil
}
