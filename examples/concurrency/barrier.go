package concurrency

import "sync"

// Barrier is a cyclic barrier: Wait blocks until `total` participants have
// arrived, then releases them all at once and re-arms itself for the next round.
type Barrier struct {
	total int
	count int
	// generation increases every time the barrier opens. Without it a woken
	// goroutine would have no predicate left to test - by then b.count has
	// already been reset to b.total.
	generation int
	// The mutex is held by value and handed to the Cond as its Locker. A separate
	// *sync.Mutex field would be redundant: cond.L already refers to it.
	mutex sync.Mutex
	cond  *sync.Cond
}

// NewBarrier panics for size < 1: such a barrier could never open, so every
// Wait would block forever - a silent deadlock is far worse than a loud panic
// at construction time.
func NewBarrier(size int) *Barrier {
	if size < 1 {
		panic("concurrency: barrier size must be at least 1")
	}
	b := &Barrier{total: size, count: size}
	b.cond = sync.NewCond(&b.mutex)
	return b
}

// Wait blocks the caller until all participants of the current round have
// called Wait, then returns in every one of them. The barrier can then be used
// again for the next round.
func (b *Barrier) Wait() {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	generation := b.generation
	b.count--

	if b.count == 0 {
		// The last participant opens the barrier and starts a new round.
		b.count = b.total
		b.generation++
		b.cond.Broadcast()
		return
	}

	// sync.Cond.Wait releases the mutex while it waits, so the condition is NOT
	// automatically true once we wake up (we may also be woken by a Broadcast
	// meant for a different round). That is why Wait always goes inside a loop
	// testing the predicate - here: "has my round finished yet?".
	for generation == b.generation {
		b.cond.Wait()
	}
}
