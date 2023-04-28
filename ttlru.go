// Package ttlru provides a simple, goroutine safe, cache with a fixed number of
// entries. Each entry has a per-cache defined TTL. This TTL is reset on both
// modification and access to the value. As a result, if the cache is full, and
// no items have expired, when adding a new item, the item with the soonest
// expiration will be evicted.
//
// It is based on the LRU implementation in golang-lru:
// github.com/hashicorp/golang-lru
//
// Which in turn is based on the LRU implementation in groupcache:
// github.com/golang/groupcache/lru
package ttlru // import "zvelo.io/ttlru"

import (
	"container/heap"
	"sync"
	"time"
)

type entry[K comparable, V any] struct {
	key     K
	value   V
	index   int
	expires time.Time
	timer   *time.Timer
}

// Cache interface.
type Cache[K comparable, V any] interface {
	// Set a key with value to the cache. Returns true if an item was
	// evicted.
	Set(key K, value V) bool

	// Get an item from the cache by key. Returns the value if it exists,
	// and a bool stating whether or not it existed.
	Get(key K) (V, bool)

	// Keys returns a slice of all the keys in the cache
	Keys() []K

	// Len returns the number of items present in the cache
	Len() int

	// Cap returns the total number of items the cache can retain
	Cap() int

	// Purge removes all items from the cache
	Purge()

	// Del deletes an item from the cache by key. Returns if an item was
	// actually deleted.
	Del(key K) bool
}

// Option type.
type Option func(*configuration)

// WithTTL option.
func WithTTL(val time.Duration) Option {
	return func(c *configuration) {
		c.ttl = val
	}
}

// WithoutReset option.
func WithoutReset() Option {
	return func(c *configuration) {
		c.NoReset = true
	}
}

// cache is the type that implements the ttlru
type cache[K comparable, V any] struct {
	cap int
	configuration
	items map[K]*entry[K, V]
	heap  *ttlHeap[K, V]
	lock  sync.RWMutex
}

// configuration type
type configuration struct {
	ttl     time.Duration
	NoReset bool
}

// New creates a new Cache with cap entries that expire after ttl has
// elapsed since the item was added, modified or accessed.
func New[K comparable, V any](cap int, opts ...Option) Cache[K, V] {
	c := cache[K, V]{cap: cap}

	for _, opt := range opts {
		opt(&c.configuration)
	}

	if c.cap <= 0 || c.ttl < 0 {
		return nil
	}

	c.items = make(map[K]*entry[K, V], cap)

	h := make(ttlHeap[K, V], 0, cap)
	c.heap = &h

	// no need to init the heap as there are no items yet

	return &c
}

func (c *cache[K, V]) Set(key K, value V) bool {
	c.lock.Lock()
	defer c.lock.Unlock()

	// Check for existing item
	if ent, ok := c.items[key]; ok {
		c.updateEntry(ent, value)
		return false
	}

	// Evict oldest if next entry would exceed capacity
	evict := len(*c.heap) == c.cap
	if evict {
		if ent := (*c.heap)[0]; ent != nil {
			c.removeEntry(ent)
		}
	}

	c.insertEntry(key, value)

	return evict
}

func (c *cache[K, V]) insertEntry(key K, value V) *entry[K, V] {
	// must already have a write lock

	ent := &entry[K, V]{
		key:     key,
		value:   value,
		expires: time.Now().Add(c.ttl),
	}

	if c.ttl > 0 {
		ent.timer = time.AfterFunc(c.ttl, func() {
			c.lock.Lock()
			defer c.lock.Unlock()
			c.removeEntry(ent)
		})
	}

	heap.Push(c.heap, ent)
	c.items[key] = ent

	return ent
}

func (c *cache[K, V]) updateEntry(e *entry[K, V], value V) {
	// must already have a write lock

	// update with the new value
	e.value = value

	// reset the ttl
	c.resetEntryTTL(e)
}

func (c *cache[K, V]) resetEntryTTL(e *entry[K, V]) {
	// must already have a write lock

	// reset the expiration timer
	if c.ttl > 0 {
		e.timer.Reset(c.ttl)
	}

	// set the new expiration time
	e.expires = time.Now().Add(c.ttl)

	// fix heap ordering
	heap.Fix(c.heap, e.index)
}

func (c *cache[K, V]) removeEntry(e *entry[K, V]) {
	// must already have a write lock

	if e.index >= 0 {
		heap.Remove(c.heap, e.index)
	}

	// if a ttl was set, stop the timer to avoid leaking timers
	if e.timer != nil {
		e.timer.Stop()
	}

	// delete the item from the map
	delete(c.items, e.key)
}

func (c *cache[K, V]) Get(key K) (V, bool) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if ent, ok := c.items[key]; ok {
		// the item should be automatically removed when it expires, but we
		// check just to be safe
		if c.ttl == 0 || time.Now().Before(ent.expires) {
			if !c.NoReset {
				c.resetEntryTTL(ent)
			}
			return ent.value, true
		}
	}

	var v V

	return v, false
}

func (c *cache[K, V]) Keys() []K {
	c.lock.RLock()
	defer c.lock.RUnlock()

	keys := make([]K, 0, len(c.items))
	for k, v := range c.items {
		// the item should be automatically removed when it expires, but we
		// check just to be safe
		if c.ttl == 0 || time.Now().Before(v.expires) {
			keys = append(keys, k)
		}
	}

	return keys
}

func (c *cache[K, V]) Len() int {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return len(c.items)
}

func (c *cache[K, V]) Cap() int {
	return c.cap
}

func (c *cache[K, V]) Purge() {
	c.lock.Lock()
	defer c.lock.Unlock()

	for _, e := range c.items {
		e.index = -1
	}

	h := make(ttlHeap[K, V], 0, c.cap)
	c.heap = &h
	c.items = make(map[K]*entry[K, V], c.cap)
}

func (c *cache[K, V]) Del(key K) bool {
	c.lock.Lock()
	defer c.lock.Unlock()

	if ent, ok := c.items[key]; ok {
		c.removeEntry(ent)
		return true
	}

	return false
}
