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

	"golang.org/x/crypto/bcrypt"
	"logvault/config"
)

const sessionCookieName = "logvault_session"
const sessionExpiry = 24 * time.Hour // Session valid for 24 hours
const roleAdmin = "admin"
const roleReadOnly = "readonly"

// In-memory session store (for simplicity)
// In a production environment, use a persistent store like Redis or a database
type sessionData struct {
	Username string
	Role     string
	Expires  time.Time
}

var sessionTokens = make(map[string]sessionData)
var sessionMutex sync.Mutex

func normalizeRole(role string) string {
	if strings.EqualFold(role, roleReadOnly) {
		return roleReadOnly
	}
	return roleAdmin
}

func getWebCredentialRole(appConfig config.Config, username, secret string) (string, bool) {
	for _, user := range appConfig.Web.Users {
		if user.Username == username && secretMatches(secret, user.SecretHash, user.Secret) {
			return normalizeRole(user.Role), true
		}
	}

	if appConfig.Web.Username != "" &&
		username == appConfig.Web.Username &&
		secretMatches(secret, appConfig.Web.SecretHash, appConfig.Web.Secret) {
		return roleAdmin, true
	}

	return "", false
}

func hasConfiguredWebCredentials(appConfig config.Config) bool {
	if len(appConfig.Web.Users) > 0 {
		for _, user := range appConfig.Web.Users {
			if user.Username != "" && (user.SecretHash != "" || user.Secret != "") {
				return true
			}
		}
	}

	return appConfig.Web.Username != "" && (appConfig.Web.SecretHash != "" || appConfig.Web.Secret != "")
}

func secretMatches(plainSecret, secretHash, legacySecret string) bool {
	if secretHash != "" {
		return bcrypt.CompareHashAndPassword([]byte(secretHash), []byte(plainSecret)) == nil
	}
	return legacySecret != "" && plainSecret == legacySecret
}

// LoginHandler handles authentication requests
func LoginHandler(w http.ResponseWriter, r *http.Request, appConfig config.Config) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Username string `json:"username"`
		Secret   string `json:"secret"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if !hasConfiguredWebCredentials(appConfig) {
		log.Println("Warning: Web credentials are not set in config.yaml. Login will always fail.")
		http.Error(w, "Server credentials not configured", http.StatusInternalServerError)
		return
	}

	role, ok := getWebCredentialRole(appConfig, req.Username, req.Secret)
	if ok {
		token, err := generateSessionToken()
		if err != nil {
			log.Printf("Failed to generate session token: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		sessionMutex.Lock()
		sessionTokens[token] = sessionData{
			Username: req.Username,
			Role:     role,
			Expires:  time.Now().Add(sessionExpiry),
		}
		sessionMutex.Unlock()

		http.SetCookie(w, &http.Cookie{
			Name:     sessionCookieName,
			Value:    token,
			Path:     "/",
			Expires:  time.Now().Add(sessionExpiry),
			HttpOnly: true, // Important for security
			Secure:   true,
			SameSite: http.SameSiteNoneMode,
		})
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "Login successful", "role": role})
	} else {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"message": "Invalid username or secret"})
	}
}

// AuthMiddleware checks for a valid session cookie
func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
		session, ok := sessionTokens[cookie.Value]
		sessionMutex.Unlock()

		if !ok || time.Now().After(session.Expires) {
			// Invalid or expired token, clear cookie and redirect
			http.SetCookie(w, &http.Cookie{
				Name:     sessionCookieName,
				Value:    "",
				Path:     "/",
				MaxAge:   -1, // Delete cookie
				HttpOnly: true,
				Secure:   true,
				SameSite: http.SameSiteNoneMode,
			})
			http.Redirect(w, r, "/login.html", http.StatusFound)
			return
		}

		// Token is valid, renew expiry (optional, but good for user experience)
		sessionMutex.Lock()
		session.Expires = time.Now().Add(sessionExpiry)
		sessionTokens[cookie.Value] = session
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
		Secure:   true,
		SameSite: http.SameSiteNoneMode,
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

// APIAuthMiddleware checks for a valid Bearer token or a valid session cookie.
// It is intended for use on API endpoints that can be accessed by both API clients and the web UI.
func APIAuthMiddleware(next http.HandlerFunc, appConfig config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// First, check for a Bearer token
		authHeader := r.Header.Get("Authorization")
		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && parts[0] == "Bearer" {
				token := parts[1]
				if appConfig.API.BearerToken != "" && token == appConfig.API.BearerToken {
					// Valid Bearer token found, proceed
					next.ServeHTTP(w, r)
					return
				}
				// Invalid Bearer token
				http.Error(w, "Unauthorized: Invalid Bearer token", http.StatusUnauthorized)
				return
			}
		}

		// If no Bearer token, check for a session cookie
		cookie, err := r.Cookie(sessionCookieName)
		if err == nil {
			sessionMutex.Lock()
			session, ok := sessionTokens[cookie.Value]
			sessionMutex.Unlock()

			if ok && time.Now().Before(session.Expires) {
				// Valid session cookie found, proceed
				next.ServeHTTP(w, r)
				return
			}
		}

		// No valid authentication method found
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	}
}

func currentSession(r *http.Request) (sessionData, bool) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return sessionData{}, false
	}

	sessionMutex.Lock()
	session, ok := sessionTokens[cookie.Value]
	sessionMutex.Unlock()

	if !ok || time.Now().After(session.Expires) {
		return sessionData{}, false
	}

	return session, true
}

func sessionInfoHandler(w http.ResponseWriter, r *http.Request) {
	session, ok := currentSession(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"username": session.Username,
		"role":     session.Role,
	})
}
