package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/julienschmidt/httprouter"
	"github.com/luizcdc/redirectory/redirector/records"
)

// loadEnv loads environment variables from a .env file if the application is not running on Google App Engine.
func loadEnv() {
	if os.Getenv("GAE_APPLICATION") != "" {
		log.Println("Running on Google App Engine, environment variables are already set.")
		// TODO: load secrets from GCP Secret Manager
	} else {
		if godotenv.Load() != nil {
			log.Fatal("Error loading .env file")
		}
	}
}

// SetRedirect sets a redirect for a given path.
// It expects a JSON payload in the request body with the following structure:
// {
//   "url": "https://example.com",
//   "duration": 10
// }
// The "url" field specifies the target URL for the redirect, and the "duration" field (optional) 
// specifies the duration of the redirect in seconds.
// The function returns a JSON response indicating the success or failure of setting the redirect.
// If the redirect is set successfully, the response will be:
// {
//   "error": null
// }
// If there is an error in setting the redirect, the response will be:
// {
//   "error": "failure message"
// }
func SetRedirect(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	type setRedirectBody struct {
		Url      string `json:"url"`
		Duration uint   `json:"duration"`
	}
	w.Header().Add("Content-Type", "application/json")
	switch {
	case len(ps.ByName("path")) < 4:
		w.WriteHeader(http.StatusBadRequest)
		resp, _ := json.Marshal(map[string]string{"error": "path must be at least 4 characters long"})
		w.Write(resp)
		return
	case !strings.Contains(r.Header.Get("content-type"), "application/json"):
		w.WriteHeader(http.StatusBadRequest)
		resp, _ := json.Marshal(map[string]string{"error": "Content-Type must be 'application/json'"})
		w.Write(resp)
		return
	}

	length, err := strconv.Atoi(r.Header.Get("content-length"))
	if err != nil {
		fmt.Println(err.Error())
		w.WriteHeader(http.StatusBadRequest)
		resp, _ := json.Marshal(map[string]string{"error": "Content-Length header is required"})
		w.Write(resp)
		return
	}
	from := ps.ByName("path")

	buffer := make([]byte, max(length, int(math.Pow(2, 16))))
	n, err := r.Body.Read(buffer)
	if err != nil && err != io.EOF {
		fmt.Println(err.Error())
		w.WriteHeader(500)
		resp, _ := json.Marshal(map[string]string{"error": fmt.Sprintf("error reading the request's body: %v", err.Error())})
		w.Write(resp)
		return
	}
	var jsonBody setRedirectBody
	err = json.Unmarshal(buffer[:n], &jsonBody)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		resp, _ := json.Marshal(map[string]string{"error": fmt.Sprintf("error parsing json in the request's body: %v", err.Error())})
		w.Write(resp)
		return
	}
	fmt.Println(from, jsonBody.Url, jsonBody.Duration)
	parsedUrl, err := url.Parse(jsonBody.Url)
	switch {
	case err != nil:
		fmt.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		resp, _ := json.Marshal(map[string]string{"error": fmt.Sprintf("the provided url is invalid: %v", err.Error())})
		w.Write(resp)
		return
	case !parsedUrl.IsAbs():
		fmt.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		resp, _ := json.Marshal(map[string]string{"error": fmt.Sprintf("the provided url must be absolute")})
		w.Write(resp)
		return
	case parsedUrl.Scheme == "":
		parsedUrl.Scheme = "https"
	}

	duration := 10
	if jsonBody.Duration != 0 {
		duration = int(jsonBody.Duration)
	}

	if records.SetKey(from, parsedUrl.String(), time.Duration(duration)*time.Second) {
		fmt.Printf("Success setting '%v' to '%v'\n", from, parsedUrl.String())
		resp, _ := json.Marshal(map[string]interface{}{"error": nil})
		w.Write(resp)
		return
	}

	resp, _ := json.Marshal(map[string]string{"error": fmt.Sprintf("failure setting '%v' to '%v'", from, parsedUrl.String())})
	w.Write(resp)
	fmt.Printf("Failure setting '%v' to '%v'\n", from, parsedUrl.String())

}

// Redirect serves the redirect request for a previously set redirect path.
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
	w.Header().Set("Location", redirecto)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

func main() {
	loadEnv()
	records.MakeCache(5)
	router := httprouter.New()
	router.POST("/set/:path", SetRedirect)
	router.GET("/*redirectpath", Redirect)
	log.Fatal(http.ListenAndServe(":8080", router))
}
