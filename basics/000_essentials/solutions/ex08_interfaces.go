package solutions

import (
	"strings"
	"time"
)

// ---- Exercise 8.1: implicit satisfaction -------------------------------------------------------

// Speaker is satisfied by every type that has a Sound() string method.
type Speaker interface {
	Sound() string
}

// Dog, Cat and Cow never mention Speaker - having the method is enough. A type needs no
// fields to satisfy an interface.
type (
	Dog struct{}
	Cat struct{}
	Cow struct{}
)

func (d Dog) Sound() string { return "Woof" }
func (c Cat) Sound() string { return "Meow" }
func (c Cow) Sound() string { return "Moo" }

// Chorus returns the sounds of all speakers, separated by spaces.
func Chorus(speakers []Speaker) string {
	sounds := make([]string, 0, len(speakers))
	for _, s := range speakers {
		sounds = append(sounds, s.Sound()) // calls Dog.Sound, Cat.Sound, ... decided at run time
	}
	return strings.Join(sounds, " ")
}

// ---- Exercise 8.2: an interface for decoupling -------------------------------------------------

// Clock is the only thing Greeting needs: the current time. Depending on this small interface
// instead of calling time.Now directly is what makes Greeting testable.
type Clock interface {
	Now() time.Time
}

// RealClock is the Clock for production code.
type RealClock struct{}

func (RealClock) Now() time.Time { return time.Now() }

// FixedClock always returns the same instant, which is ideal for tests.
type FixedClock struct {
	T time.Time
}

func (f FixedClock) Now() time.Time { return f.T }

// These lines compile only if both types satisfy Clock - a cheap compile-time check.
var (
	_ Clock = RealClock{}
	_ Clock = FixedClock{}
)

// Greeting returns "Good morning" before 12:00, "Good afternoon" before 18:00 and
// "Good evening" otherwise.
func Greeting(c Clock) string {
	hour := c.Now().Hour()
	switch {
	case hour < 12:
		return "Good morning"
	case hour < 18:
		return "Good afternoon"
	default:
		return "Good evening"
	}
}

// ---- Exercise 8.3: any and type switches -------------------------------------------------------

// Sizer is implemented by any type that knows its own size.
type Sizer interface {
	Size() int
}

// File is an example Sizer.
type File struct {
	Name  string
	Bytes int
}

func (f File) Size() int { return f.Bytes }

// TotalSize adds up the sizes of the items: a string counts its bytes, a []int its length, an
// int counts as its own value, a Sizer reports Size(), and anything else counts as 0.
func TotalSize(items []any) int {
	total := 0
	for _, item := range items {
		switch v := item.(type) { // in each case v has that case's type
		case string:
			total += len(v)
		case []int:
			total += len(v)
		case int:
			total += v
		case Sizer: // an interface as a case: matches every type with a Size() int method
			total += v.Size()
		}
	}
	return total
}
