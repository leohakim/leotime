package seed

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/leotime/leotime/apps/api/internal/db"
	"github.com/leotime/leotime/apps/api/internal/store"
)

func TestSeedCreatesDemoData(t *testing.T) {
	ctx := context.Background()
	st, user := newSeedTestStore(t, ctx)

	service := New(st)
	service.now = func() time.Time {
		return time.Date(2026, 7, 8, 15, 0, 0, 0, time.UTC)
	}

	summary, err := service.Run(ctx, Options{UserID: user.ID})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	if summary.Status != "seeded" {
		t.Fatalf("expected seeded status, got %+v", summary)
	}
	if summary.Clients < 2 || summary.Projects < 3 || summary.Tasks < 4 || summary.Tags < 3 {
		t.Fatalf("unexpected seeded counts: %+v", summary)
	}
	if summary.TimeEntries == 0 || summary.OpenTimers != 1 {
		t.Fatalf("expected time entries and one open timer, got %+v", summary)
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

func TestSeedForceRejectsExistingData(t *testing.T) {
	ctx := context.Background()
	st, user := newSeedTestStore(t, ctx)

	if _, err := st.CreateClient(ctx, user.ID, store.ClientInput{Name: "Existing client"}); err != nil {
		t.Fatalf("create client: %v", err)
	}

	_, err := New(st).Run(ctx, Options{UserID: user.ID, Force: true})
	if !errors.Is(err, ErrAlreadySeeded) {
		t.Fatalf("expected ErrAlreadySeeded, got %v", err)
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
