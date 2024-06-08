/*
Package records provides an abstract interface to the NoSQL database powering the app,
allowing for the integration of middleware such as an LRU cache to reduce the amount of
communication between microservices.
*/
package records

import (
	"context"
	"time"

	"github.com/luizcdc/redirectory/redirector/records/lru_cache"
	"github.com/luizcdc/redirectory/redirector/records/redis_client"
)

var cache *lru_cache.LRUCache

// MakeCache initializes the local cache with the specified capacity.
func MakeCache(cap uint) {
	// TODO: set up environment variable for cache duration
	if cache == nil {
		cache = lru_cache.NewCache(cap, 6e11)
	} else {
		cache.ChangeCap(cap)
	}
}

// SetKey returns a bool indicating success of the specified set operation.
func SetKey(key string, value interface{}, ttl time.Duration) bool {
	client, err := redis_client.GetClientInstance()
	if err != nil {
		return false
	}
	success := client.Set(context.Background(), key, value, ttl).Err() == nil
	if success {
		cache.Insert(key, value)
	}
	return success
}

// GetString retrieves a string value from Redis.
func GetString(key string) (string, error) {
	value, ok := cache.Fetch(key)
	if ok {
		return_value, ok := value.(string)
		if ok {
			return return_value, nil
		}
	}
	client, err := redis_client.GetClientInstance()
	if err != nil {
		return "", err
	}
	return client.Get(context.Background(), key).Result()
}

// GetAllStringsWithoutPrefix retrieves all keys that start with a prefix, with the
// prefix itself removed.
func GetAllStringsWithoutPrefix(prefix string) ([]string, error) {
	client, err := redis_client.GetClientInstance()
	if err != nil {
		return []string{}, err
	}
	keys, err := client.Keys(context.Background(), prefix + "*").Result()
	if err != nil {
		return keys, err
	}
	for i := range keys {
		keys[i] = keys[i][len(prefix):]
	}
	return keys, err
}