package solutions

import (
	"fmt"
	"time"
)

func ExampleParallelSum() {
	nums := make([]int, 100)
	for i := range nums {
		nums[i] = i + 1 // 1..100
	}
	fmt.Println(ParallelSum(nums, 4), ParallelSum(nums, 7), ParallelSum(nums, 0), ParallelSum(nil, 3))
	// Output: 5050 5050 5050 0
}

func ExampleRunWithTimeout() {
	fast := func() int { return 42 }
	slow := func() int {
		time.Sleep(time.Second)
		return 1
	}
	fmt.Println(RunWithTimeout(fast, time.Second))
	fmt.Println(RunWithTimeout(slow, 10*time.Millisecond))
	// Output:
	// 42 <nil>
	// 0 timeout
}

func ExampleCountWordsConcurrently() {
	texts := []string{"go is fun", "Go is fast", "fun fun fun"}
	fmt.Println(CountWordsConcurrently(texts))
	// Output: map[fast:1 fun:4 go:2 is:2]
}
