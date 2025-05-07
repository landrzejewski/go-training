package concurrency

import (
	"fmt"
	"sync"
	"time"
)

/*
func Run() {
	go printText("Hello")
	fmt.Println("Before sleep")
	time.Sleep(5 * time.Second)
}

func printText(text string) {
	fmt.Println(text)
}
*/

/*
// WaitGroup
func Run() {
	wg := sync.WaitGroup{}
	letters := []string{"a", "b", "c", "d", "e", "f", "g"}
	wg.Add(len(letters))
	for _, letter := range letters {
		go printText(letter, &wg)
	}
	wg.Wait()
	fmt.Println("Done")
}

func printText(text string, wg *sync.WaitGroup) {
	fmt.Println(text)
	wg.Done()
}
*/

/*
// Mutex
var counter = 0

func Run() {
	n := 1000000
	wg := sync.WaitGroup{}
	mutex := sync.Mutex{}
	wg.Add(n)
	for i := 0; i < n; i++ {
		go increment(&wg, &mutex)
	}
	wg.Wait()
	fmt.Println(counter)
}

func increment(wg *sync.WaitGroup, mutex *sync.Mutex) {
	defer wg.Done()
	defer mutex.Unlock()
	mutex.Lock()
	counter += 1
}
*/

/*
// RWMutex
type safeSlice[T any] struct {
	mutex sync.RWMutex
	data  []T
}

func (ss *safeSlice[T]) add(value T) {
	ss.mutex.Lock()
	ss.data = append(ss.data, value)
	ss.mutex.Unlock()
}

func (ss *safeSlice[T]) get(index int) (T, bool) {
	ss.mutex.RLock()
	defer ss.mutex.RUnlock()
	if index < 0 || index >= len(ss.data) {
		var empty T
		return empty, false
	}
	return ss.data[index], true
}

func (ss *safeSlice[T]) size() int {
	ss.mutex.RLock()
	defer ss.mutex.RUnlock()
	return len(ss.data)
}

func Run() {
	ss := safeSlice[int]{}

	go func() {
		ss.add(1)
		ss.add(2)
	}()

	go func() {
		fmt.Printf("Size: %d\n", ss.size())
	}()

	time.Sleep(3 * time.Second)
}
*/

/*
// Mutex + Signals
var (
	money                 = 100
	mutex                 = sync.Mutex{}
	moneyIsGraterThanZero = sync.NewCond(&mutex)
	spendValue            = 10
)

func spend() {
	for i := 1; i < 500; i++ {
		mutex.Lock()
		for money-spendValue < 10 {
			moneyIsGraterThanZero.Wait()
		}
		money -= spendValue
		fmt.Println("Spend: ", money)
		mutex.Unlock()
		time.Sleep(1 * time.Millisecond)
	}
	fmt.Println("Spend: Done")
}

func work() {
	for i := 1; i < 500; i++ {
		mutex.Lock()
		money += 5
		fmt.Println("New income, current value:", money)
		//moneyIsGraterThanZero.Broadcast() // all threads
		moneyIsGraterThanZero.Signal() // one random thread
		mutex.Unlock()
		time.Sleep(1 * time.Millisecond)
	}
	fmt.Println("Work: Done")
}

func Run() {
	go work()
	go spend()

	time.Sleep(10 * time.Second)
	fmt.Println("Current value:", money)
}
*/

/*
// Deadlocks
var (
	lock1 = sync.Mutex{}
	lock2 = sync.Mutex{}
)

func blue() {
	for {
		fmt.Println("Blue: Acquiring lock1")
		lock1.Lock()
		fmt.Println("Blue: Acquiring lock2")
		lock2.Lock()
		fmt.Println("Blue: Both locks Acquired")
		lock1.Unlock()
		lock2.Unlock()
		fmt.Println("Blue: Locks Released")
	}
}

func red() {
	for {
		fmt.Println("Red: Acquiring lock2")
		lock2.Lock()
		fmt.Println("Red: Acquiring lock1")
		lock1.Lock()
		fmt.Println("Red: Both locks Acquired")
		lock1.Unlock()
		lock2.Unlock()
		fmt.Println("Red: Locks Released")
	}
}

func Run() {
	go red()
	go blue()
	time.Sleep(20 * time.Second)
	fmt.Println("Done")
}
*/

// https://dev.to/ietxaniz/go-deadlock-detection-delock-library-1eig

// Atomics
/*var (
	money int64 = 100
	value = 10
)

func spend() {
	for i := 1; i < 500; i++ {
		atomic.AddInt64(&money, int64(-value))
		fmt.Println("Spend: ", money)
		time.Sleep(1 * time.Millisecond)
	}
	fmt.Println("Spend: Done")
}

func work() {
	for i := 1; i < 500; i++ {
		atomic.AddInt64(&money, int64(value))
		fmt.Println("New income, current value:", money)
		time.Sleep(1 * time.Millisecond)
	}
	fmt.Println("Work: Done")
}

func Run() {
	go work()
	go spend()

	time.Sleep(10 * time.Second)
	fmt.Println("Current value:", money)
}*/

// Cyclic barrier

/*func execute(name string, sleepTime int, barrier *Barrier) {
	for {
		println(name, "running")
		time.Sleep(time.Duration(sleepTime) * time.Second)
		println(name, "is waiting on barrier")
		barrier.Wait()
	}
}

func Run() {
	barrier := NewBarrier(3)
	go execute("One", 3, barrier)
	go execute("Two", 10, barrier)
	go execute("Three", 6, barrier)
	time.Sleep(100 * time.Second)
}*/

// Semaphore

/*func Run() {
	semaphore := NewSemaphore(5)
	for i := 0; i < 100; i++ {
		go func() {
			semaphore.Acquire()
			fmt.Println("Working", i)
			time.Sleep(2 * time.Second)
			fmt.Println("Releasing permit", i)
			semaphore.Release()
		}()
	}

	time.Sleep(100 * time.Second)
}*/
