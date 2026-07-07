package outbox

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/leotime/leotime/apps/api/internal/db"
	"github.com/leotime/leotime/apps/api/internal/mail"
	"github.com/leotime/leotime/apps/api/internal/store"
)

type fakeSender struct {
	mu       sync.Mutex
	calls    int
	failures int
	err      error
}

func (f *fakeSender) Send(context.Context, mail.Message) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	if f.err != nil && f.calls <= f.failures {
		return f.err
	}
	return nil
}

func setupOutboxTest(t *testing.T) (*Store, *store.Store, string, string) {
	t.Helper()

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

	return NewStore(database), st, user.ID, user.Email
}

func createOpenTimer(t *testing.T, st *store.Store, userID string) string {
	t.Helper()

	entry, err := st.StartTimer(context.Background(), userID, store.TimerStartInput{
		Description: "Long running task",
		Billable:    true,
	})
	if err != nil {
		t.Fatalf("start timer: %v", err)
	}
	return entry.ID
}

func enqueueTestEmail(t *testing.T, outboxStore *Store, userID string, timeEntryID string, email string, now time.Time) *Email {
	t.Helper()

	entry, err := outboxStore.Enqueue(context.Background(), EnqueueInput{
		UserID:      userID,
		TimeEntryID: timeEntryID,
		Kind:        KindTimerStillRunning,
		ToAddress:   email,
		Subject:     "Timer still running",
		BodyText:    "Please stop your timer.",
		MaxAttempts: 5,
	}, now)
	if err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	return entry
}

func TestEnqueueAndMarkSent(t *testing.T) {
	ctx := context.Background()
	outboxStore, st, userID, userEmail := setupOutboxTest(t)
	timerID := createOpenTimer(t, st, userID)
	now := time.Date(2026, 7, 7, 9, 0, 0, 0, time.UTC)

	entry := enqueueTestEmail(t, outboxStore, userID, timerID, userEmail, now)
	if entry.Status != StatusPending {
		t.Fatalf("expected pending status, got %q", entry.Status)
	}

	sender := &fakeSender{}
	processor := NewProcessor(outboxStore, sender, ProcessorOptions{
		RetryPolicy: DefaultRetryPolicy(time.Minute, 6*time.Hour),
		Now:         func() time.Time { return now },
	})

	result, err := processor.ProcessOnce(ctx)
	if err != nil {
		t.Fatalf("process once: %v", err)
	}
	if result.Sent != 1 || result.Retried != 0 || result.Dead != 0 {
		t.Fatalf("unexpected result: %+v", result)
	}

	updated, err := outboxStore.ByID(ctx, entry.ID)
	if err != nil {
		t.Fatalf("load outbox entry: %v", err)
	}
	if updated.Status != StatusSent {
		t.Fatalf("expected sent status, got %q", updated.Status)
	}
	if updated.SentAt == "" {
		t.Fatal("expected sent_at to be set")
	}
}

func TestEnqueueDuplicateByTimerKind(t *testing.T) {
	ctx := context.Background()
	outboxStore, st, userID, userEmail := setupOutboxTest(t)
	timerID := createOpenTimer(t, st, userID)
	now := time.Date(2026, 7, 7, 9, 0, 0, 0, time.UTC)

	enqueueTestEmail(t, outboxStore, userID, timerID, userEmail, now)
	if _, err := outboxStore.Enqueue(ctx, EnqueueInput{
		UserID:      userID,
		TimeEntryID: timerID,
		Kind:        KindTimerStillRunning,
		ToAddress:   userEmail,
		Subject:     "Timer still running",
		BodyText:    "Duplicate.",
		MaxAttempts: 5,
	}, now); !errors.Is(err, ErrDuplicate) {
		t.Fatalf("expected duplicate error, got %v", err)
	}
}

func TestProcessorRetriesTransientFailure(t *testing.T) {
	ctx := context.Background()
	outboxStore, st, userID, userEmail := setupOutboxTest(t)
	timerID := createOpenTimer(t, st, userID)
	now := time.Date(2026, 7, 7, 9, 0, 0, 0, time.UTC)

	entry := enqueueTestEmail(t, outboxStore, userID, timerID, userEmail, now)
	sender := &fakeSender{
		failures: 1,
		err:      mail.Transient(errors.New("smtp timeout")),
	}
	processor := NewProcessor(outboxStore, sender, ProcessorOptions{
		RetryPolicy: DefaultRetryPolicy(time.Minute, 6*time.Hour),
		RNG:         nil,
		Now:         func() time.Time { return now },
	})

	result, err := processor.ProcessOnce(ctx)
	if err != nil {
		t.Fatalf("process once: %v", err)
	}
	if result.Sent != 0 || result.Retried != 1 || result.Dead != 0 {
		t.Fatalf("unexpected result: %+v", result)
	}

	updated, err := outboxStore.ByID(ctx, entry.ID)
	if err != nil {
		t.Fatalf("load outbox entry: %v", err)
	}
	if updated.Attempts != 1 {
		t.Fatalf("expected attempts=1, got %d", updated.Attempts)
	}
	if updated.Status != StatusPending {
		t.Fatalf("expected pending status, got %q", updated.Status)
	}
	if updated.NextRetryAt <= now.Format(time.RFC3339Nano) {
		t.Fatalf("expected next retry in the future, got %q", updated.NextRetryAt)
	}
}

