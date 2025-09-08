package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/go-redis/redis/v8"
	"github.com/spf13/viper"
	"gopkg.in/mcuadros/go-syslog.v2"
)

const alarmPrefix = "alarm:"

// Config holds the application configuration
type Config struct {
	Syslog struct {
		Host     string `mapstructure:"host"`
		Port     int    `mapstructure:"port"`
		Protocol string `mapstructure:"protocol"`
	} `mapstructure:"syslog"`
	Redis struct {
		Address  string `mapstructure:"address"`
		Password string `mapstructure:"password"`
		DB       int    `mapstructure:"db"`
	} `mapstructure:"redis"`
	Web struct {
		Port int `mapstructure:"port"`
	} `mapstructure:"web"`
}

var appConfig Config
var rdb *redis.Client
var ctx = context.Background()

func main() {
	// Load configuration
	if err := loadConfig(); err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Init Redis
	if err := initRedis(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	log.Println("Successfully connected to Redis")

	// Start Syslog server
	syslogServer := startSyslogServer()
	log.Printf("Syslog server started. Listening on %s:%d", appConfig.Syslog.Host, appConfig.Syslog.Port)

	// Start Web server
	go startWebServer()
	log.Printf("Web UI server started. Listening on http://0.0.0.0:%d", appConfig.Web.Port)

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down logvault...")
	syslogServer.Kill()
}

func initRedis() error {
	rdb = redis.NewClient(&redis.Options{
		Addr:     appConfig.Redis.Address,
		Password: appConfig.Redis.Password,
		DB:       appConfig.Redis.DB,
	})
	_, err := rdb.Ping(ctx).Result()
	return err
}

func startSyslogServer() *syslog.Server {
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

	go processLogs(channel)
	return server
}

func startWebServer() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", serveHome)
	mux.HandleFunc("/api/alarms", alarmsHandler)
	mux.HandleFunc("/api/alarms/", alarmsHandler) // For DELETE requests with key

	addr := fmt.Sprintf(":%d", appConfig.Web.Port)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Web server failed: %v", err)
	}
}

func processLogs(channel syslog.LogPartsChannel) {
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
			if err := rdb.Set(ctx, key, message, 0).Err(); err != nil {
				log.Printf("Failed to SET key %s: %v", key, err)
			} else {
				log.Printf("ALARM: Set key %s", key)
			}
		case "CLEAR":
			if err := rdb.Del(ctx, key).Err(); err != nil {
				log.Printf("Failed to DEL key %s: %v", key, err)
			} else {
				log.Printf("CLEAR: Deleted key %s", key)
			}
		}
	}
}

func serveHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, "index.html")
}

func alarmsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		getAlarms(w, r)
	case http.MethodDelete:
		deleteAlarm(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func getAlarms(w http.ResponseWriter, r *http.Request) {
	keys, err := rdb.Keys(ctx, alarmPrefix+"*").Result()
	if err != nil {
		http.Error(w, "Failed to get keys from Redis", http.StatusInternalServerError)
		return
	}

	if len(keys) == 0 {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, "[]") // Return empty JSON array
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

func deleteAlarm(w http.ResponseWriter, r *http.Request) {
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

// loadConfig reads configuration from config.yaml
func loadConfig() error {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// Set default values
	viper.SetDefault("web.port", 8080)
	viper.SetDefault("syslog.port", 514)
	viper.SetDefault("syslog.host", "0.0.0.0")
	viper.SetDefault("redis.address", "localhost:6379")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return err // Only return error if it's not a file not found error
		}
	}

	return viper.Unmarshal(&appConfig)
}
