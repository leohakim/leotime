package billing

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/leotime/leotime/apps/api/internal/store"
)

type IssueRequest struct {
	InvoiceID string
	IssueAt   time.Time
}

type officialDocumentStore interface {
	WriteOfficial(ctx context.Context, relativePath string, sourcePath string) (StoredDocument, error)
	RemoveOfficial(relativePath string) error
}

type IssueService struct {
	store    *store.Store
	renderer Renderer
	files    officialDocumentStore
}

func NewIssueService(store *store.Store, renderer Renderer, files *DocumentStore) *IssueService {
	return &IssueService{
		store:    store,
		renderer: renderer,
		files:    files,
	}
}

func (s *IssueService) Issue(ctx context.Context, userID string, request IssueRequest) (*store.Invoice, error) {
	invoice, err := s.store.InvoiceByID(ctx, userID, request.InvoiceID)
	if err != nil {
		return nil, err
	}
	if invoice.Status != "draft" {
		return nil, store.InvoiceInputError("status", "invalid", "invoice is not editable")
	}
	if strings.TrimSpace(invoice.SeriesID) == "" {
		return nil, store.InvoiceInputError("seriesId", "required", "fiscal series is required")
	}
	if strings.TrimSpace(invoice.SellerName) == "" {
		return nil, store.InvoiceInputError("sellerName", "required", "seller name is required")
	}
	if strings.TrimSpace(invoice.ClientName) == "" {
		return nil, store.InvoiceInputError("clientName", "required", "client name is required")
	}
	if len(invoice.Lines) == 0 || invoice.TotalMinor <= 0 {
		return nil, store.InvoiceInputError("lines", "invalid", "invoice must have positive billable lines")
	}

	series, err := s.store.InvoiceSeriesByID(ctx, userID, invoice.SeriesID)
	if err != nil {
		return nil, err
	}
	if !series.Active {
		return nil, store.InvoiceInputError("seriesId", "invalid", "invoice series is inactive")
	}

	entries, err := s.store.TimeEntriesForInvoice(ctx, userID, invoice)
	if err != nil {
		return nil, err
	}

	issueAt := request.IssueAt
	if issueAt.IsZero() {
		issueAt = time.Now().UTC()
	}

	tempDir, err := os.MkdirTemp("", "leotime-billing-*")
	if err != nil {
		return nil, fmt.Errorf("create temp render dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	tx, err := s.store.DB().BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin invoice issue: %w", err)
	}
	defer tx.Rollback()

	officialNumber, fiscalSequence, err := s.store.NextInvoiceNumberTx(ctx, tx, userID, invoice.SeriesID, issueAt)
	if err != nil {
		return nil, err
	}

	snapshot, err := BuildDocumentSnapshot(invoice, entries, SnapshotOptions{
		IssueAt:    issueAt,
		SeriesCode: series.Code,
	})
	if err != nil {
		return nil, err
	}
	snapshot.Invoice.Number = officialNumber
	snapshot.Invoice.Status = "issued"
	snapshot.WorkProtocol.Number = officialNumber

	rendered, err := s.renderer.RenderPDFs(ctx, snapshot, tempDir)
	if err != nil {
		return nil, err
	}

	invoiceHashed, err := HashSourceFile(rendered.InvoicePath)
	if err != nil {
		return nil, err
	}
	protocolHashed, err := HashSourceFile(rendered.WorkProtocolPath)
	if err != nil {
		return nil, err
	}

	snapshotJSON, err := snapshot.JSON()
	if err != nil {
		return nil, err
	}

	year := issueAt.Year()
	invoiceRelative := DocumentRelativePath(year, series.Code, officialNumber, "invoice.pdf")
	protocolRelative := DocumentRelativePath(year, series.Code, officialNumber, "work-protocol.pdf")

	issuedAt := issueAt.UTC().Format(time.RFC3339Nano)
	if err := s.store.MarkInvoiceIssuedTx(ctx, tx, userID, invoice.ID, store.InvoiceIssueInput{
		InvoiceNumber:        officialNumber,
		SeriesID:             invoice.SeriesID,
		FiscalSequence:       fiscalSequence,
		IssuedAt:             issuedAt,
		DocumentSnapshotJSON: snapshotJSON,
	}); err != nil {
		return nil, err
	}

	if _, err := s.store.InsertBillingDocumentTx(ctx, tx, userID, store.BillingDocumentInput{
		InvoiceID:     invoice.ID,
		Kind:          "invoice_pdf",
		StoragePath:   invoiceRelative,
		SHA256:        invoiceHashed.SHA256,
		ByteSize:      invoiceHashed.ByteSize,
		MimeType:      invoiceHashed.MIMEType,
		RenderVersion: RenderVersion,
	}); err != nil {
		return nil, err
	}
	if _, err := s.store.InsertBillingDocumentTx(ctx, tx, userID, store.BillingDocumentInput{
		InvoiceID:     invoice.ID,
		Kind:          "work_protocol_pdf",
		StoragePath:   protocolRelative,
		SHA256:        protocolHashed.SHA256,
		ByteSize:      protocolHashed.ByteSize,
		MimeType:      protocolHashed.MIMEType,
		RenderVersion: RenderVersion,
	}); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit invoice issue: %w", err)
	}

	promoted := []string{invoiceRelative, protocolRelative}
	if _, err := s.files.WriteOfficial(ctx, invoiceRelative, rendered.InvoicePath); err != nil {
		if revertErr := s.revertIssueAfterPromotionFailure(ctx, userID, invoice.ID, invoice.SeriesID, fiscalSequence, promoted); revertErr != nil {
			return nil, fmt.Errorf("promote invoice pdf: %w (revert failed: %v)", err, revertErr)
		}
		return nil, err
	}
	if _, err := s.files.WriteOfficial(ctx, protocolRelative, rendered.WorkProtocolPath); err != nil {
		_ = s.files.RemoveOfficial(invoiceRelative)
		if revertErr := s.revertIssueAfterPromotionFailure(ctx, userID, invoice.ID, invoice.SeriesID, fiscalSequence, promoted); revertErr != nil {
			return nil, fmt.Errorf("promote work protocol pdf: %w (revert failed: %v)", err, revertErr)
		}
		return nil, err
	}

	return s.store.InvoiceByID(ctx, userID, invoice.ID)
}

