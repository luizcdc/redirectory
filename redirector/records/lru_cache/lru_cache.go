// lru_cache provides an LRUCache, an LRU cache with configurable per-entry expiration.
package lru_cache

import (
	"sync"
	"time"
)

// LRUCache is a least recently used cache for string->string entries.
type LRUCache struct {
	mu         sync.Mutex
	expiration time.Duration
	cap        uint
	hashmap    map[string]*node
	// The most recently used
	lru_head *node
}

// node is a doubly-linked list node.
type node struct {
	key          string
	value        interface{}
	last_updated time.Time
	next         *node
	previous     *node
}

// NewCache constructs an LRUCache.
func NewCache(cap uint, expiration time.Duration) *LRUCache {
	return &LRUCache{
		expiration: expiration,
		cap:        cap,
		hashmap:    make(map[string]*node, cap),
		lru_head:   nil,
	}
}

// insertNode inserts an already allocated node into the first position of the lru list.
// Time complexity: O(1)
func (cache *LRUCache) insertNode(n *node) {
	if cache == nil {
		panic("Internal method cache.insertNode() cannot receive nil cache")
	}
	last := cache.lru_head.previous
	last.next = n
	cache.lru_head.previous = n
	n.next = cache.lru_head
	n.previous = last
	cache.lru_head = n
}

// peek returns the head of the lru list (most recently used).
// Time complexity: O(1)
func (cache *LRUCache) peekMRU() *node {
	if cache == nil {
		panic("Internal method cache.peekMRU() cannot receive nil cache")
	}
	return cache.lru_head
}

// peekLRU returns the tail of the lru list (least recently used).
// Time complexity: O(1)
func (cache *LRUCache) peekLRU() *node {
	if cache == nil {
		panic("Internal method cache.peekLRU() cannot receive nil cache")
	}
	if cache.lru_head == nil {
		return nil
	}
	return cache.lru_head.previous
}

// changecap changes the capacity of the cache. If the new capacity is less than the current length
// of the cache, the least recently used entries are removed.
// Time complexity: O(cache.cap)
func (cache *LRUCache) changeCap(newcap uint) {
	newcap = max(0, newcap)
	if newcap == 0 {
		cache.hashmap = make(map[string]*node)
		cache.lru_head = nil
	} else if newcap <= cache.cap {
		for range uint(cache.len()) - newcap {
			cache.dropLRU()
		}
		cache.hashmap = make(map[string]*node, newcap)
		var stopAt *node
		for node := cache.lru_head; node != stopAt; node = node.next {
			stopAt = cache.lru_head
			cache.hashmap[node.key] = node
		}
	}
	cache.cap = newcap
}

// ChangeCap changes the capacity of the cache. If the new capacity is less than the current length
// of the cache, the least recently used entries are removed.
// Time complexity: O(cache.cap)
func (cache *LRUCache) ChangeCap(newcap uint) {
	if cache == nil {
		return
	}
	cache.mu.Lock()
	defer cache.mu.Unlock()

	cache.changeCap(newcap)
}

// len returns the numbers of entries currently in the cache.
// Time complexity: O(1)
func (cache *LRUCache) len() int {
	if cache == nil {
		panic("Internal method cache.len() cannot receive nil cache")
	}
	return len(cache.hashmap)
}

// Len returns the number of entries currently in the cache.
// Time complexity: O(1)
func (cache *LRUCache) Len() int {
	if cache == nil {
		return 0
	}

	cache.mu.Lock()
	defer cache.mu.Unlock()

	return cache.len()
}

// Cap returns the capacity of the cache.
// Time complexity: O(1)
func (cache *LRUCache) Cap() uint {
	if cache == nil {
		return 0
	}

	return cache.cap
}

// contains indicates whether the key is present at the moment (without locking).
// Time complexity: O(1)
func (cache *LRUCache) contains(key string) bool {
	if cache == nil {
		panic("Internal method cache.contains() cannot receive nil cache")
	}
	return cache.hashmap[key] != nil && time.Since(cache.hashmap[key].last_updated) < cache.expiration
}

