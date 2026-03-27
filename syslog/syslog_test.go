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

func TestValidateThreatMessageAcceptsExpectedPayload(t *testing.T) {
	values := []string{
		"5", "1742184000000", "MALWARE", "DOC", "sample.exe",
		"rule-1", "192.0.2.10", "user01", "Kim", "SOC",
	}

	if err := validateThreatMessage(values, 10); err != nil {
		t.Fatalf("expected payload to be valid, got error: %v", err)
	}
}

func TestValidateThreatMessageRejectsUnexpectedFieldCount(t *testing.T) {
	values := []string{"5", "1742184000000", "MALWARE"}

	if err := validateThreatMessage(values, 10); err == nil {
		t.Fatal("expected payload with wrong field count to be rejected")
	}
}

func TestValidateThreatMessageRejectsAllNullValues(t *testing.T) {
	values := []string{
		"NULL", "NULL", "NULL", "NULL", "NULL",
		"NULL", "NULL", "NULL", "NULL", "NULL",
	}

	if err := validateThreatMessage(values, 10); err == nil {
		t.Fatal("expected all-NULL payload to be rejected")
	}
}

func TestValidateThreatMessageRejectsInvalidDetectTime(t *testing.T) {
	values := []string{
		"5", "NULL", "MALWARE", "DOC", "sample.exe",
		"rule-1", "192.0.2.10", "user01", "Kim", "SOC",
	}

	if err := validateThreatMessage(values, 10); err == nil {
		t.Fatal("expected payload with invalid DetectTime to be rejected")
	}
}
