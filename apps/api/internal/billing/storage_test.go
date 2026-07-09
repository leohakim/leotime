package billing

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDocumentStorePathValidation(t *testing.T) {
	root := t.TempDir()
	store, err := NewDocumentStore(root)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	valid := "invoices/2026/MAIN/2026-0009/invoice.pdf"
	if err := validateDocumentRelativePath(valid); err != nil {
		t.Fatalf("expected valid path: %v", err)
	}

	cases := []string{
		"../leotime.db",
		"/etc/passwd",
		"invoices/2026/x.txt",
	}
	for _, path := range cases {
		if err := validateDocumentRelativePath(path); err == nil {
			t.Fatalf("expected invalid path %q", path)
		}
	}

	source := filepath.Join(t.TempDir(), "source.pdf")
	if err := os.WriteFile(source, []byte("%PDF-1.4 test"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	stored, err := store.WriteOfficial(t.Context(), valid, source)
	if err != nil {
		t.Fatalf("write official: %v", err)
	}
	if stored.ByteSize <= 0 || stored.SHA256 == "" || stored.MIMEType != "application/pdf" {
		t.Fatalf("unexpected stored metadata: %+v", stored)
	}

	file, reopened, err := store.Open(valid)
	if err != nil {
		t.Fatalf("open stored document: %v", err)
	}
	defer file.Close()
	if reopened.SHA256 != stored.SHA256 {
		t.Fatalf("hash mismatch: %s vs %s", reopened.SHA256, stored.SHA256)
	}
}

func TestDocumentStoreRejectsNonPDF(t *testing.T) {
	store, err := NewDocumentStore(t.TempDir())
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	source := filepath.Join(t.TempDir(), "bad.pdf")
	if err := os.WriteFile(source, []byte("not-a-pdf"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	_, err = store.WriteOfficial(t.Context(), "invoices/2026/MAIN/2026-0009/invoice.pdf", source)
	if err == nil || !strings.Contains(err.Error(), "PDF") {
		t.Fatalf("expected non-pdf error, got %v", err)
	}
}
