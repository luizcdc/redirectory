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
	redis_client "github.com/luizcdc/redirectory/redirector/records/redis_client_singleton"
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

func ResetCache() {
	currCap := cache.Cap()
	MakeCache(0)
	MakeCache(currCap)
}

// SetKey returns a bool indicating success of the specified set operation.
func SetKey(key string, value interface{}, ttl time.Duration) bool {
	client, err := redis_client.GetClientInstance()
	if err != nil {
		log.Println("Error getting Redis client instance." + err.Error())
		return false
	}
	err = client.Set(context.TODO(), AddPrefix(key), value, ttl).Err()
	if err == nil {
		cache.Insert(key, value)
	} else {
		log.Println("Error setting key in Redis. " + err.Error())
	}
	go incrCountURLsSet()
	return err == nil
}

// DelKey deletes a key, returning true and nil if the key existed and was successfully deleted,
// or false and an error if not.
func DelKey(key string) (bool, error) {
	client, err := redis_client.GetClientInstance()
	if err != nil {
		log.Println("Error getting Redis client instance. " + err.Error())
		return false, err
	}
	numRemoved, err := client.Del(context.TODO(), AddPrefix(key)).Result()
	if err == nil {
		if numRemoved > 0 {
			cache.Remove(key)
		}
	} else {
		log.Println("Error deleting key in Redis. " + err.Error())
	}
	return numRemoved > 0, err
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

// clearRedis clears all keys from the Redis database.
func clearRedis() {
	client, err := redis_client.GetClientInstance()
	if err != nil {
		return
	}
	go client.FlushAll(context.TODO())
	ResetCache()
	go clearCountURLsSet()
}

// incrCountURLsSet increments the count of all URLs ever set.
func incrCountURLsSet() {
	client, err := redis_client.GetClientInstance()
	if err != nil {
		return
	}
	client.Incr(context.TODO(), AddPrefix("count_urls_set"))
}

// GetCountURLsSet retrieves the count of all URLs ever set.
func GetCountURLsSet() (int64, error) {
	client, err := redis_client.GetClientInstance()
	if err != nil {
		return 0, err
	}
	return client.Get(context.TODO(), AddPrefix("count_urls_set")).Int64()
}

// clearCountURLsSet clears the count of all URLs ever set.
func clearCountURLsSet() {
	client, err := redis_client.GetClientInstance()
	if err != nil {
		return
	}
	client.Set(context.TODO(), AddPrefix("count_urls_set"), 0, 0)
}

// IncrCountServedRedirects increments the count of all redirects ever served.
func IncrCountServedRedirects() {
	client, err := redis_client.GetClientInstance()
	if err != nil {
		return
	}
	client.Incr(context.TODO(), AddPrefix("count_served_redirects"))
}

// GetCountServedRedirects retrieves the count of all redirects ever served.
func GetCountServedRedirects() (int64, error) {
	client, err := redis_client.GetClientInstance()
	if err != nil {
		return 0, err
	}
	return client.Get(context.TODO(), AddPrefix("count_served_redirects")).Int64()
}

// clearCountServedRedirects clears the count of all redirects ever served.
func clearCountServedRedirects() {
	client, err := redis_client.GetClientInstance()
	if err != nil {
		return
	}
	client.Set(context.TODO(), AddPrefix("count_served_redirects"), 0, 0)
}
