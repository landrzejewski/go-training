// Command 011_concurrency is module 011 of the Go course: goroutines, channels, select, sync,
// context and the classic concurrency patterns.
//
// CSP: "Do not communicate by sharing memory; share memory by communicating" - good advice, not a
// law: a channel transfers ownership or signals, a mutex guards a small piece of shared state.
// Concurrency (structure) is not parallelism (simultaneous execution, bounded by GOMAXPROCS).
//
//	go run ./basics/011_concurrency
//	go run -race ./basics/011_concurrency
package main

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

func main() {
	goroutines()      // go statement, cheap stacks, WaitGroup.Go, one recover per goroutine
	channels()        // unbuffered vs buffered, direction, close, the panic rules
	selectStatement() // first ready case, default, timeouts, the nil-channel idiom
	syncPrimitives()  // Mutex, RWMutex, typed atomics, Once
	contexts()        // cancellation, deadlines, causes, AfterFunc, values
	patterns()        // worker pool, pipeline, semaphore, early cancellation
	memoryModel()     // happens-before edges and the failure modes
}

// goroutines shows how to start goroutines and wait for them.
func goroutines() {
	fmt.Println("--- goroutines ---")
	fmt.Println("goroutines at start:", runtime.NumGoroutine())

	// go f(x) starts f in a new goroutine and returns at once. Like defer, the ARGUMENTS are
	// evaluated at the go statement, in the current goroutine. Since Go 1.22 the loop variable is
	// fresh per iteration, so the closure may capture i directly.
	var wg sync.WaitGroup
	results := make([]int, 3)
	for i := range 3 {
		wg.Add(1)
		go func(sq int) {
			defer wg.Done()
			results[i] = sq // safe: each goroutine writes its own element
		}(i * i) // evaluated here, before the goroutine starts
	}
	// main does NOT wait by itself: when it returns, the process exits and every other goroutine
	// is killed mid-statement, without running its defers. Synchronise explicitly.
	wg.Wait()
	fmt.Println("argument evaluated at the go statement, i captured per iteration:", results)

	// Go 1.25: wg.Go(f) is Add(1) + go + defer Done() in one call. Prefer it - it makes the
	// classic "Add inside the goroutine" race impossible.
	total := make([]int, 4)
	for i := range 4 {
		wg.Go(func() { total[i] = i + 1 })
	}
	wg.Wait()
	fmt.Println("sync.WaitGroup.Go (Go 1.25):", total)

	// A goroutine is not an OS thread: it starts with a 2 KB stack that grows on demand, and the
	// M:N scheduler multiplexes goroutines onto a few threads. Since Go 1.14 it is preemptive, so
	// even a tight loop with no calls can be interrupted. Hundreds of thousands is ordinary.
	const many = 10_000
	start := time.Now()
	var counter atomic.Int64
	for range many {
		wg.Go(func() { counter.Add(1) })
	}
	wg.Wait()
	fmt.Printf("%d goroutines started and finished in %v (all %d ran)\n",
		many, time.Since(start).Round(time.Microsecond), counter.Load())

	// A panic in ANY goroutine kills the whole program - a recover in another goroutine does not
	// help. Every goroutine that might panic needs its own deferred recover.
	wg.Go(func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Println("a goroutine recovered from its own panic:", r)
			}
		}()
		panic("without this recover, the whole program would die")
	})
	wg.Wait()

	// Goroutine leaks are THE characteristic Go bug: a goroutine blocked forever is never
	// collected and keeps alive everything it references. Every goroutine needs a defined way to
	// stop (a closed channel, a cancelled context, bounded work). NumGoroutine is the cheapest
	// leak check; the pprof goroutine profile is the real tool.
	fmt.Println("goroutines at end:", runtime.NumGoroutine(), "(a leak shows up here first)")
}

