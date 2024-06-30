package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	secretmanagerpb "cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"github.com/joho/godotenv"
	"github.com/luizcdc/redirectory/redirector/uint_to_any_base"
)


var intToString *uint_to_any_base.NumeralSystem
var ALLOWED_CHARS, API_KEY string
var RANDOM_SIZE, PROJECT_NUMBER int
var DEFAULT_DURATION uint

const APPLICATION_JSON = "application/json"
const API_ROOT = "/api/"

var SERVER_PORT uint16

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

	server_port, err := strconv.Atoi(os.Getenv("SERVER_PORT"))
	if err != nil {
		log.Fatalf("failure reading SERVER_PORT into an int constant: %v", err.Error())
	}
	SERVER_PORT = uint16(server_port)
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

func main() {
	loadEnv()
	// records.MakeCache(5)
	AuthSubRouter := CreateAuthSubRouter()

	// This GET wildcard is necessary because of httprouter's weird "ambiguous route" behavior
	router := DefineRoutes(AuthSubRouter)

	log.Printf("Server running on port %v", SERVER_PORT)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%v", SERVER_PORT), router))
}
