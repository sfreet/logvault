package syslog

import (
	"testing"

	"logvault/internal/allowlist"
)

func TestIsAllowedSyslogSenderAllowsAllWhenDisabled(t *testing.T) {
	allowed, err := allowlist.New(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !isAllowedSyslogSender(map[string]interface{}{"client": "203.0.113.10:514"}, allowed) {
		t.Fatal("expected sender to be allowed when allowlist is disabled")
	}
}

func TestIsAllowedSyslogSenderDeniesUnknownIP(t *testing.T) {
	allowed, err := allowlist.New([]string{"10.0.0.0/8"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if isAllowedSyslogSender(map[string]interface{}{"client": "203.0.113.10:514"}, allowed) {
		t.Fatal("expected sender to be denied")
	}
}