// channels shows how goroutines pass values and signals over channels.
func channels() {
	fmt.Println("\n--- channels ---")

	// make(chan T) is UNBUFFERED - a rendezvous: the send blocks until a receiver is ready and
	// vice versa, so it synchronises the two goroutines as well as passing a value.
	done := make(chan string)
	go func() { done <- "the send waited for this receive" }()
	fmt.Printf("unbuffered: %q\n", <-done)

	// make(chan T, n) is BUFFERED: sends block only when it is full, receives only when empty.
	// A buffer smooths bursts; it does not "make it faster", and a huge one only moves the
	// failure to memory exhaustion.
	buf := make(chan int, 3)
	buf <- 1
	buf <- 2
	fmt.Printf("buffered: len=%d cap=%d (two sends did not block)\n", len(buf), cap(buf))
	fmt.Println("receiving:", <-buf, <-buf)

	// Direction is part of the type: chan<- T is send-only, <-chan T is receive-only. A plain
	// chan T converts implicitly; put directions in signatures and the compiler enforces roles.
	numbers := make(chan int, 3)
	produce(numbers, 3)
	fmt.Println("directional channels: consumed", consume(numbers))

	// close(ch) means "no more values will be sent". range reads until the channel is closed
	// (if nobody closes it, the range never ends); after that a receive yields the zero value
	// immediately and forever, and the comma-ok form reports ok == false.
	ch := make(chan int, 3)
	for i := range 3 {
		ch <- i
	}
	close(ch)
	fmt.Print("range drains a closed channel: ")
	for v := range ch {
		fmt.Print(v, " ")
	}
	fmt.Println()
	v, ok := <-ch
	fmt.Printf("after draining: v=%d ok=%t\n", v, ok)

	// Three operations panic; only RECEIVING is always safe.
	closed := make(chan int)
	close(closed)
	fmt.Println("send on a closed channel:", catchPanic(func() { closed <- 1 }))
	fmt.Println("close a closed channel:  ", catchPanic(func() { close(closed) }))
	fmt.Println("close a nil channel:     ", catchPanic(func() { var c chan int; close(c) }))

	// Hence the convention: the SENDER closes, never the receiver. With several senders none of
	// them closes - one extra goroutine closes after wg.Wait().
	out := make(chan int)
	var wg sync.WaitGroup
	for i := range 3 {
		wg.Go(func() { out <- i })
	}
	go func() { wg.Wait(); close(out) }()
	sum := 0
	for v := range out {
		sum += v
	}
	fmt.Println("three senders, one closer after Wait(): sum =", sum)

	// A nil channel blocks forever on both send and receive - the basis of the nil-out idiom in
	// select. When EVERY goroutine is blocked, the runtime aborts and recover cannot stop it:
	//	var c chan int
	//	<-c // fatal error: all goroutines are asleep - deadlock!
	// If even one goroutine can still run, the runtime cannot tell, and the program just hangs.
}

func produce(out chan<- int, n int) {
	for i := range n {
		out <- i
	}
	close(out)
	//	<-out // ERROR: invalid operation: cannot receive from send-only channel chan<- int out (variable of type chan<- int)
}

func consume(in <-chan int) []int {
	var got []int
	for v := range in {
		got = append(got, v)
	}
	//	in <- 1   // ERROR: invalid operation: cannot send to receive-only channel <-chan int in (variable of type <-chan int)
	//	close(in) // ERROR: invalid operation: cannot close receive-only channel in (variable of type <-chan int)
	return got
}

// catchPanic runs f and returns whatever it panicked with (module 005).
func catchPanic(f func()) (recovered any) {
	defer func() {
		if r := recover(); r != nil {
			recovered = r
		}
	}()
	f()
	return "did not panic"
}

