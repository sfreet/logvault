package syslog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"gopkg.in/mcuadros/go-syslog.v2"

	"logvault/config"
	"logvault/redis"
)

const alarmPrefix = "alarm:"

// StartServer initializes and starts the syslog server
func StartServer(rdb *redis.RedisClient, appConfig config.Config) *syslog.Server {
	channel := make(syslog.LogPartsChannel)
	handler := syslog.NewChannelHandler(channel)

	server := syslog.NewServer()
	server.SetFormat(syslog.RFC5424)
	server.SetHandler(handler)

	listenAddr := fmt.Sprintf("%s:%d", appConfig.Syslog.Host, appConfig.Syslog.Port)
	if err := server.ListenUDP(listenAddr); err != nil {
		log.Fatalf("Failed to start syslog server: %v", err)
	}

	if err := server.Boot(); err != nil {
		log.Fatalf("Failed to boot syslog server: %v", err)
	}

	go processLogs(rdb, appConfig, channel)
	return server
}

func processLogs(rdb *redis.RedisClient, appConfig config.Config, channel syslog.LogPartsChannel) {
	for logParts := range channel {
		tag, _ := logParts["app_name"].(string)
		message, _ := logParts["message"].(string)
		if tag == "" || message == "" {
			continue
		}

		parts := strings.SplitN(message, " ", 2)
		if len(parts) < 1 {
			continue
		}
		key := alarmPrefix + parts[0]

		switch strings.ToUpper(tag) {
		case "ALARM":
			if err := rdb.Set(key, message, 0); err != nil {
				log.Printf("Failed to SET key %s: %v", key, err)
			} else {
				log.Printf("ALARM: Set key %s", key)
				// Call external API if enabled and tag matches
				if appConfig.ExternalAPI.Enabled && strings.ToUpper(tag) == strings.ToUpper(appConfig.ExternalAPI.TriggerTag) {
					go callExternalAPI(appConfig, map[string]string{"key": key, "message": message, "status": "ALARM"})
				}
			}
		case "CLEAR":
			if err := rdb.Del(key); err != nil {
				log.Printf("Failed to DEL key %s: %v", key, err)
			} else {
				log.Printf("CLEAR: Deleted key %s", key)
				// Call external API if enabled and tag matches
				if appConfig.ExternalAPI.Enabled && strings.ToUpper(tag) == strings.ToUpper(appConfig.ExternalAPI.TriggerTag) {
					go callExternalAPI(appConfig, map[string]string{"key": key, "message": message, "status": "CLEAR"})
				}
			}
		}
	}
}

// callExternalAPI makes an HTTP request to the configured external API
func callExternalAPI(appConfig config.Config, payload interface{}) {
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

	client := &http.Client{Timeout: 10 * time.Second} // Set a timeout for the request
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
