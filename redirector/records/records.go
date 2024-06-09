/*
Package records provides an abstract interface to the NoSQL database powering the app,
allowing for the integration of middleware such as an LRU cache to reduce the amount of
communication between microservices.
*/
package records

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/luizcdc/redirectory/redirector/records/lru_cache"
	"github.com/luizcdc/redirectory/redirector/records/redis_client"
)

var cache *lru_cache.LRUCache

// MakeCache initializes the local cache with the specified capacity.
func MakeCache(cap uint) {
	internal_cache_expire_seconds, err := strconv.Atoi(os.Getenv("INTERNAL_CACHE_EXPIRE_SECONDS"))
	if err != nil {
		log.Println("Error reading INTERNAL_CACHE_EXPIRE_SECONDS environment variable. Could not create the cache.")
		return
	}
	if cache == nil {
		cache = lru_cache.NewCache(cap, time.Duration(internal_cache_expire_seconds)*time.Second)
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
	success := client.Set(context.TODO(), AddPrefix(key), value, ttl).Err() == nil
	if success {
		cache.Insert(key, value)
	}
	return success
}

// AddPrefix adds a prefix to a key to separate keys from different environments.
func AddPrefix(key string) string {
	return fmt.Sprintf("%s:%s", os.Getenv("RUNNING_ENV"), key)
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
	return client.Get(context.TODO(), AddPrefix(key)).Result()
}

// GetAllKeys retrieves all keys that start with a prefix, with the
// prefix itself removed.
func GetAllKeys() ([]string, error) {
	prefix := AddPrefix("")
	client, err := redis_client.GetClientInstance()
	if err != nil {
		return []string{}, err
	}
	keys, err := client.Keys(context.TODO(), prefix+"*").Result()
	if err != nil {
		return keys, err
	}
	for i := range keys {
		keys[i] = keys[i][len(prefix):]
	}
	return keys, err
}

func clearRedis() {
	client, err := redis_client.GetClientInstance()
	if err != nil {
		return
	}
	client.FlushAll(context.TODO())
}