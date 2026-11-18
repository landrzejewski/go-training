package solutions

import "fmt"

func ExampleMaxOf() {
	nums := []int{4, 8, 1}
	fmt.Println(MaxOf(3), MaxOf(3, 9, 2), MaxOf(nums[0], nums[1:]...))
	// Output: 3 9 8
}

func ExampleNewIDGenerator() {
	userID := NewIDGenerator("user")
	orderID := NewIDGenerator("order")
	fmt.Println(userID(), userID(), orderID(), userID()) // two independent counters
	// Output: user-1 user-2 order-1 user-3
}

func ExampleCalculate() {
	fmt.Println(Calculate(6, 3, "*"))
	fmt.Println(Calculate(1, 0, "/"))
	fmt.Println(Calculate(2, 2, "^"))
	// Output:
	// 18 <nil>
	// 0 division by zero
	// 0 unknown operator "^"
}
