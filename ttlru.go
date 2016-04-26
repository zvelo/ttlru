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
package ttlru

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

// Cache is the type that implements the ttlru
type Cache struct {
	cap     int
	ttl     time.Duration
	items   map[interface{}]*entry
	heap    *ttlHeap
	lock    sync.RWMutex
	NoReset bool
}

// New creates a new Cache with cap entries that expire after ttl has
// elapsed since the item was added, modified or accessed.
func New(cap int, ttl time.Duration) *Cache {
	if cap <= 0 || ttl <= 0 {
		return nil
	}

	c := &Cache{
		cap:   cap,
		ttl:   ttl,
		items: make(map[interface{}]*entry, cap),
	}

	h := make(ttlHeap, 0, cap)
	c.heap = &h

	// no need to init the heap as there are no items yet

	return c
}

// Set a key with value to the cache. Returns true if an item was evicted.
func (c *Cache) Set(key, value interface{}) bool {
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

func (c *Cache) insertEntry(key, value interface{}) *entry {
	// must already have a write lock

	ent := &entry{
		key:     key,
		value:   value,
		expires: time.Now().Add(c.ttl),
	}

	ent.timer = time.AfterFunc(c.ttl, func() {
		c.lock.Lock()
		defer c.lock.Unlock()
		c.removeEntry(ent)
	})

	heap.Push(c.heap, ent)
	c.items[key] = ent

	return ent
}

func (c *Cache) updateEntry(e *entry, value interface{}) {
	// must already have a write lock

	// update with the new value
	e.value = value

	// reset the ttl
	c.resetEntryTTL(e)
}

func (c *Cache) resetEntryTTL(e *entry) {
	// must already have a write lock

	// reset the expiration timer
	e.timer.Reset(c.ttl)

	// set the new expiration time
	e.expires = time.Now().Add(c.ttl)

	// fix heap ordering
	heap.Fix(c.heap, e.index)
}

func (c *Cache) removeEntry(e *entry) {
	// must already have a write lock

	if e.index >= 0 {
		heap.Remove(c.heap, e.index)
	}

	// delete the item from the map
	delete(c.items, e.key)
}

// Get an item from the cache by key. Returns the value if it exists, and a bool
// stating whether or not it existed.
func (c *Cache) Get(key interface{}) (interface{}, bool) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if ent, ok := c.items[key]; ok {
		// the item should be automatically removed when it expires, but we
		// check just to be safe
		if time.Now().Before(ent.expires) {
			if c.NoReset != true {
				c.resetEntryTTL(ent)
			}
			return ent.value, true
		}
	}

	return nil, false
}

// Keys returns a slice of all the keys in the cache
func (c *Cache) Keys() []interface{} {
	c.lock.RLock()
	defer c.lock.RUnlock()

	keys := make([]interface{}, len(c.items))
	i := 0
	for k, v := range c.items {
		// the item should be automatically removed when it expires, but we
		// check just to be safe
		if time.Now().Before(v.expires) {
			keys[i] = k
		}
		i++
	}

	return keys
}

// Len returns the number of items present in the cache
func (c *Cache) Len() int {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return len(c.items)
}

// Cap returns the total number of items the cache can retain
func (c *Cache) Cap() int {
	return c.cap
}

// Purge removes all items from the cache
func (c *Cache) Purge() {
	c.lock.Lock()
	defer c.lock.Unlock()

	h := make(ttlHeap, 0, c.cap)
	c.heap = &h
	c.items = make(map[interface{}]*entry, c.cap)
}

// Del deletes an item from the cache by key. Returns if an item was actually
// deleted.
func (c *Cache) Del(key interface{}) bool {
	c.lock.Lock()
	defer c.lock.Unlock()

	if ent, ok := c.items[key]; ok {
		c.removeEntry(ent)
		return true
	}

	return false
}
