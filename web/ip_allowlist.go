package web

import (
	"log"
	"net"
	"net/http"
	"strings"

	"logvault/config"
)

type ipAllowlist struct {
	exactIPs map[string]struct{}
	networks []*net.IPNet
	enabled  bool
}

func newIPAllowlist(entries []string) (*ipAllowlist, error) {
	allowlist := &ipAllowlist{
		exactIPs: make(map[string]struct{}),
		networks: make([]*net.IPNet, 0),
	}

	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}

		if strings.Contains(entry, "/") {
			_, network, err := net.ParseCIDR(entry)
			if err != nil {
				return nil, err
			}
			allowlist.networks = append(allowlist.networks, network)
			allowlist.enabled = true
			continue
		}

		ip := net.ParseIP(entry)
		if ip == nil {
			return nil, &net.ParseError{Type: "IP address", Text: entry}
		}
		allowlist.exactIPs[ip.String()] = struct{}{}
		allowlist.enabled = true
	}

	return allowlist, nil
}

func (a *ipAllowlist) allows(ip net.IP) bool {
	if !a.enabled {
		return true
	}
	if ip == nil {
		return false
	}

	if _, ok := a.exactIPs[ip.String()]; ok {
		return true
	}

	for _, network := range a.networks {
		if network.Contains(ip) {
			return true
		}
	}

	return false
}

func ipAllowlistMiddleware(next http.Handler, appConfig config.Config) http.Handler {
	allowlist, err := newIPAllowlist(appConfig.Web.AllowedIPs)
	if err != nil {
		log.Fatalf("Invalid web.allowed_ips configuration: %v", err)
	}

	if !allowlist.enabled {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			host = r.RemoteAddr
		}

		clientIP := net.ParseIP(host)
		if allowlist.allows(clientIP) {
			next.ServeHTTP(w, r)
			return
		}

		log.Printf("Denied web request from %q to %s: source IP is not in web.allowed_ips", host, r.URL.Path)
		http.Error(w, "Forbidden", http.StatusForbidden)
	})
}
