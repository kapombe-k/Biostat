package cache

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestCacheLRUEvictionAndTTL(t *testing.T) {
	now := time.Date(2026, 10, 23, 12, 0, 0, 0, time.UTC)
	c := New[string, string](3, 10*time.Minute)
	c.nowFn = func() time.Time { return now }

	c.Set("a", "1")
	c.Set("b", "2")
	c.Set("c", "3")

	// Access "a" so "b" becomes least recently used.
	if v, ok := c.Get("a"); !ok || v != "1" {
		t.Fatalf("expected a=1, got %q (%v)", v, ok)
	}

	c.Set("d", "4") // Should evict "b".
	if _, ok := c.Get("b"); ok {
		t.Fatalf("expected LRU item 'b' to be evicted")
	}
	if _, ok := c.Get("a"); !ok {
		t.Fatalf("expected 'a' to still be cached")
	}

	// Advance clock past TTL.
	now = now.Add(11 * time.Minute)
	if _, ok := c.Get("a"); ok {
		t.Fatalf("expected 'a' to expire after TTL")
	}

	st := c.Stats()
	if st.Evictions != 1 {
		t.Fatalf("expected 1 eviction, got %d", st.Evictions)
	}
}

func TestFlightGroupDeduplication(t *testing.T) {
	fg := NewFlightGroup[int]()
	var executions atomic.Int32

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			v, err := fg.Do(context.Background(), "P04637", func() (int, error) {
				executions.Add(1)
				time.Sleep(30 * time.Millisecond)
				return 42, nil
			})
			if err != nil || v != 42 {
				t.Errorf("unexpected flight result: %v, %v", v, err)
			}
		}()
	}
	wg.Wait()

	if got := executions.Load(); got != 1 {
		t.Fatalf("expected single upstream execution, got %d", got)
	}
}
