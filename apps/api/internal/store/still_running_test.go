package store

import (
	"context"
	"database/sql"
	"testing"
	"time"
)

func TestListStillRunningNotificationCandidates(t *testing.T) {
	ctx := context.Background()
	st, user := newTimeEntryTestStore(t, ctx)

	timer, err := st.StartTimer(ctx, user.ID, TimerStartInput{
		Description: "Deep work",
		Billable:    true,
	})
	if err != nil {
		t.Fatalf("start timer: %v", err)
	}

	now := time.Date(2026, 7, 7, 18, 0, 0, 0, time.UTC)
	startedAt := now.Add(-9 * time.Hour)
	if _, err := st.db.ExecContext(ctx, "UPDATE time_entries SET started_at = ? WHERE id = ?", formatTime(startedAt), timer.ID); err != nil {
		t.Fatalf("backdate timer: %v", err)
	}

	candidates, err := st.ListStillRunningNotificationCandidates(ctx, now, 10)
	if err != nil {
		t.Fatalf("list candidates: %v", err)
	}
	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(candidates))
	}
	if candidates[0].TimeEntryID != timer.ID {
		t.Fatalf("unexpected candidate timer id %q", candidates[0].TimeEntryID)
	}
	if candidates[0].ThresholdHours != 8 {
		t.Fatalf("expected default threshold 8, got %d", candidates[0].ThresholdHours)
	}
}

func TestListStillRunningNotificationCandidatesSkipsShortTimers(t *testing.T) {
	ctx := context.Background()
	st, user := newTimeEntryTestStore(t, ctx)

	timer, err := st.StartTimer(ctx, user.ID, TimerStartInput{Description: "Short"})
	if err != nil {
		t.Fatalf("start timer: %v", err)
	}

	now := time.Date(2026, 7, 7, 18, 0, 0, 0, time.UTC)
	startedAt := now.Add(-7 * time.Hour)
	if _, err := st.db.ExecContext(ctx, "UPDATE time_entries SET started_at = ? WHERE id = ?", formatTime(startedAt), timer.ID); err != nil {
		t.Fatalf("backdate timer: %v", err)
	}

	candidates, err := st.ListStillRunningNotificationCandidates(ctx, now, 10)
	if err != nil {
		t.Fatalf("list candidates: %v", err)
	}
	if len(candidates) != 0 {
		t.Fatalf("expected no candidates, got %d", len(candidates))
	}
}

func TestListStillRunningNotificationCandidatesRespectsDisabledSetting(t *testing.T) {
	ctx := context.Background()
	st, user := newTimeEntryTestStore(t, ctx)

	timer, err := st.StartTimer(ctx, user.ID, TimerStartInput{Description: "Disabled"})
	if err != nil {
		t.Fatalf("start timer: %v", err)
	}

	now := time.Date(2026, 7, 7, 18, 0, 0, 0, time.UTC)
	startedAt := now.Add(-9 * time.Hour)
	if _, err := st.db.ExecContext(ctx, "UPDATE time_entries SET started_at = ? WHERE id = ?", formatTime(startedAt), timer.ID); err != nil {
		t.Fatalf("backdate timer: %v", err)
	}
	if _, err := st.db.ExecContext(ctx, `
		UPDATE app_settings
		SET timer_still_running_enabled = 0, updated_at = ?
		WHERE user_id = ?
	`, formatTime(now), user.ID); err != nil {
		t.Fatalf("disable still running setting: %v", err)
	}

	candidates, err := st.ListStillRunningNotificationCandidates(ctx, now, 10)
	if err != nil {
		t.Fatalf("list candidates: %v", err)
	}
	if len(candidates) != 0 {
		t.Fatalf("expected no candidates when disabled, got %d", len(candidates))
	}
}

func TestMarkStillRunningEmailSent(t *testing.T) {
	ctx := context.Background()
	st, user := newTimeEntryTestStore(t, ctx)

	timer, err := st.StartTimer(ctx, user.ID, TimerStartInput{Description: "Mark sent"})
	if err != nil {
		t.Fatalf("start timer: %v", err)
	}

	sentAt := time.Date(2026, 7, 7, 18, 0, 0, 0, time.UTC)
	if err := st.MarkStillRunningEmailSent(ctx, timer.ID, sentAt); err != nil {
		t.Fatalf("mark still running email sent: %v", err)
	}

	var stored sql.NullString
	if err := st.db.QueryRowContext(ctx, "SELECT still_active_email_sent_at FROM time_entries WHERE id = ?", timer.ID).Scan(&stored); err != nil {
		t.Fatalf("query still_active_email_sent_at: %v", err)
	}
	if !stored.Valid || stored.String == "" {
		t.Fatal("expected still_active_email_sent_at to be set")
	}

	candidates, err := st.ListStillRunningNotificationCandidates(ctx, sentAt.Add(time.Hour), 10)
	if err != nil {
		t.Fatalf("list candidates: %v", err)
	}
	if len(candidates) != 0 {
		t.Fatalf("expected marked timer to be excluded, got %d", len(candidates))
	}
}
