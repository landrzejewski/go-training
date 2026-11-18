package solutions

// ---- Exercise 3.1: logical operators -----------------------------------------------------------

// IsLeapYear reports whether year is a leap year: divisible by 4, except centuries, which
// must be divisible by 400.
func IsLeapYear(year int) bool {
	return year%4 == 0 && (year%100 != 0 || year%400 == 0)
}

// ---- Exercise 3.2: the integer division trap ---------------------------------------------------

// Percentage returns part as a percentage of total, or 0 when total is 0.
func Percentage(part, total int) float64 {
	if total == 0 {
		return 0
	}
	// part/total*100 in integers is 0 for 1 out of 3 (1/3 == 0). Convert to float64 first.
	return float64(part) / float64(total) * 100
}

// ---- Exercise 3.3: / and % on digits -----------------------------------------------------------

// DigitSum returns the sum and the number of the decimal digits of n. A negative n is treated
// as its absolute value; 0 has a single digit.
func DigitSum(n int) (sum, count int) {
	if n == 0 {
		return 0, 1
	}
	if n < 0 {
		n = -n
	}
	for n > 0 {
		sum += n % 10 // the last digit
		count++
		n /= 10 // drop the last digit
	}
	return sum, count
}
