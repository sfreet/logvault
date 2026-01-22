package notifier

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"logvault/config"
)

// CallExternalAPI makes an HTTP request to the configured external API
func CallExternalAPI(appConfig config.Config, payload interface{}) {
	if !appConfig.ExternalAPI.Enabled {
		return // External API integration is disabled
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshalling external API payload: %v", err)
		return
	}

	req, err := http.NewRequest(appConfig.ExternalAPI.Method, appConfig.ExternalAPI.URL, bytes.NewBuffer(jsonPayload))
	if err != nil {
		log.Printf("Error creating external API request: %v", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	if appConfig.ExternalAPI.BearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+appConfig.ExternalAPI.BearerToken)
	}

	// Create a custom transport to skip TLS verification
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client := &http.Client{
		Timeout:   10 * time.Second, // Set a timeout for the request
		Transport: tr,
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error calling external API: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		log.Printf("External API call failed with status: %s", resp.Status)
	} else {
		log.Printf("Successfully called external API, status: %s", resp.Status)
	}
}
