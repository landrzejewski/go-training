package concurrency

import "sync"

type Semaphore struct {
	mutex *sync.Mutex
	condition *sync.Cond
	permisions int
}

func NewSemaphore(permisions int) *Semaphore {
	mutex := sync.Mutex{}
	condition := sync.NewCond(&mutex)
	return &Semaphore{
		&mutex,	
		condition,
		permisions,
	}
}

func (s *Semaphore) Acquire() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	for s.permisions == 0 {
		s.condition.Wait()
	}
	s.permisions--
}

func (s *Semaphore) Release() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.permisions++
	s.condition.Signal()
}