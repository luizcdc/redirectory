package records

import (
	"context"

	redis_client "github.com/luizcdc/redirectory/redirector/records/redis_client_singleton"
)

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