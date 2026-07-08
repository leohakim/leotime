package notify

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/leotime/leotime/apps/api/internal/config"
	"github.com/leotime/leotime/apps/api/internal/outbox"
	"github.com/leotime/leotime/apps/api/internal/store"
)

func TestBackupEmailSubjectAndBodyLocalized(t *testing.T) {
	if backupEmailSubject("es", true) != "Copia de seguridad de leotime completada" {
		t.Fatal("unexpected spanish success subject")
	}
	if backupEmailSubject("en", false) != "leotime backup failed" {
		t.Fatal("unexpected english failure subject")
	}

	profile := &store.Profile{
		Name:   "Leo",
		Email:  "admin@example.com",
		Locale: "es",
	}
	body := backupEmailBody(profile, "http://127.0.0.1:8080", "leotime/backups/test.db.gz", "", true, time.Date(2026, 7, 8, 1, 0, 0, 0, time.UTC))
	if !strings.Contains(body, "Hola Leo") || !strings.Contains(body, "test.db.gz") {
		t.Fatalf("unexpected spanish body: %q", body)
	}

	profile.Locale = "en"
	body = backupEmailBody(profile, "http://127.0.0.1:8080", "leotime/backups/test.db.gz", "upload failed", false, time.Date(2026, 7, 8, 1, 0, 0, 0, time.UTC))
	if !strings.Contains(body, "upload failed") {
		t.Fatalf("expected error in english body: %q", body)
	}
}

func TestBackupNotifierRespectsProfileSettings(t *testing.T) {
	ctx := context.Background()
	st, user := newNotifyTestStore(t, ctx)

	outboxStore := outbox.NewStore(st.DB())
	notifier := NewBackupNotifier(st, outboxStore, config.Config{
		PublicBaseURL:   "http://127.0.0.1:8080",
		MailMaxAttempts: 5,
	})

	finishedAt := time.Date(2026, 7, 8, 1, 0, 0, 0, time.UTC)
	notifier.EnqueueBackupResult(ctx, user.ID, "leotime/backups/test.db.gz", "", true, finishedAt)

	pending, err := outboxStore.ListDuePending(ctx, 10, finishedAt)
	if err != nil {
		t.Fatalf("list pending: %v", err)
	}
	if len(pending) != 0 {
		t.Fatalf("expected no success email by default, got %d", len(pending))
	}

	profile, err := st.ProfileByUserID(ctx, user.ID)
	if err != nil {
		t.Fatalf("load profile: %v", err)
	}
	_, err = st.UpdateProfile(ctx, user.ID, store.ProfileUpdateInput{
		Name:                     profile.Name,
		Email:                    profile.Email,
		Locale:                   profile.Locale,
		LayoutMode:               profile.LayoutMode,
		TaskProjectRequired:      profile.Settings.TaskProjectRequired,
		DefaultCurrency:          profile.Settings.DefaultCurrency,
		Timezone:                 profile.Settings.Timezone,
		ThemeMode:                profile.Settings.ThemeMode,
		TimerStillRunningEnabled: profile.Settings.TimerStillRunningEnabled,
		TimerStillRunningHours:   profile.Settings.TimerStillRunningHours,
		BackupEmailOnSuccess:     true,
		BackupEmailOnFailure:     false,
		RestoreEmailOnSuccess:    false,
		RestoreEmailOnFailure:    true,
	})
	if err != nil {
		t.Fatalf("update profile: %v", err)
	}

	notifier.EnqueueBackupResult(ctx, user.ID, "leotime/backups/test.db.gz", "", true, finishedAt)
	pending, err = outboxStore.ListDuePending(ctx, 10, finishedAt)
	if err != nil {
		t.Fatalf("list pending after enabling success: %v", err)
	}
	if len(pending) != 1 {
		t.Fatalf("expected 1 pending success email, got %d", len(pending))
	}
	if pending[0].Kind != outbox.KindBackupSuccess {
		t.Fatalf("expected backup_success kind, got %q", pending[0].Kind)
	}

	notifier.EnqueueRestoreResult(ctx, user.ID, "leotime/backups/test.db.gz", "restore failed", false, finishedAt)
	pending, err = outboxStore.ListDuePending(ctx, 10, finishedAt)
	if err != nil {
		t.Fatalf("list pending after restore failure: %v", err)
	}
	if len(pending) != 2 {
		t.Fatalf("expected 2 pending emails, got %d", len(pending))
	}
}
