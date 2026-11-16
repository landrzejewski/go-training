package concurrency

import (
	"fmt"
	"sync"
	"time"
)

func channelProducer(index int, channel chan<- string, wg *sync.WaitGroup) {
	defer wg.Done()
	for i := 0; i < 10; i++ {
		time.Sleep(100 * time.Millisecond)
		fmt.Println("Sending message", index, i)
		channel <- fmt.Sprintf("Data from producer %d - %d", index, i)
	}
	fmt.Println("Producer finished", index)
}

// channelConsumer returns only once the channel is closed AND drained - which is
// exactly what `for range` over a channel does. No message is lost and no timeout
// is needed.
func channelConsumer(index int, channel <-chan string, wg *sync.WaitGroup) {
	defer wg.Done()
	for message := range channel {
		fmt.Printf("Consumer %d received: %v\n", index, message)
	}
	fmt.Println("Consumer finished", index)
}

func ProducerConsumerChannels() {
	channel := make(chan string, 5)

	// Two separate wait groups: we close the channel once the producers are done
	// (they are the senders), and only then wait for the consumers so they can
	// drain what is left in the buffer.
	//
	// An earlier version stopped the consumers with time.After(15s). That was wrong
	// for three reasons: (1) if both consumers had dropped out while producers were
	// still running, the 5-slot buffer would fill up and every producer would block
	// forever on `channel <-`, deadlocking wg.Wait(); (2) messages still sitting in
	// the buffer were lost; (3) close(channel) after everyone had finished was a
	// no-op anyway.
	var producers, consumers sync.WaitGroup

	for i := 0; i < 5; i++ {
		producers.Add(1)
		go channelProducer(i, channel, &producers)
	}

	for i := 0; i < 2; i++ {
		consumers.Add(1)
		go channelConsumer(i, channel, &consumers)
	}

	producers.Wait()
	close(channel) // the sending side closes, once every sender has finished
	consumers.Wait()
}
