package store

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestTimeEntryLifecycle(t *testing.T) {
	ctx := context.Background()
	st, user := newTimeEntryTestStore(t, ctx)

	client, err := st.CreateClient(ctx, user.ID, ClientInput{Name: "Acme", DefaultCurrency: "EUR"})
	if err != nil {
		t.Fatalf("create client: %v", err)
	}
	project, err := st.CreateProject(ctx, user.ID, ProjectInput{ClientID: client.ID, Name: "Portal", Color: "#2563eb"})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	task, err := st.CreateTask(ctx, user.ID, TaskInput{ProjectID: project.ID, Name: "API", Billable: true})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	tag, err := st.CreateTag(ctx, user.ID, TagInput{Name: "Deep Work", Color: "#64748b"})
	if err != nil {
		t.Fatalf("create tag: %v", err)
	}

	startedAt := time.Date(2026, 6, 29, 8, 4, 30, 0, time.UTC)
	endedAt := startedAt.Add(2*time.Hour + 51*time.Minute)

	entry, err := st.CreateTimeEntry(ctx, user.ID, TimeEntryInput{
		ClientID:    client.ID,
		ProjectID:   project.ID,
		TaskID:      task.ID,
		TagIDs:      []string{tag.ID},
		Description: "Refactor API",
		StartedAt:   startedAt.Format(time.RFC3339Nano),
		EndedAt:     endedAt.Format(time.RFC3339Nano),
		Billable:    true,
	})
	if err != nil {
		t.Fatalf("create time entry: %v", err)
	}
	if entry.DurationSeconds != 10260 || entry.ProjectName != "Portal" || len(entry.Tags) != 1 {
		t.Fatalf("unexpected created entry: %+v", entry)
	}

	entries, err := st.ListTimeEntries(ctx, user.ID, TimeEntryListOptions{})
	if err != nil {
		t.Fatalf("list time entries: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected one entry, got %d", len(entries))
	}

	updatedEnd := startedAt.Add(3 * time.Hour)
	updated, err := st.UpdateTimeEntry(ctx, user.ID, entry.ID, TimeEntryInput{
		Description: "Refactor API updated",
		StartedAt:   startedAt.Format(time.RFC3339Nano),
		EndedAt:     updatedEnd.Format(time.RFC3339Nano),
		Billable:    false,
	})
	if err != nil {
		t.Fatalf("update time entry: %v", err)
	}
	if updated.Description != "Refactor API updated" || updated.Billable || updated.DurationSeconds != 10800 {
		t.Fatalf("unexpected updated entry: %+v", updated)
	}

	if err := st.DeleteTimeEntry(ctx, user.ID, entry.ID); err != nil {
		t.Fatalf("delete time entry: %v", err)
	}

	remaining, err := st.ListTimeEntries(ctx, user.ID, TimeEntryListOptions{})
	if err != nil {
		t.Fatalf("list after delete: %v", err)
	}
	if len(remaining) != 0 {
		t.Fatalf("expected no entries, got %d", len(remaining))
	}
}

func TestCreateTimeEntrySetsOverlapWarningWithoutBlocking(t *testing.T) {
	ctx := context.Background()
	st, user := newTimeEntryTestStore(t, ctx)

	start := time.Date(2026, 6, 29, 9, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)

	first, err := st.CreateTimeEntry(ctx, user.ID, TimeEntryInput{
		Description: "First",
		StartedAt:   start.Format(time.RFC3339Nano),
		EndedAt:     end.Format(time.RFC3339Nano),
		Billable:    true,
	})
	if err != nil {
		t.Fatalf("create first entry: %v", err)
	}
	if first.OverlapWarning {
		t.Fatal("expected first entry without overlap warning")
	}

	second, err := st.CreateTimeEntry(ctx, user.ID, TimeEntryInput{
		Description: "Second",
		StartedAt:   start.Add(30 * time.Minute).Format(time.RFC3339Nano),
		EndedAt:     end.Add(30 * time.Minute).Format(time.RFC3339Nano),
		Billable:    true,
	})
	if err != nil {
		t.Fatalf("create overlapping entry: %v", err)
	}
	if !second.OverlapWarning {
		t.Fatal("expected overlap warning on second entry")
	}
}

func TestCreateTimeEntryInfersClientFromProject(t *testing.T) {
	ctx := context.Background()
	st, user := newTimeEntryTestStore(t, ctx)

	client, err := st.CreateClient(ctx, user.ID, ClientInput{Name: "Osoigo", DefaultCurrency: "EUR"})
	if err != nil {
		t.Fatalf("create client: %v", err)
	}
	project, err := st.CreateProject(ctx, user.ID, ProjectInput{ClientID: client.ID, Name: "RTVE", Color: "#2563eb"})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	startedAt := time.Date(2026, 7, 1, 9, 0, 0, 0, time.UTC)
	entry, err := st.CreateTimeEntry(ctx, user.ID, TimeEntryInput{
		ProjectID:   project.ID,
		Description: "Broadcast work",
		StartedAt:   startedAt.Format(time.RFC3339Nano),
		EndedAt:     startedAt.Add(2 * time.Hour).Format(time.RFC3339Nano),
		Billable:    true,
	})
	if err != nil {
		t.Fatalf("create time entry: %v", err)
	}
	if entry.ClientID != client.ID || entry.ClientName != "Osoigo" {
		t.Fatalf("expected client inferred from project, got %+v", entry)
	}
}

func TestCreateTimeEntryValidatesInput(t *testing.T) {
	ctx := context.Background()
	st, user := newTimeEntryTestStore(t, ctx)

	start := time.Date(2026, 6, 29, 9, 0, 0, 0, time.UTC)

	tests := []struct {
		name  string
		input TimeEntryInput
	}{
		{
			name: "missing end",
			input: TimeEntryInput{
				StartedAt: start.Format(time.RFC3339Nano),
			},
		},
		{
			name: "end before start",
			input: TimeEntryInput{
				StartedAt: start.Format(time.RFC3339Nano),
				EndedAt:   start.Add(-time.Hour).Format(time.RFC3339Nano),
			},
		},
		{
			name: "shorter than one minute",
			input: TimeEntryInput{
				StartedAt: start.Format(time.RFC3339Nano),
				EndedAt:   start.Add(30 * time.Second).Format(time.RFC3339Nano),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := st.CreateTimeEntry(ctx, user.ID, test.input); !errors.Is(err, ErrInvalidTimeEntryInput) {
				t.Fatalf("expected invalid input, got %v", err)
			}
		})
	}
}

func newTimeEntryTestStore(t *testing.T, ctx context.Context) (*Store, *User) {
	t.Helper()
	return newTagTestStore(t, ctx)
}
