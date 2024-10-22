package concurrency

import (
	"fmt"
	"time"
)

func Channels() {
	channel := make(chan int)

	go func() {
		channel <- 1
		channel <- 2
	}()

	value, isClose := <-channel
	if !isClose {
		fmt.Printf("channel closed, value is %v", value)
	}
	fmt.Println(value)
	value = <-channel
	fmt.Println(value)

	// var sendChannel chan<- int = make(chan int)    // Send-only channel
	// var receiveChannel <-chan int = make(chan int) // Receive-only channel

	buffChannel := make(chan int, 3) // Create a buffered channel with a capacity of 3
	buffChannel <- 1                 // Non-blocking, buffer has space
	buffChannel <- 2                 // Non-blocking, buffer has space
	buffChannel <- 3                 // Non-blocking, buffer is now full
	/*
		You can close a channel using the close function to indicate that no more
		values will be sent on it. Receiving from a closed channel will continue
		to return values until the buffer is empty, after which it will return
		the zero value for the channel’s type.
	*/
	close(buffChannel)

	// You can use the range keyword to iterate over a channel until it is closed:

	/*ch := make(chan int)
	go func() {
		for i := 1; i <= 5; i++ {
			ch <- i
			time.Sleep(2 * time.Second)
		}
		close(ch) // Close the channel when done sending
	}()

	for value := range ch { // Iterate over the channel
		fmt.Println(value) // Prints values from 1 to 5
	}*/

	/*
		The select statement in Go allows you to wait on multiple channel operations.
		It chooses a case that is ready to proceed, allowing you to handle multiple channels
		concurrently. If multiple channels are ready, one of them is selected at random.
	*/

	ch1 := make(chan string)
	ch2 := make(chan string)

	go func() { ch1 <- "message from channel 1" }()
	go func() { ch2 <- "message from channel 2" }()

	for range 2 {
		select {
		case msg1 := <-ch1:
			fmt.Println(msg1)
		case msg2 := <-ch2:
			fmt.Println(msg2)
		case <-time.After(2 * time.Second):
			fmt.Println("Task timed out")
		}
	}

}
