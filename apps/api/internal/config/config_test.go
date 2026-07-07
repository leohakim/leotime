package config

import (
	"testing"
	"time"
)

func TestFromLookupUsesDefaults(t *testing.T) {
	cfg := FromLookup(func(string) (string, bool) {
		return "", false
	})

	if cfg.HTTPAddr != ":8080" {
		t.Fatalf("expected default http addr, got %q", cfg.HTTPAddr)
	}
	if cfg.SessionTTL != 168*time.Hour {
		t.Fatalf("expected default session ttl, got %s", cfg.SessionTTL)
	}
	if cfg.CookieSecure {
		t.Fatal("expected insecure cookie default for local development")
	}
}

func TestFromLookupOverridesValues(t *testing.T) {
	env := map[string]string{
		"LEOTIME_HTTP_ADDR":     ":9090",
		"LEOTIME_SESSION_TTL":   "24h",
		"LEOTIME_COOKIE_SECURE": "true",
	}

	cfg := FromLookup(func(key string) (string, bool) {
		value, ok := env[key]
		return value, ok
	})

	if cfg.HTTPAddr != ":9090" {
		t.Fatalf("expected override http addr, got %q", cfg.HTTPAddr)
	}
	if cfg.SessionTTL != 24*time.Hour {
		t.Fatalf("expected override session ttl, got %s", cfg.SessionTTL)
	}
	if !cfg.CookieSecure {
		t.Fatal("expected secure cookies")
	}
}

func TestFromLookupMailAndSchedulerDefaults(t *testing.T) {
	cfg := FromLookup(func(string) (string, bool) {
		return "", false
	})

	if !cfg.SchedulerEnabled {
		t.Fatal("expected scheduler enabled by default")
	}
	if cfg.MailMode != "log" {
		t.Fatalf("expected log mail mode, got %q", cfg.MailMode)
	}
	if cfg.MailMaxAttempts != 5 {
		t.Fatalf("expected 5 mail attempts, got %d", cfg.MailMaxAttempts)
	}
	if cfg.MailRetryBase != time.Minute {
		t.Fatalf("expected 1m retry base, got %s", cfg.MailRetryBase)
	}
	if cfg.PublicBaseURL != "http://127.0.0.1:8080" {
		t.Fatalf("unexpected public base url %q", cfg.PublicBaseURL)
	}
}
