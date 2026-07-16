package seed

import (
	"context"
	"testing"
	"time"

	"github.com/leotime/leotime/apps/api/internal/db"
	"github.com/leotime/leotime/apps/api/internal/store"
)

func TestSeedCreatesDemoData(t *testing.T) {
	ctx := context.Background()
	st, user := newSeedTestStore(t, ctx)

	fixedNow := time.Date(2026, 7, 16, 15, 0, 0, 0, time.UTC)
	service := NewWithNow(st, func() time.Time { return fixedNow })

	summary, err := service.Run(ctx, Options{UserID: user.ID})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	if summary.Status != "seeded" {
		t.Fatalf("expected seeded status, got %+v", summary)
	}
	if summary.Clients < 3 || summary.Projects < 5 || summary.Tasks < 6 || summary.Tags < 5 {
		t.Fatalf("unexpected seeded catalog counts: %+v", summary)
	}
	if summary.TimeEntries < 300 || summary.OpenTimers != 1 {
		t.Fatalf("expected six months of entries and one open timer, got %+v", summary)
	}
	if summary.Invoices < 2 {
		t.Fatalf("expected invoice drafts, got %+v", summary)
	}

	entries, err := st.ListTimeEntries(ctx, user.ID, store.TimeEntryListOptions{
		From: fixedNow.AddDate(0, -seedHistoryMonths, 0).Format(time.RFC3339),
		To:   fixedNow.Format(time.RFC3339),
	})
	if err != nil {
		t.Fatalf("list seeded entries: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("expected seeded time entries in the requested range")
	}
}

func TestNewWithNowPinsSeedTimeline(t *testing.T) {
	ctx := context.Background()
	st, user := newSeedTestStore(t, ctx)

	fixed := time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC)
	summary, err := NewWithNow(st, func() time.Time { return fixed }).Run(ctx, Options{UserID: user.ID})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	if summary.Status != "seeded" {
		t.Fatalf("expected seeded status, got %+v", summary)
	}
}

func TestSeedSkipsWhenDataExists(t *testing.T) {
	ctx := context.Background()
	st, user := newSeedTestStore(t, ctx)

	if _, err := st.CreateClient(ctx, user.ID, store.ClientInput{Name: "Existing client"}); err != nil {
		t.Fatalf("create client: %v", err)
	}

	summary, err := New(st).Run(ctx, Options{UserID: user.ID})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	if summary.Status != "skipped" {
		t.Fatalf("expected skipped status, got %+v", summary)
	}
}

func TestSeedForceWipesAndReseeds(t *testing.T) {
	ctx := context.Background()
	st, user := newSeedTestStore(t, ctx)

	if _, err := st.CreateClient(ctx, user.ID, store.ClientInput{Name: "Existing client"}); err != nil {
		t.Fatalf("create client: %v", err)
	}

	summary, err := New(st).Run(ctx, Options{UserID: user.ID, Force: true})
	if err != nil {
		t.Fatalf("seed with force: %v", err)
	}
	if summary.Status != "seeded" {
		t.Fatalf("expected seeded status after force, got %+v", summary)
	}
	if summary.Clients < 3 {
		t.Fatalf("expected reseeded catalog, got %+v", summary)
	}
}

func newSeedTestStore(t *testing.T, ctx context.Context) (*store.Store, *store.User) {
	t.Helper()

	database, err := db.Open(ctx, t.TempDir()+"/leotime.db")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

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
	return st, user
}
