package syslog

import (
	"fmt"
	"log"
	"strings"

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

	go processLogs(rdb, channel)
	return server
}

func processLogs(rdb *redis.RedisClient, channel syslog.LogPartsChannel) {
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
			}
		case "CLEAR":
			if err := rdb.Del(key); err != nil {
				log.Printf("Failed to DEL key %s: %v", key, err)
			} else {
				log.Printf("CLEAR: Deleted key %s", key)
			}
		}
	}
}
