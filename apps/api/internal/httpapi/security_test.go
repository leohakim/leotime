package httpapi

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/leotime/leotime/apps/api/internal/backup"
	"github.com/leotime/leotime/apps/api/internal/config"
	"github.com/leotime/leotime/apps/api/internal/db"
	"github.com/leotime/leotime/apps/api/internal/maintenance"
	"github.com/leotime/leotime/apps/api/internal/notify"
	"github.com/leotime/leotime/apps/api/internal/outbox"
	"github.com/leotime/leotime/apps/api/internal/store"
)

func TestSafeStaticFilePathRejectsTraversal(t *testing.T) {
	root := t.TempDir()
	indexPath := filepath.Join(root, "index.html")
	if err := os.WriteFile(indexPath, []byte("<html></html>"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, ok := safeStaticFilePath(root, "../index.html"); ok {
		t.Fatal("expected traversal path to be rejected")
	}
	if _, ok := safeStaticFilePath(root, "../../etc/passwd"); ok {
		t.Fatal("expected traversal path to be rejected")
	}

	fullPath, ok := safeStaticFilePath(root, "index.html")
	if !ok || fullPath != indexPath {
		t.Fatalf("expected index.html path, got %q ok=%v", fullPath, ok)
	}
}

func TestMetricsHiddenInProductionWithoutToken(t *testing.T) {
	cfg := testSecurityConfig()
	cfg.Production = true
	cfg.MetricsToken = ""

	router := newTestRouterWithConfig(t, cfg)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	router.ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", response.Code)
	}
}

func TestMetricsRequiresBearerTokenWhenConfigured(t *testing.T) {
	cfg := testSecurityConfig()
	cfg.MetricsToken = "secret-metrics"

	router := newTestRouterWithConfig(t, cfg)

	unauth := httptest.NewRecorder()
	router.ServeHTTP(unauth, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if unauth.Code != http.StatusNotFound {
		t.Fatalf("expected 404 without token, got %d", unauth.Code)
	}

	queryToken := httptest.NewRecorder()
	router.ServeHTTP(queryToken, httptest.NewRequest(http.MethodGet, "/metrics?token=secret-metrics", nil))
	if queryToken.Code != http.StatusNotFound {
		t.Fatalf("expected 404 with query token, got %d", queryToken.Code)
	}

	auth := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	req.Header.Set("Authorization", "Bearer secret-metrics")
	router.ServeHTTP(auth, req)
	if auth.Code != http.StatusOK {
		t.Fatalf("expected 200 with token, got %d", auth.Code)
	}
}

func TestSecurityHeadersMiddleware(t *testing.T) {
	router := newTestRouterWithConfig(t, testSecurityConfig())

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/health", nil))

	if response.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("expected nosniff header, got %q", response.Header().Get("X-Content-Type-Options"))
	}
	if response.Header().Get("Referrer-Policy") != "no-referrer" {
		t.Fatalf("expected no-referrer header, got %q", response.Header().Get("Referrer-Policy"))
	}
	if response.Header().Get("X-Frame-Options") != "DENY" {
		t.Fatalf("expected DENY framing header, got %q", response.Header().Get("X-Frame-Options"))
	}
}

func TestLoginRateLimitIgnoresForwardedHeaderByDefault(t *testing.T) {
	router := newTestRouterWithConfig(t, testSecurityConfig())

	for i := 0; i < 10; i++ {
		response := httptest.NewRecorder()
		body := bytes.NewBufferString(`{"email":"admin@example.com","password":"wrong"}`)
		request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", body)
		request.RemoteAddr = "203.0.113.10:1234"
		request.Header.Set("X-Forwarded-For", "198.51.100.99")
		router.ServeHTTP(response, request)
		if response.Code == http.StatusTooManyRequests {
			t.Fatalf("unexpected rate limit at attempt %d", i+1)
		}
	}

	response := httptest.NewRecorder()
	body := bytes.NewBufferString(`{"email":"admin@example.com","password":"wrong"}`)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", body)
	request.RemoteAddr = "203.0.113.10:1234"
	request.Header.Set("X-Forwarded-For", "198.51.100.200")
	router.ServeHTTP(response, request)
	if response.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 for same peer address, got %d", response.Code)
	}
}

