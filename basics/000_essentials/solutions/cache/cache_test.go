package cache

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"sync"
	"testing"
	"time"
)

// fakeClock is a Clock the test moves forward by hand. It has a mutex because the concurrency
// test reads it from many goroutines.
type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

func newClock() *fakeClock {
	return &fakeClock{now: time.Date(2026, time.March, 1, 12, 0, 0, 0, time.UTC)}
}

// newCache creates a cache or stops the test. It is generic, like New.
func newCache[K comparable, V any](t *testing.T, ttl time.Duration, capacity int, clock Clock) *Cache[K, V] {
	t.Helper()
	c, err := New[K, V](ttl, capacity, clock)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return c
}

func TestNewValidates(t *testing.T) {
	tests := []struct {
		name     string
		ttl      time.Duration
		capacity int
		want     error
	}{
		{"zero ttl", 0, 10, ErrInvalidTTL},
		{"negative ttl", -time.Second, 10, ErrInvalidTTL},
		{"zero capacity", time.Second, 0, ErrInvalidCapacity},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := New[string, int](tt.ttl, tt.capacity, RealClock{}); !errors.Is(err, tt.want) {
				t.Errorf("New(%v, %d) error = %v, want %v", tt.ttl, tt.capacity, err, tt.want)
			}
		})
	}
}

func TestSetGetDelete(t *testing.T) {
	c := newCache[string, int](t, time.Minute, 10, newClock())
	c.Set("a", 1)
	c.Set("a", 2) // overwrite
	if v, ok := c.Get("a"); !ok || v != 2 {
		t.Errorf("Get(a) = %d, %t; want 2, true", v, ok)
	}
	c.Delete("a")
	if v, ok := c.Get("a"); ok || v != 0 {
		t.Errorf("Get after Delete = %d, %t; want 0, false", v, ok)
	}
}

func TestExpiry(t *testing.T) {
	clock := newClock()
	c := newCache[string, string](t, time.Minute, 10, clock)
	c.Set("k", "v")

	clock.Advance(59 * time.Second)
	if _, ok := c.Get("k"); !ok {
		t.Fatal("entry expired too early")
	}
	clock.Advance(time.Second) // exactly the TTL: an entry lives while now < expires
	if _, ok := c.Get("k"); ok {
		t.Error("entry still alive after its TTL")
	}
	if n := c.Len(); n != 0 {
		t.Errorf("Len = %d, want 0", n)
	}
}

func TestEvictionWhenFull(t *testing.T) {
	clock := newClock()
	c := newCache[string, int](t, time.Hour, 2, clock)
	c.Set("old", 1)
	clock.Advance(time.Minute)
	c.Set("newer", 2)
	clock.Advance(time.Minute)
	c.Set("old", 10)   // an existing key: no eviction, and "old" now expires last
	c.Set("newest", 3) // full: "newer" expires soonest, so it goes

	if _, ok := c.Get("newer"); ok {
		t.Error(`"newer" should have been evicted`)
	}
	for _, k := range []string{"old", "newest"} {
		if _, ok := c.Get(k); !ok {
			t.Errorf("%q should still be cached", k)
		}
	}
	if s := c.Stats(); s.Evictions != 1 {
		t.Errorf("Evictions = %d, want 1", s.Evictions)
	}
}

func TestExpiredEntriesMakeRoomWithoutEviction(t *testing.T) {
	clock := newClock()
	c := newCache[int, int](t, time.Minute, 2, clock)
	c.Set(1, 1)
	c.Set(2, 2)
	clock.Advance(2 * time.Minute) // both expired
	c.Set(3, 3)
	if s := c.Stats(); s.Evictions != 0 {
		t.Errorf("Evictions = %d, want 0: expired entries are dropped, not evicted", s.Evictions)
	}
}

