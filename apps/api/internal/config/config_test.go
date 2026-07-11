package config

import (
	"strings"
	"testing"
	"time"
)

func TestFromLookupUsesDefaults(t *testing.T) {
	cfg, err := FromLookup(func(string) (string, bool) {
		return "", false
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

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

	cfg, err := FromLookup(func(key string) (string, bool) {
		value, ok := env[key]
		return value, ok
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

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

func TestFromLookupRejectsInvalidBoolean(t *testing.T) {
	_, err := FromLookup(func(key string) (string, bool) {
		if key == "LEOTIME_COOKIE_SECURE" {
			return "not-a-bool", true
		}
		return "", false
	})
	if err == nil || !strings.Contains(err.Error(), "LEOTIME_COOKIE_SECURE") {
		t.Fatalf("expected boolean parse error, got %v", err)
	}
}

func TestFromLookupRejectsInvalidDuration(t *testing.T) {
	_, err := FromLookup(func(key string) (string, bool) {
		if key == "LEOTIME_SESSION_TTL" {
			return "not-a-duration", true
		}
		return "", false
	})
	if err == nil || !strings.Contains(err.Error(), "LEOTIME_SESSION_TTL") {
		t.Fatalf("expected duration parse error, got %v", err)
	}
}

func TestFromLookupRejectsInvalidInteger(t *testing.T) {
	_, err := FromLookup(func(key string) (string, bool) {
		if key == "LEOTIME_SMTP_PORT" {
			return "abc", true
		}
		return "", false
	})
	if err == nil || !strings.Contains(err.Error(), "LEOTIME_SMTP_PORT") {
		t.Fatalf("expected integer parse error, got %v", err)
	}
}

func TestFromLookupMailAndSchedulerDefaults(t *testing.T) {
	cfg, err := FromLookup(func(string) (string, bool) {
		return "", false
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

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
	if cfg.PublicBaseURL != defaultPublicBaseURL {
		t.Fatalf("unexpected public base url %q", cfg.PublicBaseURL)
	}
	if cfg.DocumentRoot != "/data/documents" {
		t.Fatalf("expected default document root, got %q", cfg.DocumentRoot)
	}
	if cfg.TrustForwardedHeaders {
		t.Fatal("expected forwarded headers to be untrusted by default")
	}
}

func TestValidateRequiresBootstrapPasswordInProduction(t *testing.T) {
	cfg, err := FromLookup(func(key string) (string, bool) {
		if key == "LEOTIME_ENV" {
			return "production", true
		}
		return "", false
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected production validation error")
	}

	cfg, err = FromLookup(func(key string) (string, bool) {
		switch key {
		case "LEOTIME_ENV":
			return "production", true
		case "LEOTIME_BOOTSTRAP_PASSWORD":
			return "change-me-now", true
		}
		return "", false
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := cfg.Validate(); err != ErrBootstrapPasswordDefault {
		t.Fatalf("expected default password error, got %v", err)
	}
}

func productionConfigLookup(extra map[string]string) func(string) (string, bool) {
	return func(key string) (string, bool) {
		if value, ok := extra[key]; ok {
			return value, true
		}
		switch key {
		case "LEOTIME_ENV":
			return "production", true
		case "LEOTIME_BOOTSTRAP_PASSWORD":
			return "strong-production-password", true
		case "LEOTIME_COOKIE_SECURE":
			return "true", true
		case "LEOTIME_PUBLIC_BASE_URL":
			return "https://leotime.example.com", true
		case "LEOTIME_MAIL_MODE":
			return "smtp", true
		}
		return "", false
	}
}

func TestValidateRequiresSecureCookiesInProduction(t *testing.T) {
	cfg, err := FromLookup(productionConfigLookup(map[string]string{
		"LEOTIME_COOKIE_SECURE": "false",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := cfg.Validate(); err != ErrCookieSecureRequired {
		t.Fatalf("expected cookie secure error, got %v", err)
	}
}

func TestValidateRequiresExplicitPublicBaseURLInProduction(t *testing.T) {
	cfg, err := FromLookup(func(key string) (string, bool) {
		switch key {
		case "LEOTIME_ENV":
			return "production", true
		case "LEOTIME_BOOTSTRAP_PASSWORD":
			return "strong-production-password", true
		case "LEOTIME_COOKIE_SECURE":
			return "true", true
		case "LEOTIME_MAIL_MODE":
			return "smtp", true
		}
		return "", false
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := cfg.Validate(); err != ErrPublicBaseURLRequired {
		t.Fatalf("expected public base url required error, got %v", err)
	}

	cfg, err = FromLookup(productionConfigLookup(map[string]string{
		"LEOTIME_PUBLIC_BASE_URL": defaultPublicBaseURL,
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := cfg.Validate(); err != ErrPublicBaseURLDefault {
		t.Fatalf("expected default public base url error, got %v", err)
	}
}

func TestValidateRequiresMailLogOptInForProductionLogMode(t *testing.T) {
	cfg, err := FromLookup(productionConfigLookup(map[string]string{
		"LEOTIME_MAIL_MODE": "log",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := cfg.Validate(); err != ErrMailLogProduction {
		t.Fatalf("expected mail log production error, got %v", err)
	}

	cfg, err = FromLookup(productionConfigLookup(map[string]string{
		"LEOTIME_MAIL_MODE":        "log",
		"LEOTIME_MAIL_LOG_ENABLED": "true",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected valid production log mail config, got %v", err)
	}
}

func TestValidateAcceptsValidProductionConfig(t *testing.T) {
	cfg, err := FromLookup(productionConfigLookup(nil))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected valid production config, got %v", err)
	}
}
