package allowlist

import (
	"net"
	"testing"
)

func TestAllowlistAllowsAllWhenDisabled(t *testing.T) {
	allowed, err := New(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !allowed.Allows(net.ParseIP("203.0.113.50")) {
		t.Fatal("expected disabled allowlist to allow all IPs")
	}
}

func TestAllowlistAllowsExactIP(t *testing.T) {
	allowed, err := New([]string{"203.0.113.50"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !allowed.Allows(net.ParseIP("203.0.113.50")) {
		t.Fatal("expected exact IP to be allowed")
	}
}

func TestAllowlistAllowsCIDR(t *testing.T) {
	allowed, err := New([]string{"10.10.0.0/16"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !allowed.Allows(net.ParseIP("10.10.20.15")) {
		t.Fatal("expected CIDR to be allowed")
	}
}

func TestAllowlistDeniesUnknownIP(t *testing.T) {
	allowed, err := New([]string{"10.10.0.0/16"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if allowed.Allows(net.ParseIP("203.0.113.50")) {
		t.Fatal("expected unknown IP to be denied")
	}
}
