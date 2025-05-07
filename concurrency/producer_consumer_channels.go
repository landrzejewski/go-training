package concurrency

import (
	"fmt"
	"sync"
	"time"
)

func channelProducer(index int, channel chan<- string, wg *sync.WaitGroup) {
	defer wg.Done()
	for i := 0; i < 10; i++ {
		time.Sleep(1 * time.Second)
		fmt.Println("Sending message", index, i)
		channel <- fmt.Sprintf("Data from producer %d - %d", index, i)
	}
	fmt.Println("Producer finished", index)
}

func channelConsumer(index int, channel <-chan string, wg *sync.WaitGroup) {
	defer wg.Done()
	done := false
	for !done {
		select {
		case message := <-channel:
			fmt.Printf("Consumer %d received: %v\n", index, message)
		case <-time.After(15 * time.Second):
			done = true
		}
	}
}

func ProducerConsumerChannels() {
	channel := make(chan string, 5)
	wg := sync.WaitGroup{}

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go channelProducer(i, channel, &wg)
	}

	for i := 0; i < 2; i++ {
		wg.Add(1)
		go channelConsumer(i, channel, &wg)
	}

	wg.Wait()
	close(channel)
}
