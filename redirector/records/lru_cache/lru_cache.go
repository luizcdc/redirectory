package lru_cache

// LRUCache is a least recently used cache for string->string entries.
type LRUCache struct {
	cap 		uint
	hashmap map[string]*node
	// The most recently used
	lru_head *node
}

// node is a doubly-linked list node.
type node struct {
	key 	 string
	value    string
	next     *node
	previous *node
}

// NewCache constructs an LRUCache.
func NewCache(cap uint) * LRUCache {
	return &LRUCache {
		cap: cap,
		hashmap: make(map[string]*node, cap),
		lru_head: nil,
	}
}

// insertNode inserts an already allocated node into the first position of the lru list.
func (cache *LRUCache) insertNode(n *node) {
	last := cache.lru_head.previous
	last.next = n
	cache.lru_head.previous = n
	n.next = cache.lru_head
	n.previous = last
	cache.lru_head = n
}

// peek returns the head of the lru list (most recently used).
func (cache LRUCache) peekMRU() *node {
	return cache.lru_head
}

// peekLRU returns the tail of the lru list (least recently used).
func (cache LRUCache) peekLRU() *node {
	if cache.lru_head == nil {
		return nil
	}
	return cache.lru_head.previous
}

// ChangeCap changes the capacity of the cache. If the new capacity is less than the current length 
// of the cache, the least recently used entries are removed.
func (cache *LRUCache) ChangeCap(newcap int) {
	// TODO: Implement
	return
} 

func (cache *LRUCache) Len() int {
	if cache == nil {
		return 0
	}
	return len(cache.hashmap)
}

// Contains indicates whether the key is present at the moment.
func (cache *LRUCache) Contains(key string) bool {
	return cache != nil && cache.hashmap[key] != nil
}

// Insert creates an entry and inserts it as the most recently used, returning the value to the 
// caller.
func (cache *LRUCache) Insert(key string, val string) string {
	if cache == nil {
		return val
	}

	// When the key is present, we bump it to most recently used and update it.
	if cache.hashmap[key] != nil {
		cache.Hit(key)
		cache.hashmap[key].value = val
		return val
	}

	if cache.Len() == int(cache.cap) {
		cache.DropLRU()
	}

	n := node{key: key, value: val}
	cache.hashmap[key] = &n

	if cache.lru_head == nil {
		n.next, n.previous = &n, &n
		cache.lru_head = &n
	} else {
		cache.insertNode(&n)
	}
	return val
}

// Hit moves the given entry into the head of the lru list (as the most recently used entry).
func (cache *LRUCache) Hit(key string) {
	if !cache.Contains(key) {
		return
	}
	node := cache.hashmap[key]
	node.previous.next = node.next
	node.next.previous = node.previous

	cache.insertNode(node)
}

// Remove removes a node from the lru list, freeing memory.
func (cache *LRUCache) Remove(key string) {
	if cache == nil || cache.hashmap[key] == nil {
		return
	}
	node := cache.hashmap[key]
	delete(cache.hashmap, key)
	if cache.Len() == 1 {
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

// DropLRU removes the least recently used entry.
func (cache *LRUCache) DropLRU() {
	if cache == nil || cache.Len() == 0 {
		return
	}
	key := cache.lru_head.previous.key
	cache.Remove(key)
}

func (cache *LRUCache) Fetch(key string) (value string, ok bool) {
	if cache == nil {
		return
	}
	node, ok := cache.hashmap[key]
	if !ok {
		return
	}
	cache.Hit(key)
	value = node.value
	return
}