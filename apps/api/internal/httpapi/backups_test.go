package httpapi

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/leotime/leotime/apps/api/internal/backup"
	"github.com/leotime/leotime/apps/api/internal/backup/crypto"
	"github.com/leotime/leotime/apps/api/internal/backup/storage"
	"github.com/leotime/leotime/apps/api/internal/config"
	"github.com/leotime/leotime/apps/api/internal/db"
	"github.com/leotime/leotime/apps/api/internal/maintenance"
	"github.com/leotime/leotime/apps/api/internal/notify"
	"github.com/leotime/leotime/apps/api/internal/outbox"
	"github.com/leotime/leotime/apps/api/internal/store"
)

func TestBackupRoutesRequireAuthentication(t *testing.T) {
	router := newTestRouter(t)
	routes := []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodGet, "/api/v1/backups/settings", ""},
		{http.MethodPut, "/api/v1/backups/settings", `{"enabled":true}`},
		{http.MethodPost, "/api/v1/backups/test", ""},
		{http.MethodPost, "/api/v1/backups/run", ""},
		{http.MethodGet, "/api/v1/backups/objects", ""},
		{http.MethodPost, "/api/v1/backups/restore", `{"confirm":true,"latest":true}`},
		{http.MethodGet, "/api/v1/backups/status", ""},
	}

	for _, route := range routes {
		response := httptest.NewRecorder()
		var body io.Reader
		if route.body != "" {
			body = bytes.NewBufferString(route.body)
		}
		request := httptest.NewRequest(route.method, route.path, body)
		router.ServeHTTP(response, request)
		if response.Code != http.StatusUnauthorized {
			t.Fatalf("%s %s expected 401, got %d", route.method, route.path, response.Code)
		}
	}
}

func TestListBackupObjectsWithoutSecretsKey(t *testing.T) {
	router, _ := newBackupHTTPTestRouterWithoutSecretsKey(t)
	cookies := loginCookies(t, router)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/backups/objects", nil)
	for _, cookie := range cookies {
		request.AddCookie(cookie)
	}
	router.ServeHTTP(response, request)

	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d: %s", response.Code, response.Body.String())
	}
}

func TestRestoreBackupRequiresConfirm(t *testing.T) {
	router, _ := newBackupHTTPTestRouter(t)
	cookies := loginCookies(t, router)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/backups/restore", bytes.NewBufferString(`{"latest":true}`))
	for _, cookie := range cookies {
		request.AddCookie(cookie)
	}
	router.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", response.Code, response.Body.String())
	}
}

func TestBackupTestConnectionReturnsValidationFields(t *testing.T) {
	router, _ := newBackupHTTPTestRouter(t)
	cookies := loginCookies(t, router)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/backups/test", bytes.NewBufferString(`{
		"enabled": true,
		"accessKeyId": "test-key"
	}`))
	request.Header.Set("Content-Type", "application/json")
	for _, cookie := range cookies {
		request.AddCookie(cookie)
	}
	router.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", response.Code, response.Body.String())
	}

	var payload struct {
		Error struct {
			Fields []struct {
				Field string `json:"field"`
			} `json:"fields"`
		} `json:"error"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(payload.Error.Fields) == 0 || payload.Error.Fields[0].Field != "bucket" {
		t.Fatalf("expected bucket field error, got %+v", payload.Error.Fields)
	}
}

func TestListBackupObjectsRemoteStorageErrorIsGeneric(t *testing.T) {
	router, service := newBackupHTTPTestRouter(t)
	service.SetClientFactory(func(ctx context.Context, cfg storage.S3Config) (storage.Client, error) {
		return nil, fmt.Errorf("AccessDenied: secret bucket arn:aws:s3:::private")
	})
	cookies := loginCookies(t, router)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/backups/objects", nil)
	for _, cookie := range cookies {
		request.AddCookie(cookie)
	}
	router.ServeHTTP(response, request)

	if response.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d: %s", response.Code, response.Body.String())
	}
	body := response.Body.String()
	if strings.Contains(body, "AccessDenied") || strings.Contains(body, "arn:aws") {
		t.Fatalf("expected generic backup error, got %s", body)
	}

	var payload struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Error.Code != "backup_remote_storage_failed" {
		t.Fatalf("expected backup_remote_storage_failed, got %q", payload.Error.Code)
	}
}

