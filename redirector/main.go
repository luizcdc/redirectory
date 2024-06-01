package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
)

const STATUS_TEMPORARY_REDIRECTION = 302

func GetTarget(path string) string {
	path = strings.ToLower(path[1:])
	return "https://" + "www.google.com/search?q=" + url.QueryEscape(path)
}

func Redirect(w http.ResponseWriter, r *http.Request) {
	fmt.Println(r.URL.Path[1:])
	w.Header().Set("Location", GetTarget(r.URL.Path))
	w.WriteHeader(STATUS_TEMPORARY_REDIRECTION)
}

func main() {
	http.HandleFunc("/", Redirect)
	log.Fatal(http.ListenAndServe(":8080", nil))
}