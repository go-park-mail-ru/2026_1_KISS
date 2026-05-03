package mail

import (
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	s := New("from@example.com", "https://example.com", "localhost", "25")
	if s.from != "from@example.com" {
		t.Errorf("want from@example.com, got %s", s.from)
	}
	if s.appURL != "https://example.com" {
		t.Errorf("want https://example.com, got %s", s.appURL)
	}
	if s.smtpHost != "localhost" {
		t.Errorf("want localhost, got %s", s.smtpHost)
	}
	if s.smtpPort != "25" {
		t.Errorf("want 25, got %s", s.smtpPort)
	}
}

func TestSendVerification_FailsWithNoSMTP(t *testing.T) {
	s := New("from@example.com", "https://example.com", "127.0.0.1", "19999")
	err := s.SendVerification("to@example.com", "test-token")
	if err == nil {
		t.Error("expected error when SMTP is unavailable")
	}
}

func TestSendMultipart_BuildsMessage(t *testing.T) {
	s := New("from@example.com", "https://example.com", "127.0.0.1", "19999")
	err := s.sendMultipart("to@example.com", "Test Subject", "text body", "<b>html body</b>")
	if err == nil {
		t.Error("expected error when SMTP is unavailable")
	}
	// error should be connection-related, not message-building related
	errStr := err.Error()
	if strings.Contains(errStr, "invalid") {
		t.Errorf("unexpected error type: %s", errStr)
	}
}
