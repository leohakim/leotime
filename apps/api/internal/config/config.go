package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const defaultPublicBaseURL = "http://127.0.0.1:8080"

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
	publicBaseURLSet     bool

	SchedulerEnabled        bool
	SchedulerScanInterval   time.Duration
	OutboxProcessInterval   time.Duration
	MailMode                string
	MailLogEnabled          bool
	MailLogBody             bool
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
	TrustForwardedHeaders   bool
}

var (
	ErrBootstrapPasswordRequired = errors.New("LEOTIME_BOOTSTRAP_PASSWORD must be set in production")
	ErrBootstrapPasswordDefault  = errors.New("LEOTIME_BOOTSTRAP_PASSWORD must not use the default value in production")
	ErrCookieSecureRequired      = errors.New("LEOTIME_COOKIE_SECURE must be true in production")
	ErrPublicBaseURLRequired     = errors.New("LEOTIME_PUBLIC_BASE_URL must be set in production")
	ErrPublicBaseURLDefault      = errors.New("LEOTIME_PUBLIC_BASE_URL must not use the development default in production")
	ErrMailLogProduction         = errors.New("LEOTIME_MAIL_MODE=log requires LEOTIME_MAIL_LOG_ENABLED=true in production")
)

func Load() (Config, error) {
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
	if !c.CookieSecure {
		return ErrCookieSecureRequired
	}
	if !c.publicBaseURLSet {
		return ErrPublicBaseURLRequired
	}
	if strings.TrimSpace(c.PublicBaseURL) == defaultPublicBaseURL {
		return ErrPublicBaseURLDefault
	}
	if strings.EqualFold(strings.TrimSpace(c.MailMode), "log") && !c.MailLogEnabled {
		return ErrMailLogProduction
	}
	return nil
}

