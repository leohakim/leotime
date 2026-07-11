package billing

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/leotime/leotime/apps/api/internal/db"
	"github.com/leotime/leotime/apps/api/internal/store"
)

func TestIssueRollsBackWhenRendererFails(t *testing.T) {
	ctx := context.Background()
	st, user, series, invoice := newIssueFixture(t, ctx)

	files, err := NewDocumentStore(t.TempDir())
	if err != nil {
		t.Fatalf("document store: %v", err)
	}
	service := NewIssueService(st, failingRenderer{err: assertErr("render failed")}, files)

	_, err = service.Issue(ctx, user.ID, IssueRequest{
		InvoiceID: invoice.ID,
		IssueAt:   time.Date(2026, 7, 8, 12, 0, 0, 0, time.UTC),
	})
	if err == nil {
		t.Fatal("expected render failure")
	}

	reloaded, err := st.InvoiceByID(ctx, user.ID, invoice.ID)
	if err != nil {
		t.Fatalf("reload invoice: %v", err)
	}
	if reloaded.Status != "draft" || !strings.HasPrefix(reloaded.InvoiceNumber, "DRAFT-") {
		t.Fatalf("invoice should remain draft, got %+v", reloaded)
	}

	updatedSeries, err := st.InvoiceSeriesByID(ctx, user.ID, series.ID)
	if err != nil {
		t.Fatalf("reload series: %v", err)
	}
	if updatedSeries.NextSequence != 1 {
		t.Fatalf("expected unchanged sequence, got %d", updatedSeries.NextSequence)
	}

	docs, err := st.ListInvoiceDocuments(ctx, user.ID, invoice.ID)
	if err != nil {
		t.Fatalf("list docs: %v", err)
	}
	if len(docs) != 0 {
		t.Fatalf("expected no documents after rollback, got %d", len(docs))
	}
}

func TestIssueRollsBackWhenPromotionFails(t *testing.T) {
	ctx := context.Background()
	st, user, series, invoice := newIssueFixture(t, ctx)

	root := t.TempDir()
	baseStore, err := NewDocumentStore(root)
	if err != nil {
		t.Fatalf("document store: %v", err)
	}
	files := &failingDocumentStore{DocumentStore: baseStore, failOn: 1}
	service := &IssueService{store: st, renderer: stubRenderer{}, files: files}

	_, err = service.Issue(ctx, user.ID, IssueRequest{
		InvoiceID: invoice.ID,
		IssueAt:   time.Date(2026, 7, 8, 12, 0, 0, 0, time.UTC),
	})
	if err == nil {
		t.Fatal("expected promotion failure")
	}

	reloaded, err := st.InvoiceByID(ctx, user.ID, invoice.ID)
	if err != nil {
		t.Fatalf("reload invoice: %v", err)
	}
	if reloaded.Status != "draft" || !strings.HasPrefix(reloaded.InvoiceNumber, "DRAFT-") {
		t.Fatalf("invoice should remain draft, got %+v", reloaded)
	}

	updatedSeries, err := st.InvoiceSeriesByID(ctx, user.ID, series.ID)
	if err != nil {
		t.Fatalf("reload series: %v", err)
	}
	if updatedSeries.NextSequence != 1 {
		t.Fatalf("expected unchanged sequence, got %d", updatedSeries.NextSequence)
	}

	docs, err := st.ListInvoiceDocuments(ctx, user.ID, invoice.ID)
	if err != nil {
		t.Fatalf("list docs: %v", err)
	}
	if len(docs) != 0 {
		t.Fatalf("expected no documents after rollback, got %d", len(docs))
	}

	if countPDFsUnder(root) != 0 {
		t.Fatalf("expected no official documents under root")
	}
}

func TestIssueRollsBackWhenSecondPromotionFails(t *testing.T) {
	ctx := context.Background()
	st, user, series, invoice := newIssueFixture(t, ctx)

	root := t.TempDir()
	baseStore, err := NewDocumentStore(root)
	if err != nil {
		t.Fatalf("document store: %v", err)
	}
	files := &failingDocumentStore{DocumentStore: baseStore, failOn: 2}
	service := &IssueService{store: st, renderer: stubRenderer{}, files: files}

	_, err = service.Issue(ctx, user.ID, IssueRequest{
		InvoiceID: invoice.ID,
		IssueAt:   time.Date(2026, 7, 8, 12, 0, 0, 0, time.UTC),
	})
	if err == nil {
		t.Fatal("expected promotion failure")
	}

	reloaded, err := st.InvoiceByID(ctx, user.ID, invoice.ID)
	if err != nil {
		t.Fatalf("reload invoice: %v", err)
	}
	if reloaded.Status != "draft" {
		t.Fatalf("invoice should remain draft, got %+v", reloaded)
	}

	updatedSeries, err := st.InvoiceSeriesByID(ctx, user.ID, series.ID)
	if err != nil {
		t.Fatalf("reload series: %v", err)
	}
	if updatedSeries.NextSequence != 1 {
		t.Fatalf("expected unchanged sequence, got %d", updatedSeries.NextSequence)
	}

	if countPDFsUnder(root) != 0 {
		t.Fatalf("expected no official documents under root after partial promotion failure")
	}
}

