package common

type Stack struct {
	data []int
}

func NewStack() *Stack {
	return &Stack{}
}

func (s *Stack) Push(element int) {
	s.data = append(s.data, element)
} 

func (s *Stack) Pop() (int, bool) {
	if s.isEmpty() {
		return 0, false
	}
	lastIndex := s.Size() - 1
	element := s.data[lastIndex]
	s.data = s.data[:lastIndex]
	return element, true
}

func (s *Stack) isEmpty() bool {
	return s.Size() == 0 
}

func (s *Stack) Size() int {
	return len(s.data)
}
