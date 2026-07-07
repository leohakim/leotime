package notify

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	"github.com/leotime/leotime/apps/api/internal/config"
	"github.com/leotime/leotime/apps/api/internal/db"
	"github.com/leotime/leotime/apps/api/internal/outbox"
	"github.com/leotime/leotime/apps/api/internal/store"
)

func TestStillRunningSubjectAndBodyLocalized(t *testing.T) {
	candidate := store.StillRunningCandidate{
		UserName:    "Leo",
		Locale:      "es",
		ProjectName: "Portal",
		TaskName:    "API",
		Description: "Refactor",
		StartedAt:   time.Date(2026, 7, 7, 8, 0, 0, 0, time.UTC).Format(time.RFC3339Nano),
	}

	if stillRunningSubject("es") != "Tu cronómetro sigue activo" {
		t.Fatalf("unexpected spanish subject")
	}

	body := stillRunningBody(candidate, "http://127.0.0.1:8080", time.Date(2026, 7, 7, 17, 30, 0, 0, time.UTC))
	if !strings.Contains(body, "Tu cronómetro sigue activo") {
		t.Fatalf("expected spanish body, got %q", body)
	}
	if !strings.Contains(body, "Portal") || !strings.Contains(body, "9h 30m") {
		t.Fatalf("unexpected body content: %q", body)
	}

	if stillRunningSubject("en") != "Your time tracker is still running" {
		t.Fatalf("unexpected english subject")
	}
}

func TestStillRunningNotifierEnqueueAndHandleSent(t *testing.T) {
	ctx := context.Background()
	st, user := newNotifyTestStore(t, ctx)

	timer, err := st.StartTimer(ctx, user.ID, store.TimerStartInput{Description: "Long task"})
	if err != nil {
		t.Fatalf("start timer: %v", err)
	}

	now := time.Date(2026, 7, 7, 18, 0, 0, 0, time.UTC)
	startedAt := now.Add(-9 * time.Hour)
	if _, err := st.DB().ExecContext(ctx, "UPDATE time_entries SET started_at = ? WHERE id = ?", startedAt.UTC().Format(time.RFC3339Nano), timer.ID); err != nil {
		t.Fatalf("backdate timer: %v", err)
	}

	outboxStore := outbox.NewStore(st.DB())
	notifier := NewStillRunningNotifier(st, outboxStore, config.Config{
		PublicBaseURL:   "http://127.0.0.1:8080",
		MailMaxAttempts: 5,
	})
	notifier.now = func() time.Time { return now }

	enqueued, err := notifier.EnqueueDue(ctx)
	if err != nil {
		t.Fatalf("enqueue due: %v", err)
	}
	if enqueued != 1 {
		t.Fatalf("expected 1 enqueued notification, got %d", enqueued)
	}

	pending, err := outboxStore.ListDuePending(ctx, 10, now)
	if err != nil {
		t.Fatalf("list pending: %v", err)
	}
	if len(pending) != 1 {
		t.Fatalf("expected 1 pending outbox email, got %d", len(pending))
	}

	if err := notifier.HandleSent(ctx, pending[0]); err != nil {
		t.Fatalf("handle sent: %v", err)
	}

	var stillActive sql.NullString
	if err := st.DB().QueryRowContext(ctx, "SELECT still_active_email_sent_at FROM time_entries WHERE id = ?", timer.ID).Scan(&stillActive); err != nil {
		t.Fatalf("query still_active_email_sent_at: %v", err)
	}
	if !stillActive.Valid || stillActive.String == "" {
		t.Fatal("expected still_active_email_sent_at to be set")
	}
}

func newNotifyTestStore(t *testing.T, ctx context.Context) (*store.Store, *store.User) {
	t.Helper()

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
	return st, user
}
