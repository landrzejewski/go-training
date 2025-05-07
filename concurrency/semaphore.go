package concurrency

import "sync"

type Semaphore struct {
	mutex       *sync.Mutex
	condition   *sync.Cond
	permissions int
}

func NewSemaphore(permissions int) *Semaphore {
	mutex := sync.Mutex{}
	condition := sync.NewCond(&mutex)
	return &Semaphore{
		&mutex,
		condition,
		permissions,
	}
}

func (s *Semaphore) Acquire() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	for s.permissions == 0 {
		s.condition.Wait()
	}
	s.permissions--
}

func (s *Semaphore) Release() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.permissions++
	s.condition.Signal()
}
