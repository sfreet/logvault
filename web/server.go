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
	mux.HandleFunc("/api/data", APIAuthMiddleware(getAllRedisDataHandler(rdb), appConfig))

	// Protected web UI routes
	mux.HandleFunc("/logout", LogoutHandler) // Add logout handler
	mux.HandleFunc("/", AuthMiddleware(serveHome(rdb)))
	mux.HandleFunc("/api/alarms", APIAuthMiddleware(alarmsHandler(rdb, appConfig), appConfig))
	mux.HandleFunc("/api/alarms/", APIAuthMiddleware(alarmsHandler(rdb, appConfig), appConfig)) // For DELETE requests with key

	addr := fmt.Sprintf(":%d", appConfig.Web.Port)
	if appConfig.Web.CertFile != "" && appConfig.Web.KeyFile != "" {
		log.Printf("Web UI server starting with HTTPS. Listening on https://0.0.0.0:%d", appConfig.Web.Port)
		if err := http.ListenAndServeTLS(addr, appConfig.Web.CertFile, appConfig.Web.KeyFile, corsMiddleware(mux)); err != nil {
			log.Fatalf("Web server failed: %v", err)
		}
	} else {
		log.Printf("Web UI server starting with HTTP. Listening on http://0.0.0.0:%d", appConfig.Web.Port)
		if err := http.ListenAndServe(addr, corsMiddleware(mux)); err != nil {
			log.Fatalf("Web server failed: %v", err)
		}
	}
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Call the next handler
		next.ServeHTTP(w, r)
	})
}
