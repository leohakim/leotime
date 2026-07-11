package store

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func seedSyntheticFinishedEntries(t *testing.T, ctx context.Context, st *Store, userID, clientID string, count int, billable bool) {
	t.Helper()

	now := nowString()
	base := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	billableInt := 0
	if billable {
		billableInt = 1
	}

	tx, err := st.DB().BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	defer tx.Rollback()

	for i := 0; i < count; i++ {
		entryID, err := newID("ten")
		if err != nil {
			t.Fatalf("new entry id: %v", err)
		}
		started := base.Add(time.Duration(i) * time.Minute)
		ended := started.Add(time.Minute)
		_, err = tx.ExecContext(ctx, `
			INSERT INTO time_entries (
				id, user_id, client_id, description, started_at, ended_at,
				duration_seconds, billable, overlap_warning, source, sync_state, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, 60, ?, 0, 'manual', 'synced', ?, ?)
		`, entryID, userID, clientID, fmt.Sprintf("Synthetic %d", i),
			formatTime(started), formatTime(ended), billableInt, now, now)
		if err != nil {
			t.Fatalf("insert entry %d: %v", i, err)
		}
	}

	if err := tx.Commit(); err != nil {
		t.Fatalf("commit synthetic entries: %v", err)
	}
}
