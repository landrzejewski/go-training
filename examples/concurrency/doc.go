// Package concurrency collects the goroutine and channel exercises: the
// sleeping-barber problem (Barbers), a cyclic Barrier and a counting
// Semaphore built on sync.Cond, two producer/consumer variants
// (ProducerConsumerClassic with a mutex and conditions, ProducerConsumerChannels
// with a channel), a file-search pipeline (FindFiles) and a tour of channel
// operations (ChannelsDemo).
//
// classic.go is a commented-out walkthrough of the primitives (WaitGroup,
// Mutex, RWMutex, Cond, atomics, deadlocks) and is not compiled.
package concurrency
