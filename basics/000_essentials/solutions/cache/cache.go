// Package cache is the reference solution for Project 2 in basic_exercises.md: a generic
// in-memory cache whose entries expire after a time-to-live (TTL) and which holds at most a
// fixed number of entries.
//
// A Cache is safe for concurrent use.
package cache

import (
	"errors"
	"fmt"
	"iter"
	"sync"
	"time"
)

// Errors returned by New.
var (
	ErrInvalidTTL      = errors.New("ttl must be positive")
	ErrInvalidCapacity = errors.New("capacity must be at least 1")
)

// Clock tells the cache what time it is. Tests pass a fake clock to make entries expire
// without sleeping.
type Clock interface {
	Now() time.Time
}

// RealClock is the Clock for production use.
type RealClock struct{}

func (RealClock) Now() time.Time { return time.Now() }

// Stats counts what happened to the cache since it was created.
type Stats struct {
	Hits      int // Get found a live entry
	Misses    int // Get found nothing, or only an expired entry
	Evictions int // a live entry was removed to make room for a new one
}

// HitRate returns Hits / (Hits + Misses), or 0 before the first Get.
func (s Stats) HitRate() float64 {
	total := s.Hits + s.Misses
	if total == 0 {
		return 0
	}
	return float64(s.Hits) / float64(total)
}

// String implements fmt.Stringer: "hits=3 misses=1 evictions=0 hit-rate=75%".
func (s Stats) String() string {
	return fmt.Sprintf("hits=%d misses=%d evictions=%d hit-rate=%.0f%%", s.Hits, s.Misses, s.Evictions, s.HitRate()*100)
}

// entry is a stored value plus the moment it stops being valid. It is generic too, because
// it stores a V.
type entry[V any] struct {
	value   V
	expires time.Time
}

// Cache maps keys of type K to values of type V. K must be comparable because it is used as
// a map key; V can be anything. Create one with New.
type Cache[K comparable, V any] struct {
	mu       sync.Mutex
	clock    Clock
	ttl      time.Duration
	capacity int
	items    map[K]entry[V]
	stats    Stats
}

// New returns an empty cache. The type arguments cannot be inferred from the parameters, so
// callers write them out: cache.New[string, int](time.Minute, 100, cache.RealClock{}).
func New[K comparable, V any](ttl time.Duration, capacity int, clock Clock) (*Cache[K, V], error) {
	if ttl <= 0 {
		return nil, fmt.Errorf("cache: %w (got %v)", ErrInvalidTTL, ttl)
	}
	if capacity < 1 {
		return nil, fmt.Errorf("cache: %w (got %d)", ErrInvalidCapacity, capacity)
	}
	return &Cache[K, V]{
		clock:    clock,
		ttl:      ttl,
		capacity: capacity,
		items:    make(map[K]entry[V], capacity),
	}, nil
}

// Set stores value under key for the cache's TTL, replacing any previous value. When a new key
// arrives and the cache is full, expired entries are dropped first; if that is not enough,
// the entry that would expire soonest is evicted.
func (c *Cache[K, V]) Set(key K, value V) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := c.clock.Now()
	if _, exists := c.items[key]; !exists && len(c.items) >= c.capacity {
		c.removeExpired(now)
		if len(c.items) >= c.capacity {
			c.evictSoonestExpiring()
		}
	}
	c.items[key] = entry[V]{value: value, expires: now.Add(c.ttl)}
}

// Get returns the value stored under key and true, or the zero value of V and false when the
// key is missing or its entry has expired.
func (c *Cache[K, V]) Get(key K) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	e, ok := c.items[key]
	if ok && c.expired(e, c.clock.Now()) {
		delete(c.items, key) // clean up lazily, on the way
		ok = false
	}
	if !ok {
		c.stats.Misses++
		var zero V
		return zero, false
	}
	c.stats.Hits++
	return e.value, true
}

// Delete removes key from the cache. Deleting a missing key does nothing.
func (c *Cache[K, V]) Delete(key K) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, key)
}

// Len returns the number of live (not expired) entries.
func (c *Cache[K, V]) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.removeExpired(c.clock.Now())
	return len(c.items)
}

// Stats returns a copy of the counters.
func (c *Cache[K, V]) Stats() Stats {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.stats // Stats is a plain struct, so returning it copies it
}

// All yields every live entry, in no particular order. It works on a snapshot taken when the
// loop starts, so the loop body may use the cache freely.
func (c *Cache[K, V]) All() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		c.mu.Lock()
		now := c.clock.Now()
		snapshot := make(map[K]V, len(c.items))
		for k, e := range c.items {
			if !c.expired(e, now) {
				snapshot[k] = e.value
			}
		}
		c.mu.Unlock()

		for k, v := range snapshot {
			if !yield(k, v) {
				return
			}
		}
	}
}

// GetOrLoad returns the cached value for key. On a miss it calls load, stores the result and
// returns it. An error from load is returned wrapped, and nothing is stored.
//
// load runs WITHOUT the lock held, so a slow load does not block other keys. The price is that
// two goroutines missing the same key at the same time may both call load.
func (c *Cache[K, V]) GetOrLoad(key K, load func(K) (V, error)) (V, error) {
	if v, ok := c.Get(key); ok {
		return v, nil
	}
	v, err := load(key)
	if err != nil {
		var zero V
		return zero, fmt.Errorf("cache: load %v: %w", key, err)
	}
	c.Set(key, v)
	return v, nil
}

// expired reports whether e is no longer valid at now. An entry lives while now < expires.
func (c *Cache[K, V]) expired(e entry[V], now time.Time) bool {
	return !now.Before(e.expires)
}

// removeExpired deletes every expired entry. The caller must hold c.mu. (Deleting from a map
// while ranging over it is allowed in Go.)
func (c *Cache[K, V]) removeExpired(now time.Time) {
	for k, e := range c.items {
		if c.expired(e, now) {
			delete(c.items, k)
		}
	}
}

// evictSoonestExpiring removes the entry with the earliest expiry. The caller must hold c.mu.
// With one TTL for all entries that is also the oldest entry. A linear scan is fine for a
// small cache; a big one would keep the entries in a heap or a list.
func (c *Cache[K, V]) evictSoonestExpiring() {
	var (
		victim K
		first  time.Time
		found  bool
	)
	for k, e := range c.items {
		if !found || e.expires.Before(first) {
			victim, first, found = k, e.expires, true
		}
	}
	if found {
		delete(c.items, victim)
		c.stats.Evictions++
	}
}