func TestProcessorMarksPermanentFailureDead(t *testing.T) {
	ctx := context.Background()
	outboxStore, st, userID, userEmail := setupOutboxTest(t)
	timerID := createOpenTimer(t, st, userID)
	now := time.Date(2026, 7, 7, 9, 0, 0, 0, time.UTC)

	entry := enqueueTestEmail(t, outboxStore, userID, timerID, userEmail, now)
	sender := &fakeSender{
		failures: 1,
		err:      mail.Permanent(errors.New("550 recipient rejected")),
	}
	processor := NewProcessor(outboxStore, sender, ProcessorOptions{
		RetryPolicy: DefaultRetryPolicy(time.Minute, 6*time.Hour),
		Now:         func() time.Time { return now },
	})

	result, err := processor.ProcessOnce(ctx)
	if err != nil {
		t.Fatalf("process once: %v", err)
	}
	if result.Dead != 1 {
		t.Fatalf("expected dead=1, got %+v", result)
	}

	updated, err := outboxStore.ByID(ctx, entry.ID)
	if err != nil {
		t.Fatalf("load outbox entry: %v", err)
	}
	if updated.Status != StatusDead {
		t.Fatalf("expected dead status, got %q", updated.Status)
	}
}

func TestProcessorMarksMaxAttemptsDead(t *testing.T) {
	ctx := context.Background()
	outboxStore, st, userID, userEmail := setupOutboxTest(t)
	timerID := createOpenTimer(t, st, userID)
	now := time.Date(2026, 7, 7, 9, 0, 0, 0, time.UTC)

	entry, err := outboxStore.Enqueue(ctx, EnqueueInput{
		UserID:      userID,
		TimeEntryID: timerID,
		Kind:        KindTimerStillRunning,
		ToAddress:   userEmail,
		Subject:     "Timer still running",
		BodyText:    "Please stop your timer.",
		MaxAttempts: 2,
	}, now)
	if err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	if _, err := outboxStore.db.ExecContext(ctx, `
		UPDATE email_outbox
		SET attempts = 1, next_retry_at = ?
		WHERE id = ?
	`, now.Format(time.RFC3339Nano), entry.ID); err != nil {
		t.Fatalf("seed attempts: %v", err)
	}

	sender := &fakeSender{
		failures: 2,
		err:      mail.Transient(errors.New("connection reset")),
	}
	processor := NewProcessor(outboxStore, sender, ProcessorOptions{
		RetryPolicy: DefaultRetryPolicy(time.Minute, 6*time.Hour),
		Now:         func() time.Time { return now },
	})

	result, err := processor.ProcessOnce(ctx)
	if err != nil {
		t.Fatalf("process once: %v", err)
	}
	if result.Dead != 1 {
		t.Fatalf("expected dead=1, got %+v", result)
	}

	updated, err := outboxStore.ByID(ctx, entry.ID)
	if err != nil {
		t.Fatalf("load outbox entry: %v", err)
	}
	if updated.Status != StatusDead {
		t.Fatalf("expected dead status, got %q", updated.Status)
	}
	if updated.Attempts != 2 {
		t.Fatalf("expected attempts=2 after max retries, got %d", updated.Attempts)
	}
}

func TestListDuePendingRespectsNextRetryAt(t *testing.T) {
	ctx := context.Background()
	outboxStore, st, userID, userEmail := setupOutboxTest(t)
	timerID := createOpenTimer(t, st, userID)
	now := time.Date(2026, 7, 7, 9, 0, 0, 0, time.UTC)
	future := now.Add(10 * time.Minute)

	entry := enqueueTestEmail(t, outboxStore, userID, timerID, userEmail, future)
	if _, err := outboxStore.db.ExecContext(ctx, `
		UPDATE email_outbox
		SET next_retry_at = ?
		WHERE id = ?
	`, future.Format(time.RFC3339Nano), entry.ID); err != nil {
		t.Fatalf("update next retry: %v", err)
	}

	due, err := outboxStore.ListDuePending(ctx, 10, now)
	if err != nil {
		t.Fatalf("list due pending: %v", err)
	}
	if len(due) != 0 {
		t.Fatalf("expected no due emails, got %d", len(due))
	}
}
