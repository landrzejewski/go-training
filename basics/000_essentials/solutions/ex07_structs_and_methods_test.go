package solutions

import "fmt"

func ExampleRectangle() {
	r := Rectangle{Width: 3, Height: 4}
	fmt.Println(r, r.Area(), r.Perimeter())
	r.Scale(2) // Go takes &r automatically for the pointer receiver
	fmt.Println(r, r.Area(), r.Perimeter())
	// Output:
	// 3x4 12 14
	// 6x8 48 28
}

func ExampleShoppingCart() {
	cart := NewCart()
	cart.Add("apple", 0.5, 4)
	cart.Add("bread", 3.2, 1)
	cart.Add("apple", 0.5, 2) // now 6 apples
	fmt.Printf("%.2f\n", cart.Total())
	fmt.Println(cart.Remove("bread"), cart.Remove("milk"))
	fmt.Printf("%.2f\n", cart.Total())
	// Output:
	// 6.20
	// true false
	// 3.00
}

func ExampleCar() {
	car := Car{Vehicle: Vehicle{Brand: "Fiat", Wheels: 4}, Doors: 5}
	fmt.Println(car.Brand) // promoted field
	fmt.Println(car.Describe())
	fmt.Println(car.Vehicle.Describe())
	// Output:
	// Fiat
	// Fiat with 4 wheels and 5 doors
	// Fiat with 4 wheels
}
