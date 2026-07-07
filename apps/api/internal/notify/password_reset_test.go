package notify

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/leotime/leotime/apps/api/internal/config"
	"github.com/leotime/leotime/apps/api/internal/db"
	"github.com/leotime/leotime/apps/api/internal/outbox"
	"github.com/leotime/leotime/apps/api/internal/store"
)

func TestPasswordResetServiceEnqueuesEmail(t *testing.T) {
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

	outboxStore := outbox.NewStore(database)
	service := NewPasswordResetService(st, outboxStore, config.Config{
		PublicBaseURL:    "http://127.0.0.1:8080",
		PasswordResetTTL: time.Hour,
		MailMaxAttempts:  5,
	})

	if err := service.RequestReset(ctx, "admin@example.com"); err != nil {
		t.Fatalf("request reset: %v", err)
	}

	var kind string
	if err := database.QueryRowContext(ctx, `
		SELECT kind FROM email_outbox LIMIT 1
	`).Scan(&kind); err != nil {
		t.Fatalf("query outbox kind: %v", err)
	}
	if kind != outbox.KindPasswordReset {
		t.Fatalf("expected password_reset kind, got %q", kind)
	}

	if err := service.RequestReset(ctx, "missing@example.com"); err != nil {
		t.Fatalf("missing user should not fail: %v", err)
	}
}

func TestPasswordResetURLIncludesToken(t *testing.T) {
	url := passwordResetURL("http://127.0.0.1:8080/", "abc123")
	if !strings.Contains(url, "#reset-password?token=abc123") {
		t.Fatalf("unexpected reset url %q", url)
	}
}
