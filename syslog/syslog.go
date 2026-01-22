package syslog

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"strings"

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

		if tag == "" {
			saveWithRandomKey(rdb, message, "NONE")
			continue
		}

		switch strings.ToUpper(tag) {
		case "ALARM", "CLEAR":
			parts := strings.SplitN(message, " ", 2)
			if len(parts) < 1 {
				continue
			}
			key := alarmPrefix + parts[0]

			if strings.ToUpper(tag) == "ALARM" {
				data := map[string]string{"tag": tag, "message": message}
				jsonBytes, err := json.Marshal(data)
				if err != nil {
					log.Printf("Failed to marshal ALARM data: %v", err)
					continue
				}
				jsonString := string(jsonBytes)

				if err := rdb.Set(key, jsonString, 0); err != nil {
					log.Printf("Failed to SET key %s: %v", key, err)
				} else {
					log.Printf("ALARM: Set key %s", key)
					if appConfig.ExternalAPI.Enabled && shouldTriggerNotifier(tag, appConfig.ExternalAPI.TriggerTags) {
						go notifier.CallExternalAPI(appConfig, map[string]string{"key": key, "message": jsonString, "status": "ALARM"})
					}
				}
			} else { // CLEAR
				if err := rdb.Del(key); err != nil {
					log.Printf("Failed to DEL key %s: %v", key, err)
				} else {
					log.Printf("CLEAR: Deleted key %s", key)
					if appConfig.ExternalAPI.Enabled {
						go notifier.CallExternalAPI(appConfig, map[string]string{"key": key, "message": message, "status": "CLEAR"})
					}
				}
			}
		case "INSIGHTS":
			parseAndSaveAsJSON(rdb, message, appConfig, tag)
		default:
			saveWithRandomKey(rdb, message, tag)
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
