package solutions

import "fmt"

// IsLeapYear is tested in ex14_testing_test.go - writing that test is exercise 14.1.
func ExampleIsLeapYear() {
	fmt.Println(IsLeapYear(2024), IsLeapYear(2023), IsLeapYear(1900), IsLeapYear(2000))
	// Output: true false false true
}

func ExamplePercentage() {
	fmt.Printf("%.2f %.2f %.2f\n", Percentage(1, 3), Percentage(1, 4), Percentage(5, 0))
	// Output: 33.33 25.00 0.00
}

func ExampleDigitSum() {
	fmt.Println(DigitSum(1234))
	fmt.Println(DigitSum(-505))
	fmt.Println(DigitSum(0))
	// Output:
	// 10 4
	// 10 3
	// 0 1
}
