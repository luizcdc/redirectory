package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/julienschmidt/httprouter"
	"github.com/luizcdc/redirectory/redirector/records"
	"github.com/luizcdc/redirectory/redirector/uint_to_any_base"
)

var intToString *uint_to_any_base.NumeralSystem
var ALLOWED_CHARS string
var RANDOM_SIZE int

// loadEnv loads environment variables from a .env file if the application is not running on
// Google App Engine.
func loadEnv() {
	if os.Getenv("GAE_APPLICATION") != "" {
		log.Println("Running on Google App Engine, environment variables are already set.")
		// TODO: load secrets from GCP Secret Manager
	} else {
		if godotenv.Load() != nil {
			log.Fatal("Error loading .env file")
		}
	}
	ALLOWED_CHARS = os.Getenv("ALLOWED_CHARS")
	var err error
	RANDOM_SIZE, err = strconv.Atoi(os.Getenv("DEFAULT_RANDOM_STRING_SIZE"))
	if err != nil {
		log.Fatalf("failure loading DEFAULT_RANDOM_STRING_SIZE environment variable")
	}
	if intToString == nil {
		var err error
		intToString, err = uint_to_any_base.NewNumeralSystem(uint32(len(ALLOWED_CHARS)), ALLOWED_CHARS, uint32(RANDOM_SIZE))
		if err != nil {
			log.Fatalf("failure creating NumeralSystem to generate strings from ints: %v", err.Error())
		}
	}
}

// simpleErrorJSONReply is a higher-order function that returns a function
// responsible for sending a JSON response with the specified status code
// and error message in the "error" field.
//
// Parameters:
//   - w: The http.ResponseWriter that will write the response.
//
// Returns:
//
//	A function (status int, err interface{}) that and sends a JSON response with the specified
//
// status code and error message in the "error" field.
//
// Example usage:
//
//	errorHandler := simpleErrorJSONReply(w)
//	errorHandler(http.StatusInternalServerError, "Internal Server Error because...")
func simpleErrorJSONReply(w http.ResponseWriter) func(int, interface{}) {
	return func(status int, err interface{}) {
		w.WriteHeader(status)
		resp, _ := json.Marshal(struct {
			Error interface{} `json:"error"`
		}{err})
		w.Write(resp)
	}
}

// SetSpecificRedirect sets a redirect for a given path.
// It expects a JSON payload in the request body with the following structure:
//
//	{
//	  "url": "https://example.com",
//	  "duration": 10
//	}
//
// The "url" field specifies the target URL for the redirect, and the "duration" field (optional)
// specifies the duration of the redirect in seconds.
// The function returns a JSON response indicating the success or failure of setting the redirect.
// If the redirect is set successfully, the response will be:
//
//	{
//	  "error": null
//	}
//
// If there is an error in setting the redirect, the response will be:
//
//	{
//	  "error": "failure message"
//	}
func SetSpecificRedirect(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	type setRedirectBody struct {
		Url      string `json:"url"`
		Duration uint   `json:"duration"`
	}

	reply := simpleErrorJSONReply(w)

	w.Header().Add("Content-Type", "application/json")
	// TODO: use readJSONIntoBuffer here
	switch {
	case len(ps.ByName("path")) < 4:
		reply(http.StatusBadRequest, "path must be at least 4 characters long")
		return
	case !strings.Contains(r.Header.Get("content-type"), "application/json"):
		reply(http.StatusBadRequest, "Content-Type must be 'application/json'")
		return
	}

	length, err := strconv.Atoi(r.Header.Get("content-length"))
	if err != nil {
		log.Println(err.Error())
		reply(http.StatusBadRequest, "Content-Length header is required")
		return
	}
	from := ps.ByName("path")

	buffer := make([]byte, min(length, int(math.Pow(2, 16))))
	sizeRead, err := r.Body.Read(buffer)
	if err != nil && err != io.EOF {
		log.Println(err.Error())
		reply(http.StatusInternalServerError, fmt.Sprintf("error reading the request's body: %v", err.Error()))
		return
	}
	var jsonBody setRedirectBody

	if err := json.Unmarshal(buffer[:sizeRead], &jsonBody); err != nil {
		log.Println(err)
		reply(http.StatusBadRequest, fmt.Sprintf("error parsing json in the request's body: %v", err.Error()))
		return
	}
	log.Println(from, jsonBody.Url, jsonBody.Duration)
	parsedUrl, err := url.Parse(jsonBody.Url)
	switch {
	case err != nil:
		log.Println(err)
		reply(http.StatusBadRequest, fmt.Sprintf("the provided url is invalid: %v", err.Error()))
		return
	case !parsedUrl.IsAbs():
		log.Println(err)
		reply(http.StatusBadRequest, "the provided url must be absolute")
		return
	}

	duration := 60
	if jsonBody.Duration != 0 {
		duration = int(jsonBody.Duration)
	}

	if records.SetKey(from, parsedUrl.String(), time.Duration(duration)*time.Second) {
		log.Printf("Success setting '%v' to '%v'\n", from, parsedUrl.String())
		reply(http.StatusOK, nil)
		return
	}

	reply(http.StatusInternalServerError, fmt.Sprintf("failure setting '%v' to '%v'", from, parsedUrl.String()))
	log.Printf("Failure setting '%v' to '%v'\n", from, parsedUrl.String())

}

