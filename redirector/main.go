package main

import (
	"context"
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

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	secretmanagerpb "cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"github.com/joho/godotenv"
	"github.com/julienschmidt/httprouter"
	"github.com/luizcdc/redirectory/redirector/records"
	"github.com/luizcdc/redirectory/redirector/uint_to_any_base"
)

type Auth struct {
	handler httprouter.Router
}

// ServeHTTP is implements the http.Handler interface for the Auth struct, checking the
// Authorization header for the API_KEY before serving the request.
func (a *Auth) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if API_KEY != "" && r.Header.Get("Authorization") != fmt.Sprintf("Bearer %s", API_KEY) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	a.handler.ServeHTTP(w, r)
}

var intToString *uint_to_any_base.NumeralSystem
var ALLOWED_CHARS, API_KEY string
var RANDOM_SIZE, PROJECT_NUMBER int
var DEFAULT_DURATION uint
var APPLICATION_JSON = "application/json"

// initConstants sets the global constants from the environment variables.
func initConstants() {
	ALLOWED_CHARS = os.Getenv("ALLOWED_CHARS")

	intRandomChars, err := strconv.Atoi(os.Getenv("DEFAULT_RANDOM_STRING_SIZE"))
	if err != nil {
		log.Fatalf("failure reading RANDOM_SIZE into an int constant: %v", err.Error())
	}
	RANDOM_SIZE = intRandomChars

	intToString, err = uint_to_any_base.NewNumeralSystem(uint32(len(ALLOWED_CHARS)), ALLOWED_CHARS, uint32(RANDOM_SIZE))
	if err != nil {
		log.Fatalf("failure creating NumeralSystem to generate strings from ints: %v", err.Error())
	}

	API_KEY = os.Getenv("API_KEY")

	duration, err := strconv.Atoi(os.Getenv("DEFAULT_DURATION"))
	if err != nil {
		log.Fatalf("failure reading DEFAULT_DURATION into an int constant: %v", err.Error())
	}
	DEFAULT_DURATION = uint(duration)
}

// getProjectNumber retrieves the project number from the environment variables or from the metadata server.
func getProjectNumber() {
	if os.Getenv("PROJECT_NUMBER") == "" {

		req, _ := http.NewRequest(
			"GET",
			"http://metadata.google.internal/computeMetadata/v1/project/numeric-project-id",
			nil,
		)
		req.Header.Add("Metadata-Flavor", "Google")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Fatalf("failure getting project number from metadata server: %v", err.Error())
		}
		defer resp.Body.Close()
		body := make([]byte, 100)
		n, err := resp.Body.Read(body)
		if err != nil && err != io.EOF {
			log.Fatalf("failure reading project number from metadata server: %v", err.Error())
		}
		os.Setenv("PROJECT_NUMBER", string(body[:n]))
	}
	var err error
	PROJECT_NUMBER, err = strconv.Atoi(os.Getenv("PROJECT_NUMBER"))
	if err != nil {
		log.Fatalf("failure converting project number to int: %v", err.Error())
	}
}

// getSecrets retrieves sensitive environment variables from GCP Secret Manager
// and sets them in the runtime environment.
func getSecrets() {
	getProjectNumber()
	log.Println("Getting secrets from GCP Secret Manager")
	ctx := context.Background()
	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		log.Fatalf("failed to create secretmanager client: %v", err)
	}

	result, err := client.AccessSecretVersion(ctx, &secretmanagerpb.AccessSecretVersionRequest{
		Name: os.Getenv("REDIS_PASSWORD_RESOURCE_ID"),
	})
	if err != nil {
		log.Fatalf("failed to access REDIS_PASSWORD_RESOURCE_ID: %v", err)
	}

	os.Setenv("REDIS_PASSWORD", string(result.Payload.Data))

	result, err = client.AccessSecretVersion(ctx, &secretmanagerpb.AccessSecretVersionRequest{
		Name: os.Getenv("API_KEY_RESOURCE_ID"),
	})

	if err != nil {
		log.Fatalf("failed to access API_KEY_RESOURCE_ID: %v", err)
	}

	apiKey := result.Payload.Data

	os.Setenv("API_KEY", string(apiKey))

	result, err = client.AccessSecretVersion(ctx, &secretmanagerpb.AccessSecretVersionRequest{
		Name: os.Getenv("REDIS_HOST_RESOURCE_ID"),
	})

	if err != nil {
		log.Fatalf("failed to access REDIS_HOST_RESOURCE_ID: %v", err)
	}

	os.Setenv("REDIS_HOST", string(result.Payload.Data))

	log.Println("Secrets loaded successfully")
}

