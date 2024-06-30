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
	"strconv"
	"strings"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/luizcdc/redirectory/redirector/records"
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

// createAuthSubRouter initializes an auth-only subrouter, setting up routes and handlers.
func CreateAuthSubRouter() *Auth {
	requireAuthRouter := httprouter.New()
	requireAuthRouter.POST(API_ROOT+"set/:path", SetSpecificRedirect)
	requireAuthRouter.POST(API_ROOT+"set", SetRandomRedirect)
	requireAuthRouter.DELETE(API_ROOT+"del/:path", DelRedirect)
	requireAuthRouter.GET(API_ROOT+"stats/urlcount", GetTotalSetRedirects)
	requireAuthRouter.GET(API_ROOT+"stats/redirectcount", GetTotalServedRedirects)
	AuthSubRouter := &Auth{*requireAuthRouter}
	return AuthSubRouter
}

func DefineRoutes(AuthSubRouter *Auth) *httprouter.Router {
	router := httprouter.New()

	router.Handler(http.MethodGet, "/:redirectpath/*any", AuthSubRouter)
	router.Handler(http.MethodPost, API_ROOT+"*any", AuthSubRouter)
	router.Handler(http.MethodDelete, API_ROOT+"*any", AuthSubRouter)
	router.Handler(http.MethodPut, API_ROOT+"*any", AuthSubRouter)

	router.GET("/:redirectpath", Redirect)
	return router
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
	key := ps.ByName("redirectpath")
	key = strings.Trim(key, "/")
	redirectTo, err := records.GetString(key)
	if err != nil {
		log.Printf("Error: no redirect for key '%v'\n", key)
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(fmt.Sprintf("<h1>Error %v: URL not found!</h1>", http.StatusNotFound)))
		return
	}
	go records.IncrCountServedRedirects()
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

// GetTotalServedRedirects returns the total number of served redirects.
// The function returns a JSON response, where the body is the total number of served redirects.
// If there is an error in retrieving the count, the response will be 'null'.
func GetTotalServedRedirects(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	returnCount(w, records.GetCountServedRedirects)
}

// GetTotalSetRedirects returns the total number of set redirects.
// The function returns a JSON response, where the body is the total number of set redirects.
// If there is an error in retrieving the count, the response will be 'null'.
func GetTotalSetRedirects(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	returnCount(w, records.GetCountURLsSet)
}

// returnCount is a helper function that returns the value of a specified counter.
func returnCount(w http.ResponseWriter, getCount func() (int64, error)) {
	w.Header().Add("Content-Type", APPLICATION_JSON)
	totalURLs, err := getCount()
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("null"))
		return
	}
	w.Write([]byte(fmt.Sprint(totalURLs)))
}