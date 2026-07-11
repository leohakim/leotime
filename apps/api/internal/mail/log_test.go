package mail

import (
	"strings"
	"testing"
)

func TestRedactSensitiveMailContent(t *testing.T) {
	body := "Reset here: https://app.example.com/#reset-password?token=super-secret-token&foo=bar"
	redacted := redactSensitiveMailContent(body)
	if redacted == body {
		t.Fatal("expected body to be redacted")
	}
	if strings.Contains(redacted, "super-secret-token") {
		t.Fatalf("expected token to be redacted, got %q", redacted)
	}
	if !strings.Contains(redacted, "token=<redacted>") {
		t.Fatalf("expected token placeholder, got %q", redacted)
	}
}
