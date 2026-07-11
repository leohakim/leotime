package solidtimeimport

import (
	"archive/zip"
	"os"
	"strings"
	"testing"
)

func TestParseRejectsUnexpectedFileInMap(t *testing.T) {
	files := validFixture()
	files["evil.txt"] = []byte("nope")

	if _, err := Parse(files); err == nil || !strings.Contains(err.Error(), "unexpected file") {
		t.Fatalf("expected unexpected file error, got %v", err)
	}
}

func TestParseFileRejectsTraversalLikeZipEntry(t *testing.T) {
	path := writeZipWithEntries(t, map[string][]byte{
		"../meta.json": []byte(`{"id":"export-1","version":"1.0","organizations":["org-1"],"exported_at":"2025-02-01T10:00:00Z"}`),
	})

	if _, err := ParseFile(path); err == nil || !strings.Contains(err.Error(), "traversal-like") {
		t.Fatalf("expected traversal-like error, got %v", err)
	}
}

func TestParseFileRejectsUnknownZipEntry(t *testing.T) {
	files := validFixture()
	files["notes.txt"] = []byte("extra")
	path := writeZipFixture(t, files)

	if _, err := ParseFile(path); err == nil || !strings.Contains(err.Error(), "unexpected file") {
		t.Fatalf("expected unexpected file error, got %v", err)
	}
}

func TestParseFileRejectsOversizedCSV(t *testing.T) {
	files := validFixture()
	oversized := make([]byte, maxSolidtimeCSVBytes+1)
	copy(oversized, files["clients.csv"])
	files["clients.csv"] = oversized
	path := writeZipFixture(t, files)

	if _, err := ParseFile(path); err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("expected oversized csv error, got %v", err)
	}
}

func TestSourcePathBasenameSanitizesFullPath(t *testing.T) {
	if got := SourcePathBasename("/Users/me/exports/solidtime-export.zip"); got != "solidtime-export.zip" {
		t.Fatalf("expected basename only, got %q", got)
	}
	if got := SourcePathBasename(`C:\exports\solidtime-export.zip`); got != "solidtime-export.zip" {
		t.Fatalf("expected windows basename only, got %q", got)
	}
}

func writeZipWithEntries(t testing.TB, entries map[string][]byte) string {
	t.Helper()

	path := t.TempDir() + "/solidtime-export.zip"
	output, err := os.Create(path)
	if err != nil {
		t.Fatalf("create zip: %v", err)
	}
	defer output.Close()

	writer := zip.NewWriter(output)
	for name, body := range entries {
		fileWriter, err := writer.Create(name)
		if err != nil {
			t.Fatalf("create zip file %s: %v", name, err)
		}
		if _, err := fileWriter.Write(body); err != nil {
			t.Fatalf("write zip file %s: %v", name, err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}
	return path
}
