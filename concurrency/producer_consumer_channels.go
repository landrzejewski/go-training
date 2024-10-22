package concurrency

import (
	"fmt"
	"sync"
	"time"
)

func channelProducer(index int, ch chan string, wg *sync.WaitGroup) {
	for i := 0; i < 5; i++ {
		fmt.Println("Sending message", index)
		ch <- fmt.Sprintf("Data from producer %v with value %v", index, i)
		time.Sleep(1 * time.Second)
	}
	wg.Done()
}

func channelConsumer(index int, ch chan string, wg *sync.WaitGroup) {
	done := false
	for !done {
		select {
		case msg, isClosed := <-ch:
			if !isClosed {
				done = true
			}
			fmt.Printf("Consumer %v Received: %s\n", index, msg)
		case <-time.After(10 * time.Second):
			fmt.Println("Consumer is going down", index)
			wg.Done()
		}
	}
	wg.Done()
}

func ProducerConsumerChannels() {
	ch := make(chan string, 10)
	var wg sync.WaitGroup

	for i := 0; i < 4; i++ {
		wg.Add(1)
		go channelProducer(i, ch, &wg)
	}

	for i := 0; i < 2; i++ {
		wg.Add(1)
		go channelConsumer(i, ch, &wg)
	}

	wg.Wait()
	close(ch)
}
