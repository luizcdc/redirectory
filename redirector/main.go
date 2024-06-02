package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/redis/go-redis/v9"
)

const STATUS_TEMPORARY_REDIRECTION = 302
const SECONDS = 1e9

var redis_client = redis.NewClient(
	&redis.Options{
		Addr: "localhost:6379",
		Password: "",
		DB: 0,
	})

func GetTarget(path string) string {
	path = strings.ToLower(path)
	return "https://" + "www.google.com/search?q=" + url.QueryEscape(path)
}

func SetRedirect(client * redis.Client) func (http.ResponseWriter, *http.Request) {
	setKey := func (key string, value string) bool {
		return client.Set(context.Background(), "test:"+ key, value, 10 * SECONDS).Err() == nil
	}
	return func (w http.ResponseWriter, r *http.Request) {
		params := strings.Split(r.URL.Path[len("/SetRedirect/"):], "/")[:2]
		key, value := params[0], params[1]
		fmt.Println(key, value)
		if setKey(key, value) {
			fmt.Printf("Success setting %v to %v\n", key, value)
			return
		}
		fmt.Println("FAILED!")
	}
}


func Redirect(client *redis.Client) func (http.ResponseWriter, * http.Request) {
	

	getKey := func (key string) string {
		result := client.Get(context.Background(), "test:" + key)
		if result.Err() == nil {
			return result.Val()	
		}
		fmt.Println("Returning key")
		return key
	}
	
	return func (w http.ResponseWriter, r *http.Request) {
		key := r.URL.Path[1:]
		redirectTo := getKey(key)
		w.Header().Set("Location", GetTarget(redirectTo))
		w.WriteHeader(STATUS_TEMPORARY_REDIRECTION)
	}
}

func main() {
	http.HandleFunc("/setredirect/", SetRedirect(redis_client))
	http.HandleFunc("/", Redirect(redis_client))
	log.Println(http.ListenAndServe(":8080", nil))

}