func FromLookup(lookup func(string) (string, bool)) (Config, error) {
	var parseErrors []error

	bootstrapPassword, bootstrapPasswordSet := lookupString(lookup, "LEOTIME_BOOTSTRAP_PASSWORD", "change-me-now")
	publicBaseURL, publicBaseURLSet := lookupString(lookup, "LEOTIME_PUBLIC_BASE_URL", defaultPublicBaseURL)

	sessionTTL, err := durationEnv(lookup, "LEOTIME_SESSION_TTL", 168*time.Hour)
	if err != nil {
		parseErrors = append(parseErrors, err)
	}
	cookieSecure, err := boolEnv(lookup, "LEOTIME_COOKIE_SECURE", false)
	if err != nil {
		parseErrors = append(parseErrors, err)
	}
	schedulerEnabled, err := boolEnv(lookup, "LEOTIME_SCHEDULER_ENABLED", true)
	if err != nil {
		parseErrors = append(parseErrors, err)
	}
	schedulerScanInterval, err := durationEnv(lookup, "LEOTIME_SCHEDULER_SCAN_INTERVAL", 10*time.Minute)
	if err != nil {
		parseErrors = append(parseErrors, err)
	}
	outboxProcessInterval, err := durationEnv(lookup, "LEOTIME_OUTBOX_PROCESS_INTERVAL", 30*time.Second)
	if err != nil {
		parseErrors = append(parseErrors, err)
	}
	smtpPort, err := intEnv(lookup, "LEOTIME_SMTP_PORT", 587)
	if err != nil {
		parseErrors = append(parseErrors, err)
	}
	smtpTLS, err := boolEnv(lookup, "LEOTIME_SMTP_TLS", true)
	if err != nil {
		parseErrors = append(parseErrors, err)
	}
	mailMaxAttempts, err := intEnv(lookup, "LEOTIME_MAIL_MAX_ATTEMPTS", 5)
	if err != nil {
		parseErrors = append(parseErrors, err)
	}
	mailRetryBase, err := durationEnv(lookup, "LEOTIME_MAIL_RETRY_BASE", time.Minute)
	if err != nil {
		parseErrors = append(parseErrors, err)
	}
	mailRetryMax, err := durationEnv(lookup, "LEOTIME_MAIL_RETRY_MAX", 6*time.Hour)
	if err != nil {
		parseErrors = append(parseErrors, err)
	}
	passwordResetTTL, err := durationEnv(lookup, "LEOTIME_PASSWORD_RESET_TTL", time.Hour)
	if err != nil {
		parseErrors = append(parseErrors, err)
	}
	backupSchedulerEnabled, err := boolEnv(lookup, "LEOTIME_BACKUP_SCHEDULER_ENABLED", true)
	if err != nil {
		parseErrors = append(parseErrors, err)
	}
	backupSchedulerInterval, err := durationEnv(lookup, "LEOTIME_BACKUP_SCHEDULER_INTERVAL", time.Minute)
	if err != nil {
		parseErrors = append(parseErrors, err)
	}
	mailLogEnabled, err := boolEnv(lookup, "LEOTIME_MAIL_LOG_ENABLED", false)
	if err != nil {
		parseErrors = append(parseErrors, err)
	}
	mailLogBody, err := boolEnv(lookup, "LEOTIME_MAIL_LOG_BODY", false)
	if err != nil {
		parseErrors = append(parseErrors, err)
	}
	trustForwardedHeaders, err := boolEnv(lookup, "LEOTIME_TRUST_FORWARDED_HEADERS", false)
	if err != nil {
		parseErrors = append(parseErrors, err)
	}

	cfg := Config{
		HTTPAddr:          stringEnv(lookup, "LEOTIME_HTTP_ADDR", ":8080"),
		DBPath:            stringEnv(lookup, "LEOTIME_DB_PATH", "data/leotime.db"),
		StaticDir:         stringEnv(lookup, "LEOTIME_STATIC_DIR", ""),
		BootstrapEmail:    stringEnv(lookup, "LEOTIME_BOOTSTRAP_EMAIL", "admin@example.com"),
		BootstrapPassword: bootstrapPassword,
		SessionCookieName: stringEnv(lookup, "LEOTIME_SESSION_COOKIE", "leotime_session"),
		SessionTTL:        sessionTTL,
		CookieSecure:      cookieSecure,
		Production:        productionEnv(lookup),
		MetricsToken:      stringEnv(lookup, "LEOTIME_METRICS_TOKEN", ""),

		SchedulerEnabled:        schedulerEnabled,
		SchedulerScanInterval:   schedulerScanInterval,
		OutboxProcessInterval:   outboxProcessInterval,
		MailMode:                stringEnv(lookup, "LEOTIME_MAIL_MODE", "log"),
		MailLogEnabled:          mailLogEnabled,
		MailLogBody:             mailLogBody,
		MailFrom:                stringEnv(lookup, "LEOTIME_MAIL_FROM", "no-reply@localhost"),
		MailFromName:            stringEnv(lookup, "LEOTIME_MAIL_FROM_NAME", "leotime"),
		SMTPHost:                stringEnv(lookup, "LEOTIME_SMTP_HOST", ""),
		SMTPPort:                smtpPort,
		SMTPUsername:            stringEnv(lookup, "LEOTIME_SMTP_USERNAME", ""),
		SMTPPassword:            stringEnv(lookup, "LEOTIME_SMTP_PASSWORD", ""),
		SMTPTLS:                 smtpTLS,
		MailMaxAttempts:         mailMaxAttempts,
		MailRetryBase:           mailRetryBase,
		MailRetryMax:            mailRetryMax,
		PublicBaseURL:           publicBaseURL,
		PasswordResetTTL:        passwordResetTTL,
		SecretsKey:              stringEnv(lookup, "LEOTIME_SECRETS_KEY", ""),
		BackupSchedulerEnabled:  backupSchedulerEnabled,
		BackupSchedulerInterval: backupSchedulerInterval,
		DocumentRoot:            stringEnv(lookup, "LEOTIME_DOCUMENT_ROOT", "/data/documents"),
		TrustForwardedHeaders:   trustForwardedHeaders,

		bootstrapPasswordSet: bootstrapPasswordSet,
		publicBaseURLSet:     publicBaseURLSet,
	}
	if len(parseErrors) > 0 {
		return Config{}, errors.Join(parseErrors...)
	}
	return cfg, nil
}

func lookupString(lookup func(string) (string, bool), key string, fallback string) (string, bool) {
	value, ok := lookup(key)
	if !ok || strings.TrimSpace(value) == "" {
		return fallback, false
	}
	return strings.TrimSpace(value), true
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
	if !ok || strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
}

func durationEnv(lookup func(string) (string, bool), key string, fallback time.Duration) (time.Duration, error) {
	value, ok := lookup(key)
	if !ok || strings.TrimSpace(value) == "" {
		return fallback, nil
	}
	parsed, err := time.ParseDuration(strings.TrimSpace(value))
	if err != nil {
		return 0, fmt.Errorf("%s: invalid duration", key)
	}
	return parsed, nil
}

func boolEnv(lookup func(string) (string, bool), key string, fallback bool) (bool, error) {
	value, ok := lookup(key)
	if !ok || strings.TrimSpace(value) == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseBool(strings.TrimSpace(value))
	if err != nil {
		return false, fmt.Errorf("%s: invalid boolean", key)
	}
	return parsed, nil
}

func intEnv(lookup func(string) (string, bool), key string, fallback int) (int, error) {
	value, ok := lookup(key)
	if !ok || strings.TrimSpace(value) == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0, fmt.Errorf("%s: invalid integer", key)
	}
	return parsed, nil
}