// loadEnv loads environment variables from a .env file if the application is not running on
// Google App Engine.
func loadEnv() {
	if os.Getenv("GAE_APPLICATION") == "" {
		if godotenv.Load() != nil {
			log.Fatal("Error loading .env file")
		}
	} else {
		log.Println("Running on Google App Engine, environment variables are already set.")
		getSecrets()
	}
	initConstants()
	log.Println("Environment variables loaded successfully")
}

// setErrorJSONReply is a higher-order function that returns a function
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
//	errorHandler := setErrorJSONReply(w)
//	errorHandler(http.StatusInternalServerError, "Internal Server Error because...")
func setErrorJSONReply(w http.ResponseWriter) func(int, string) {
	return func(status int, err string) {
		w.WriteHeader(status)
		resp, _ := json.Marshal(struct {
			Error    string `json:"error"`
			Path     string `json:"path"`
			Duration uint   `json:"duration"`
		}{err, "", 0})
		w.Write(resp)
	}
}

// setSuccessJSONReply is a higher-order function that returns a function
// responsible for sending a JSON response with the specified status code
// and success message.
//
// Parameters:
//   - w: The http.ResponseWriter that will write the response.
//
// Returns:
//
//	A function (path string, duration uint) that sends a JSON response with the specified
//
// path and duration in the "path" and "duration" fields, respectively.
//
// Example usage:
//
//	successHandler := setSuccessJSONReply(w)
//	successHandler("path", 10)
func setSuccessJSONReply(w http.ResponseWriter) func(string, uint) {
	return func(path string, duration uint) {
		w.WriteHeader(http.StatusOK)
		resp, _ := json.Marshal(struct {
			Error    interface{} `json:"error"`
			Path     string      `json:"path"`
			Duration uint        `json:"duration"`
		}{nil, path, duration})
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
//		{
//		  "error": null,
//	   "path": "path"
//	   "duration": 10
//		}
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

	replyError := setErrorJSONReply(w)
	replySuccess := setSuccessJSONReply(w)

	w.Header().Add("Content-Type", APPLICATION_JSON)

	if len(ps.ByName("path")) < 4 {
		replyError(http.StatusBadRequest, "path must be at least 4 characters long")
		return
	}
	from := ps.ByName("path")

	buffer, sizeRead, err := readJSONIntoBuffer(r, replyError)
	if err != nil {
		log.Println(err.Error())
		return
	}

	var jsonBody setRedirectBody
	if err := json.Unmarshal(buffer[:sizeRead], &jsonBody); err != nil {
		log.Println(err)
		replyError(http.StatusBadRequest, fmt.Sprintf("error parsing json in the request's body: %v", err.Error()))
		return
	}
	parsedUrl, err := url.Parse(jsonBody.Url)
	switch {
	case err != nil:
		log.Println(err)
		replyError(http.StatusBadRequest, fmt.Sprintf("the provided url is invalid: %v", err.Error()))
		return
	case !parsedUrl.IsAbs():
		log.Println(err)
		replyError(http.StatusBadRequest, "the provided url must be absolute")
		return
	}

	duration := DEFAULT_DURATION
	if jsonBody.Duration != 0 {
		duration = jsonBody.Duration
	}

	if records.SetKey(from, parsedUrl.String(), time.Duration(duration)*time.Second) {
		log.Printf("Success setting '%v' to '%v', for '%v' seconds\n", from, parsedUrl.String(), duration)
		replySuccess(from, duration)
		return
	}

	replyError(http.StatusInternalServerError, fmt.Sprintf("failure setting '%v' to '%v'", from, parsedUrl.String()))
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

	replyError := setErrorJSONReply(w)
	replySuccess := setSuccessJSONReply(w)

	w.Header().Add("Content-Type", APPLICATION_JSON)
	buffer, sizeRead, err := readJSONIntoBuffer(r, replyError)
	if err != nil {
		log.Println(err)
		return
	}

	var jsonBody setRandomRedirectBody
	if err := json.Unmarshal(buffer[:sizeRead], &jsonBody); err != nil {
		replyError(http.StatusBadRequest, err.Error())
		log.Println(err)
		return
	}

	parsedUrl, err := url.Parse(jsonBody.Url)
	switch {
	case err != nil:
		replyError(http.StatusBadRequest, err.Error())
		log.Println(err)
		return
	case !parsedUrl.IsAbs():
		replyError(http.StatusBadRequest, "the provided url must be absolute")
		return
	}

	nPossibilities := int32(math.Pow(float64(len(ALLOWED_CHARS)), float64(RANDOM_SIZE)))

	var chosen string
	for {
		var err error
		chosen, err = intToString.IntegerToString(uint32(rand.Int31n(nPossibilities)))
		if err != nil {
			replyError(http.StatusInternalServerError, err.Error())
			return
		}
		if _, err := records.GetString(chosen); err != nil {
			break
		}
	}

	duration := DEFAULT_DURATION
	if jsonBody.Duration != 0 {
		duration = jsonBody.Duration
	}

	if records.SetKey(chosen, parsedUrl.String(), time.Duration(duration)*time.Second) {
		replySuccess(chosen, duration)
		log.Printf("Success setting '%v' to '%v'\n", chosen, parsedUrl.String())
		return
	}

	replyError(http.StatusInternalServerError, fmt.Sprintf("failure setting '%v' to '%v'", chosen, parsedUrl.String()))
	log.Printf("Failure setting '%v' to '%v'\n", chosen, parsedUrl.String())

}

// readJSONIntoBuffer reads JSON data from the request body into a buffer (prior to unmarshalling it).
// It checks if the appropriate headers are set and if the content length is valid.
// If any of the checks fail, it replies to the request with an error and returns the error,
// a nil buffer, and 0 bytes read.
// Otherwise, it reads the JSON data into the buffer and returns the buffer and the number of
// bytes read.
func readJSONIntoBuffer(r *http.Request, replyError func(int, string)) ([]byte, int, error) {
	if !strings.Contains(r.Header.Get("content-type"), APPLICATION_JSON) {
		err := fmt.Errorf("Content-Type must be 'application/json'")
		replyError(http.StatusBadRequest, err.Error())
		return nil, 0, err
	}

	length, err := strconv.Atoi(r.Header.Get("content-length"))
	if err != nil {
		log.Println(err.Error())
		err := fmt.Errorf("Content-Length header is required and must be valid")
		replyError(http.StatusBadRequest, err.Error())
		return nil, 0, err
	}

	buffer := make([]byte, min(length, int(math.Pow(2, 16))))
	sizeRead, err := r.Body.Read(buffer)
	if err == io.EOF {
		err = nil
	}
	if err != nil {
		log.Println(err.Error())
		err := fmt.Errorf("error reading the request's body: %v", err.Error())
		replyError(http.StatusInternalServerError, err.Error())
		return nil, 0, err
	}
	return buffer, sizeRead, err
}

// Redirect serves the redirect request for a previously set redirect path.
func Redirect(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	key := ps.ByName("redirectpath")[1:]
	redirectTo, err := records.GetString(key)
	if err != nil {
		log.Printf("Error: no redirect for key '%v'\n", key)
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(fmt.Sprintf("<h1>Error %v: URL not found!</h1>", http.StatusNotFound)))
		return
	}
	w.Header().Set("Location", redirectTo)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

// DelRedirect deletes the redirect for a given path.
func DelRedirect(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	delErrorJSONReply := func(status int, err string) {
		w.WriteHeader(status)
		resp, _ := json.Marshal(struct {
			Error interface{} `json:"error"`
		}{err})
		w.Write(resp)
	}
	if len(ps.ByName("path")) == 0 {
		replyError := setErrorJSONReply(w)
		replyError(http.StatusBadRequest, "no redirect to delete was specified")
		return
	}
	path := ps.ByName("path")
	deleted, err := records.DelKey(path)
	if deleted {
		w.WriteHeader(http.StatusOK)
		resp, _ := json.Marshal(struct {
			Error interface{} `json:"error"`
		}{nil})
		w.Write(resp)
	} else if err == nil {
		delErrorJSONReply(http.StatusNotFound, fmt.Sprintf("no redirect found for path '%v'", path))
	} else {
		delErrorJSONReply(http.StatusInternalServerError, fmt.Sprintf("error deleting redirect for path '%v': %v", path, err.Error()))
	}

}

func main() {
	loadEnv()
	// records.MakeCache(5)
	requireAuthRouter := httprouter.New()
	requireAuthRouter.POST("/set/:path", SetSpecificRedirect)
	requireAuthRouter.POST("/set", SetRandomRedirect)
	requireAuthRouter.DELETE("/del/:path", DelRedirect)
	auth := &Auth{*requireAuthRouter}

	router := httprouter.New()
	router.Handler(http.MethodPost, "/set/:path", auth)
	router.Handler(http.MethodPost, "/set", auth)
	router.Handler(http.MethodDelete, "/del/:path", auth)

	router.GET("/*redirectpath", Redirect)
	log.Println("Server running on port 8080")

	log.Fatal(http.ListenAndServe(":8080", router))
}
