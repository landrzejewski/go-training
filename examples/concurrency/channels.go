package concurrency

import (
	"fmt"
	"sync"
	"time"
)

// ChannelsDemo shows two channel idioms: waiting on several channels with
// select (including a timeout case), and a broadcaster fanning messages out to
// a set of listeners, with clean shutdown by closing channels in the right
// order.
func ChannelsDemo() {
	// Capacity 1 so that neither sender is stranded if the timeout branch below
	// ever wins: on an unbuffered channel the unmatched `channel <- n` goroutine
	// would block forever, which is a goroutine leak. (The timer itself is not a
	// leak - since Go 1.23 an unreferenced Timer is garbage-collected even if it
	// has not fired.)
	channel1 := make(chan int, 1)
	channel2 := make(chan int, 1)

	go func() { channel1 <- 1 }()
	go func() { channel2 <- 2 }()

	// select waits on several channel operations at once and runs the case that
	// is ready. If several cases are ready, one is picked uniformly at random -
	// the order of the cases in the source code does not give any of them
	// priority.
	for range 2 {
		select {
		case msg1 := <-channel1:
			fmt.Println("msg1", msg1)
		case msg2 := <-channel2:
			fmt.Println("msg2", msg2)
		case <-time.After(time.Second * 5):
		}
	}

	channel := make(chan string)
	listeners := make([]chan string, 3)

	var wg sync.WaitGroup
	for i := range listeners {
		listeners[i] = make(chan string)
		wg.Add(1)
		go func() {
			defer wg.Done()
			listener(i, listeners[i])
		}()
	}
	go broadcaster(channel, listeners)

	for i := range 5 {
		channel <- fmt.Sprintf("Message %d", i)
	}

	// Closing the source channel ends the broadcaster, which in its defer closes
	// the listener channels - so every goroutine finishes on its own and nothing
	// leaks.
	close(channel)

	// Wait for the listeners to drain what is still queued. An earlier version
	// used time.Sleep here, which is not a synchronisation primitive: the demo
	// could return while the listeners were still printing.
	wg.Wait()
}

func listener(id int, channel <-chan string) {
	for msg := range channel {
		fmt.Println(msg, id)
	}
}

func broadcaster(channel <-chan string, listeners []chan string) {
	// The broadcaster is the sole sender on the listener channels, so it is the one
	// that must close them. Without that the `for range` in each listener never
	// ends, and closing `channel` leaves three goroutines hanging.
	defer func() {
		for _, listener := range listeners {
			close(listener)
		}
	}()
	for msg := range channel {
		for _, listener := range listeners {
			listener <- msg
		}
	}
}
