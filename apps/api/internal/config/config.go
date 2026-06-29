package config

import (
	"os"
	"strconv"
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
}

func Load() Config {
	return FromLookup(os.LookupEnv)
}

func FromLookup(lookup func(string) (string, bool)) Config {
	return Config{
		HTTPAddr:          stringEnv(lookup, "LEOTIME_HTTP_ADDR", ":8080"),
		DBPath:            stringEnv(lookup, "LEOTIME_DB_PATH", "data/leotime.db"),
		StaticDir:         stringEnv(lookup, "LEOTIME_STATIC_DIR", ""),
		BootstrapEmail:    stringEnv(lookup, "LEOTIME_BOOTSTRAP_EMAIL", "admin@example.com"),
		BootstrapPassword: stringEnv(lookup, "LEOTIME_BOOTSTRAP_PASSWORD", "change-me-now"),
		SessionCookieName: stringEnv(lookup, "LEOTIME_SESSION_COOKIE", "leotime_session"),
		SessionTTL:        durationEnv(lookup, "LEOTIME_SESSION_TTL", 168*time.Hour),
		CookieSecure:      boolEnv(lookup, "LEOTIME_COOKIE_SECURE", false),
	}
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
