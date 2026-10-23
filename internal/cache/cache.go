package cache

import (
	"container/list"
	"context"
	"sync"
	"time"
)

type cacheEntry[K comparable, V any] struct {
	key       K
	value     V
	expiresAt time.Time
}

// Stats exposes bounded cache telemetry for health and diagnostics.
type Stats struct {
	Items     int    `json:"items"`
	MaxItems  int    `json:"max_items"`
	Hits      uint64 `json:"hits"`
	Misses    uint64 `json:"misses"`
	Evictions uint64 `json:"evictions"`
}

// Cache is a thread-safe, bounded LRU cache with per-entry TTL expiration.
type Cache[K comparable, V any] struct {
	mu        sync.Mutex
	maxItems  int
	ttl       time.Duration
	ll        *list.List
	items     map[K]*list.Element
	hits      uint64
	misses    uint64
	evictions uint64
	nowFn     func() time.Time
}

// New creates a bounded LRU cache with default TTL.
func New[K comparable, V any](maxItems int, ttl time.Duration) *Cache[K, V] {
	if maxItems <= 0 {
		maxItems = 128
	}
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	return &Cache[K, V]{
		maxItems: maxItems,
		ttl:      ttl,
		ll:       list.New(),
		items:    make(map[K]*list.Element, maxItems),
		nowFn:    time.Now,
	}
}

// Get retrieves an item if present and unexpired, promoting it to MRU position.
func (c *Cache[K, V]) Get(key K) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var zero V
	el, ok := c.items[key]
	if !ok {
		c.misses++
		return zero, false
	}
	ent := el.Value.(*cacheEntry[K, V])
	if c.nowFn().After(ent.expiresAt) {
		c.removeElement(el)
		c.misses++
		return zero, false
	}
	c.ll.MoveToFront(el)
	c.hits++
	return ent.value, true
}

// Set inserts or updates key with default TTL, evicting the least-recently-used entry if full.
func (c *Cache[K, V]) Set(key K, value V) {
	c.SetWithTTL(key, value, c.ttl)
}

// SetWithTTL inserts or updates key with a custom TTL.
func (c *Cache[K, V]) SetWithTTL(key K, value V, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	exp := c.nowFn().Add(ttl)
	if el, ok := c.items[key]; ok {
		c.ll.MoveToFront(el)
		ent := el.Value.(*cacheEntry[K, V])
		ent.value = value
		ent.expiresAt = exp
		return
	}

	ent := &cacheEntry[K, V]{
		key:       key,
		value:     value,
		expiresAt: exp,
	}
	el := c.ll.PushFront(ent)
	c.items[key] = el

	for c.ll.Len() > c.maxItems {
		oldest := c.ll.Back()
		if oldest != nil {
			c.removeElement(oldest)
			c.evictions++
		}
	}
}

// Delete removes a key if present.
func (c *Cache[K, V]) Delete(key K) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if el, ok := c.items[key]; ok {
		c.removeElement(el)
	}
}

// Stats returns current cache occupancy and hit/miss counters.
func (c *Cache[K, V]) Stats() Stats {
	c.mu.Lock()
	defer c.mu.Unlock()
	return Stats{
		Items:     c.ll.Len(),
		MaxItems:  c.maxItems,
		Hits:      c.hits,
		Misses:    c.misses,
		Evictions: c.evictions,
	}
}

func (c *Cache[K, V]) removeElement(el *list.Element) {
	c.ll.Remove(el)
	ent := el.Value.(*cacheEntry[K, V])
	delete(c.items, ent.key)
}

type flightCall[V any] struct {
	done chan struct{}
	val  V
	err  error
}

// FlightGroup deduplicates concurrent in-flight upstream API calls for identical keys.
type FlightGroup[V any] struct {
	mu    sync.Mutex
	calls map[string]*flightCall[V]
}

// NewFlightGroup constructs a concurrent request deduplicator.
func NewFlightGroup[V any]() *FlightGroup[V] {
	return &FlightGroup[V]{
		calls: make(map[string]*flightCall[V]),
	}
}

// Do executes fn once for concurrent callers sharing key, while honoring each caller's context.
func (g *FlightGroup[V]) Do(ctx context.Context, key string, fn func() (V, error)) (V, error) {
	g.mu.Lock()
	if call, ok := g.calls[key]; ok {
		g.mu.Unlock()
		select {
		case <-call.done:
			return call.val, call.err
		case <-ctx.Done():
			var zero V
			return zero, ctx.Err()
		}
	}

	call := &flightCall[V]{done: make(chan struct{})}
	g.calls[key] = call
	g.mu.Unlock()

	go func() {
		defer func() {
			g.mu.Lock()
			delete(g.calls, key)
			g.mu.Unlock()
			close(call.done)
		}()
		call.val, call.err = fn()
	}()

	select {
	case <-call.done:
		return call.val, call.err
	case <-ctx.Done():
		var zero V
		return zero, ctx.Err()
	}
}
