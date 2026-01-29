package syslog

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"strconv" // Added for string to int conversion
	"strings"
	"time"    // Added for time formatting

	"gopkg.in/mcuadros/go-syslog.v2"

	"logvault/config"
	"logvault/notifier"
	"logvault/redis"
)

const alarmPrefix = "alarm:"

// StartServer initializes and starts the syslog server
func StartServer(rdb *redis.RedisClient, appConfig config.Config) *syslog.Server {
	channel := make(syslog.LogPartsChannel)
	handler := syslog.NewChannelHandler(channel)

	server := syslog.NewServer()
	server.SetFormat(syslog.Automatic)
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

func shouldTriggerNotifier(tag string, triggerTags string) bool {
	if triggerTags == "" {
		return false
	}
	incomingTag := strings.ToUpper(tag)
	tags := strings.Split(triggerTags, ",")
	for _, t := range tags {
		if incomingTag == strings.ToUpper(strings.TrimSpace(t)) {
			return true
		}
	}
	return false
}

func processLogs(rdb *redis.RedisClient, appConfig config.Config, channel syslog.LogPartsChannel) {
	for logParts := range channel {
		log.Printf("DEBUG: Received raw syslog parts: %+v", logParts)
		tag, message := "", "" // Declare once

		tagVal, ok := logParts["tag"]
		if ok && tagVal != nil {
			tag = tagVal.(string)
		}

		if tag == "" { // If RFC3164 tag is empty, try RFC5424 app_name
			appnameVal, ok := logParts["app_name"]
			if ok && appnameVal != nil {
				tag = appnameVal.(string)
			}
		}

		contentVal, ok := logParts["content"]
		if ok && contentVal != nil {
			message = contentVal.(string)
		}

		if message == "" { // If RFC3164 content is empty, try RFC5424 message
			messageVal, ok := logParts["message"]
			if ok && messageVal != nil {
				message = messageVal.(string)
			}
		}

		if message == "" {
			continue
		}

		switch strings.ToUpper(tag) {
		case "INSIGHTS":
			parseThreatMessageAndSave(rdb, message, appConfig, tag)
		default:
			saveWithRandomKey(rdb, message, tag)
		}
	}
}

func parseThreatMessageAndSave(rdb *redis.RedisClient, message string, appConfig config.Config, tag string) {
	// Generate a random key for the new entry
	randomBytes := make([]byte, 16)
	if _, err := rand.Read(randomBytes); err != nil {
		log.Printf("Failed to generate random key for THREAT log: %v", err)
		return
	}
	key := alarmPrefix + hex.EncodeToString(randomBytes)

	// Define the field names in order
	fields := []string{
		"Score", "DetectTime", "DetectType", "DetectSubType", "FileName",
		"RuleName", "IP", "AuthID", "AuthName", "AuthDeptName",
	}

	// Split the message by the backtick delimiter
	message = strings.Trim(message, "`")
	values := strings.Split(message, "`")

	// Create a map to hold the structured data
	jsonData := make(map[string]interface{})
	jsonData["tag"] = tag

	// Populate the map with parsed data
	for i, field := range fields {
		if i < len(values) {
			jsonData[field] = values[i]
		} else {
			jsonData[field] = "" // Assign empty string if value is missing
		}
	}

	// Add any extra fields from the log message
	if len(values) > len(fields) {
		jsonData["extra_data"] = strings.Join(values[len(fields):], "`")
	}

	// Format DetectTime if it's a Unix timestamp
	if dtVal, ok := jsonData["DetectTime"].(string); ok && dtVal != "" {
		if timestamp, err := strconv.ParseInt(dtVal, 10, 64); err == nil {
			// Assuming milliseconds, convert to seconds and nanoseconds
			t := time.Unix(timestamp/1000, (timestamp%1000)*int64(time.Millisecond))
			jsonData["DetectTimeFormatted"] = t.Format("2006-01-02 15:04:05 MST") // Example format
		} else {
			log.Printf("Failed to parse DetectTime '%s': %v", dtVal, err)
		}
	}

	// Marshal the map into a JSON string
	jsonBytes, err := json.Marshal(jsonData)
	if err != nil {
		log.Printf("Failed to marshal THREAT data: %v. Falling back to raw log.", err)
		// Fallback to saving the raw message if JSON marshaling fails
		saveWithRandomKey(rdb, message, tag)
		return
	}

	// Save the JSON string to Redis
	jsonString := string(jsonBytes)
	if err := rdb.Set(key, jsonString, 0); err != nil {
		log.Printf("Failed to SET key %s for THREAT log: %v", key, err)
	} else {
		log.Printf("SAVED: Set key %s for THREAT message (as JSON)", key)
		if appConfig.ExternalAPI.Enabled && shouldTriggerNotifier(tag, appConfig.ExternalAPI.TriggerTags) {
			go notifier.CallExternalAPI(appConfig, map[string]string{"key": key, "message": jsonString, "status": tag})
		}
	}
}

func saveWithRandomKey(rdb *redis.RedisClient, message string, tag string) {
	randomBytes := make([]byte, 16)
	if _, err := rand.Read(randomBytes); err != nil {
		log.Printf("Failed to generate random key: %v", err)
		return
	}
	key := alarmPrefix + hex.EncodeToString(randomBytes)

	data := map[string]string{"tag": tag, "message": message}
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		log.Printf("Failed to marshal data for random key: %v", err)
		return
	}

	if err := rdb.Set(key, string(jsonBytes), 0); err != nil {
		log.Printf("Failed to SET key %s: %v", key, err)
	} else {
		log.Printf("SAVED: Set key %s for message with tag %s", key, tag)
	}
}

func parseAndSaveAsJSON(rdb *redis.RedisClient, message string, appConfig config.Config, tag string) {
	randomBytes := make([]byte, 16)
	if _, err := rand.Read(randomBytes); err != nil {
		log.Printf("Failed to generate random key: %v", err)
		return
	}
	key := alarmPrefix + hex.EncodeToString(randomBytes)

	parts := strings.Split(message, "`")
	jsonData := make(map[string]interface{})
	jsonData["tag"] = tag // Add the tag to the JSON data

	for i, part := range parts {
		var v interface{}
		// Attempt to unmarshal the part as a JSON object
		if err := json.Unmarshal([]byte(part), &v); err == nil {
			jsonData[fmt.Sprintf("field_%d", i)] = v
		} else {
			jsonData[fmt.Sprintf("field_%d", i)] = part
		}
	}

	jsonBytes, err := json.Marshal(jsonData)
	if err != nil {
		log.Printf("Failed to marshal JSON: %v", err)
		// Fallback to saving the raw message if JSON marshaling fails
		data := map[string]string{"tag": tag, "message": message}
		fallbackBytes, _ := json.Marshal(data)
		if err := rdb.Set(key, string(fallbackBytes), 0); err != nil {
			log.Printf("Failed to SET key %s (raw): %v", key, err)
		}
		return
	}

	jsonString := string(jsonBytes)
	if err := rdb.Set(key, jsonString, 0); err != nil {
		log.Printf("Failed to SET key %s (JSON): %v", key, err)
	} else {
		log.Printf("SAVED: Set key %s for message (as JSON)", key)
		if appConfig.ExternalAPI.Enabled && shouldTriggerNotifier(tag, appConfig.ExternalAPI.TriggerTags) {
			go notifier.CallExternalAPI(appConfig, map[string]string{"key": key, "message": jsonString, "status": tag})
		}
	}
}