// Contains indicates whether the key is present at the moment.
// Time complexity: O(1)
func (cache *LRUCache) Contains(key string) bool {
	if cache == nil {
		return false
	}

	cache.mu.Lock()
	defer cache.mu.Unlock()

	return cache.contains(key)
}

// insert creates an entry and inserts it as the most recently used, returning the value to the
// caller (without locking).
// Time complexity: O(1)
func (cache *LRUCache) insert(key string, val interface{}) interface{} {
	if cache == nil {
		panic("Internal method cache.insert() cannot receive nil cache")
	}

	// When the key is present, we bump it to most recently used and update it.
	if cache.hashmap[key] != nil {
		cache.hit(key)
		cache.hashmap[key].value = val
		cache.hashmap[key].last_updated = time.Now()
		return val
	}

	if cache.len() == int(cache.cap) {
		cache.dropLRU()
	}

	n := node{key: key, value: val, last_updated: time.Now()}
	cache.hashmap[key] = &n

	if cache.lru_head == nil {
		n.next, n.previous = &n, &n
		cache.lru_head = &n
	} else {
		cache.insertNode(&n)
	}
	return val

}

// Insert creates an entry and inserts it as the most recently used, returning the value to the
// caller.
// Time complexity: O(1)
func (cache *LRUCache) Insert(key string, val interface{}) interface{} {
	if cache == nil {
		return val
	}

	cache.mu.Lock()
	defer cache.mu.Unlock()

	return cache.insert(key, val)
}

// hit moves the given entry into the head of the lru list (as the most recently used entry)
// (without locking).
// Time complexity: O(1)
func (cache *LRUCache) hit(key string) {
	node := cache.hashmap[key]
	node.previous.next = node.next
	node.next.previous = node.previous

	cache.insertNode(node)
}

// Hit moves the given entry into the head of the lru list (as the most recently used entry).
// Time complexity: O(1)
func (cache *LRUCache) Hit(key string) {
	if !cache.Contains(key) {
		return
	}

	cache.mu.Lock()
	defer cache.mu.Unlock()

	cache.hit(key)
}

// remove removes a node from the lru list, freeing memory (without locking).
// Time complexity: O(1)
func (cache *LRUCache) remove(key string) {
	if cache.hashmap[key] == nil {
		return
	}

	node := cache.hashmap[key]
	delete(cache.hashmap, key)
	if cache.len() == 1 {
		// It is the only node in the cache.
		cache.lru_head = nil
		return
	}
	if cache.lru_head == node {
		cache.lru_head = node.next
	}
	node.previous.next = node.next
	node.next.previous = node.previous
}

// Remove removes a node from the lru list, freeing memory.
// Time complexity: O(1)
func (cache *LRUCache) Remove(key string) {
	if cache == nil {
		return
	}

	cache.mu.Lock()
	defer cache.mu.Unlock()

	cache.remove(key)
}

// dropLRU removes the least recently used entry (without locking).
func (cache *LRUCache) dropLRU() {
	key := cache.lru_head.previous.key
	cache.remove(key)
}

// DropLRU removes the least recently used entry.
// Time complexity: O(1)
func (cache *LRUCache) DropLRU() {
	if cache == nil || cache.Len() == 0 {
		return
	}

	cache.mu.Lock()
	defer cache.mu.Unlock()

	cache.dropLRU()
}

// fetch gets the value for the key if it exists in the cache, also returning whether it
// found the value. If it did, the key is bumped to most recently used. Doesn't lock the mutex.
// Time complexity: O(1)
func (cache *LRUCache) fetch(key string) (value interface{}, ok bool) {
	node, ok := cache.hashmap[key]
	if !ok {
		return
	}
	if time.Since(node.last_updated) > cache.expiration {
		cache.remove(key)
		ok = false
		return
	}
	cache.hit(key)
	value = node.value
	return
}

// Fetch gets the value for the key if it exists in the cache, also returning whether it
// found the value. If it did, the key is bumped to most recently used.
// Time complexity: O(1)
func (cache *LRUCache) Fetch(key string) (value interface{}, ok bool) {
	if cache == nil {
		return
	}

	cache.mu.Lock()
	defer cache.mu.Unlock()

	return cache.fetch(key)
}
