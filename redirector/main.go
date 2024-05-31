package main

import (
	"fmt"
	"log"
	"net/http"
)

const STATUS_TEMPORARY_REDIRECTION = 302

func Redirect(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Response now!")
	w.Header().Set("Location", "https://google.com/")
	w.WriteHeader(STATUS_TEMPORARY_REDIRECTION)
}

func main() {
	http.HandleFunc("/", Redirect)
	log.Fatal(http.ListenAndServe(":8080", nil))
}