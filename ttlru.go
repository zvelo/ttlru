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

type entry struct {
	key     interface{}
	value   interface{}
	index   int
	expires time.Time
	timer   *time.Timer
}

type Cache interface {
	// Set a key with value to the cache. Returns true if an item was
	// evicted.
	Set(key, value interface{}) bool

	// Get an item from the cache by key. Returns the value if it exists,
	// and a bool stating whether or not it existed.
	Get(key interface{}) (interface{}, bool)

	// Keys returns a slice of all the keys in the cache
	Keys() []interface{}

	// Len returns the number of items present in the cache
	Len() int

	// Cap returns the total number of items the cache can retain
	Cap() int

	// Purge removes all items from the cache
	Purge()

	// Del deletes an item from the cache by key. Returns if an item was
	// actually deleted.
	Del(key interface{}) bool
}

type Option func(*cache)

func WithTTL(val time.Duration) Option {
	return func(c *cache) {
		c.ttl = val
	}
}

func WithoutReset() Option {
	return func(c *cache) {
		c.NoReset = true
	}
}

// cache is the type that implements the ttlru
type cache struct {
	cap     int
	ttl     time.Duration
	items   map[interface{}]*entry
	heap    *ttlHeap
	lock    sync.RWMutex
	NoReset bool
}

// New creates a new Cache with cap entries that expire after ttl has
// elapsed since the item was added, modified or accessed.
func New(cap int, opts ...Option) Cache {
	c := cache{cap: cap}

	for _, opt := range opts {
		opt(&c)
	}

	if c.cap <= 0 || c.ttl < 0 {
		return nil
	}

	c.items = make(map[interface{}]*entry, cap)

	h := make(ttlHeap, 0, cap)
	c.heap = &h

	// no need to init the heap as there are no items yet

	return &c
}

func (c *cache) Set(key, value interface{}) bool {
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

func (c *cache) insertEntry(key, value interface{}) *entry {
	// must already have a write lock

	ent := &entry{
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

func (c *cache) updateEntry(e *entry, value interface{}) {
	// must already have a write lock

	// update with the new value
	e.value = value

	// reset the ttl
	c.resetEntryTTL(e)
}

func (c *cache) resetEntryTTL(e *entry) {
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

func (c *cache) removeEntry(e *entry) {
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

func (c *cache) Get(key interface{}) (interface{}, bool) {
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

	return nil, false
}

func (c *cache) Keys() []interface{} {
	c.lock.RLock()
	defer c.lock.RUnlock()

	keys := make([]interface{}, 0, len(c.items))
	for k, v := range c.items {
		// the item should be automatically removed when it expires, but we
		// check just to be safe
		if c.ttl == 0 || time.Now().Before(v.expires) {
			keys = append(keys, k)
		}
	}

	return keys
}

func (c *cache) Len() int {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return len(c.items)
}

func (c *cache) Cap() int {
	return c.cap
}

func (c *cache) Purge() {
	c.lock.Lock()
	defer c.lock.Unlock()

	for _, e := range c.items {
		e.index = -1
	}

	h := make(ttlHeap, 0, c.cap)
	c.heap = &h
	c.items = make(map[interface{}]*entry, c.cap)
}

func (c *cache) Del(key interface{}) bool {
	c.lock.Lock()
	defer c.lock.Unlock()

	if ent, ok := c.items[key]; ok {
		c.removeEntry(ent)
		return true
	}

	return false
}
