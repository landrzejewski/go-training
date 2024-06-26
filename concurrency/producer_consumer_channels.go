package concurrency

import (
	"fmt"
	"sync"
	"time"
)

func producer(index int, ch chan string, wg *sync.WaitGroup) {
	for i := 0; i < 5; i++ {
		ch <- fmt.Sprintf("Producer %v send %v", index, i)
	}
	wg.Done()
}
   
func consumer(index int, ch chan string, wg *sync.WaitGroup) {
	done := false
	for !done {
		select {
	 	case msg, ok := <-ch:
	  		if !ok {
	   			done = true
	  		}
	  		fmt.Printf("Consumer %v Received: %s\n", index, msg)
	 	case <-time.After(1 * time.Second):
	  		wg.Done()
	 	}
   
	}
	wg.Done()
}

func ProducerConsumer() {
	ch := make(chan string)
	var wg sync.WaitGroup
   
	for i := 0; i < 50; i++ {
		wg.Add(1)
	 	go producer(i, ch, &wg)
	}
	
	for i := 0; i < 100; i++ {
		wg.Add(1)
	 	go consumer(i, ch, &wg)
	}
   
	wg.Wait()
	close(ch)
}