func TestGetOrLoad(t *testing.T) {
	c := newCache[int, string](t, time.Minute, 10, newClock())
	calls := 0
	load := func(k int) (string, error) {
		calls++
		return "value-" + strconv.Itoa(k), nil
	}
	for range 3 {
		v, err := c.GetOrLoad(7, load)
		if err != nil || v != "value-7" {
			t.Fatalf("GetOrLoad = %q, %v", v, err)
		}
	}
	if calls != 1 {
		t.Errorf("load called %d times, want 1", calls)
	}

	errDown := errors.New("database down")
	_, err := c.GetOrLoad(8, func(int) (string, error) { return "", errDown })
	if !errors.Is(err, errDown) {
		t.Errorf("error = %v, want it to wrap errDown", err)
	}
	if _, ok := c.Get(8); ok {
		t.Error("a failed load must not store anything")
	}
}

func TestAllSkipsExpiredAndStopsOnBreak(t *testing.T) {
	clock := newClock()
	c := newCache[string, int](t, time.Minute, 10, clock)
	c.Set("expired", 0)
	clock.Advance(2 * time.Minute)
	c.Set("a", 1)
	c.Set("b", 2)

	got := maps.Collect(c.All()) // maps.Collect turns an iter.Seq2 into a map
	if len(got) != 2 || got["a"] != 1 || got["b"] != 2 {
		t.Errorf("All = %v, want map[a:1 b:2]", got)
	}

	n := 0
	for range c.All() {
		n++
		break
	}
	if n != 1 {
		t.Errorf("loop ran %d times after break, want 1", n)
	}
}

func TestStats(t *testing.T) {
	c := newCache[string, int](t, time.Minute, 10, newClock())
	if got := c.Stats().String(); got != "hits=0 misses=0 evictions=0 hit-rate=0%" {
		t.Errorf("empty stats = %q", got)
	}
	c.Set("a", 1)
	c.Get("a")
	c.Get("a")
	c.Get("a")
	c.Get("missing")
	if got := c.Stats().String(); got != "hits=3 misses=1 evictions=0 hit-rate=75%" {
		t.Errorf("stats = %q", got)
	}
}

// TestConcurrentUse hammers one cache from many goroutines. It checks nothing by itself:
// its job is to let go test -race find unsynchronised access.
func TestConcurrentUse(t *testing.T) {
	c := newCache[int, int](t, time.Minute, 50, newClock())
	var wg sync.WaitGroup
	for g := range 20 {
		wg.Go(func() {
			for i := range 200 {
				key := (g * i) % 100
				c.Set(key, i)
				c.Get(key)
				_, _ = c.GetOrLoad(key+1, func(k int) (int, error) { return k, nil })
				if i%50 == 0 {
					c.Len()
					for range c.All() {
					}
				}
			}
		})
	}
	wg.Wait()
	if n := c.Len(); n > 50 {
		t.Errorf("Len = %d, larger than the capacity 50", n)
	}
}

func ExampleCache() {
	clock := &fakeClock{now: time.Date(2026, time.March, 1, 12, 0, 0, 0, time.UTC)}
	prices, err := New[string, float64](10*time.Minute, 100, clock)
	if err != nil {
		fmt.Println(err)
		return
	}

	prices.Set("BTC", 61000)
	prices.Set("ETH", 3100)
	v, ok := prices.Get("BTC")
	fmt.Println(v, ok)

	clock.Advance(11 * time.Minute) // both entries expire
	_, ok = prices.Get("BTC")
	fmt.Println(ok)

	price, err := prices.GetOrLoad("ETH", func(symbol string) (float64, error) {
		return 3200, nil // e.g. ask an exchange API
	})
	fmt.Println(price, err)
	fmt.Println(slices.Collect(maps.Keys(maps.Collect(prices.All()))))
	fmt.Println(prices.Stats())
	// Output:
	// 61000 true
	// false
	// 3200 <nil>
	// [ETH]
	// hits=1 misses=2 evictions=0 hit-rate=33%
}
