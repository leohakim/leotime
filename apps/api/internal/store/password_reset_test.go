package store

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/leotime/leotime/apps/api/internal/auth"
	"github.com/leotime/leotime/apps/api/internal/db"
)

func TestCreateAndResetPasswordWithToken(t *testing.T) {
	ctx := context.Background()
	database := openStoreTestDB(t, ctx)
	st := New(database)

	rawToken, err := st.CreatePasswordResetToken(ctx, mustAuthUserID(t, st), time.Hour)
	if err != nil {
		t.Fatalf("create password reset token: %v", err)
	}

	if err := st.ResetPasswordWithToken(ctx, rawToken, "new-secret-123"); err != nil {
		t.Fatalf("reset password with token: %v", err)
	}

	user, err := st.Authenticate(ctx, "admin@example.com", "new-secret-123")
	if err != nil {
		t.Fatalf("authenticate with new password: %v", err)
	}
	if user.Email != "admin@example.com" {
		t.Fatalf("unexpected user email %q", user.Email)
	}

	if err := st.ResetPasswordWithToken(ctx, rawToken, "another-secret-456"); err == nil {
		t.Fatal("expected token reuse to fail")
	}
}

func TestResetPasswordRejectsShortPassword(t *testing.T) {
	ctx := context.Background()
	database := openStoreTestDB(t, ctx)
	st := New(database)

	rawToken, err := st.CreatePasswordResetToken(ctx, mustAuthUserID(t, st), time.Hour)
	if err != nil {
		t.Fatalf("create password reset token: %v", err)
	}

	if err := st.ResetPasswordWithToken(ctx, rawToken, "short"); err == nil {
		t.Fatal("expected short password error")
	}
}

func TestResetPasswordRejectsExpiredToken(t *testing.T) {
	ctx := context.Background()
	database := openStoreTestDB(t, ctx)
	st := New(database)
	userID := mustAuthUserID(t, st)

	rawToken, err := st.CreatePasswordResetToken(ctx, userID, time.Millisecond)
	if err != nil {
		t.Fatalf("create password reset token: %v", err)
	}

	time.Sleep(5 * time.Millisecond)

	if err := st.ResetPasswordWithToken(ctx, rawToken, "new-secret-123"); err == nil {
		t.Fatal("expected expired token error")
	}
}

func openStoreTestDB(t *testing.T, ctx context.Context) *sql.DB {
	t.Helper()

	database, err := db.Open(ctx, t.TempDir()+"/leotime.db")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	if err := db.Migrate(ctx, database); err != nil {
		t.Fatalf("migrate db: %v", err)
	}
	return database
}

func mustAuthUserID(t *testing.T, st *Store) string {
	t.Helper()

	if err := st.BootstrapAdmin(context.Background(), "admin@example.com", "change-me-now"); err != nil {
		t.Fatalf("bootstrap admin: %v", err)
	}
	user, err := st.Authenticate(context.Background(), "admin@example.com", "change-me-now")
	if err != nil {
		t.Fatalf("authenticate: %v", err)
	}
	return user.ID
}

func TestUserByEmail(t *testing.T) {
	ctx := context.Background()
	st := New(openStoreTestDB(t, ctx))
	mustAuthUserID(t, st)

	user, err := st.UserByEmail(ctx, "admin@example.com")
	if err != nil {
		t.Fatalf("user by email: %v", err)
	}
	if user.Email != "admin@example.com" {
		t.Fatalf("unexpected email %q", user.Email)
	}

	if _, err := st.UserByEmail(ctx, "missing@example.com"); err == nil {
		t.Fatal("expected missing user error")
	}
}

func TestResetPasswordClearsSessions(t *testing.T) {
	ctx := context.Background()
	st := New(openStoreTestDB(t, ctx))
	userID := mustAuthUserID(t, st)

	token, _, err := st.CreateSession(ctx, userID, time.Hour)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	rawReset, err := st.CreatePasswordResetToken(ctx, userID, time.Hour)
	if err != nil {
		t.Fatalf("create password reset token: %v", err)
	}
	if err := st.ResetPasswordWithToken(ctx, rawReset, "new-secret-123"); err != nil {
		t.Fatalf("reset password: %v", err)
	}

	if _, err := st.UserBySessionToken(ctx, token); err == nil {
		t.Fatal("expected session to be cleared after password reset")
	}

	if !auth.VerifyPassword(mustPasswordHash(t, st, userID), "new-secret-123") {
		t.Fatal("expected password hash to match new password")
	}
}

func mustPasswordHash(t *testing.T, st *Store, userID string) string {
	t.Helper()

	var passwordHash string
	if err := st.db.QueryRowContext(context.Background(), "SELECT password_hash FROM users WHERE id = ?", userID).Scan(&passwordHash); err != nil {
		t.Fatalf("query password hash: %v", err)
	}
	return passwordHash
}
