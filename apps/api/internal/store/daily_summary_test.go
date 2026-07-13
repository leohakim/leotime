package store

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestBuildDailySummarySpanishBulletFormat(t *testing.T) {
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
	if !strings.Contains(text, "- RTVE:\n    - corrección de rutas API para el despliegue nuevo") {
		t.Fatalf("expected RTVE bullet group, got:\n%s", text)
	}
	if !strings.Contains(text, "- Colegio de Farmaceuticos:\n    - endpoints de membresía y mensaje de difusión") {
		t.Fatalf("expected CFG bullet group, got:\n%s", text)
	}
	if strings.Contains(text, "Por la mañana") {
		t.Fatalf("did not expect period prose, got:\n%s", text)
	}
	if !strings.HasSuffix(strings.TrimSpace(text), "Hasta mañana team!") {
		t.Fatalf("expected closing line, got:\n%s", text)
	}
}

func TestBuildDailySummaryPlacesMeetingsByScope(t *testing.T) {
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
	meetProject, err := st.CreateProject(ctx, user.ID, ProjectInput{Name: "Meet Tech", Color: "#111111"})
	if err != nil {
		t.Fatalf("create meet project: %v", err)
	}

	_, err = st.CreateTimeEntry(ctx, user.ID, TimeEntryInput{
		ClientID:    client.ID,
		ProjectID:   project.ID,
		Description: "cerramos la rama de recommendations y la metimos en master",
		StartedAt:   "2026-07-12T09:00:00Z",
		EndedAt:     "2026-07-12T10:00:00Z",
	})
	if err != nil {
		t.Fatalf("create work entry: %v", err)
	}
	_, err = st.CreateTimeEntry(ctx, user.ID, TimeEntryInput{
		ClientID:    client.ID,
		ProjectID:   project.ID,
		Description: "reunión con Miguel Angel para revisar incidencias de webhooks",
		StartedAt:   "2026-07-12T11:00:00Z",
		EndedAt:     "2026-07-12T12:00:00Z",
	})
	if err != nil {
		t.Fatalf("create project meeting entry: %v", err)
	}
	_, err = st.CreateTimeEntry(ctx, user.ID, TimeEntryInput{
		ProjectID:   meetProject.ID,
		Description: "weekly de tech",
		StartedAt:   "2026-07-12T17:00:00Z",
		EndedAt:     "2026-07-12T18:00:00Z",
	})
	if err != nil {
		t.Fatalf("create weekly entry: %v", err)
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
		t.Fatalf("build daily summary: %v", err)
	}

	text := summary.Text
	if !strings.Contains(text, "- RTVE:\n    - cerramos la rama de recommendations y la metimos en master") {
		t.Fatalf("expected RTVE work bullet, got:\n%s", text)
	}
	if !strings.Contains(text, "    - reunión con Miguel Angel para revisar incidencias de webhooks") {
		t.Fatalf("expected project meeting under RTVE, got:\n%s", text)
	}
	if !strings.Contains(text, "- Meet Tech") {
		t.Fatalf("expected standalone weekly meeting at end, got:\n%s", text)
	}
	if strings.Index(text, "- Meet Tech") < strings.Index(text, "reunión con Miguel Angel") {
		t.Fatalf("expected standalone meeting after project bullets, got:\n%s", text)
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
	if !strings.Contains(clientOnly.Text, "- OUT:\n    - pixel de conversión") {
		t.Fatalf("expected client-only bullet group, got:\n%s", clientOnly.Text)
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
	if !strings.Contains(projectOnly.Text, "- Generador de landings:\n    - pixel de conversión") {
		t.Fatalf("expected project-only bullet group, got:\n%s", projectOnly.Text)
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

func TestBuildDailySummaryIncludesManualNote(t *testing.T) {
	ctx := context.Background()
	st, user := newTaskTestStore(t, ctx)

	summary, err := st.BuildDailySummary(ctx, user.ID, DailySummaryOptions{
		Date:           "2026-07-12",
		Timezone:       "UTC",
		Locale:         "es",
		IncludeClosing: true,
		ManualNote:     "Reunión con RTVE sobre el deploy pendiente.",
	})
	if err != nil {
		t.Fatalf("build daily summary: %v", err)
	}

	text := summary.Text
	if !strings.Contains(text, "Reunión con RTVE sobre el deploy pendiente.") {
		t.Fatalf("expected manual note in summary, got:\n%s", text)
	}
	if !strings.HasSuffix(strings.TrimSpace(text), "Hasta mañana team!") {
		t.Fatalf("expected closing after manual note, got:\n%s", text)
	}
}

func TestBuildDailySummaryFiltersByClient(t *testing.T) {
	ctx := context.Background()
	st, user := newTaskTestStore(t, ctx)

	clientA, err := st.CreateClient(ctx, user.ID, ClientInput{Name: "Cliente A", DefaultCurrency: "EUR"})
	if err != nil {
		t.Fatalf("create client A: %v", err)
	}
	clientB, err := st.CreateClient(ctx, user.ID, ClientInput{Name: "Cliente B", DefaultCurrency: "EUR"})
	if err != nil {
		t.Fatalf("create client B: %v", err)
	}
	projectA, err := st.CreateProject(ctx, user.ID, ProjectInput{ClientID: clientA.ID, Name: "Proyecto A", Color: "#2563eb"})
	if err != nil {
		t.Fatalf("create project A: %v", err)
	}
	projectB, err := st.CreateProject(ctx, user.ID, ProjectInput{ClientID: clientB.ID, Name: "Proyecto B", Color: "#0f7a5b"})
	if err != nil {
		t.Fatalf("create project B: %v", err)
	}

	_, err = st.CreateTimeEntry(ctx, user.ID, TimeEntryInput{
		ClientID:    clientA.ID,
		ProjectID:   projectA.ID,
		Description: "trabajo cliente A",
		StartedAt:   "2026-07-12T09:00:00Z",
		EndedAt:     "2026-07-12T10:00:00Z",
	})
	if err != nil {
		t.Fatalf("create entry A: %v", err)
	}
	_, err = st.CreateTimeEntry(ctx, user.ID, TimeEntryInput{
		ClientID:    clientB.ID,
		ProjectID:   projectB.ID,
		Description: "trabajo cliente B",
		StartedAt:   "2026-07-12T11:00:00Z",
		EndedAt:     "2026-07-12T12:00:00Z",
	})
	if err != nil {
		t.Fatalf("create entry B: %v", err)
	}

	summary, err := st.BuildDailySummary(ctx, user.ID, DailySummaryOptions{
		Date:           "2026-07-12",
		Timezone:       "UTC",
		Locale:         "es",
		IncludeClient:  true,
		IncludeProject: true,
		ClientID:       clientA.ID,
	})
	if err != nil {
		t.Fatalf("build scoped summary: %v", err)
	}

	if !strings.Contains(summary.Text, "Cliente A") || !strings.Contains(summary.Text, "trabajo cliente A") {
		t.Fatalf("expected client A activity, got:\n%s", summary.Text)
	}
	if strings.Contains(summary.Text, "Cliente B") || strings.Contains(summary.Text, "trabajo cliente B") {
		t.Fatalf("did not expect client B activity, got:\n%s", summary.Text)
	}
	if summary.EntryCount != 1 {
		t.Fatalf("expected one entry, got %+v", summary)
	}
}

func TestDailySummaryOptionsJSONRoundTrip(t *testing.T) {
	original := DailySummaryOptions{
		Date:           "2026-03-12",
		Timezone:       "Europe/Madrid",
		Locale:         "es",
		IncludeClient:  true,
		IncludeProject: true,
		IncludeClosing: true,
		BillableOnly:   false,
		ManualNote:     "Reunion con Huesca",
		ClientID:       "cli_d8251f3fa8f86df0ec74d16215b29529",
		ProjectID:      "",
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal options: %v", err)
	}

	var decoded DailySummaryOptions
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal options: %v", err)
	}
	if decoded.ClientID != original.ClientID || decoded.Date != original.Date || decoded.ManualNote != original.ManualNote {
		t.Fatalf("unexpected decoded options: %+v", decoded)
	}
}
