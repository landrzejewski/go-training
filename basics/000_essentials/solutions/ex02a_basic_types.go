package solutions

import (
	"strconv"
	"unicode"
)

// ---- Exercise 2a.1: explicit conversions -------------------------------------------------------

// CelsiusToFahrenheit converts degrees Celsius to degrees Fahrenheit: F = C*9/5 + 32.
func CelsiusToFahrenheit(c float64) float64 {
	return c*9/5 + 32
}

// Average3 returns the average of three integers as a float64.
func Average3(a, b, c int) float64 {
	// Convert BEFORE dividing. float64((a+b+c)/3) would do integer division first and lose
	// the fraction: the average of 1, 2, 2 would come out as 1 instead of 1.67.
	return float64(a+b+c) / 3
}

// ---- Exercise 2a.2: bytes, runes and characters ------------------------------------------------

// CharStats returns the length of s in bytes and the number of letters and digits in it.
func CharStats(s string) (bytes, letters, digits int) {
	for _, r := range s { // range decodes UTF-8, so r is a whole character (a rune)
		if unicode.IsLetter(r) {
			letters++
		} else if unicode.IsDigit(r) {
			digits++
		}
	}
	return len(s), letters, digits // len counts bytes, not characters
}

// ---- Exercise 2a.3: strconv --------------------------------------------------------------------

// AddStrings adds two integers given as strings and returns the sum as a string. If either
// input is not an integer it returns "" and the parsing error.
func AddStrings(a, b string) (string, error) {
	x, err := strconv.Atoi(a)
	if err != nil {
		return "", err
	}
	y, err := strconv.Atoi(b) // := is allowed: y is new, err is simply reused
	if err != nil {
		return "", err
	}
	return strconv.Itoa(x + y), nil // NOT string(x + y): that would give a character
}
