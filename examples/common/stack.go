// Package common holds small generic helpers shared by the other example
// packages: a Stack, the Add function with its Number constraint, and gob
// encoding helpers (ToBytes, FromBytes).
package common

// Stack is a LIFO container of T. The zero value is an empty stack ready to
// use. Stack is not safe for concurrent use; callers that share one between
// goroutines must guard it with a mutex (see
// examples/concurrency/producer_consumer_classic.go).
type Stack[T any] struct {
	data []T
}

// Push adds element on top of the stack.
func (s *Stack[T]) Push(element T) {
	s.data = append(s.data, element)
}

// Pop removes and returns the top element. The boolean is false when the
// stack is empty, in which case the zero value of T is returned.
func (s *Stack[T]) Pop() (T, bool) {
	if s.isEmpty() {
		var empty T
		return empty, false
	}
	lastIndex := s.Size() - 1
	element := s.data[lastIndex]
	// Reslicing only shrinks len - the popped element is still reachable through
	// the backing array, so the GC cannot free it. For Stack[*T] or Stack[string]
	// that is a real memory leak, hence zeroing the slot first.
	var zero T
	s.data[lastIndex] = zero
	s.data = s.data[:lastIndex] // [0:lastIndex)
	return element, true
}

func (s *Stack[T]) isEmpty() bool {
	return s.Size() == 0
}

// Size returns the number of elements on the stack.
func (s *Stack[T]) Size() int {
	return len(s.data)
}
