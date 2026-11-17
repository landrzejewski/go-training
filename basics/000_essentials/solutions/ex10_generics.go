package solutions

import (
	"cmp"
	"errors"
	"maps"
	"slices"
)

// ---- Exercise 10.1: a constrained type parameter -----------------------------------------------

// Largest returns the largest element of items. T must be cmp.Ordered because the code uses
// >. For an empty slice it returns the zero value of T and an error.
func Largest[T cmp.Ordered](items []T) (T, error) {
	var zero T // works for every T: 0, "", 0.0, ...
	if len(items) == 0 {
		return zero, errors.New("largest: empty slice")
	}
	best := items[0]
	for _, v := range items[1:] {
		if v > best {
			best = v
		}
	}
	return best, nil
}

// ---- Exercise 10.2: two type parameters --------------------------------------------------------

// GroupBy puts every item into the group chosen by key, keeping input order within a group.
// T can be anything. K must be comparable because it becomes a map key.
func GroupBy[T any, K comparable](items []T, key func(T) K) map[K][]T {
	groups := make(map[K][]T)
	for _, item := range items {
		k := key(item)
		groups[k] = append(groups[k], item)
	}
	return groups
}

// ---- Exercise 10.3: a generic type -------------------------------------------------------------

// Set is a collection of unique values. T must be comparable because the values are used as
// map keys. The zero value is an empty set that is ready to use.
type Set[T comparable] struct {
	items map[T]struct{} // struct{} takes no memory: only the keys matter
}

// NewSet returns a set holding the given items; duplicates are stored once.
func NewSet[T comparable](items ...T) *Set[T] {
	s := &Set[T]{}
	for _, v := range items {
		s.Add(v)
	}
	return s
}

// Add inserts v. Adding a value that is already present does nothing.
func (s *Set[T]) Add(v T) {
	if s.items == nil {
		s.items = make(map[T]struct{}) // created lazily, so the zero Set works too
	}
	s.items[v] = struct{}{}
}

// Has reports whether v is in the set. Reading a nil map is fine, so no check is needed.
func (s *Set[T]) Has(v T) bool {
	_, ok := s.items[v]
	return ok
}

// Len returns the number of items.
func (s *Set[T]) Len() int { return len(s.items) }

// Union returns a NEW set with the items of both s and other; neither input changes.
func (s *Set[T]) Union(other *Set[T]) *Set[T] {
	out := NewSet[T]() // no arguments to infer T from, so it is written out
	for v := range s.items {
		out.Add(v)
	}
	for v := range other.items {
		out.Add(v)
	}
	return out
}

// Items returns the items in no particular order (map order is random).
func (s *Set[T]) Items() []T {
	return slices.Collect(maps.Keys(s.items))
}
