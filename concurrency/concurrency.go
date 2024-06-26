package concurrency

import (
)

/*
import (
	"fmt"
	"time"
)

func print(text string) {
	fmt.Println(text)
}

func Run() {
	go print("Hello")
	time.Sleep(1 * time.Second)
	print("Go")
}
*/

// WaitGroup

/*
func print(text string, wg *sync.WaitGroup) {
	defer wg.Done()
	fmt.Println(text)
}

func Run() {
	wg := sync.WaitGroup{}
	letters := []string{"a", "b", "c", "d", "e", "f"}
	wg.Add(len(letters))
	for _, l := range letters {
		go print(l, &wg)
	}
	wg.Wait()
	wg.Add(1)
	print("Done", &wg)
}
*/

// Mutex

/*
var message string = "Initial text"
var wg sync.WaitGroup

func updateMessage(newValue string) {
	defer wg.Done()
	fmt.Printf("Before: %v\n", message)
	message = newValue
	fmt.Printf("After: %v\n", message)
}

func Run() {
	wg.Add(2)
	go updateMessage("Hello")
	go updateMessage("Go")
	wg.Wait()
	fmt.Println(message)
}
*/

/*
var message string = "Initial text"
var wg sync.WaitGroup

func updateMessage(newValue string, mutex *sync.Mutex) {
	defer wg.Done()
	mutex.Lock()
	fmt.Printf("Before: %v\n", message)
	message = newValue
	fmt.Printf("After: %v\n", message)
	mutex.Unlock()
}

func Run() {
	mutex := sync.Mutex{}

	wg.Add(2)
	go updateMessage("Hello", &mutex)
	go updateMessage("Go", &mutex)
	wg.Wait()
	fmt.Println(message)
}
*/

// RWMutex

/*
type safeSlice struct {
	mutex sync.RWMutex
	data []int
}

func (ss *safeSlice) add(value int) {
	ss.mutex.Lock()
	ss.data = append(ss.data, value)
	ss.mutex.Unlock()
}

func (ss *safeSlice) get(index int) (int, bool) {
	ss.mutex.RLock()
	defer ss.mutex.RUnlock()
	if index < 0 || index >= len(ss.data) {
		return 0, false
	}
	return ss.data[index], true
}

func (ss *safeSlice) size() int {
	ss.mutex.RLock()
	defer ss.mutex.RUnlock()
	return len(ss.data)
}

func newSafeSlice() *safeSlice {
	return &safeSlice{
		data: make([]int, 0),
	}
}

func Run() {
	ss := newSafeSlice()

	go func () {
		ss.add(1)
		ss.add(2)
	}()

	go func () {
		fmt.Printf("Size: %d\n", ss.size())
	}()

	time.Sleep(1 * time.Second)
}
*/

// Mutex i WaitGroup

/*
var wg sync.WaitGroup

type Income struct {
	Source string
	Amount int
}

func main() {
	var bankBalance int
	var balance sync.Mutex

	fmt.Printf("Initial account balance: $%d.00", bankBalance)
	fmt.Println()

	incomes := []Income{
		{Source: "Main job", Amount: 500},
		{Source: "Gifts", Amount: 10},
		{Source: "Part time job", Amount: 50},
		{Source: "Investments", Amount: 100},
	}

	wg.Add(len(incomes))

	for i, income := range incomes {

		go func(i int, income Income) {
			defer wg.Done()

			for week := 1; week <= 52; week++ {
				balance.Lock()
				temp := bankBalance
				temp += income.Amount
				bankBalance = temp
				balance.Unlock()

				fmt.Printf("On week %d, you earned $%d.00 from %s\n", week, income.Amount, income.Source)
			}
		}(i, income)
	}

	wg.Wait()

	fmt.Printf("Final bank balance: $%d.00", bankBalance)
	fmt.Println()
}
*/

// Mutex + Signals

/*
var (
	money = 100
	mutex = sync.Mutex{}
	moneyIsGraterThanZero = sync.NewCond(&mutex)
	spendValue = 10
)

func spend() {
	for i := 1; i < 500; i++ {
		mutex.Lock()
		for money - spendValue < 10 {
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
		// moneyIsGraterThanZero.Broadcast() // all threads
		moneyIsGraterThanZero.Signal()  // one random thread
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

// Deadlocks

/*
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

// Atomics

/*
var (
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
}
*/

// Cyclic barrier

/*
func execute(name string, sleepTime int, barrier *Barrier) {
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
}
*/

// Semaphore

/*
func Run() {
	semaphore := NewSemaphore(5)
	for i := 0; i < 100; i++ {
		go func ()  {
			semaphore.Acquire()
			fmt.Println("Working", i)
			time.Sleep(2 * time.Second)
			fmt.Println("Releaseing permit", i)
			semaphore.Release()
		}()
	}

	time.Sleep(100 * time.Second)
}
*/

// Channels

/*
func execute(in <-chan string, out chan<- string) {
	for {
		// message, isCLosed := <-in
		message := <-in
		out <- fmt.Sprintf("Echo: %s", message)
	}
}

func Run() {
	in := make(chan string)
	out := make(chan string)
	go execute(in, out)
	var input string
	for {
		fmt.Scanln(&input)
		if input == "q" {
			break
		}
		in <- input
		echo := <- out
		fmt.Println(echo)
	}
	close(in)
	close(out)
}
*/

/*
func producerOne(channel chan string) {
	for {
		time.Sleep(5 * time.Second)
		channel <- "Data from producer one"
	}
}

func producerTwo(channel chan string) {
	for {
		time.Sleep(2 * time.Second)
		channel <- "Data from producer two"
	}
}

func Run() {
	channelOne := make(chan string)
	channelTwo := make(chan string)

	go producerOne(channelOne)
	go producerTwo(channelTwo)

	for {
		select {
		case messageOne := <- channelOne:
			fmt.Println(messageOne, "case1") // w przypadku case z tym samym źródłem wybierany jest przypadkowy 
		case messageOne := <- channelOne:
			fmt.Println(messageOne, "case2")
		case messageTwo := <- channelTwo:
			fmt.Println(messageTwo, "case3")
		}
	}
}
*/

/*
func listener(channel chan int) {
	for {
		value := <- channel
		fmt.Println("Consuming", value)
		time.Sleep(1 * time.Second)
	}
}

func Run() {
	channel := make(chan int, 10)
	defer close(channel)
	go listener(channel)

	for i := 0; i <= 50; i++ {
		channel <- i
		fmt.Println("Sent", i)
	}
	fmt.Println("Done")
}
*/

func Run() {
	Orders()
}

// https://en.wikipedia.org/wiki/Dining_philosophers_problem
// https://en.wikipedia.org/wiki/Sleeping_barber_problem