package web

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"logvault/redis"
)

const alarmPrefix = "alarm:"

func serveHome(rdb *redis.RedisClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, "index.html")
	}
}

func alarmsHandler(rdb *redis.RedisClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			getAlarms(w, r, rdb)
		case http.MethodDelete:
			deleteAlarm(w, r, rdb)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}
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

	alarms := make(map[string]string)
	for _, key := range keys {
		val, err := rdb.Get(key)
		if err != nil {
			log.Printf("Failed to get value for key %s: %v", key, err)
			continue
		}
		cleanKey := strings.TrimPrefix(key, alarmPrefix)
		alarms[cleanKey] = val
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(alarms)
}

func deleteAlarm(w http.ResponseWriter, r *http.Request, rdb *redis.RedisClient) {
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
