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

type IssueService struct {
	store    *store.Store
	renderer Renderer
	files    *DocumentStore
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

	snapshotJSON, err := snapshot.JSON()
	if err != nil {
		return nil, err
	}

	tempDir, err := os.MkdirTemp("", "leotime-billing-*")
	if err != nil {
		return nil, fmt.Errorf("create temp render dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	rendered, err := s.renderer.RenderPDFs(ctx, snapshot, tempDir)
	if err != nil {
		return nil, err
	}

	year := issueAt.Year()
	invoiceRelative := DocumentRelativePath(year, series.Code, officialNumber, "invoice.pdf")
	protocolRelative := DocumentRelativePath(year, series.Code, officialNumber, "work-protocol.pdf")

	invoiceStored, err := s.files.WriteOfficial(ctx, invoiceRelative, rendered.InvoicePath)
	if err != nil {
		return nil, err
	}
	protocolStored, err := s.files.WriteOfficial(ctx, protocolRelative, rendered.WorkProtocolPath)
	if err != nil {
		return nil, err
	}

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
		StoragePath:   invoiceStored.RelativePath,
		SHA256:        invoiceStored.SHA256,
		ByteSize:      invoiceStored.ByteSize,
		MimeType:      invoiceStored.MIMEType,
		RenderVersion: RenderVersion,
	}); err != nil {
		return nil, err
	}
	if _, err := s.store.InsertBillingDocumentTx(ctx, tx, userID, store.BillingDocumentInput{
		InvoiceID:     invoice.ID,
		Kind:          "work_protocol_pdf",
		StoragePath:   protocolStored.RelativePath,
		SHA256:        protocolStored.SHA256,
		ByteSize:      protocolStored.ByteSize,
		MimeType:      protocolStored.MIMEType,
		RenderVersion: RenderVersion,
	}); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit invoice issue: %w", err)
	}

	return s.store.InvoiceByID(ctx, userID, invoice.ID)
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
