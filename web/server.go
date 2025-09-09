package web

import (
	"fmt"
	"log"
	"net/http"

	"logvault/config"
	"logvault/redis"
)

// StartServer initializes and starts the web server
func StartServer(rdb *redis.RedisClient, appConfig config.Config) {
	mux := http.NewServeMux()

	// Public routes
	mux.HandleFunc("/login.html", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "login.html")
	})
	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		LoginHandler(w, r, appConfig)
	})

	// API routes
	mux.HandleFunc("/api/data", BearerAuthMiddleware(getAllRedisDataHandler(rdb), appConfig))

	// Protected web UI routes
	mux.HandleFunc("/logout", LogoutHandler) // Add logout handler
	mux.HandleFunc("/", AuthMiddleware(serveHome(rdb)))
	mux.HandleFunc("/api/alarms", AuthMiddleware(alarmsHandler(rdb)))
	mux.HandleFunc("/api/alarms/", AuthMiddleware(alarmsHandler(rdb))) // For DELETE requests with key

	addr := fmt.Sprintf(":%d", appConfig.Web.Port)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Web server failed: %v", err)
	}
}
