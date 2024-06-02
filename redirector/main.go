package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/luizcdc/redirectory/redirector/records"
)

const TEMPORARY_REDIRECTION = 302
const BAD_REQUEST = 400
const NOT_FOUND = 404
const SECONDS = 1e9



func GetTarget(path string) string {
	path = strings.ToLower(path)
	return "https://" + "www.google.com/search?q=" + url.QueryEscape(path)
}

func SetRedirect(w http.ResponseWriter, r *http.Request) {
	if len(r.URL.Path) < len("/setredirect/") {
		w.WriteHeader(BAD_REQUEST)
		w.Write([]byte(fmt.Sprintf("<h1>Error %v: bad request", BAD_REQUEST)))
		return
	}
	params := strings.Split(r.URL.Path[len("/SetRedirect/"):], "/")[:2]
	key, value := params[0], params[1]
	fmt.Println(key, value)
	if records.SetKey(key, value, 10 * SECONDS) {
		fmt.Printf("Success setting '%v' to '%v'\n", key, value)
		return
	}
	fmt.Println("FAILED!")
}


func Redirect (w http.ResponseWriter, r * http.Request) {
		key := r.URL.Path[1:]
		redirectTo, err := records.GetString(key)
		if err != nil {
			log.Printf("Error (%v): no redirect for key '%v'\n", err.Error(), key)
			w.WriteHeader(NOT_FOUND)
			w.Write([]byte(fmt.Sprintf("<h1>Error %v: URL not found!</h1>", NOT_FOUND)))
			return
		}
		w.Header().Set("Location", GetTarget(redirectTo))
		w.WriteHeader(TEMPORARY_REDIRECTION)
}

func main() {
	http.HandleFunc("/setredirect", SetRedirect)
	http.HandleFunc("/setredirect/", SetRedirect)
	http.HandleFunc("/", Redirect)
	log.Println(http.ListenAndServe(":8080", nil))

}