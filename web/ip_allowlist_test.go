package web

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"logvault/config"
)

func TestIPAllowlistAllowsAllWhenDisabled(t *testing.T) {
	cfg := config.Config{}
	handler := ipAllowlistMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}), cfg)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "203.0.113.50:12345"
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected disabled allowlist to pass through, got %d", rr.Code)
	}
}

func TestIPAllowlistAllowsExactIP(t *testing.T) {
	cfg := config.Config{}
	cfg.Web.AllowedIPs = []string{"203.0.113.50"}

	handler := ipAllowlistMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}), cfg)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "203.0.113.50:12345"
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected exact IP to be allowed, got %d", rr.Code)
	}
}

func TestIPAllowlistAllowsCIDR(t *testing.T) {
	cfg := config.Config{}
	cfg.Web.AllowedIPs = []string{"10.10.0.0/16"}

	handler := ipAllowlistMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}), cfg)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.10.20.15:12345"
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected CIDR to be allowed, got %d", rr.Code)
	}
}

func TestIPAllowlistDeniesUnknownIP(t *testing.T) {
	cfg := config.Config{}
	cfg.Web.AllowedIPs = []string{"10.10.0.0/16"}

	handler := ipAllowlistMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}), cfg)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "203.0.113.50:12345"
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected unknown IP to be forbidden, got %d", rr.Code)
	}
}
