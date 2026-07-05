package store

import (
	"context"
	"errors"
	"testing"

	"github.com/leotime/leotime/apps/api/internal/db"
)

func TestProjectLifecycle(t *testing.T) {
	ctx := context.Background()
	st, user := newProjectTestStore(t, ctx)

	client, err := st.CreateClient(ctx, user.ID, ClientInput{
		Name:            "Acme",
		DefaultCurrency: "EUR",
	})
	if err != nil {
		t.Fatalf("create client: %v", err)
	}

	rate := int64(8500)
	project, err := st.CreateProject(ctx, user.ID, ProjectInput{
		ClientID:               client.ID,
		Name:                   "  API Migration  ",
		Color:                  "#0f7a5b",
		DefaultHourlyRateMinor: &rate,
	})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	if project.Name != "API Migration" || project.ClientName != "Acme" || project.DefaultHourlyRateMinor == nil {
		t.Fatalf("expected normalized project, got %+v", project)
	}

	projects, err := st.ListProjects(ctx, user.ID, false, "")
	if err != nil {
		t.Fatalf("list projects: %v", err)
	}
	if len(projects) != 1 {
		t.Fatalf("expected one project, got %d", len(projects))
	}

	updatedRate := int64(9900)
	updated, err := st.UpdateProject(ctx, user.ID, project.ID, ProjectInput{
		Name:                   "API Migration Updated",
		Color:                  "#2563eb",
		DefaultHourlyRateMinor: &updatedRate,
	})
	if err != nil {
		t.Fatalf("update project: %v", err)
	}
	if updated.ClientID != "" || updated.Name != "API Migration Updated" || *updated.DefaultHourlyRateMinor != 9900 {
		t.Fatalf("unexpected updated project: %+v", updated)
	}

	if err := st.ArchiveProject(ctx, user.ID, project.ID); err != nil {
		t.Fatalf("archive project: %v", err)
	}

	activeProjects, err := st.ListProjects(ctx, user.ID, false, "")
	if err != nil {
		t.Fatalf("list active projects: %v", err)
	}
	if len(activeProjects) != 0 {
		t.Fatalf("expected no active projects, got %d", len(activeProjects))
	}

	allProjects, err := st.ListProjects(ctx, user.ID, true, "")
	if err != nil {
		t.Fatalf("list all projects: %v", err)
	}
	if len(allProjects) != 1 || allProjects[0].ArchivedAt == "" {
		t.Fatalf("expected archived project, got %+v", allProjects)
	}

	restored, err := st.RestoreProject(ctx, user.ID, project.ID)
	if err != nil {
		t.Fatalf("restore project: %v", err)
	}
	if restored.ArchivedAt != "" {
		t.Fatalf("expected restored project without archivedAt, got %+v", restored)
	}

	activeProjectsAfterRestore, err := st.ListProjects(ctx, user.ID, false, "")
	if err != nil {
		t.Fatalf("list active projects after restore: %v", err)
	}
	if len(activeProjectsAfterRestore) != 1 {
		t.Fatalf("expected one active project after restore, got %d", len(activeProjectsAfterRestore))
	}
}

func TestCreateProjectValidatesInput(t *testing.T) {
	ctx := context.Background()
	st, user := newProjectTestStore(t, ctx)

	negativeRate := int64(-1)
	tests := []struct {
		name  string
		input ProjectInput
	}{
		{
			name:  "missing name",
			input: ProjectInput{Name: "", Color: "#2563eb"},
		},
		{
			name:  "invalid color",
			input: ProjectInput{Name: "Project", Color: "blue"},
		},
		{
			name:  "negative rate",
			input: ProjectInput{Name: "Project", Color: "#2563eb", DefaultHourlyRateMinor: &negativeRate},
		},
		{
			name:  "unknown client",
			input: ProjectInput{Name: "Project", Color: "#2563eb", ClientID: "cli_missing"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := st.CreateProject(ctx, user.ID, test.input); !errors.Is(err, ErrInvalidProjectInput) {
				t.Fatalf("expected invalid input, got %v", err)
			}
		})
	}
}

func newProjectTestStore(t *testing.T, ctx context.Context) (*Store, *User) {
	t.Helper()

	database, err := db.Open(ctx, t.TempDir()+"/leotime.db")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() {
		database.Close()
	})

	if err := db.Migrate(ctx, database); err != nil {
		t.Fatalf("migrate db: %v", err)
	}

	st := New(database)
	if err := st.BootstrapAdmin(ctx, "admin@example.com", "change-me-now"); err != nil {
		t.Fatalf("bootstrap admin: %v", err)
	}
	user, err := st.Authenticate(ctx, "admin@example.com", "change-me-now")
	if err != nil {
		t.Fatalf("authenticate: %v", err)
	}
	return st, user
}
