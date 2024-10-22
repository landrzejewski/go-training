package concurrency

import "sync"

type Barrier struct {
	total int
	count int
	mutex *sync.Mutex
	cond  *sync.Cond
}

func NewBarrier(size int) *Barrier {
	mutex := &sync.Mutex{}
	condition := sync.NewCond(mutex)
	return &Barrier{size, size, mutex, condition}
}

func (b *Barrier) Wait() {
	b.mutex.Lock()
	b.count -= 1
	if b.count == 0 {
		b.count = b.total
		b.cond.Broadcast()
	} else {
		b.cond.Wait()
	}
	b.mutex.Unlock()
}
