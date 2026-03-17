package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"golang.org/x/crypto/bcrypt"
	"logvault/config"
)

func TestLoginHandlerRequiresUsernameAndSecret(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("s3cr3t"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("failed to generate hash: %v", err)
	}

	cfg := config.Config{}
	cfg.Web.Users = []struct {
		Username   string `mapstructure:"username"`
		Secret     string `mapstructure:"secret"`
		SecretHash string `mapstructure:"secret_hash"`
		Role       string `mapstructure:"role"`
	}{
		{Username: "admin", SecretHash: string(hash), Role: roleAdmin},
	}

	body, _ := json.Marshal(map[string]string{
		"username": "admin",
		"secret":   "s3cr3t",
	})

	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	LoginHandler(rr, req, cfg)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected login success, got %d", rr.Code)
	}
}

func TestLoginHandlerRejectsInvalidUsername(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("s3cr3t"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("failed to generate hash: %v", err)
	}

	cfg := config.Config{}
	cfg.Web.Users = []struct {
		Username   string `mapstructure:"username"`
		Secret     string `mapstructure:"secret"`
		SecretHash string `mapstructure:"secret_hash"`
		Role       string `mapstructure:"role"`
	}{
		{Username: "admin", SecretHash: string(hash), Role: roleAdmin},
	}

	body, _ := json.Marshal(map[string]string{
		"username": "other",
		"secret":   "s3cr3t",
	})

	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	LoginHandler(rr, req, cfg)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized for invalid username, got %d", rr.Code)
	}
}

func TestLoginHandlerStoresReadOnlyRole(t *testing.T) {
	sessionTokens = make(map[string]sessionData)
	hash, err := bcrypt.GenerateFromPassword([]byte("s3cr3t"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("failed to generate hash: %v", err)
	}

	cfg := config.Config{}
	cfg.Web.Users = []struct {
		Username   string `mapstructure:"username"`
		Secret     string `mapstructure:"secret"`
		SecretHash string `mapstructure:"secret_hash"`
		Role       string `mapstructure:"role"`
	}{
		{Username: "viewer", SecretHash: string(hash), Role: roleReadOnly},
	}

	body, _ := json.Marshal(map[string]string{
		"username": "viewer",
		"secret":   "s3cr3t",
	})

	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	LoginHandler(rr, req, cfg)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected login success, got %d", rr.Code)
	}

	var found bool
	for _, session := range sessionTokens {
		if session.Username == "viewer" {
			found = true
			if session.Role != roleReadOnly {
				t.Fatalf("expected readonly role, got %q", session.Role)
			}
		}
	}

	if !found {
		t.Fatal("expected session to be stored")
	}
}

func TestLoginHandlerSupportsLegacySingleUserCredentials(t *testing.T) {
	cfg := config.Config{}
	cfg.Web.Username = "admin"
	cfg.Web.Secret = "s3cr3t"

	body, _ := json.Marshal(map[string]string{
		"username": "admin",
		"secret":   "s3cr3t",
	})

	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	LoginHandler(rr, req, cfg)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected legacy single-user login success, got %d", rr.Code)
	}
}

func TestLoginHandlerSupportsLegacySingleUserHash(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("s3cr3t"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("failed to generate hash: %v", err)
	}

	cfg := config.Config{}
	cfg.Web.Username = "admin"
	cfg.Web.SecretHash = string(hash)

	body, _ := json.Marshal(map[string]string{
		"username": "admin",
		"secret":   "s3cr3t",
	})

	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	LoginHandler(rr, req, cfg)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected legacy hashed login success, got %d", rr.Code)
	}
}