func TestLoginRateLimitUsesForwardedHeaderWhenTrusted(t *testing.T) {
	cfg := testSecurityConfig()
	cfg.TrustForwardedHeaders = true
	router := newTestRouterWithConfig(t, cfg)

	for i := 0; i < 10; i++ {
		response := httptest.NewRecorder()
		body := bytes.NewBufferString(`{"email":"admin@example.com","password":"wrong"}`)
		request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", body)
		request.RemoteAddr = "203.0.113.10:1234"
		request.Header.Set("X-Forwarded-For", "198.51.100.50")
		router.ServeHTTP(response, request)
	}

	response := httptest.NewRecorder()
	body := bytes.NewBufferString(`{"email":"admin@example.com","password":"wrong"}`)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", body)
	request.RemoteAddr = "203.0.113.99:1234"
	request.Header.Set("X-Forwarded-For", "198.51.100.50")
	router.ServeHTTP(response, request)
	if response.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 for trusted forwarded client, got %d", response.Code)
	}
}

func TestLoginRateLimit(t *testing.T) {
	router := newTestRouterWithConfig(t, testSecurityConfig())

	for i := 0; i < 10; i++ {
		response := httptest.NewRecorder()
		body := bytes.NewBufferString(`{"email":"admin@example.com","password":"wrong"}`)
		request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", body)
		request.RemoteAddr = "203.0.113.10:1234"
		router.ServeHTTP(response, request)
		if response.Code == http.StatusTooManyRequests {
			t.Fatalf("unexpected rate limit at attempt %d", i+1)
		}
	}

	response := httptest.NewRecorder()
	body := bytes.NewBufferString(`{"email":"admin@example.com","password":"wrong"}`)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", body)
	request.RemoteAddr = "203.0.113.10:1234"
	router.ServeHTTP(response, request)
	if response.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", response.Code)
	}
}

func TestJSONBodyTooLarge(t *testing.T) {
	router := newTestRouter(t)

	largeBody := bytes.NewBufferString(`{"email":"` + strings.Repeat("a", 2<<20) + `","password":"x"}`)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", largeBody)
	router.ServeHTTP(response, request)

	if response.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413, got %d", response.Code)
	}
}

func TestMaintenanceModeBlocksAPI(t *testing.T) {
	router := newTestRouter(t)
	cookies := loginCookies(t, router)

	maintenance.Enter()
	t.Cleanup(maintenance.Leave)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/clients", nil)
	for _, cookie := range cookies {
		request.AddCookie(cookie)
	}
	router.ServeHTTP(response, request)
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", response.Code)
	}

	health := httptest.NewRecorder()
	router.ServeHTTP(health, httptest.NewRequest(http.MethodGet, "/api/health", nil))
	if health.Code != http.StatusOK {
		t.Fatalf("expected health 200, got %d", health.Code)
	}
}

func testSecurityConfig() config.Config {
	return config.Config{
		HTTPAddr:          ":0",
		DBPath:            "unused",
		BootstrapEmail:    "admin@example.com",
		BootstrapPassword: "change-me-now",
		SessionCookieName: "leotime_session",
		SessionTTL:        time.Hour,
		PublicBaseURL:     "http://127.0.0.1:8080",
		PasswordResetTTL:  time.Hour,
		MailMaxAttempts:   5,
	}
}

func newTestRouterWithConfig(t *testing.T, cfg config.Config) http.Handler {
	t.Helper()
	maintenance.Leave()
	t.Cleanup(maintenance.Leave)
	if cfg.DocumentRoot == "" {
		cfg.DocumentRoot = filepath.Join(t.TempDir(), "documents")
	}

	ctx := context.Background()
	database, err := db.Open(ctx, t.TempDir()+"/leotime.db")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() {
		database.Close()
	})

	if err := db.Migrate(ctx, database); err != nil {
		t.Fatalf("migrate db: %v", err)
	}

	st := store.New(database)
	if err := st.BootstrapAdmin(ctx, "admin@example.com", "change-me-now"); err != nil {
		t.Fatalf("bootstrap admin: %v", err)
	}

	outboxStore := outbox.NewStore(database)
	passwordReset := notify.NewPasswordResetService(st, outboxStore, cfg)
	backupService := backup.NewService(cfg, st, database, nil)
	router, err := NewRouter(cfg, st, passwordReset, backupService)
	if err != nil {
		t.Fatalf("new router: %v", err)
	}
	return router
}
