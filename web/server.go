package web

import (
	"fmt"
	"log"
	"net/http"

	"github.com/go-redis/redis/v8"

	"logvault/config"
)

// StartServer initializes and starts the web server
func StartServer(rdb *redis.Client, appConfig config.Config) {
	mux := http.NewServeMux()

	// Public routes
	mux.HandleFunc("/login.html", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "login.html")
	})
	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		LoginHandler(w, r, appConfig)
	})

	// Protected routes
	mux.HandleFunc("/logout", LogoutHandler) // Add logout handler
	mux.HandleFunc("/", AuthMiddleware(serveHome(rdb)))
	mux.HandleFunc("/api/alarms", AuthMiddleware(alarmsHandler(rdb)))
	mux.HandleFunc("/api/alarms/", AuthMiddleware(alarmsHandler(rdb))) // For DELETE requests with key

	addr := fmt.Sprintf(":%d", appConfig.Web.Port)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Web server failed: %v", err)
	}
}
