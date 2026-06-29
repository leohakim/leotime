package store

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/leotime/leotime/apps/api/internal/db"
)

func TestBootstrapAuthenticateAndSession(t *testing.T) {
	ctx := context.Background()
	database, err := db.Open(ctx, t.TempDir()+"/leotime.db")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer database.Close()

	if err := db.Migrate(ctx, database); err != nil {
		t.Fatalf("migrate db: %v", err)
	}

	st := New(database)
	if err := st.BootstrapAdmin(ctx, "admin@example.com", "change-me-now"); err != nil {
		t.Fatalf("bootstrap admin: %v", err)
	}

	user, err := st.Authenticate(ctx, "ADMIN@example.com", "change-me-now")
	if err != nil {
		t.Fatalf("authenticate: %v", err)
	}
	if user.Email != "admin@example.com" {
		t.Fatalf("expected normalized email, got %q", user.Email)
	}

	if _, err := st.Authenticate(ctx, "admin@example.com", "wrong"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected invalid credentials, got %v", err)
	}

	token, _, err := st.CreateSession(ctx, user.ID, time.Hour)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	sessionUser, err := st.UserBySessionToken(ctx, token)
	if err != nil {
		t.Fatalf("query session user: %v", err)
	}
	if sessionUser.ID != user.ID {
		t.Fatalf("expected session user %q, got %q", user.ID, sessionUser.ID)
	}

	if err := st.DeleteSession(ctx, token); err != nil {
		t.Fatalf("delete session: %v", err)
	}
	if _, err := st.UserBySessionToken(ctx, token); !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("expected missing session, got %v", err)
	}
}
