// Package redis manages the redis_client_singleton singleton and offers useful methods.
package redis_client

import (
	"context"
	"log"

	"github.com/redis/go-redis/v9"
)

// redis_client_singleton is the only instance of a redis.Client within the whole application.
var redis_client_singleton *redis.Client

// instantiateClient instantiates the client into the redis_client_singleton global.
func instantiateClient() error {
	redis_client_singleton = redis.NewClient(
		&redis.Options{
		Addr: "localhost:6379",
		Password: "",
		DB: 0,
	})
	err := redis_client_singleton.Ping(context.Background()).Err()
	if err != nil {
		log.Printf("Error (%v): failed creating Redis client\n", err)
	}else {
		log.Println("Created Redis client.")
	}
	return err
}

// GetClientInstance provides a global access point to redis_client_singleton, initializing it
// if necessary.
func GetClientInstance () (redis.Client, error) {
	err := error(nil)

	if redis_client_singleton == nil {
		err = instantiateClient()
	}

	if err != nil {
		return redis.Client{}, err
	}

	return *redis_client_singleton, err
}