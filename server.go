package main

import (
	"cloud.google.com/go/compute/metadata"
	"context"
	"encoding/json"
	"fmt"
	"golang.org/x/oauth2/google"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

const K_SERVICE = "K_SERVICE"

func main() {
	// Create the channel and notify it in case of SIGTERM signal
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	// Graceful termination handling
	go gracefulTermination(sigs)

	// Register paths
	http.HandleFunc("/", Helloworld)

	// Determine port for HTTP service.
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		log.Printf("defaulting to port %s", port)
	}

	// Start HTTP server.
	log.Printf("listening on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}

func gracefulTermination(sigs chan os.Signal) {
	// Bloc on the signal and wait for event
	sig := <-sigs

	fmt.Printf("Signal received %s\n", sig)

	// Get the projectNumber and the region of the current instance to call the right service
	projectNumber, region, err := getProjectAndRegion()
	if err != nil {
		fmt.Printf("impossible to get the projectNumber and region from the metadata server with error %+v\n", err)
		return
	}

	// Get the service name from the environment variables
	service := os.Getenv(K_SERVICE)
	if service == "" {
		fmt.Printf("impossible to get the Cloud Run service name from Environment Variable with error %+v\n", err)
		return
	}

	// With the region, the projectNumber and the serviceName, it's possible to recover the Cloud Run service URL
	cloudRunUrl, err := getCloudRunUrl(region, projectNumber, service)
	if err != nil {
		return
	}

	// And then to perform a self call on the current service to start a new instance.
	selfCall(cloudRunUrl)
}

func selfCall(url string) {

	// To be generic, here a service-to-service call is performed
	// An idToken is requested to the metadata server with the correct audience (URL of the service (the base URL))
	tokenURL := fmt.Sprintf("/instance/service-accounts/default/identity?audience=%s", url)
	idToken, err := metadata.Get(tokenURL)
	if err != nil {
		fmt.Errorf("metadata.Get: failed to query id_token: %+v\n", err)
		return
	}

	// Create a GET request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Errorf("error when creating the client %+v\n", err)
		return
	}
	// Add the idToken in the authorization header
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", idToken))

	// Perform a call in loop until having a http 200 response code.
	// Prevent potential network disruptions and issues.
	// If the correct HTTP code is never received, after 10s the instance is killed and the loop vanish
	for resp, err := http.DefaultClient.Do(req); err != nil || resp.StatusCode >= 300; {
		fmt.Println("self call not successful, retry")
	}
	fmt.Println("Self call success. Goodbye")

	// Exit gracefully
	os.Exit(0)
}

func getCloudRunUrl(region string, projectNumber string, service string) (url string, err error) {
	ctx := context.Background()
	// To perform a call the Cloud Run API, the current service, through the service account, needs to be authenticated
	// The Google Auth default client add automatically the authorization header for that.
	client, err := google.DefaultClient(ctx)

	// Build the request to the API
	cloudRunApi := fmt.Sprintf("https://%s-run.googleapis.com/apis/serving.knative.dev/v1/namespaces/%s/services/%s", region, projectNumber, service)
	// Perform the call
	resp, err := client.Get(cloudRunApi)
	if err != nil {
		fmt.Printf("error when calling the Cloud Run API %s with error %+v\n", cloudRunApi, err)
		return
	}
	defer resp.Body.Close()

	// Read the body of the response
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("impossible to read the Cloud Run API call body with error %+v\n", err)
		return
	}

	// Map the JSON body in a minimal struct. We need only the URL, the struct match only this part in the JSON.
	cloudRunResp := &CloudRunAPIUrlOnly{}
	json.Unmarshal(body, cloudRunResp)
	url = cloudRunResp.Status.URL
	return
}

// Minimal type to get only the interesting part in the answer
type CloudRunAPIUrlOnly struct {
	Status struct {
		URL string `json:"url"`
	} `json:"status"`
}

func getProjectAndRegion() (projectNumber string, region string, err error) {
	// Get the region from the metadata server. The project number is returned in the response
	resp, err := metadata.Get("/instance/region")
	if err != nil {
		return
	}
	// response pattern is projects/<projectNumber>/regions/<region>
	r := strings.Split(resp, "/")
	projectNumber = r[1]
	region = r[3]
	return
}

func Helloworld(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "hello world")
}
