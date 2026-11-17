package solutions

import "fmt"

func ExampleSplitDuration() {
	fmt.Println(SplitDuration(90061)) // 1 day + 1 hour + 1 minute + 1 second
	fmt.Println(SplitDuration(59))
	fmt.Println(SplitDuration(SecondsPerDay * 3))
	// Output:
	// 1 1 1 1
	// 0 0 0 59
	// 3 0 0 0
}

func ExampleNextLight() {
	l := Red
	for range 4 {
		fmt.Print(LightName(l), " -> ")
		l = NextLight(l)
	}
	fmt.Println(LightName(l))
	// Output: red -> yellow -> green -> red -> yellow
}

func ExampleLightName() {
	fmt.Println(LightName(Green), LightName(TrafficLight(7)))
	// Output: green unknown
}

func ExampleSumStrings() {
	fmt.Println(SumStrings([]string{"1", "2", "3"}))
	fmt.Println(SumStrings([]string{"1", "x", "3"}))
	// Output:
	// 6 <nil>
	// 0 strconv.Atoi: parsing "x": invalid syntax
}
