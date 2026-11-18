package solutions

import "fmt"

func ExampleCelsiusToFahrenheit() {
	fmt.Println(CelsiusToFahrenheit(100), CelsiusToFahrenheit(-40))
	fmt.Printf("%.2f\n", CelsiusToFahrenheit(36.6))
	// Output:
	// 212 -40
	// 97.88
}

func ExampleAverage3() {
	fmt.Printf("%.2f %.2f\n", Average3(1, 2, 2), Average3(10, 20, 30))
	// Output: 1.67 20.00
}

func ExampleCharStats() {
	fmt.Println(CharStats("Łódź 2026!"))
	fmt.Println(CharStats("Go"))
	// Output:
	// 13 4 4
	// 2 2 0
}

func ExampleAddStrings() {
	sum, err := AddStrings("40", "2")
	fmt.Printf("%q %v\n", sum, err)
	sum, err = AddStrings("4o", "2")
	fmt.Printf("%q %v\n", sum, err)
	// Output:
	// "42" <nil>
	// "" strconv.Atoi: parsing "4o": invalid syntax
}
