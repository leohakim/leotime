package backup

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateManifestRejectsInvalidDocumentPath(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, archiveDBFileName)
	if err := os.WriteFile(dbPath, []byte("sqlite-db"), 0o644); err != nil {
		t.Fatal(err)
	}

	hash, _, err := hashFile(dbPath)
	if err != nil {
		t.Fatal(err)
	}

	manifest := &Manifest{
		GeneratedAt:    "2026-07-09T00:00:00Z",
		DatabaseSHA256: hash,
		Documents: []ManifestDocument{
			{Path: "../escape.pdf", SHA256: "abc", ByteSize: 1},
		},
	}

	if err := ValidateManifest(manifest, dir); err == nil {
		t.Fatal("expected invalid document path error")
	}
}

func TestValidateManifestRejectsNegativeByteSize(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, archiveDBFileName)
	if err := os.WriteFile(dbPath, []byte("sqlite-db"), 0o644); err != nil {
		t.Fatal(err)
	}

	hash, _, err := hashFile(dbPath)
	if err != nil {
		t.Fatal(err)
	}

	manifest := &Manifest{
		GeneratedAt:    "2026-07-09T00:00:00Z",
		DatabaseSHA256: hash,
		Documents: []ManifestDocument{
			{Path: "invoice.pdf", SHA256: "abc", ByteSize: -1},
		},
	}

	if err := ValidateManifest(manifest, dir); err == nil {
		t.Fatal("expected negative byte size error")
	}
}

func TestValidateManifestRejectsMissingDocument(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, archiveDBFileName)
	if err := os.WriteFile(dbPath, []byte("sqlite-db"), 0o644); err != nil {
		t.Fatal(err)
	}

	hash, _, err := hashFile(dbPath)
	if err != nil {
		t.Fatal(err)
	}

	manifest := &Manifest{
		GeneratedAt:    "2026-07-09T00:00:00Z",
		DatabaseSHA256: hash,
		Documents: []ManifestDocument{
			{Path: "invoice.pdf", SHA256: "abc", ByteSize: 4},
		},
	}

	if err := ValidateManifest(manifest, dir); err == nil {
		t.Fatal("expected missing document error")
	}
}

func TestValidateManifestRejectsWrongHash(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, archiveDBFileName)
	if err := os.WriteFile(dbPath, []byte("sqlite-db"), 0o644); err != nil {
		t.Fatal(err)
	}

	hash, _, err := hashFile(dbPath)
	if err != nil {
		t.Fatal(err)
	}

	docDir := filepath.Join(dir, archiveDocumentsDir)
	if err := os.MkdirAll(docDir, 0o755); err != nil {
		t.Fatal(err)
	}
	docPath := filepath.Join(docDir, "invoice.pdf")
	if err := os.WriteFile(docPath, []byte("%PDF-1.4"), 0o644); err != nil {
		t.Fatal(err)
	}

	manifest := &Manifest{
		GeneratedAt:    "2026-07-09T00:00:00Z",
		DatabaseSHA256: hash,
		Documents: []ManifestDocument{
			{Path: "invoice.pdf", SHA256: "deadbeef", ByteSize: 8},
		},
	}

	if err := ValidateManifest(manifest, dir); err == nil {
		t.Fatal("expected document hash mismatch error")
	}
}

func TestCreateAndExtractArchiveRoundTrip(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "source.db")
	if err := os.WriteFile(dbPath, []byte("sqlite-db"), 0o644); err != nil {
		t.Fatal(err)
	}

	documentRoot := filepath.Join(dir, "documents")
	docPath := filepath.Join(documentRoot, "invoices", "2026", "MAIN", "2026-0001", "invoice.pdf")
	if err := os.MkdirAll(filepath.Dir(docPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(docPath, []byte("%PDF-1.4 test"), 0o644); err != nil {
		t.Fatal(err)
	}

	archivePath := filepath.Join(dir, "backup.tar.gz")
	if err := CreateArchive(dbPath, documentRoot, archivePath); err != nil {
		t.Fatalf("create archive: %v", err)
	}

	extractDir := filepath.Join(dir, "extracted")
	manifest, err := ExtractArchive(archivePath, extractDir)
	if err != nil {
		t.Fatalf("extract archive: %v", err)
	}
	if manifest == nil || manifest.DatabaseSHA256 == "" {
		t.Fatal("expected manifest with database hash")
	}
	if len(manifest.Documents) != 1 {
		t.Fatalf("expected one document in manifest, got %d", len(manifest.Documents))
	}

	restoredDB := filepath.Join(extractDir, archiveDBFileName)
	data, err := os.ReadFile(restoredDB)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "sqlite-db" {
		t.Fatalf("unexpected restored database content: %q", string(data))
	}
}
