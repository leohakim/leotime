package backup

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStageAndPromoteRestoredDocuments(t *testing.T) {
	dir := t.TempDir()
	documentRoot := filepath.Join(dir, "documents")
	liveDoc := filepath.Join(documentRoot, "invoices", "live.pdf")
	if err := os.MkdirAll(filepath.Dir(liveDoc), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(liveDoc, []byte("live-version"), 0o644); err != nil {
		t.Fatal(err)
	}

	sourceDir := filepath.Join(dir, "extracted", archiveDocumentsDir)
	restoredDoc := filepath.Join(sourceDir, "invoices", "restored.pdf")
	if err := os.MkdirAll(filepath.Dir(restoredDoc), 0o755); err != nil {
		t.Fatal(err)
	}
	restoredContent := []byte("%PDF restored")
	if err := os.WriteFile(restoredDoc, restoredContent, 0o644); err != nil {
		t.Fatal(err)
	}

	hash, size, err := hashFile(restoredDoc)
	if err != nil {
		t.Fatal(err)
	}
	manifest := &Manifest{
		GeneratedAt:    "2026-07-11T00:00:00Z",
		DatabaseSHA256: "db",
		Documents: []ManifestDocument{
			{Path: "invoices/restored.pdf", SHA256: hash, ByteSize: size},
		},
	}

	stagingDir, err := stageRestoredDocuments(manifest, sourceDir, documentRoot)
	if err != nil {
		t.Fatalf("stage restored documents: %v", err)
	}
	if _, err := os.Stat(filepath.Join(stagingDir, "invoices", "restored.pdf")); err != nil {
		t.Fatalf("expected staged document: %v", err)
	}

	backupDir, err := backupLiveDocuments(documentRoot)
	if err != nil {
		t.Fatalf("backup live documents: %v", err)
	}
	liveHash, _, err := hashFile(liveDoc)
	if err != nil {
		t.Fatal(err)
	}

	if err := promoteStagedDocuments(stagingDir, documentRoot); err != nil {
		t.Fatalf("promote staged documents: %v", err)
	}

	promotedHash, _, err := hashFile(filepath.Join(documentRoot, "invoices", "restored.pdf"))
	if err != nil {
		t.Fatal(err)
	}
	if promotedHash != hash {
		t.Fatalf("expected promoted hash %s, got %s", hash, promotedHash)
	}
	if _, err := os.Stat(filepath.Join(documentRoot, "invoices", "live.pdf")); !os.IsNotExist(err) {
		t.Fatal("expected live document to be replaced by staged tree")
	}

	if err := restoreBackedUpDocuments(backupDir, documentRoot); err != nil {
		t.Fatalf("restore backed up documents: %v", err)
	}
	rolledBackHash, _, err := hashFile(liveDoc)
	if err != nil {
		t.Fatal(err)
	}
	if rolledBackHash != liveHash {
		t.Fatalf("expected rolled back hash %s, got %s", liveHash, rolledBackHash)
	}
}
