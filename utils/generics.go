package utils

import "fmt"

func addInt(a, b int) int {
	return a + b
}

func addFloat(a, b float64) float64 {
	return a + b
}

// func add[T int | float64 | int16](a, b T) T {
// 	return a + b
// }

type Number interface {
	int64 | float64 | int16
}

func add[T Number](a, b T) T {
	return a + b
}

type Stack[T any] struct {
	data []T
}

func (s *Stack[T]) Push(value T) {
	s.data = append(s.data, value)
}

func (s *Stack[T]) Pop() (T, bool) {
	if len(s.data) == 0 {
		var empty T
		return empty, false
	}
	lastIndex := s.Size() - 1
	value := s.data[lastIndex]
	s.data = s.data[:lastIndex]
	return value, true
}

func (s *Stack[T]) Size() int {
	return len(s.data)
}

func Generics() {
	var a float64 = 10
	var b float64 = 10
	add(a, b)

	intStack := Stack[int]{}
	intStack.Push(1)
	intStack.Push(2)
	fmt.Println(intStack.Pop())

	stringStack := Stack[string]{}
	stringStack.Push("a")
	stringStack.Push("b")
	fmt.Println(stringStack.Pop())
}
