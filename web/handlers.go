package web

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/go-redis/redis/v8"
)

const alarmPrefix = "alarm:"

func serveHome(rdb *redis.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, "index.html")
	}
}

func alarmsHandler(rdb *redis.Client) http.HandlerFunc {
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

func getAlarms(w http.ResponseWriter, r *http.Request, rdb *redis.Client) {
	ctx := context.Background()
	keys, err := rdb.Keys(ctx, alarmPrefix+"*").Result()
	if err != nil {
		http.Error(w, "Failed to get keys from Redis", http.StatusInternalServerError)
		return
	}

	if len(keys) == 0 {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, "{}") // Return empty JSON object
		return
	}

	values, err := rdb.MGet(ctx, keys...).Result()
	if err != nil {
		http.Error(w, "Failed to get values from Redis", http.StatusInternalServerError)
		return
	}

	alarms := make(map[string]string)
	for i, key := range keys {
		cleanKey := strings.TrimPrefix(key, alarmPrefix)
		if values[i] != nil {
			alarms[cleanKey] = values[i].(string)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(alarms)
}

func deleteAlarm(w http.ResponseWriter, r *http.Request, rdb *redis.Client) {
	ctx := context.Background()
	key := strings.TrimPrefix(r.URL.Path, "/api/alarms/")
	if key == "" {
		http.Error(w, "Key is missing", http.StatusBadRequest)
		return
	}

	fullKey := alarmPrefix + key
	if err := rdb.Del(ctx, fullKey).Err(); err != nil {
		log.Printf("Failed to DEL key %s via API: %v", fullKey, err)
		http.Error(w, "Failed to delete key from Redis", http.StatusInternalServerError)
		return
	}

	log.Printf("API: Deleted key %s", fullKey)
	w.WriteHeader(http.StatusNoContent)
}
