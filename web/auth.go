package web

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"logvault/config"
)

const sessionCookieName = "logvault_session"
const sessionExpiry = 24 * time.Hour // Session valid for 24 hours

// In-memory session store (for simplicity)
// In a production environment, use a persistent store like Redis or a database
var sessionTokens = make(map[string]time.Time)
var sessionMutex sync.Mutex

// LoginHandler handles authentication requests
func LoginHandler(w http.ResponseWriter, r *http.Request, appConfig config.Config) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Secret string `json:"secret"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if appConfig.Web.Secret == "" {
		log.Println("Warning: Web secret is not set in config.yaml. Login will always fail.")
		http.Error(w, "Server secret not configured", http.StatusInternalServerError)
		return
	}

	if req.Secret == appConfig.Web.Secret {
		token, err := generateSessionToken()
		if err != nil {
			log.Printf("Failed to generate session token: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		sessionMutex.Lock()
		sessionTokens[token] = time.Now().Add(sessionExpiry)
		sessionMutex.Unlock()

		http.SetCookie(w, &http.Cookie{
			Name:     sessionCookieName,
			Value:    token,
			Path:     "/",
			Expires:  time.Now().Add(sessionExpiry),
			HttpOnly: true,  // Important for security
			Secure:   false, // Set to true in production with HTTPS
			SameSite: http.SameSiteLaxMode,
		})
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "Login successful"})
	} else {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"message": "Invalid secret"})
	}
}

// AuthMiddleware checks for a valid session cookie
func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Skip authentication for API routes handled by APIKeyAuthMiddleware
		if strings.HasPrefix(r.URL.Path, "/api/") {
			next.ServeHTTP(w, r)
			return
		}

		cookie, err := r.Cookie(sessionCookieName)
		if err != nil {
			if err == http.ErrNoCookie {
				http.Redirect(w, r, "/login.html", http.StatusFound)
				return
			}
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		sessionMutex.Lock()
		expiry, ok := sessionTokens[cookie.Value]
		sessionMutex.Unlock()

		if !ok || time.Now().After(expiry) {
			// Invalid or expired token, clear cookie and redirect
			http.SetCookie(w, &http.Cookie{
				Name:     sessionCookieName,
				Value:    "",
				Path:     "/",
				MaxAge:   -1, // Delete cookie
				HttpOnly: true,
				Secure:   false, // Set to true in production with HTTPS
				SameSite: http.SameSiteLaxMode,
			})
			http.Redirect(w, r, "/login.html", http.StatusFound)
			return
		}

		// Token is valid, renew expiry (optional, but good for user experience)
		sessionMutex.Lock()
		sessionTokens[cookie.Value] = time.Now().Add(sessionExpiry)
		sessionMutex.Unlock()

		next.ServeHTTP(w, r)
	}
}

// LogoutHandler invalidates the session and redirects to the login page
func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(sessionCookieName)
	if err == nil { // If cookie exists, invalidate the session
		sessionMutex.Lock()
		delete(sessionTokens, cookie.Value)
		sessionMutex.Unlock()
	}

	// Clear the session cookie from the browser
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1, // Delete cookie
		HttpOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		SameSite: http.SameSiteLaxMode,
	})

	http.Redirect(w, r, "/login.html", http.StatusFound)
}

// generateSessionToken generates a random, URL-safe string for session token
func generateSessionToken() (string, error) {
	b := make([]byte, 32) // 32 bytes for a 256-bit token
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// BearerAuthMiddleware checks for a valid Bearer token in the Authorization header
func BearerAuthMiddleware(next http.HandlerFunc, appConfig config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Unauthorized: Missing Authorization header", http.StatusUnauthorized)
			return
		}

		// Expecting "Bearer <token>"
		parts := strings.SplitN(authHeader, " ", 2)
		if !(len(parts) == 2 && parts[0] == "Bearer") {
			http.Error(w, "Unauthorized: Invalid Authorization header format", http.StatusUnauthorized)
			return
		}

		token := parts[1]

		if appConfig.API.BearerToken == "" {
			log.Println("Warning: Bearer token is not set in config.yaml. Bearer token authentication will always fail.")
			http.Error(w, "Bearer token not configured on server", http.StatusInternalServerError)
			return
		}

		if token != appConfig.API.BearerToken {
			http.Error(w, "Unauthorized: Invalid Bearer token", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	}
}
