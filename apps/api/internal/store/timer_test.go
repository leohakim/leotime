package store

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestTimerLifecycle(t *testing.T) {
	ctx := context.Background()
	st, user := newTimeEntryTestStore(t, ctx)

	project, err := st.CreateProject(ctx, user.ID, ProjectInput{Name: "Portal", Color: "#2563eb"})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	task, err := st.CreateTask(ctx, user.ID, TaskInput{ProjectID: project.ID, Name: "API", Billable: true})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}

	timer, err := st.StartTimer(ctx, user.ID, TimerStartInput{
		ProjectID:   project.ID,
		TaskID:      task.ID,
		Description: "Live coding",
		Billable:    true,
	})
	if err != nil {
		t.Fatalf("start timer: %v", err)
	}
	if timer.Source != "timer" || timer.EndedAt != "" || timer.ProjectName != "Portal" {
		t.Fatalf("unexpected started timer: %+v", timer)
	}

	openTimers, err := st.ListOpenTimers(ctx, user.ID)
	if err != nil {
		t.Fatalf("list open timers: %v", err)
	}
	if len(openTimers) != 1 {
		t.Fatalf("expected one open timer, got %d", len(openTimers))
	}

	startedAt := time.Now().UTC().Add(-2 * time.Hour).Truncate(time.Minute)
	if _, err := st.db.ExecContext(ctx, "UPDATE time_entries SET started_at = ? WHERE id = ?", formatTime(startedAt), timer.ID); err != nil {
		t.Fatalf("backdate timer start: %v", err)
	}

	stopped, err := st.StopTimer(ctx, user.ID, timer.ID)
	if err != nil {
		t.Fatalf("stop timer: %v", err)
	}
	if stopped.EndedAt == "" || stopped.DurationSeconds < 60 || stopped.Source != "timer" {
		t.Fatalf("unexpected stopped timer: %+v", stopped)
	}

	remaining, err := st.ListOpenTimers(ctx, user.ID)
	if err != nil {
		t.Fatalf("list open timers after stop: %v", err)
	}
	if len(remaining) != 0 {
		t.Fatalf("expected no open timers, got %d", len(remaining))
	}

	entries, err := st.ListTimeEntries(ctx, user.ID, TimeEntryListOptions{})
	if err != nil {
		t.Fatalf("list finalized entries: %v", err)
	}
	if len(entries) != 1 || entries[0].Description != "Live coding" {
		t.Fatalf("unexpected finalized entries: %+v", entries)
	}
}

func TestStartTimerAllowsMultipleOpenTimers(t *testing.T) {
	ctx := context.Background()
	st, user := newTimeEntryTestStore(t, ctx)

	first, err := st.StartTimer(ctx, user.ID, TimerStartInput{Description: "First"})
	if err != nil {
		t.Fatalf("start first timer: %v", err)
	}
	second, err := st.StartTimer(ctx, user.ID, TimerStartInput{Description: "Second"})
	if err != nil {
		t.Fatalf("start second timer: %v", err)
	}
	if first.ID == second.ID {
		t.Fatalf("expected distinct timer ids")
	}

	openTimers, err := st.ListOpenTimers(ctx, user.ID)
	if err != nil {
		t.Fatalf("list open timers: %v", err)
	}
	if len(openTimers) != 2 {
		t.Fatalf("expected two open timers, got %d", len(openTimers))
	}
}

