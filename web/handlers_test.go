package web

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"logvault/config"
)

func TestCanDeleteAlarmsAllowsAdminSession(t *testing.T) {
	sessionTokens = map[string]sessionData{
		"token": {
			Username: "admin",
			Role:     roleAdmin,
			Expires:  sessionExpiryLater(),
		},
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/alarms/test", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "token"})

	if !canDeleteAlarms(req, config.Config{}) {
		t.Fatal("expected admin session to be allowed to delete alarms")
	}
}

func TestCanDeleteAlarmsDeniesReadOnlySession(t *testing.T) {
	sessionTokens = map[string]sessionData{
		"token": {
			Username: "viewer",
			Role:     roleReadOnly,
			Expires:  sessionExpiryLater(),
		},
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/alarms/test", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "token"})

	if canDeleteAlarms(req, config.Config{}) {
		t.Fatal("expected readonly session to be denied delete access")
	}
}

func TestCanDeleteAlarmsAllowsBearerToken(t *testing.T) {
	cfg := config.Config{}
	cfg.API.BearerToken = "api-token"

	req := httptest.NewRequest(http.MethodDelete, "/api/alarms/test", nil)
	req.Header.Set("Authorization", "Bearer api-token")

	if !canDeleteAlarms(req, cfg) {
		t.Fatal("expected valid bearer token to be allowed")
	}
}

func sessionExpiryLater() (later time.Time) {
	return time.Now().Add(sessionExpiry)
}
