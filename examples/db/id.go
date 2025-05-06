package db

type IdGenerator interface {
	next() int64
}

type Sequence struct {
	counter int64
}

func (s *Sequence) next() int64 {
	s.counter++
	return s.counter
}