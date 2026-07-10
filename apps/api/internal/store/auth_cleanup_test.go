package store

import (
	"context"
	"testing"
	"time"
)

func TestPurgeExpiredAuthArtifactsRemovesStaleRows(t *testing.T) {
	ctx := context.Background()
	st, user := newTimeEntryTestStore(t, ctx)

	usedToken, err := st.CreatePasswordResetToken(ctx, user.ID, time.Hour)
	if err != nil {
		t.Fatalf("create used reset token: %v", err)
	}
	if err := st.ResetPasswordWithToken(ctx, usedToken, "new-password-123"); err != nil {
		t.Fatalf("consume reset token: %v", err)
	}

	expiredResetToken, err := st.CreatePasswordResetToken(ctx, user.ID, time.Hour)
	if err != nil {
		t.Fatalf("create expired reset token: %v", err)
	}
	purgeNow := time.Now().UTC()
	if _, err := st.db.ExecContext(ctx, `
		UPDATE password_reset_tokens
		SET expires_at = ?
		WHERE token_hash = ?
	`, formatTime(purgeNow.Add(-time.Minute)), hashToken(expiredResetToken)); err != nil {
		t.Fatalf("expire reset token row: %v", err)
	}

	activeToken, _, err := st.CreateSession(ctx, user.ID, time.Hour)
	if err != nil {
		t.Fatalf("create active session: %v", err)
	}

	expiredToken, _, err := st.CreateSession(ctx, user.ID, time.Hour)
	if err != nil {
		t.Fatalf("create expired session: %v", err)
	}
	updateResult, err := st.db.ExecContext(ctx, `
		UPDATE sessions
		SET expires_at = ?
		WHERE token_hash = ?
	`, formatTime(purgeNow.Add(-time.Hour)), hashToken(expiredToken))
	if err != nil {
		t.Fatalf("expire session row: %v", err)
	}
	if affected, err := updateResult.RowsAffected(); err != nil || affected != 1 {
		t.Fatalf("expected to expire one session, affected=%d err=%v", affected, err)
	}

	result, err := st.PurgeExpiredAuthArtifacts(ctx, purgeNow)
	if err != nil {
		t.Fatalf("purge expired auth artifacts: %v", err)
	}
	if result.Sessions != 1 || result.PasswordResetTokens != 2 {
		t.Fatalf("unexpected purge counts: %+v", result)
	}

	if _, err := st.UserBySessionToken(ctx, activeToken); err != nil {
		t.Fatalf("expected active session to remain: %v", err)
	}
	if _, err := st.UserBySessionToken(ctx, expiredToken); err == nil {
		t.Fatal("expected expired session to be removed")
	}
}
