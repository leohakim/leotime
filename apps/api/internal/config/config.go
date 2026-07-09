package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	HTTPAddr          string
	DBPath            string
	StaticDir         string
	BootstrapEmail    string
	BootstrapPassword string
	SessionCookieName string
	SessionTTL        time.Duration
	CookieSecure      bool
	Production        bool
	MetricsToken      string

	bootstrapPasswordSet bool

	SchedulerEnabled        bool
	SchedulerScanInterval   time.Duration
	OutboxProcessInterval   time.Duration
	MailMode                string
	MailFrom                string
	MailFromName            string
	SMTPHost                string
	SMTPPort                int
	SMTPUsername            string
	SMTPPassword            string
	SMTPTLS                 bool
	MailMaxAttempts         int
	MailRetryBase           time.Duration
	MailRetryMax            time.Duration
	PublicBaseURL           string
	PasswordResetTTL        time.Duration
	SecretsKey              string
	BackupSchedulerEnabled  bool
	BackupSchedulerInterval time.Duration
	DocumentRoot            string
}

var (
	ErrBootstrapPasswordRequired = errors.New("LEOTIME_BOOTSTRAP_PASSWORD must be set in production")
	ErrBootstrapPasswordDefault  = errors.New("LEOTIME_BOOTSTRAP_PASSWORD must not use the default value in production")
)

func Load() Config {
	return FromLookup(os.LookupEnv)
}

func (c Config) Validate() error {
	if !c.Production {
		return nil
	}
	if !c.bootstrapPasswordSet {
		return ErrBootstrapPasswordRequired
	}
	if c.BootstrapPassword == "change-me-now" {
		return ErrBootstrapPasswordDefault
	}
	return nil
}

func FromLookup(lookup func(string) (string, bool)) Config {
	bootstrapPassword, bootstrapPasswordSet := lookupString(lookup, "LEOTIME_BOOTSTRAP_PASSWORD", "change-me-now")
	return Config{
		HTTPAddr:          stringEnv(lookup, "LEOTIME_HTTP_ADDR", ":8080"),
		DBPath:            stringEnv(lookup, "LEOTIME_DB_PATH", "data/leotime.db"),
		StaticDir:         stringEnv(lookup, "LEOTIME_STATIC_DIR", ""),
		BootstrapEmail:    stringEnv(lookup, "LEOTIME_BOOTSTRAP_EMAIL", "admin@example.com"),
		BootstrapPassword: bootstrapPassword,
		SessionCookieName: stringEnv(lookup, "LEOTIME_SESSION_COOKIE", "leotime_session"),
		SessionTTL:        durationEnv(lookup, "LEOTIME_SESSION_TTL", 168*time.Hour),
		CookieSecure:      boolEnv(lookup, "LEOTIME_COOKIE_SECURE", false),
		Production:        productionEnv(lookup),
		MetricsToken:      stringEnv(lookup, "LEOTIME_METRICS_TOKEN", ""),

		SchedulerEnabled:        boolEnv(lookup, "LEOTIME_SCHEDULER_ENABLED", true),
		SchedulerScanInterval:   durationEnv(lookup, "LEOTIME_SCHEDULER_SCAN_INTERVAL", 10*time.Minute),
		OutboxProcessInterval:   durationEnv(lookup, "LEOTIME_OUTBOX_PROCESS_INTERVAL", 30*time.Second),
		MailMode:                stringEnv(lookup, "LEOTIME_MAIL_MODE", "log"),
		MailFrom:                stringEnv(lookup, "LEOTIME_MAIL_FROM", "no-reply@localhost"),
		MailFromName:            stringEnv(lookup, "LEOTIME_MAIL_FROM_NAME", "leotime"),
		SMTPHost:                stringEnv(lookup, "LEOTIME_SMTP_HOST", ""),
		SMTPPort:                intEnv(lookup, "LEOTIME_SMTP_PORT", 587),
		SMTPUsername:            stringEnv(lookup, "LEOTIME_SMTP_USERNAME", ""),
		SMTPPassword:            stringEnv(lookup, "LEOTIME_SMTP_PASSWORD", ""),
		SMTPTLS:                 boolEnv(lookup, "LEOTIME_SMTP_TLS", true),
		MailMaxAttempts:         intEnv(lookup, "LEOTIME_MAIL_MAX_ATTEMPTS", 5),
		MailRetryBase:           durationEnv(lookup, "LEOTIME_MAIL_RETRY_BASE", time.Minute),
		MailRetryMax:            durationEnv(lookup, "LEOTIME_MAIL_RETRY_MAX", 6*time.Hour),
		PublicBaseURL:           stringEnv(lookup, "LEOTIME_PUBLIC_BASE_URL", "http://127.0.0.1:8080"),
		PasswordResetTTL:        durationEnv(lookup, "LEOTIME_PASSWORD_RESET_TTL", time.Hour),
		SecretsKey:              stringEnv(lookup, "LEOTIME_SECRETS_KEY", ""),
		BackupSchedulerEnabled:  boolEnv(lookup, "LEOTIME_BACKUP_SCHEDULER_ENABLED", true),
		BackupSchedulerInterval: durationEnv(lookup, "LEOTIME_BACKUP_SCHEDULER_INTERVAL", time.Minute),
		DocumentRoot:            stringEnv(lookup, "LEOTIME_DOCUMENT_ROOT", "/data/documents"),
		bootstrapPasswordSet:    bootstrapPasswordSet,
	}
}

func lookupString(lookup func(string) (string, bool), key string, fallback string) (string, bool) {
	value, ok := lookup(key)
	if !ok || value == "" {
		return fallback, false
	}
	return value, true
}

func productionEnv(lookup func(string) (string, bool)) bool {
	value, ok := lookup("LEOTIME_ENV")
	if !ok {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(value), "production")
}

func stringEnv(lookup func(string) (string, bool), key string, fallback string) string {
	value, ok := lookup(key)
	if !ok || value == "" {
		return fallback
	}
	return value
}

func durationEnv(lookup func(string) (string, bool), key string, fallback time.Duration) time.Duration {
	value, ok := lookup(key)
	if !ok || value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func boolEnv(lookup func(string) (string, bool), key string, fallback bool) bool {
	value, ok := lookup(key)
	if !ok || value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func intEnv(lookup func(string) (string, bool), key string, fallback int) int {
	value, ok := lookup(key)
	if !ok || value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}
