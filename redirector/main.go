package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/luizcdc/redirectory/redirector/records"
)

const SECONDS = 1e9

func GetTarget(path string) string {
	path = strings.ToLower(path)
	return "https://" + "www.google.com/search?q=" + url.QueryEscape(path)
}

func SetRedirect(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	type setRedirectBody struct {
		Url      string `json:"url"`
		Duration uint   `json:"duration"`
	}
	// TODO: respond to error handling with json
	switch {
	case len(ps.ByName("path")) < 4:
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("<h1>Error %v: bad request1", http.StatusBadRequest)))
		return
	case !strings.Contains(r.Header.Get("content-type"), "application/json"):
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("<h1>Error %v: content-type must be 'application/json'.", http.StatusBadRequest)))
		return
	}

	jsonBody := setRedirectBody{}
	length, err := strconv.Atoi(r.Header.Get("content-length"))
	if err != nil {
		fmt.Println(err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("<h1>Error %v: bad request2", http.StatusBadRequest)))
		return
	}
	from := ps.ByName("path")

	buffer := make([]byte, max(length, int(math.Pow(2, 16))))
	n, err := r.Body.Read(buffer)
	if err != nil && err != io.EOF {
		fmt.Println(err.Error())
		w.WriteHeader(500)
		w.Write([]byte(fmt.Sprintf("<h1>Error %v: internal server error", http.StatusBadRequest)))
		return
	}
	err = json.Unmarshal(buffer[:n], &jsonBody)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("<h1>Error %v: bad request3", http.StatusBadRequest)))
		return
	
	}
	fmt.Println(from, jsonBody.Url, jsonBody.Duration)

	duration := 10
	if jsonBody.Duration != 0 {
		duration = int(jsonBody.Duration)
	}

	if records.SetKey(from, jsonBody.Url, time.Duration(duration*SECONDS)) {
		fmt.Printf("Success setting '%v' to '%v'\n", from, jsonBody.Url)
		// TODO: Response
		return
	}
	fmt.Println("FAILED!")
}

func Redirect(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	key := ps.ByName("redirectpath")[1:]
	fmt.Println(key)
	redirecto, err := records.GetString(key)
	if err != nil {
		log.Printf("Error: no redirect for key '%v'\n", key)
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(fmt.Sprintf("<h1>Error %v: URL not found!</h1>", http.StatusNotFound)))
		return
	}
	w.Header().Set("Location", GetTarget(redirecto))
	w.WriteHeader(http.StatusTemporaryRedirect)
}

func main() {
	records.MakeCache(5)
	router := httprouter.New()
	router.POST("/set/:path", SetRedirect)
	router.GET("/*redirectpath", Redirect)
	log.Fatal(http.ListenAndServe(":8080", router))
}
