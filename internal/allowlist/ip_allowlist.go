package allowlist

import (
	"net"
	"strings"
)

type IPAllowlist struct {
	exactIPs map[string]struct{}
	networks []*net.IPNet
	enabled  bool
}

func New(entries []string) (*IPAllowlist, error) {
	allowlist := &IPAllowlist{
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

func (a *IPAllowlist) Allows(ip net.IP) bool {
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

func (a *IPAllowlist) Enabled() bool {
	return a.enabled
}

func ParseRemoteHost(remoteAddr string) net.IP {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}
	return net.ParseIP(host)
}