// selectStatement shows select: waiting on several channel operations at once.
func selectStatement() {
	fmt.Println("\n--- select ---")

	// select proceeds with the first case that becomes ready.
	fast := make(chan string)
	// slow is buffered: nobody receives from it once select picks fast, so with an unbuffered
	// channel its goroutine would block forever - a textbook goroutine leak.
	slow := make(chan string, 1)
	go func() { time.Sleep(5 * time.Millisecond); fast <- "fast" }()
	go func() { time.Sleep(50 * time.Millisecond); slow <- "slow" }()
	select {
	case v := <-fast:
		fmt.Printf("first ready case: %q\n", v)
	case v := <-slow:
		fmt.Printf("first ready case: %q\n", v)
	}

	// A default case makes select non-blocking: if nothing is ready, default runs at once.
	// Works for sends too - a send to a full buffer is skipped instead of blocking.
	full := make(chan int, 1)
	full <- 1
	select {
	case full <- 2:
		fmt.Println("sent")
	default:
		fmt.Println("default: the buffer was full, so the send was skipped")
	}

	// A timeout is select + time.After. Before Go 1.23 an unfired time.After timer was not
	// collected until it fired; in a hot loop time.NewTimer + Stop is still the cheaper choice.
	silent := make(chan int)
	select {
	case v := <-silent:
		fmt.Println("received", v)
	case <-time.After(20 * time.Millisecond):
		fmt.Println("timed out after 20ms via select + time.After")
	}

	// When several cases are ready, select picks one uniformly at random, so none can starve.
	a, b := make(chan int, 10), make(chan int, 10)
	for range 10 {
		a <- 1
		b <- 2
	}
	counts := map[int]int{}
	for range 10 {
		select {
		case v := <-a:
			counts[v]++
		case v := <-b:
			counts[v]++
		}
	}
	fmt.Println("both ready 10 times -> a random split:", counts)

	// An empty select{} blocks forever: it parks main in a server, or triggers the deadlock
	// detector if nothing else runs. for { select { ... case <-ctx.Done(): return } } is the
	// canonical worker loop (see contexts).

	// The nil-channel idiom: a nil channel is never ready, so setting a drained input to nil
	// disables its case. That is how merge stops reading each input as it closes.
	var got []int
	for v := range merge(count(1, 3), count(10, 2)) {
		got = append(got, v)
	}
	fmt.Println("fan-in via the nil-out idiom:", got, "(order varies)")
}

func count(start, n int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for i := range n {
			out <- start + i
		}
	}()
	return out
}

// merge fans two channels into one. The loop condition MUST be in the header: checked at the
// bottom, the continue after a close would skip it and the next select would wait on two nil
// channels - forever, since no case can become ready and there is no default.
func merge[T any](ch1, ch2 <-chan T) <-chan T {
	out := make(chan T)
	go func() {
		defer close(out)
		for ch1 != nil || ch2 != nil {
			select {
			case v, ok := <-ch1:
				if !ok {
					ch1 = nil // disables this case in every later select
					continue
				}
				out <- v
			case v, ok := <-ch2:
				if !ok {
					ch2 = nil
					continue
				}
				out <- v
			}
		}
	}()
	return out
}