func (s *IssueService) revertIssueAfterPromotionFailure(ctx context.Context, userID, invoiceID, seriesID string, fiscalSequence int, relativePaths []string) error {
	for _, relativePath := range relativePaths {
		_ = s.files.RemoveOfficial(relativePath)
	}

	tx, err := s.store.DB().BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin invoice revert: %w", err)
	}
	defer tx.Rollback()

	if err := s.store.RevertInvoiceIssueTx(ctx, tx, userID, invoiceID, seriesID, fiscalSequence); err != nil {
		return err
	}
	return tx.Commit()
}

type failingRenderer struct {
	err error
}

func (f failingRenderer) RenderPreviewHTML(context.Context, DocumentSnapshot) ([]byte, error) {
	return nil, f.err
}

func (f failingRenderer) RenderPDFs(context.Context, DocumentSnapshot, string) (RenderedPDFs, error) {
	return RenderedPDFs{}, f.err
}

type stubRenderer struct {
	dir string
}

func (s stubRenderer) RenderPreviewHTML(context.Context, DocumentSnapshot) ([]byte, error) {
	return []byte("<html>preview</html>"), nil
}

func (s stubRenderer) RenderPDFs(_ context.Context, _ DocumentSnapshot, outputDir string) (RenderedPDFs, error) {
	invoicePath := filepath.Join(outputDir, "invoice.pdf")
	protocolPath := filepath.Join(outputDir, "work-protocol.pdf")
	for _, path := range []string{invoicePath, protocolPath} {
		if err := os.WriteFile(path, []byte("%PDF-1.4\n% stub"), 0o644); err != nil {
			return RenderedPDFs{}, err
		}
	}
	return RenderedPDFs{InvoicePath: invoicePath, WorkProtocolPath: protocolPath}, nil
}

type failingDocumentStore struct {
	*DocumentStore
	failOn int
	calls  int
}

func (f *failingDocumentStore) WriteOfficial(ctx context.Context, relativePath string, sourcePath string) (StoredDocument, error) {
	f.calls++
	if f.calls >= f.failOn {
		return StoredDocument{}, fmt.Errorf("forced promotion failure")
	}
	return f.DocumentStore.WriteOfficial(ctx, relativePath, sourcePath)
}
