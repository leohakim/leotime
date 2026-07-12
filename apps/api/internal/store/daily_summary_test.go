package store

import (
	"context"
	"strings"
	"testing"
)

func TestBuildDailySummarySpanishProse(t *testing.T) {
	ctx := context.Background()
	st, user := newTaskTestStore(t, ctx)

	client, err := st.CreateClient(ctx, user.ID, ClientInput{
		Name:            "RTVE",
		DefaultCurrency: "EUR",
	})
	if err != nil {
		t.Fatalf("create client: %v", err)
	}

	project, err := st.CreateProject(ctx, user.ID, ProjectInput{
		ClientID: client.ID,
		Name:     "Participa",
		Color:    "#2563eb",
	})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	_, err = st.CreateTimeEntry(ctx, user.ID, TimeEntryInput{
		ClientID:    client.ID,
		ProjectID:   project.ID,
		Description: "corrección de rutas API para el despliegue nuevo",
		StartedAt:   "2026-07-12T08:30:00Z",
		EndedAt:     "2026-07-12T10:00:00Z",
		Billable:    true,
	})
	if err != nil {
		t.Fatalf("create morning entry: %v", err)
	}

	cfgProject, err := st.CreateProject(ctx, user.ID, ProjectInput{
		Name:  "Colegio de Farmaceuticos",
		Color: "#0f7a5b",
	})
	if err != nil {
		t.Fatalf("create cfg project: %v", err)
	}

	_, err = st.CreateTimeEntry(ctx, user.ID, TimeEntryInput{
		ProjectID:   cfgProject.ID,
		Description: "endpoints de membresía y mensaje de difusión",
		StartedAt:   "2026-07-12T15:00:00Z",
		EndedAt:     "2026-07-12T17:00:00Z",
		Billable:    true,
	})
	if err != nil {
		t.Fatalf("create afternoon entry: %v", err)
	}

	summary, err := st.BuildDailySummary(ctx, user.ID, DailySummaryOptions{
		Date:           "2026-07-12",
		Timezone:       "Europe/Madrid",
		Locale:         "es",
		IncludeClient:  true,
		IncludeProject: true,
		IncludeClosing: true,
	})
	if err != nil {
		t.Fatalf("build daily summary: %v", err)
	}

	if summary.EntryCount != 2 || summary.TotalSeconds != 12600 {
		t.Fatalf("unexpected totals: %+v", summary)
	}

	text := summary.Text
	if !strings.Contains(text, "12/7:") {
		t.Fatalf("expected Spanish date header, got:\n%s", text)
	}
	if !strings.Contains(text, "Resumen de hoy:") {
		t.Fatalf("expected header, got:\n%s", text)
	}
	if !strings.Contains(text, "Por la mañana avancé con RTVE — Participa:") {
		t.Fatalf("expected morning RTVE sentence, got:\n%s", text)
	}
	if !strings.Contains(text, "corrección de rutas API") {
		t.Fatalf("expected morning activity, got:\n%s", text)
	}
	if !strings.Contains(text, "Por la tarde avancé con Colegio de Farmaceuticos:") {
		t.Fatalf("expected afternoon CFG sentence, got:\n%s", text)
	}
	if !strings.HasSuffix(strings.TrimSpace(text), "Hasta mañana team!") {
		t.Fatalf("expected closing line, got:\n%s", text)
	}
}

func TestBuildDailySummaryRespectsClientAndProjectToggles(t *testing.T) {
	ctx := context.Background()
	st, user := newTaskTestStore(t, ctx)

	client, err := st.CreateClient(ctx, user.ID, ClientInput{
		Name:            "OUT",
		DefaultCurrency: "EUR",
	})
	if err != nil {
		t.Fatalf("create client: %v", err)
	}

	project, err := st.CreateProject(ctx, user.ID, ProjectInput{
		ClientID: client.ID,
		Name:     "Generador de landings",
		Color:    "#2563eb",
	})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	_, err = st.CreateTimeEntry(ctx, user.ID, TimeEntryInput{
		ClientID:    client.ID,
		ProjectID:   project.ID,
		Description: "pixel de conversión",
		StartedAt:   "2026-07-12T09:00:00Z",
		EndedAt:     "2026-07-12T10:00:00Z",
	})
	if err != nil {
		t.Fatalf("create entry: %v", err)
	}

	clientOnly, err := st.BuildDailySummary(ctx, user.ID, DailySummaryOptions{
		Date:           "2026-07-12",
		Timezone:       "UTC",
		Locale:         "es",
		IncludeClient:  true,
		IncludeProject: false,
	})
	if err != nil {
		t.Fatalf("client-only summary: %v", err)
	}
	if !strings.Contains(clientOnly.Text, "avancé con OUT:") {
		t.Fatalf("expected client-only context, got:\n%s", clientOnly.Text)
	}
	if strings.Contains(clientOnly.Text, "Generador de landings") {
		t.Fatalf("did not expect project in client-only summary:\n%s", clientOnly.Text)
	}

	projectOnly, err := st.BuildDailySummary(ctx, user.ID, DailySummaryOptions{
		Date:           "2026-07-12",
		Timezone:       "UTC",
		Locale:         "es",
		IncludeClient:  false,
		IncludeProject: true,
	})
	if err != nil {
		t.Fatalf("project-only summary: %v", err)
	}
	if !strings.Contains(projectOnly.Text, "avancé con Generador de landings:") {
		t.Fatalf("expected project-only context, got:\n%s", projectOnly.Text)
	}
	if strings.Contains(projectOnly.Text, "OUT") {
		t.Fatalf("did not expect client in project-only summary:\n%s", projectOnly.Text)
	}
}

func TestBuildDailySummaryEmptyDay(t *testing.T) {
	ctx := context.Background()
	st, user := newTaskTestStore(t, ctx)

	summary, err := st.BuildDailySummary(ctx, user.ID, DailySummaryOptions{
		Date:           "2026-07-12",
		Timezone:       "UTC",
		Locale:         "es",
		IncludeClosing: false,
	})
	if err != nil {
		t.Fatalf("build empty summary: %v", err)
	}
	if summary.EntryCount != 0 {
		t.Fatalf("expected zero entries, got %+v", summary)
	}
	if !strings.Contains(summary.Text, "Sin entradas registradas hoy.") {
		t.Fatalf("expected empty body, got:\n%s", summary.Text)
	}
	if strings.Contains(summary.Text, "Hasta mañana team!") {
		t.Fatalf("did not expect closing line, got:\n%s", summary.Text)
	}
}