func TestStopTimerSetsOverlapWarningWithoutBlocking(t *testing.T) {
	ctx := context.Background()
	st, user := newTimeEntryTestStore(t, ctx)

	start := time.Date(2026, 6, 29, 9, 0, 0, 0, time.UTC)
	end := start.Add(2 * time.Hour)

	if _, err := st.CreateTimeEntry(ctx, user.ID, TimeEntryInput{
		Description: "Existing",
		StartedAt:   start.Format(time.RFC3339Nano),
		EndedAt:     end.Format(time.RFC3339Nano),
		Billable:    true,
	}); err != nil {
		t.Fatalf("create existing entry: %v", err)
	}

	timer, err := st.StartTimer(ctx, user.ID, TimerStartInput{Description: "Overlap timer"})
	if err != nil {
		t.Fatalf("start timer: %v", err)
	}

	if _, err := st.db.ExecContext(ctx, "UPDATE time_entries SET started_at = ? WHERE id = ?", formatTime(start.Add(30*time.Minute)), timer.ID); err != nil {
		t.Fatalf("align timer start: %v", err)
	}

	stopped, err := st.StopTimer(ctx, user.ID, timer.ID)
	if err != nil {
		t.Fatalf("stop timer: %v", err)
	}
	if !stopped.OverlapWarning {
		t.Fatalf("expected overlap warning on stopped timer")
	}
}

func TestDiscardOpenTimer(t *testing.T) {
	ctx := context.Background()
	st, user := newTimeEntryTestStore(t, ctx)

	timer, err := st.StartTimer(ctx, user.ID, TimerStartInput{Description: "Discard me"})
	if err != nil {
		t.Fatalf("start timer: %v", err)
	}

	if err := st.DiscardTimer(ctx, user.ID, timer.ID); err != nil {
		t.Fatalf("discard timer: %v", err)
	}

	openTimers, err := st.ListOpenTimers(ctx, user.ID)
	if err != nil {
		t.Fatalf("list open timers: %v", err)
	}
	if len(openTimers) != 0 {
		t.Fatalf("expected no open timers after discard")
	}
}

func TestUpdateOpenTimerStartedAt(t *testing.T) {
	ctx := context.Background()
	st, user := newTimeEntryTestStore(t, ctx)

	timer, err := st.StartTimer(ctx, user.ID, TimerStartInput{Description: "Adjust start"})
	if err != nil {
		t.Fatalf("start timer: %v", err)
	}

	updatedStart := time.Now().UTC().Add(-45 * time.Minute).Truncate(time.Minute)
	updated, err := st.UpdateOpenTimer(ctx, user.ID, timer.ID, TimerStartInput{
		Description: "Adjust start",
		StartedAt:   updatedStart.Format(time.RFC3339Nano),
		Billable:    true,
	})
	if err != nil {
		t.Fatalf("update timer startedAt: %v", err)
	}

	parsed, err := parseRFC3339(updated.StartedAt)
	if err != nil {
		t.Fatalf("parse updated startedAt: %v", err)
	}
	if !parsed.Equal(updatedStart) {
		t.Fatalf("expected startedAt %s, got %s", formatTime(updatedStart), updated.StartedAt)
	}
}

func TestUpdateOpenTimerRejectsFutureStartedAt(t *testing.T) {
	ctx := context.Background()
	st, user := newTimeEntryTestStore(t, ctx)

	timer, err := st.StartTimer(ctx, user.ID, TimerStartInput{Description: "Future start"})
	if err != nil {
		t.Fatalf("start timer: %v", err)
	}

	future := time.Now().UTC().Add(2 * time.Hour).Truncate(time.Minute)
	if _, err := st.UpdateOpenTimer(ctx, user.ID, timer.ID, TimerStartInput{
		Description: "Future start",
		StartedAt:   future.Format(time.RFC3339Nano),
		Billable:    true,
	}); !errors.Is(err, ErrInvalidTimeEntryInput) {
		t.Fatalf("expected ErrInvalidTimeEntryInput, got %v", err)
	}
}

func TestStopTimerNotFound(t *testing.T) {
	ctx := context.Background()
	st, user := newTimeEntryTestStore(t, ctx)

	if _, err := st.StopTimer(ctx, user.ID, "ten_missing"); !errors.Is(err, ErrTimerNotFound) {
		t.Fatalf("expected ErrTimerNotFound, got %v", err)
	}
}
