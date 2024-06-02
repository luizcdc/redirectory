/*
Package records provides an abstract interface to the NoSQL database powering the app,
allowing for the integration of middleware such as an LRU cache to reduce the amount of
communication between microservices.
*/
package records

import (
	"context"
	"time"

	"github.com/luizcdc/redirectory/redirector/records/redis_client"
)

// SetKey returns a bool indicating success of the specified set operation.
func SetKey(key string, value interface{}, ttl time.Duration) bool {
	client, err := redis_client.GetClientInstance()
	if err != nil {
		return false
	}
	return client.Set(context.Background(), key, value, ttl).Err() == nil
}

// GetString retrieves a string value from Redis.
func GetString(key string)  (string, error) {
	client, err := redis_client.GetClientInstance()
	if err != nil {
		return "", err
	}
	return client.Get(context.Background(), key).Result()
}