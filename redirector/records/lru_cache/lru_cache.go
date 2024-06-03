package lru_cache

type LRUCache struct {
	hashmap map[string]*Node
	// The most recently used
	lru_head *Node
}

type Node struct {
	value    string
	next     *Node
	previous *Node
}

// InsertNode moves the given node into the head of the lru list (as the most recently used entry).
func (cache *LRUCache) Hit(key string) {
	if cache == nil {
		return
	}
	// TODO: implement
}

// Insert creates a node and inserts the given value into the head of the lru list (as the most
// recently used entry).
func (cache *LRUCache) Insert(key string, val string) {
	if cache == nil {
		return
	}
	_, ok := cache.hashmap[key]
	if ok {
		return
	}
	n := Node{value: val}
	cache.hashmap[key] = &n

	if cache.lru_head == nil {
		n.next, n.previous = nil, nil
	} else {
		last := cache.lru_head.previous
		last.next = &n
		cache.lru_head.previous = &n
		n.next = cache.lru_head
		n.previous = last
	}
	cache.lru_head = &n
}

// Remove removes a node from the lru list, freeing memory.
func (cache *LRUCache) Remove(key string) {
	if cache == nil {
		return
	}
	// TODO: implement
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