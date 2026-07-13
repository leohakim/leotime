package store

import (
	"context"
	"strings"
	"testing"
)

func TestSplitDailySummaryTopics(t *testing.T) {
	got := splitDailySummaryTopics("Cropper Imagenes + Reunion Nico + Visibilidad de procesos")
	if len(got) != 3 {
		t.Fatalf("expected 3 topics, got %v", got)
	}
	if got[0] != "Cropper Imagenes" || got[1] != "Reunion Nico" || got[2] != "Visibilidad de procesos" {
		t.Fatalf("unexpected topics: %v", got)
	}
}

func TestDailySummaryEntryBulletsSplitsCompoundTaskTitle(t *testing.T) {
	bullets := dailySummaryEntryBullets(TimeEntry{
		TaskName:    "Cropper Imagenes + Reunion Nico + Visibilidad de procesos",
		Description: "ADR cropper y sync con Nico",
	})
	if len(bullets) != 3 {
		t.Fatalf("expected 3 bullets, got %v", bullets)
	}
	if bullets[0] != "Cropper Imagenes" || bullets[1] != "Reunion Nico" || bullets[2] != "Visibilidad de procesos" {
		t.Fatalf("unexpected bullets: %v", bullets)
	}
}

func TestBuildDailySummarySplitsCompoundTaskIntoProjectBullets(t *testing.T) {
	ctx := context.Background()
	st, user := newTaskTestStore(t, ctx)

	client, err := st.CreateClient(ctx, user.ID, ClientInput{Name: "RTVE", DefaultCurrency: "EUR"})
	if err != nil {
		t.Fatalf("create client: %v", err)
	}
	project, err := st.CreateProject(ctx, user.ID, ProjectInput{ClientID: client.ID, Name: "Participa", Color: "#2563eb"})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	task, err := st.CreateTask(ctx, user.ID, TaskInput{
		ProjectID: project.ID,
		Name:      "Cropper Imagenes + Reunion Nico + Visibilidad de procesos",
		Billable:  true,
	})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}

	_, err = st.CreateTimeEntry(ctx, user.ID, TimeEntryInput{
		ClientID:    client.ID,
		ProjectID:   project.ID,
		TaskID:      task.ID,
		Description: "ADR cropper y revisión con Nico",
		StartedAt:   "2026-07-12T09:00:00Z",
		EndedAt:     "2026-07-12T12:00:00Z",
	})
	if err != nil {
		t.Fatalf("create entry: %v", err)
	}

	summary, err := st.BuildDailySummary(ctx, user.ID, DailySummaryOptions{
		Date:           "2026-07-12",
		Timezone:       "UTC",
		Locale:         "es",
		IncludeClient:  true,
		IncludeProject: true,
		IncludeClosing: true,
	})
	if err != nil {
		t.Fatalf("build summary: %v", err)
	}

	text := summary.Text
	if !containsAll(text,
		"- RTVE:",
		"    - Cropper Imagenes",
		"    - Reunion Nico",
		"    - Visibilidad de procesos",
	) {
		t.Fatalf("expected split bullets under RTVE, got:\n%s", text)
	}
	if len(summary.EntryFacts) != 1 || len(summary.EntryFacts[0].Topics) != 3 {
		t.Fatalf("expected entry facts with 3 topics, got %+v", summary.EntryFacts)
	}
}

func containsAll(text string, parts ...string) bool {
	for _, part := range parts {
		if !strings.Contains(text, part) {
			return false
		}
	}
	return true
}