// syncPrimitives shows the sync and sync/atomic packages.
func syncPrimitives() {
	fmt.Println("\n--- sync and sync/atomic ---")

	// Concurrent writes to a plain map are a FATAL error, not a panic - recover cannot catch it:
	//	fatal error: concurrent map writes
	// so shared state goes behind a mutex (see registry).
	reg := newRegistry()
	var wg sync.WaitGroup
	for range 100 {
		wg.Go(func() { reg.inc("hits") })
	}
	wg.Wait()
	fmt.Println("RWMutex-guarded map after 100 concurrent inc:", reg.get("hits"))

	// A mutex must not be copied after first use, which is why structs holding one are passed
	// by pointer (module 007). go vet's copylocks check catches the copy:
	//	snapshot := *reg // vet: assignment copies lock value to snapshot: training.pl/go/basics/011_concurrency.registry contains sync.RWMutex

	// Typed atomics (Go 1.19: atomic.Int64, Bool, Pointer[T], ...) beat the old free functions:
	// the zero value is ready to use and the value cannot be accessed non-atomically by mistake.
	var hits atomic.Int64
	var ready atomic.Bool
	for range 1000 {
		wg.Go(func() { hits.Add(1) })
	}
	wg.Wait()
	ready.Store(true)
	fmt.Printf("atomic.Int64 after 1000 concurrent Add: %d (ready=%t)\n", hits.Load(), ready.Load())

	// CompareAndSwap is the building block of every lock-free algorithm.
	var state atomic.Int32
	swapped := state.CompareAndSwap(0, 5)
	failed := state.CompareAndSwap(0, 9)
	fmt.Printf("CompareAndSwap(0,5)=%t, then CompareAndSwap(0,9)=%t, value=%d\n",
		swapped, failed, state.Load())

	// atomic.Pointer[T] is the type-safe successor of atomic.Value (Go 1.4, untyped, any-based).
	var current atomic.Pointer[registry]
	current.Store(reg)
	fmt.Println("atomic.Pointer[registry] loaded a registry with", current.Load().get("hits"), "hits")

	// sync.Once runs a function exactly once; sync.OnceValue / OnceValues (Go 1.21) are the
	// generic versions that also cache a result (module 010).
	var once sync.Once
	for range 3 {
		once.Do(func() { fmt.Println("sync.Once: this line runs exactly once") })
	}
	config := sync.OnceValue(func() string { fmt.Println("(loading config)"); return "cfg-v1" })
	fmt.Println("sync.OnceValue:", config(), config())

	// Rarely needed, but worth recognising:
	//	sync.Cond - wait for a condition; a channel is usually clearer (examples/concurrency/barrier.go)
	//	sync.Pool - a free list of temporary objects, emptied at any GC; use only after profiling
	//	sync.Map  - only for write-once/read-many keys or disjoint keys per goroutine; else map + RWMutex
	// The race detector (go run -race, go test -race; roughly 2-20x slower) is the only reliable
	// way to find data races. Run your tests with it in CI.
}

// registry is the idiomatic shape: the mutex sits directly above the fields it guards.
type registry struct {
	mu     sync.RWMutex
	counts map[string]int // guarded by mu
}

func newRegistry() *registry { return &registry{counts: make(map[string]int)} }

// The zero Mutex is unlocked and needs no constructor. It is NOT reentrant: locking it twice in
// one goroutine deadlocks. The idiom is always Lock(); defer Unlock().
func (r *registry) inc(key string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.counts[key]++
}

// RWMutex allows many readers OR one writer. It pays off only when reads dominate and are slow;
// for short critical sections a plain Mutex is faster (less bookkeeping).
func (r *registry) get(key string) int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.counts[key]
}

// contexts shows context.Context: cancellation, deadlines and request-scoped values.
func contexts() {
	fmt.Println("\n--- context ---")

	// Conventions: ctx is the FIRST parameter of anything that can block; never store it in a
	// struct; never pass nil (use context.TODO() as a placeholder). context.Background() is the
	// root, at the top of main or of a request. A Context is immutable: each With... derives a child.

	// WithTimeout / WithDeadline / WithCancel also return a cancel function. ALWAYS defer it,
	// even when the timeout will fire anyway: otherwise the child stays registered in its parent
	// (and its timer alive) until the parent ends. go vet's lostcancel check catches this:
	//	ctx, _ := context.WithTimeout(parent, d) // vet: the cancel function returned by context.WithTimeout should be called, not discarded, to avoid a context leak
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	if err := slowWork(ctx, 200*time.Millisecond); err != nil {
		// ctx.Err() says why: context.Canceled or context.DeadlineExceeded.
		fmt.Printf("work cancelled: %v (ctx.Err() = %v)\n", err, ctx.Err())
	}

	// Cancellation propagates DOWN the tree to every descendant, never up.
	parent, cancelParent := context.WithCancel(context.Background())
	child, cancelChild := context.WithCancel(parent)
	defer cancelChild()
	cancelParent()
	<-child.Done() // Done() is a channel closed on cancellation
	fmt.Println("cancelling the parent cancelled the child:", child.Err())

	// WithCancelCause (Go 1.20): cancel with a reason, read it back with context.Cause.
	ctx3, cancelCause := context.WithCancelCause(context.Background())
	cancelCause(errors.New("upstream returned 503"))
	fmt.Printf("WithCancelCause: Err()=%v Cause()=%v\n", ctx3.Err(), context.Cause(ctx3))

	// AfterFunc (Go 1.21): run a function once the context is done.
	ctx4, cancel4 := context.WithCancel(context.Background())
	fired := make(chan struct{})
	context.AfterFunc(ctx4, func() { close(fired) })
	cancel4()
	<-fired
	fmt.Println("context.AfterFunc ran on cancellation")

	// WithValue is for request-scoped data only (trace ID, authenticated user), never for
	// optional parameters. The key is an unexported type, so no other package can collide with
	// it or read it - a plain string key is a bug waiting to happen.
	valued := context.WithValue(context.Background(), requestIDKey, "req-42")
	fmt.Println("request-scoped value:", handleRequest(valued))

	// WithoutCancel (Go 1.21): keep the values, drop the cancellation - e.g. for cleanup work
	// that must outlive the request.
	cancelled, cancel5 := context.WithCancel(valued)
	cancel5()
	detached := context.WithoutCancel(cancelled)
	fmt.Printf("WithoutCancel keeps values (%v) but not cancellation (Err=%v)\n",
		detached.Value(requestIDKey), detached.Err())
}