func TestGetBackupStatus(t *testing.T) {
	router, _ := newBackupHTTPTestRouter(t)
	cookies := loginCookies(t, router)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/backups/status", nil)
	for _, cookie := range cookies {
		request.AddCookie(cookie)
	}
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
}

func TestRestoreBackupRejectsForeignObjectKeyWithGenericError(t *testing.T) {
	router, _ := newBackupHTTPTestRouter(t)
	cookies := loginCookies(t, router)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/backups/restore", bytes.NewBufferString(`{
		"confirm": true,
		"objectKey": "foreign-prefix/leotime.db.gz"
	}`))
	request.Header.Set("Content-Type", "application/json")
	for _, cookie := range cookies {
		request.AddCookie(cookie)
	}
	router.ServeHTTP(response, request)

	if response.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d: %s", response.Code, response.Body.String())
	}
	body := response.Body.String()
	if strings.Contains(body, "configured prefix") {
		t.Fatalf("expected generic backup error, got %s", body)
	}

	var payload struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Error.Code != "backup_operation_failed" {
		t.Fatalf("expected backup_operation_failed, got %q", payload.Error.Code)
	}
	if payload.Error.Message != "backup operation failed" {
		t.Fatalf("expected generic message, got %q", payload.Error.Message)
	}
}

func newBackupHTTPTestRouter(t *testing.T) (http.Handler, *backup.Service) {
	secretsKey := base64.StdEncoding.EncodeToString([]byte("01234567890123456789012345678901"))
	router, service, _ := newBackupHTTPTestRouterWithSecretsKey(t, secretsKey)
	return router, service
}

func newBackupHTTPTestRouterWithoutSecretsKey(t *testing.T) (http.Handler, *backup.Service) {
	router, service, _ := newBackupHTTPTestRouterWithSecretsKey(t, "")
	return router, service
}

func newBackupHTTPTestRouterWithSecretsKey(t *testing.T, secretsKey string) (http.Handler, *backup.Service, config.Config) {
	t.Helper()
	maintenance.Leave()
	t.Cleanup(maintenance.Leave)

	ctx := context.Background()
	database, err := db.Open(ctx, t.TempDir()+"/leotime.db")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	if err := db.Migrate(ctx, database); err != nil {
		t.Fatalf("migrate db: %v", err)
	}

	st := store.New(database)
	if err := st.BootstrapAdmin(ctx, "admin@example.com", "change-me-now"); err != nil {
		t.Fatalf("bootstrap admin: %v", err)
	}
	user, err := st.Authenticate(ctx, "admin@example.com", "change-me-now")
	if err != nil {
		t.Fatalf("authenticate: %v", err)
	}

	cfg := config.Config{
		DBPath:            t.TempDir() + "/leotime-live.db",
		DocumentRoot:      filepath.Join(t.TempDir(), "documents"),
		SecretsKey:        secretsKey,
		BootstrapEmail:    "admin@example.com",
		BootstrapPassword: "change-me-now",
		SessionCookieName: "leotime_session",
		SessionTTL:        time.Hour,
		PublicBaseURL:     "http://127.0.0.1:8080",
		PasswordResetTTL:  time.Hour,
		MailMaxAttempts:   5,
	}

	secretEnc, err := crypto.Encrypt([]byte("secret"), []byte("01234567890123456789012345678901"))
	if err != nil {
		t.Fatalf("encrypt secret: %v", err)
	}
	if _, err := st.UpsertBackupSettings(ctx, user.ID, store.BackupSettingsInput{
		Enabled: true, Bucket: "bucket", AccessKeyID: "key", ScheduleHour: 1, RetentionDays: 365,
	}, secretEnc); err != nil {
		t.Fatalf("save backup settings: %v", err)
	}

	outboxStore := outbox.NewStore(database)
	passwordReset := notify.NewPasswordResetService(st, outboxStore, cfg)
	backupService := backup.NewService(cfg, st, database, nil)
	backupService.SetClientFactory(func(ctx context.Context, cfg storage.S3Config) (storage.Client, error) {
		return storage.NewMemoryClient(), nil
	})
	router, err := NewRouter(cfg, st, passwordReset, backupService)
	if err != nil {
		t.Fatalf("new router: %v", err)
	}
	return router, backupService, cfg
}
