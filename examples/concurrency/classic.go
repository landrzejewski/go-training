// This file is a walkthrough of the classic synchronisation primitives -
// goroutines, WaitGroup, Mutex, RWMutex, Cond, deadlocks, atomics, the Barrier
// and the Semaphore. Every snippet is deliberately commented out (they all
// define a Run function and could not coexist), so nothing here is compiled:
// uncomment one block at a time, add the imports it needs, and run it.

package concurrency

/*func Run() {
	go printText("Hello")
	fmt.Println("Before sleep")
	time.Sleep(5 * time.Second)
}

func printText(text string) {
	fmt.Println(text)
}
*/

/*// WaitGroup
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
	defer wg.Done()
	fmt.Println(text)
}*/

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
	// Idiom: Lock first, and only then defer Unlock. The reverse order happens to
	// work here, but any return or panic inserted between the defer and the Lock
	// gives "fatal error: sync: unlock of unlocked mutex".
	mutex.Lock()
	defer mutex.Unlock()
	counter += 1
}*/

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
}*/

/*// Mutex + Signals
var (
	money = 100
	mutex = sync.Mutex{}
	// The name describes the real predicate of the Wait loop below: money-spendValue >= 10.
	enoughMoney = sync.NewCond(&mutex)
	spendValue  = 10
)

func spend() {
	for i := 1; i < 500; i++ {
		mutex.Lock()
		for money-spendValue < 10 {
			enoughMoney.Wait()
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
		//enoughMoney.Broadcast() // wakes every waiting goroutine
		// Signal wakes ONE goroutine waiting on the condition - a goroutine, not an OS
		// thread. WHICH one is deliberately unspecified: the docs promise only "wakes one
		// goroutine waiting on c, if there is any". The current runtime happens to use a
		// FIFO notify list, but that is an implementation detail - never rely on it.
		enoughMoney.Signal()
		mutex.Unlock()
		time.Sleep(1 * time.Millisecond)
	}
	fmt.Println("Work: Done")
}

func Run() {
	go work()
	go spend()

	time.Sleep(10 * time.Second)
	// money is guarded by the mutex, so reading it must take the lock too -
	// time.Sleep is not a synchronisation primitive.
	mutex.Lock()
	currentMoney := money
	mutex.Unlock()
	fmt.Println("Current value:", currentMoney)
}*/

/*// Deadlocks
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
// Further reading on detecting the deadlock above:
// https://dev.to/ietxaniz/go-deadlock-detection-delock-library-1eig

/*
// Atomics
var (
	money int64 = 100
	value = 10
)

func spend() {
	for i := 1; i < 500; i++ {
		// AddInt64 returns the new value - we use it instead of reading `money` with
		// a plain load, which would be a data race (go run -race reports it).
		current := atomic.AddInt64(&money, int64(-value))
		fmt.Println("Spend: ", current)
		time.Sleep(1 * time.Millisecond)
	}
	fmt.Println("Spend: Done")
}

func work() {
	for i := 1; i < 500; i++ {
		current := atomic.AddInt64(&money, int64(value))
		fmt.Println("New income, current value:", current)
		time.Sleep(1 * time.Millisecond)
	}
	fmt.Println("Work: Done")
}

func Run() {
	go work()
	go spend()

	time.Sleep(10 * time.Second)
	// A variable written atomically must be read atomically too.
	fmt.Println("Current value:", atomic.LoadInt64(&money))
}

// The Go 1.19+ idiom: the atomic.Int64 type instead of the atomic.*Int64
// functions on a raw variable - then it cannot accidentally be read non-atomically:
//
//	var money atomic.Int64
//	money.Store(100)
//	current := money.Add(-int64(value))
//	fmt.Println(money.Load())
*/

/*// Cyclic barrier
func execute(name string, sleepTime int, barrier *Barrier) {
	for {
		println(name, "running")
		time.Sleep(time.Duration(sleepTime) * time.Second)
		println(name, "is waiting on barrier")
		barrier.Wait()
		println(name, "is after barrier")
	}
}

func Run() {
	barrier := NewBarrier(3)
	go execute("One", 3, barrier)
	go execute("Two", 10, barrier)
	go execute("Three", 6, barrier)
	time.Sleep(100 * time.Second)
}*/

/*
// Semaphore
func Run() {
	semaphore := NewSemaphore(5)
	// NOTE: since Go 1.22 each iteration has its own `i`, so capturing it in
	// the goroutine below is CORRECT. Before Go 1.22 all closures shared one
	// loop variable and printed whatever value it happened to have when they
	// ran (typically 100, after the loop had finished) - the fix then was to
	// pass i as a function argument.
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
}
*/