// SetRandomRedirect sets a random redirect URL with a specified duration.
// The function reads a JSON body from the request, parses the URL, and generates a random string
// which will be the path that will redirect to the specified URL.
// The duration of the redirect can be specified in the JSON body, otherwise it follows the default.
// The function returns a JSON response with the generated string as the path of the redirect.
// If the redirect is set successfully, the response will be:
//
//	{
//	  "error": null,
//	  "path": "generated_path"
//	}
//
// If any errors occur during the process, an appropriate error response is returned:
//
//	{
//	  "error": "error message",
//	  "path": ""
//	}
func SetRandomRedirect(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	type setRandomRedirectBody struct {
		Url      string `json:"url"`
		Duration uint   `json:"duration"`
	}

	reply := func(status int, err interface{}, redirectPath string, duration uint) {
		w.WriteHeader(status)
		resp, _ := json.Marshal(struct {
			Error interface{} `json:"error"`
			Path  string      `json:"path"`
			Duration uint   `json:"duration"`
		}{err, redirectPath, duration})
		w.Write(resp)
	}

	w.Header().Add("Content-Type", "application/json")
	err, buffer, sizeRead := readJSONIntoBuffer(r)
	if err != nil {
		reply(http.StatusBadRequest, err.Error(), "", 0)
		log.Println(err)
		return
	}

	var jsonBody setRandomRedirectBody
	if err := json.Unmarshal(buffer[:sizeRead], &jsonBody); err != nil {
		reply(http.StatusBadRequest, err.Error(), "", 0)
		log.Println(err)
		return
	}

	parsedUrl, err := url.Parse(jsonBody.Url)
	switch {
	case err != nil:
		reply(http.StatusBadRequest, err.Error(), "", 0)
		log.Println(err)
		return
	case !parsedUrl.IsAbs():
		reply(http.StatusBadRequest, "the provided url must be absolute", "", 0)
		return
	}

	nPossibilities := int32(math.Pow(float64(len(ALLOWED_CHARS)), float64(RANDOM_SIZE)))

	var chosen string
	for {
		var err error
		chosen, err = intToString.IntegerToString(uint32(rand.Int31n(nPossibilities)))
		if err != nil {
			reply(http.StatusInternalServerError, err.Error(), "", 0)
			return
		}
		if _, err := records.GetString(chosen); err != nil {
			break
		}
	}

	duration := uint(60)
	if jsonBody.Duration != 0 {
		duration = jsonBody.Duration
	}

	if records.SetKey(chosen, parsedUrl.String(), time.Duration(duration)*time.Second) {
		reply(http.StatusOK, nil, chosen, duration)
		log.Printf("Success setting '%v' to '%v'\n", chosen, parsedUrl.String())
		return
	}

	reply(http.StatusInternalServerError, fmt.Sprintf("failure setting '%v' to '%v'", chosen, parsedUrl.String()), "", 0)
	log.Printf("Failure setting '%v' to '%v'\n", chosen, parsedUrl.String())

}

// readJSONIntoBuffer reads JSON data from the request body into a buffer (prior to unmarshalling it).
// It checks if the appropriate headers are set and if the content length is valid.
// If any of the checks fail, it returns an error along with a nil buffer and sizeRead of 0.
// Otherwise, it reads the JSON data into the buffer and returns the buffer and the number of 
// bytes read.
func readJSONIntoBuffer(r *http.Request) (error, []byte, int) {
	if !strings.Contains(r.Header.Get("content-type"), "application/json") {
		return fmt.Errorf("Content-Type must be 'application/json'"), nil, 0
	}

	length, err := strconv.Atoi(r.Header.Get("content-length"))
	if err != nil {
		log.Println(err.Error())

		return fmt.Errorf("Content-Length header is required"), nil, 0
	}

	buffer := make([]byte, min(length, int(math.Pow(2, 16))))
	sizeRead, err := r.Body.Read(buffer)
	if err == io.EOF {
		err = nil
	}
	if err != nil {

		log.Println(err.Error())
		return err, nil, 0
	}
	return err, buffer, sizeRead
}

// Redirect serves the redirect request for a previously set redirect path.
func Redirect(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	key := ps.ByName("redirectpath")[1:]
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
	// records.MakeCache(5)
	router := httprouter.New()
	router.POST("/set/:path", SetSpecificRedirect)
	router.POST("/set", SetRandomRedirect)
	router.GET("/*redirectpath", Redirect)
	log.Fatal(http.ListenAndServe(":8080", router))
}
