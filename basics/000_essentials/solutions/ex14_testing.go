package solutions

import "strings"

// ---- Exercise 14.3: two implementations to benchmark ------------------------------------------
// (Exercises 14.1-14.3 are about writing tests. Their solutions are in ex14_testing_test.go.)

// JoinPlus joins parts with sep using +=. It is simple, but every += copies the whole string
// built so far, so the cost grows with the square of the length.
func JoinPlus(parts []string, sep string) string {
	result := ""
	for i, p := range parts {
		if i > 0 {
			result += sep
		}
		result += p
	}
	return result
}

// JoinBuilder does the same with a strings.Builder, which appends to one growing buffer.
func JoinBuilder(parts []string, sep string) string {
	var sb strings.Builder
	for i, p := range parts {
		if i > 0 {
			sb.WriteString(sep)
		}
		sb.WriteString(p)
	}
	return sb.String()
}