type ctxKey string

const requestIDKey ctxKey = "requestID"

// slowWork is the canonical cancellable operation: select on the work and on ctx.Done().
func slowWork(ctx context.Context, d time.Duration) error {
	select {
	case <-time.After(d):
		return nil
	case <-ctx.Done():
		return fmt.Errorf("slowWork: %w", ctx.Err())
	}
}

func handleRequest(ctx context.Context) string {
	id, ok := ctx.Value(requestIDKey).(string)
	if !ok {
		return "no request id"
	}
	return "handling " + id
}

// patterns shows worker pools, pipelines, fan-out/fan-in and semaphores.
func patterns() {
	fmt.Println("\n--- patterns ---")

	// Worker pool: N goroutines read one jobs channel (fan-out) and write one results channel.
	// It bounds concurrency - runtime.NumCPU() workers for CPU-bound work, or a downstream rate
	// limit. Do NOT start an unbounded goroutine per item.
	jobs := make(chan int, 20)
	results := make(chan int, 20)
	workers := min(4, runtime.NumCPU())
	var wg sync.WaitGroup
	for range workers {
		wg.Go(func() {
			for j := range jobs { // every worker reads the SAME channel
				results <- j * j
			}
		})
	}
	for i := 1; i <= 9; i++ {
		jobs <- i
	}
	close(jobs)    // no more work: each worker's range ends
	wg.Wait()      // wait for all of them
	close(results) // only now is it safe to close the output
	sum := 0
	for r := range results {
		sum += r
	}
	fmt.Printf("worker pool: %d workers squared 1..9, sum=%d\n", workers, sum)

	// Pipeline: a chain of stages, each taking a receive-only channel and returning a new one
	// that it owns and closes. Fan-in (merging channels) is merge from selectStatement.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	even := func(n int) bool { return n%2 == 0 }
	var out []int
	for v := range square(ctx, filter(ctx, generate(ctx, 1, 10), even)) {
		out = append(out, v)
	}
	fmt.Println("pipeline generate -> filter even -> square:", out)

	// Semaphore: a buffered channel used as a counting semaphore - send to acquire, receive to
	// release (examples/concurrency/semaphore.go shows the sync.Cond version).
	const limit = 3
	sem := make(chan struct{}, limit)
	var peak, running atomic.Int32
	for range 20 {
		wg.Go(func() {
			sem <- struct{}{}        // acquire
			defer func() { <-sem }() // release
			n := running.Add(1)
			for { // record the peak with a CAS loop
				p := peak.Load()
				if n <= p || peak.CompareAndSwap(p, n) {
					break
				}
			}
			time.Sleep(time.Millisecond)
			running.Add(-1)
		})
	}
	wg.Wait()
	fmt.Printf("buffered channel as a semaphore: limit=%d, observed peak=%d\n", limit, peak.Load())

	// Every pattern needs a cancellation story: a stage blocked sending to a consumer that went
	// away is a leak. Each stage selects on ctx.Done() (see generate).
	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel2() // lostcancel insists, even though we cancel below
	taken := 0
	for range generate(ctx2, 1, 1_000_000) {
		if taken++; taken == 3 {
			cancel2() // generate sees ctx.Done() and returns instead of leaking
			break
		}
	}
	// Sleep + NumGoroutine is only illustrative (racy and slow); real tests use testing/synctest,
	// which waits until every goroutine is durably blocked (module 014).
	time.Sleep(5 * time.Millisecond)
	fmt.Printf("cancelled a 1,000,000-item generator after %d items; goroutines now %d\n",
		taken, runtime.NumGoroutine())

	// golang.org/x/sync/errgroup (not stdlib, but used almost universally) wraps all of this:
	// it runs a group of goroutines, cancels the shared ctx when one fails, returns the first
	// error. Worked exercises live in examples/concurrency: find_files.go (a three-stage
	// pipeline), barbers.go (the sleeping barber), barrier.go, semaphore.go, producer_consumer_*.go.
}

