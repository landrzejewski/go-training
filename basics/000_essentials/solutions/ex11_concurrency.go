package solutions

import (
	"errors"
	"sync"
	"time"
)

// ---- Exercise 11.1: goroutines and a WaitGroup -------------------------------------------------

// ParallelSum sums nums by splitting it into parts chunks that are summed in separate
// goroutines. parts below 1 is treated as 1.
func ParallelSum(nums []int, parts int) int {
	parts = max(parts, 1)
	size := (len(nums) + parts - 1) / parts // chunk length, rounded up
	partial := make([]int, parts)           // one slot per goroutine: no shared writes, no mutex

	var wg sync.WaitGroup
	for p := range parts {
		start := min(p*size, len(nums))
		end := min(start+size, len(nums))
		wg.Go(func() {
			for _, n := range nums[start:end] {
				partial[p] += n
			}
		})
	}
	wg.Wait() // every partial sum is ready after this line

	total := 0
	for _, s := range partial {
		total += s
	}
	return total
}

// ---- Exercise 11.2: channels and select --------------------------------------------------------

// ErrTimeout is returned by RunWithTimeout when f does not finish in time.
var ErrTimeout = errors.New("timeout")

// RunWithTimeout runs f in a goroutine and returns its result, or ErrTimeout if f takes
// longer than d.
func RunWithTimeout(f func() int, d time.Duration) (int, error) {
	// Buffered: if we stop waiting, a late f can still send and its goroutine can finish.
	// With an unbuffered channel it would block on the send forever (a goroutine leak).
	result := make(chan int, 1)
	go func() { result <- f() }()

	select {
	case v := <-result:
		return v, nil
	case <-time.After(d):
		return 0, ErrTimeout
	}
}

// ---- Exercise 11.3: a mutex around shared data -------------------------------------------------

// CountWordsConcurrently counts words across all texts, like WordCount (exercise 2b.2) but
// with one goroutine per text. The partial counts are merged into one map under a mutex.
func CountWordsConcurrently(texts []string) map[string]int {
	var (
		wg    sync.WaitGroup
		mu    sync.Mutex
		total = make(map[string]int)
	)
	for _, text := range texts {
		wg.Go(func() {
			counts := WordCount(text) // the work runs in parallel, without holding the lock

			mu.Lock() // only one goroutine at a time may write to total
			defer mu.Unlock()
			for word, n := range counts {
				total[word] += n
			}
		})
	}
	wg.Wait()
	return total
}
