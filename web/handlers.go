package web

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"logvault/config"
	"logvault/notifier"
	"logvault/redis"
)

const alarmPrefix = "alarm:"

func serveHome(rdb *redis.RedisClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		// Get the absolute path for index.html
		absPath, err := filepath.Abs("index.html")
		if err != nil {
			log.Printf("Error getting absolute path for index.html: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Check if the file exists and we can stat it.
		if _, err := os.Stat(absPath); err != nil {
			// Log any error from os.Stat, not just os.IsNotExist
			log.Printf("FATAL: Could not stat file at path %s: %v", absPath, err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		http.ServeFile(w, r, absPath)
	}
}

func alarmsHandler(rdb *redis.RedisClient, appConfig config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			getAlarms(w, r, rdb)
		case http.MethodDelete:
			// If the path is just /api/alarms, delete all.
			// Otherwise, it's /api/alarms/{key}, so delete one.
			if r.URL.Path == "/api/alarms" || r.URL.Path == "/api/alarms/" {
				deleteAllAlarms(w, r, rdb, appConfig)
			} else {
				deleteAlarm(w, r, rdb, appConfig)
			}
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}
}

func deleteAllAlarms(w http.ResponseWriter, r *http.Request, rdb *redis.RedisClient, appConfig config.Config) {
	ctx := context.Background()
	keys, err := rdb.GetKeysByPattern(ctx, alarmPrefix+"*")
	if err != nil {
		http.Error(w, "Failed to get keys from Redis", http.StatusInternalServerError)
		return
	}

	if len(keys) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// The underlying go-redis Del method accepts multiple keys
	if err := rdb.Del(keys...); err != nil {
		log.Printf("Failed to DEL all alarm keys via API: %v", err)
		http.Error(w, "Failed to delete keys from Redis", http.StatusInternalServerError)
		return
	}

	log.Printf("API: Deleted %d alarm keys", len(keys))

	// Call external API if enabled
	if appConfig.ExternalAPI.Enabled {
		go notifier.CallExternalAPI(appConfig, map[string]string{
			"key":     "ALL_ALARMS",
			"message": fmt.Sprintf("Deleted %d alarms via web UI", len(keys)),
			"status":  "CLEAR",
		})
	}

	w.WriteHeader(http.StatusNoContent)
}

func getAlarms(w http.ResponseWriter, r *http.Request, rdb *redis.RedisClient) {
	ctx := context.Background()
	keys, err := rdb.GetKeysByPattern(ctx, alarmPrefix+"*")
	if err != nil {
		http.Error(w, "Failed to get keys from Redis", http.StatusInternalServerError)
		return
	}

	if len(keys) == 0 {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, "{}") // Return empty JSON object
		return
	}

	alarms := make(map[string]interface{})
	for _, key := range keys {
		val, err := rdb.Get(key)
		if err != nil {
			log.Printf("Failed to get value for key %s: %v", key, err)
			continue
		}
		cleanKey := strings.TrimPrefix(key, alarmPrefix)

		var v interface{}
		// Try to unmarshal the value as JSON
		if err := json.Unmarshal([]byte(val), &v); err == nil {
			alarms[cleanKey] = v // It's JSON, store the parsed object
		} else {
			alarms[cleanKey] = val // It's not JSON, store as a plain string
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(alarms)
}

func deleteAlarm(w http.ResponseWriter, r *http.Request, rdb *redis.RedisClient, appConfig config.Config) {
	key := strings.TrimPrefix(r.URL.Path, "/api/alarms/")
	if key == "" {
		http.Error(w, "Key is missing", http.StatusBadRequest)
		return
	}

	fullKey := alarmPrefix + key
	if err := rdb.Del(fullKey); err != nil {
		log.Printf("Failed to DEL key %s via API: %v", fullKey, err)
		http.Error(w, "Failed to delete key from Redis", http.StatusInternalServerError)
		return
	}

	log.Printf("API: Deleted key %s", fullKey)

	// Call external API if enabled
	if appConfig.ExternalAPI.Enabled {
		go notifier.CallExternalAPI(appConfig, map[string]string{
			"key":     fullKey,
			"message": fmt.Sprintf("Alarm cleared for %s via web UI", key),
			"status":  "CLEAR",
		})
	}

	w.WriteHeader(http.StatusNoContent)
}

func getAllRedisDataHandler(rdb *redis.RedisClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := context.Background()
		keys, err := rdb.GetAllKeys(ctx)
		if err != nil {
			http.Error(w, "Failed to get Redis keys: "+err.Error(), http.StatusInternalServerError)
			return
		}

		data := make(map[string]string)
		for _, key := range keys {
			val, err := rdb.Get(key)
			if err != nil {
				// Log the error but continue to get other keys
				log.Printf("Failed to get value for key %s: %v", key, err)
				continue
			}
			data[key] = val
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(data)
	}
}
