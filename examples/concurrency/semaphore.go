package concurrency

import "sync"

// Semaphore is a counting semaphore: it holds a pool of `permissions` permits,
// Acquire takes one (blocking while the pool is empty) and Release gives one back.
//
// Idiomatic Go would usually reach for a buffered channel instead - acquire by
// sending, release by receiving. This sync.Cond version exists to show the
// primitive that a channel hides.
type Semaphore struct {
	// The mutex is held by value and handed to the Cond as its Locker; cond.L
	// already refers to it, so a separate *sync.Mutex field would be redundant.
	mutex       sync.Mutex
	condition   *sync.Cond
	permissions int
	max         int
}

// NewSemaphore panics for permissions < 1: a semaphore with no permits can never
// be acquired, so every Acquire would block forever.
func NewSemaphore(permissions int) *Semaphore {
	if permissions < 1 {
		panic("concurrency: semaphore needs at least 1 permit")
	}
	s := &Semaphore{permissions: permissions, max: permissions}
	s.condition = sync.NewCond(&s.mutex)
	return s
}

// Acquire takes one permit, blocking until one is available.
func (s *Semaphore) Acquire() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	for s.permissions == 0 {
		s.condition.Wait()
	}
	s.permissions--
}

// Release returns one permit and wakes one goroutine blocked in Acquire.
//
// Release panics when it is called more times than Acquire: an unmatched
// Release would conjure a permit out of thin air and silently raise the limit,
// and a programming error like that is better caught loudly.
func (s *Semaphore) Release() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.permissions >= s.max {
		panic("concurrency: semaphore Release without a matching Acquire")
	}
	s.permissions++
	// Signal is enough: we released exactly one permit, so we wake exactly one
	// goroutine waiting in Acquire.
	s.condition.Signal()
}