func generate(ctx context.Context, from, to int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for i := from; i <= to; i++ {
			select {
			case out <- i:
			case <-ctx.Done(): // without this the generator leaks when the consumer stops early
				return
			}
		}
	}()
	return out
}

func filter(ctx context.Context, in <-chan int, keep func(int) bool) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for v := range in {
			if !keep(v) {
				continue
			}
			select {
			case out <- v:
			case <-ctx.Done():
				return
			}
		}
	}()
	return out
}

func square(ctx context.Context, in <-chan int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for v := range in {
			select {
			case out <- v * v:
			case <-ctx.Done():
				return
			}
		}
	}()
	return out
}

// memoryModel shows the happens-before edges that make concurrent code correct.
func memoryModel() {
	fmt.Println("\n--- memory model, races and failures ---")

	// The memory model says when a write in one goroutine is guaranteed visible to a read in
	// another. The happens-before edges you get:
	//	a go statement             -> the start of that goroutine
	//	a send (or a close)        -> the receive that gets it (or sees the close)
	//	an unbuffered receive      -> the completion of the matching send
	//	mu.Unlock()                -> a later mu.Lock() returning
	//	f in once.Do(f)            -> any once.Do returning
	//	sync/atomic operations are sequentially consistent (stated explicitly since Go 1.19)
	var data string
	ready := make(chan struct{})
	go func() {
		data = "written before the close"
		close(ready)
	}()
	<-ready
	fmt.Printf("channel close as a happens-before edge: %q\n", data)

	// (Unlock -> Lock is what made registry correct in syncPrimitives.)
	var flag atomic.Bool
	var payload atomic.Pointer[string]
	msg := "published atomically"
	go func() { payload.Store(&msg); flag.Store(true) }()
	for !flag.Load() {
		runtime.Gosched()
	}
	fmt.Printf("atomics are sequentially consistent: %q\n", *payload.Load())

	// Without one of those edges there is NO guarantee - not even that a write ever becomes
	// visible. A "benign" race is not benign: with a plain bool the compiler may hoist the read
	// out of the loop, and this spins forever:
	//	for !done {} // data race

	// The failure modes, and whether recover helps:
	//	data race               undefined behaviour; only -race finds it
	//	concurrent map writes   fatal error, NOT recoverable
	//	all goroutines asleep   fatal error, NOT recoverable
	//	partial deadlock        undetected - some goroutines blocked, the program just hangs
	//	                        (the common case in real servers)
	//	goroutine leak          blocked forever, never collected
	//	panic in a goroutine    kills the process unless THAT goroutine recovers
	// Debugging tools, in order: go test -race; NumGoroutine before/after; the pprof goroutine
	// profile (import _ "net/http/pprof"); GOTRACEBACK=all; testing/synctest (Go 1.25) for
	// deterministic concurrent tests with a fake clock (module 014).
	fmt.Println("goroutines still alive at the end:", runtime.NumGoroutine())
}
