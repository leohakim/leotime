package store

import (
	"context"
	"testing"
)

func TestResolveBillableFlagUsesClientOrProjectRateOnCreate(t *testing.T) {
	ctx := context.Background()
	st, user := newTimeEntryTestStore(t, ctx)

	client, err := st.CreateClient(ctx, user.ID, ClientInput{
		Name:                   "Osoigo",
		DefaultCurrency:        "EUR",
		DefaultHourlyRateMinor: 3500,
	})
	if err != nil {
		t.Fatalf("create client: %v", err)
	}
	rate := int64(3500)
	project, err := st.CreateProject(ctx, user.ID, ProjectInput{
		ClientID:               client.ID,
		Name:                   "RTVE",
		Color:                  "#2563eb",
		DefaultHourlyRateMinor: &rate,
	})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	billable, err := st.resolveBillableFlag(ctx, user.ID, client.ID, project.ID, true, true)
	if err != nil {
		t.Fatalf("resolve billable: %v", err)
	}
	if !billable {
		t.Fatal("expected billable true when project has a rate")
	}

	billable, err = st.resolveBillableFlag(ctx, user.ID, client.ID, project.ID, true, false)
	if err != nil {
		t.Fatalf("resolve billable on update: %v", err)
	}
	if !billable {
		t.Fatal("expected update path to preserve requested billable true")
	}

	billable, err = st.resolveBillableFlag(ctx, user.ID, client.ID, project.ID, false, true)
	if err != nil {
		t.Fatalf("resolve billable opt-out: %v", err)
	}
	if billable {
		t.Fatal("expected explicit opt-out to stay non-billable")
	}
}

func TestCreateTimeEntryInfersBillableFromProjectRate(t *testing.T) {
	ctx := context.Background()
	st, user := newTimeEntryTestStore(t, ctx)

	client, err := st.CreateClient(ctx, user.ID, ClientInput{Name: "Osoigo", DefaultCurrency: "EUR", DefaultHourlyRateMinor: 3500})
	if err != nil {
		t.Fatalf("create client: %v", err)
	}
	rate := int64(3500)
	project, err := st.CreateProject(ctx, user.ID, ProjectInput{ClientID: client.ID, Name: "RTVE", Color: "#2563eb", DefaultHourlyRateMinor: &rate})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	entry, err := st.CreateTimeEntry(ctx, user.ID, TimeEntryInput{
		ProjectID:   project.ID,
		Description: "Broadcast work",
		StartedAt:   "2026-07-01T09:00:00Z",
		EndedAt:     "2026-07-01T11:00:00Z",
		Billable:    true,
	})
	if err != nil {
		t.Fatalf("create time entry: %v", err)
	}
	if !entry.Billable {
		t.Fatalf("expected billable entry when project has a rate, got %+v", entry)
	}
}
