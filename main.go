package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"logvault/config"
	"logvault/redis"
	"logvault/syslog"
	"logvault/web"
)

var appConfig config.Config
var ctx = context.Background()

func main() {
	// Load configuration
	var err error
	appConfig, err = config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Init Redis
	rdb, err := redis.InitRedis(appConfig)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	log.Println("Successfully connected to Redis")

	// Start Syslog server
	syslogServer := syslog.StartServer(rdb, appConfig)
	log.Printf("Syslog server started. Listening on %s:%d", appConfig.Syslog.Host, appConfig.Syslog.Port)

	// Start Web server
	go web.StartServer(rdb, appConfig)
	log.Printf("Web UI server started. Listening on http://0.0.0.0:%d", appConfig.Web.Port)

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down logvault...")
	syslogServer.Kill()
}
