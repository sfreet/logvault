package web

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"logvault/config"
	"logvault/redis"
)

// MimeTypeMiddleware sets the correct Content-Type for static assets.
func MimeTypeMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("STATIC: Request for %s", r.URL.Path)
		if strings.HasSuffix(r.URL.Path, ".css") {
			w.Header().Set("Content-Type", "text/css")
			log.Printf("STATIC: Set Content-Type to text/css for %s", r.URL.Path)
		} else if strings.HasSuffix(r.URL.Path, ".js") {
			w.Header().Set("Content-Type", "application/javascript")
			log.Printf("STATIC: Set Content-Type to application/javascript for %s", r.URL.Path)
		}
		next.ServeHTTP(w, r)
	})
}

// StartServer initializes and starts the web server
func StartServer(rdb *redis.RedisClient, appConfig config.Config) {
	mux := http.NewServeMux()

	// Serve static files from the client directory
	fs := http.FileServer(http.Dir("./client"))
	mux.Handle("/static/", MimeTypeMiddleware(http.StripPrefix("/static/", fs)))

	// Public routes
	mux.HandleFunc("/login.html", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./client/login.html")
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
		if err := http.ListenAndServeTLS(addr, appConfig.Web.CertFile, appConfig.Web.KeyFile, corsMiddleware(mux, []string{appConfig.Web.CORSOrigin}, true, true)); err != nil {
			log.Fatalf("Web server failed: %v", err)
		}
	} else {
		log.Printf("Web UI server starting with HTTP. Listening on http://0.0.0.0:%d", appConfig.Web.Port)
		if err := http.ListenAndServe(addr, corsMiddleware(mux, []string{appConfig.Web.CORSOrigin}, true, true)); err != nil {
			log.Fatalf("Web server failed: %v", err)
		}
	}
}

func corsMiddleware(next http.Handler, allowedOrigins []string, allowNull bool, allowCredentials bool) http.Handler {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, o := range allowedOrigins {
		allowed[o] = struct{}{}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		originAllowed := false
		if origin != "" {
			if _, ok := allowed[origin]; ok {
				originAllowed = true
			} else if allowNull && origin == "null" {
				originAllowed = true
			}
		}

		if originAllowed {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Add("Vary", "Origin")

			if allowCredentials {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")

		// 브라우저가 요청한 헤더를 그대로 허용 (프리플라이트 호환성 ↑)
		if reqHdr := r.Header.Get("Access-Control-Request-Headers"); reqHdr != "" {
			w.Header().Set("Access-Control-Allow-Headers", reqHdr)
		} else {
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		}

		// preflight
		if r.Method == http.MethodOptions {
			if !originAllowed && origin != "" {
				http.Error(w, "CORS origin not allowed", http.StatusForbidden)
				return
			}
			w.WriteHeader(http.StatusNoContent) // 204
			return
		}

		next.ServeHTTP(w, r)
	})
}
