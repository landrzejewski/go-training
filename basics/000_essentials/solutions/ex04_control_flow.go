package solutions

import "strconv"

// ---- Exercise 4.1: FizzBuzz --------------------------------------------------------------------

// FizzBuzz returns the numbers 1..n as strings, with multiples of 3 replaced by "Fizz",
// multiples of 5 by "Buzz" and multiples of both by "FizzBuzz".
func FizzBuzz(n int) []string {
	out := make([]string, 0, max(n, 0)) // a negative capacity would panic
	for i := 1; i <= n; i++ {
		switch { // a tagless switch: the first true case wins, so check 15 first
		case i%15 == 0:
			out = append(out, "FizzBuzz")
		case i%3 == 0:
			out = append(out, "Fizz")
		case i%5 == 0:
			out = append(out, "Buzz")
		default:
			out = append(out, strconv.Itoa(i))
		}
	}
	return out
}

// ---- Exercise 4.2: switch ----------------------------------------------------------------------

// Grade turns points from 0 to 100 into a letter: 90+ "A", 80+ "B", 70+ "C", 60+ "D" and
// below 60 "F". Points outside 0-100 give "invalid".
func Grade(points int) string {
	switch {
	case points < 0 || points > 100:
		return "invalid"
	case points >= 90:
		return "A"
	case points >= 80:
		return "B"
	case points >= 70:
		return "C"
	case points >= 60:
		return "D"
	default:
		return "F"
	}
}

// ---- Exercise 4.3: nested loops ----------------------------------------------------------------

// Primes returns all prime numbers up to and including limit, in increasing order.
func Primes(limit int) []int {
	var primes []int
	for n := 2; n <= limit; n++ {
		isPrime := true
		for d := 2; d*d <= n; d++ { // a divisor larger than sqrt(n) would pair with a smaller one
			if n%d == 0 {
				isPrime = false
				break // one divisor is enough: leave the inner loop
			}
		}
		if isPrime {
			primes = append(primes, n)
		}
	}
	return primes
}
