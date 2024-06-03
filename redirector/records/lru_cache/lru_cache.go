package lru_cache

type LRUCache struct {
	hashmap map[string]Node
	// The most recently used
	lru_head *Node
}

type Node struct {
	value    string
	next     *Node
	previous *Node
}

func (cache LRUCache) InsertNode(n Node) {
	// TODO: implement
	return
}

func (cache LRUCache) Remove(n Node) {
	// TODO: implement
	return
}

func (cache LRUCache) Fetch(key string) (string, bool) {
	// TODO: redo
	v, ok := cache.hashmap[key]
	if !ok {
		return "", false
	}
	cache.Remove(v)
	cache.InsertNode(v)
	return key, true
}