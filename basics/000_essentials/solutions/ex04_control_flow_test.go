package solutions

import (
	"fmt"
	"strings"
)

func ExampleFizzBuzz() {
	fmt.Println(strings.Join(FizzBuzz(15), " "))
	fmt.Println(len(FizzBuzz(0)))
	// Output:
	// 1 2 Fizz 4 Buzz Fizz 7 8 Fizz Buzz 11 Fizz 13 14 FizzBuzz
	// 0
}

func ExampleGrade() {
	for _, p := range []int{95, 80, 61, 12, 101} {
		fmt.Println(p, Grade(p))
	}
	// Output:
	// 95 A
	// 80 B
	// 61 D
	// 12 F
	// 101 invalid
}

func ExamplePrimes() {
	fmt.Println(Primes(30))
	fmt.Println(Primes(1))
	// Output:
	// [2 3 5 7 11 13 17 19 23 29]
	// []
}
