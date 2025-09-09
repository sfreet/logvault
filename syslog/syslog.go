package syslog

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/go-redis/redis/v8"
	"gopkg.in/mcuadros/go-syslog.v2"

	"logvault/config"
)

const alarmPrefix = "alarm:"

// StartServer initializes and starts the syslog server
func StartServer(rdb *redis.Client, appConfig config.Config) *syslog.Server {
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

	go processLogs(rdb, channel)
	return server
}

func processLogs(rdb *redis.Client, channel syslog.LogPartsChannel) {
	ctx := context.Background()
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
