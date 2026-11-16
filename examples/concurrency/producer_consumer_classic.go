package concurrency

import (
	"fmt"
	"sync"

	"training.pl/go/examples/common"
)

// boundedBuffer is the pre-channel way to write producer/consumer: shared state
// under one mutex, with a separate sync.Cond per waiting side. Two conditions
// on ONE mutex is the point of the example - a producer blocked on "buffer is
// full" and a consumer blocked on "buffer is empty" must not wake each other
// up.
//
// The closed flag is what lets the demo terminate no matter how many producers
// and consumers there are: once every producer is done, close() wakes the
// consumers, which drain what is left and return instead of waiting on a
// condition nobody will signal again.
type boundedBuffer struct {
	mutex    sync.Mutex
	notFull  *sync.Cond // producers wait here
	notEmpty *sync.Cond // consumers wait here
	items    common.Stack[int]
	capacity int
	closed   bool
}

func newBoundedBuffer(capacity int) *boundedBuffer {
	b := &boundedBuffer{capacity: capacity}
	b.notFull = sync.NewCond(&b.mutex)
	b.notEmpty = sync.NewCond(&b.mutex)
	return b
}

// put blocks while the buffer is full. It panics when called after close, like
// a send on a closed channel would.
func (b *boundedBuffer) put(item int) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	// Always test the predicate in a loop: Wait releases the mutex, so the
	// condition may no longer hold by the time we are woken up.
	for b.items.Size() >= b.capacity && !b.closed {
		fmt.Println("Producer waiting - buffer is full")
		b.notFull.Wait()
	}
	if b.closed {
		panic("concurrency: put on a closed boundedBuffer")
	}
	b.items.Push(item)
	b.notEmpty.Signal()
}

// take blocks while the buffer is empty. The boolean is false once the buffer
// has been closed and drained - the same contract as a receive from a channel.
func (b *boundedBuffer) take() (int, bool) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	for b.items.Size() == 0 && !b.closed {
		fmt.Println("Consumer waiting - buffer is empty")
		b.notEmpty.Wait()
	}
	item, ok := b.items.Pop()
	if !ok {
		return 0, false // closed and empty
	}
	b.notFull.Signal()
	return item, true
}

// close marks the end of production and wakes every waiter so it can re-check
// the predicate.
func (b *boundedBuffer) close() {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	b.closed = true
	b.notEmpty.Broadcast()
	b.notFull.Broadcast()
}

func producer(id, count int, buffer *boundedBuffer, wg *sync.WaitGroup) {
	defer wg.Done()
	for i := range count {
		fmt.Printf("Producer %d: producing %d\n", id, i)
		buffer.put(i)
	}
}

func consumer(id int, buffer *boundedBuffer, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		item, ok := buffer.take()
		if !ok {
			fmt.Printf("Consumer %d: buffer closed, done\n", id)
			return
		}
		fmt.Printf("Consumer %d: consuming %d\n", id, item)
	}
}

// ProducerConsumerClassic runs two producers and one consumer over a bounded
// buffer guarded by a mutex and two conditions. Unlike an earlier version that
// relied on the producer and consumer counts balancing exactly, the buffer is
// closed once the producers finish, so any combination of counts terminates.
func ProducerConsumerClassic() {
	buffer := newBoundedBuffer(10)

	var producers, consumers sync.WaitGroup
	for id := range 2 {
		producers.Add(1)
		go producer(id, 100, buffer, &producers)
	}
	for id := range 1 {
		consumers.Add(1)
		go consumer(id, buffer, &consumers)
	}

	producers.Wait()
	buffer.close() // the producing side closes, once every producer has finished
	consumers.Wait()
}