func countPDFsUnder(root string) int {
	count := 0
	_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if strings.HasSuffix(strings.ToLower(entry.Name()), ".pdf") {
			count++
		}
		return nil
	})
	return count
}

func TestIssueCreatesImmutableDocuments(t *testing.T) {
	ctx := context.Background()
	st, user, series, invoice := newIssueFixture(t, ctx)

	files, err := NewDocumentStore(t.TempDir())
	if err != nil {
		t.Fatalf("document store: %v", err)
	}
	service := NewIssueService(st, stubRenderer{}, files)

	issued, err := service.Issue(ctx, user.ID, IssueRequest{
		InvoiceID: invoice.ID,
		IssueAt:   time.Date(2026, 7, 8, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("issue invoice: %v", err)
	}
	if issued.Status != "issued" || issued.InvoiceNumber != "2026-0001" {
		t.Fatalf("unexpected issued invoice: %+v", issued)
	}
	if issued.DocumentSnapshotJSON == "" {
		t.Fatal("expected document snapshot json")
	}
	if len(issued.Documents) != 2 {
		t.Fatalf("expected two documents, got %+v", issued.Documents)
	}

	updatedSeries, err := st.InvoiceSeriesByID(ctx, user.ID, series.ID)
	if err != nil {
		t.Fatalf("reload series: %v", err)
	}
	if updatedSeries.NextSequence != 2 {
		t.Fatalf("expected next sequence 2, got %d", updatedSeries.NextSequence)
	}

	_, err = st.UpdateInvoice(ctx, user.ID, invoice.ID, store.InvoiceUpdateInput{
		Notes: strPtr("changed"),
	})
	if err != store.ErrInvoiceNotEditable {
		t.Fatalf("expected not editable error, got %v", err)
	}
}

func newIssueFixture(t *testing.T, ctx context.Context) (*store.Store, *store.User, *store.InvoiceSeries, *store.Invoice) {
	t.Helper()
	st, user := newBillingTestStore(t, ctx)

	series, err := st.DefaultInvoiceSeries(ctx, user.ID)
	if err != nil {
		t.Fatalf("default series: %v", err)
	}

	client, err := st.CreateClient(ctx, user.ID, store.ClientInput{
		Name:                   "Acme",
		DefaultCurrency:        "EUR",
		DefaultHourlyRateMinor: 10000,
	})
	if err != nil {
		t.Fatalf("create client: %v", err)
	}

	project, err := st.CreateProject(ctx, user.ID, store.ProjectInput{
		ClientID: client.ID,
		Name:     "Portal",
		Color:    "#2563eb",
	})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	_, err = st.CreateTimeEntry(ctx, user.ID, store.TimeEntryInput{
		ClientID:    client.ID,
		ProjectID:   project.ID,
		Description: "Design",
		StartedAt:   "2026-07-01T08:00:00Z",
		EndedAt:     "2026-07-01T10:00:00Z",
		Billable:    true,
	})
	if err != nil {
		t.Fatalf("create entry: %v", err)
	}

	invoice, err := st.CreateInvoiceDraftFromTime(ctx, user.ID, store.InvoiceDraftFromTimeInput{
		ClientID: client.ID,
		From:     "2026-07-01T00:00:00Z",
		To:       "2026-07-31T23:59:59Z",
		SeriesID: series.ID,
	})
	if err != nil {
		t.Fatalf("create draft: %v", err)
	}

	return st, user, series, invoice
}

func newBillingTestStore(t *testing.T, ctx context.Context) (*store.Store, *store.User) {
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

func assertErr(message string) error {
	return &testError{message: message}
}

type testError struct {
	message string
}

func (e *testError) Error() string {
	return e.message
}

func strPtr(value string) *string {
	return &value
}
