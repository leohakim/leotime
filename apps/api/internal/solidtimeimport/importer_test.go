package solidtimeimport

import (
	"archive/zip"
	"bytes"
	"context"
	"database/sql"
	"encoding/csv"
	"os"
	"strings"
	"testing"

	"github.com/leotime/leotime/apps/api/internal/db"
	"github.com/leotime/leotime/apps/api/internal/store"
)

func TestParseFileReadsSyntheticSolidtimeExport(t *testing.T) {
	path := writeZipFixture(t, validFixture())

	export, err := ParseFile(path)
	if err != nil {
		t.Fatalf("parse solidtime export: %v", err)
	}

	if export.Meta.Version != "1.0" {
		t.Fatalf("expected version 1.0, got %q", export.Meta.Version)
	}
	if len(export.Clients) != 1 || len(export.TimeEntries) != 1 {
		t.Fatalf("expected one client and one time entry, got %d/%d", len(export.Clients), len(export.TimeEntries))
	}
}

func TestParseFileRejectsInvalidZip(t *testing.T) {
	path := t.TempDir() + "/invalid.zip"
	if err := os.WriteFile(path, []byte("not a zip"), 0o644); err != nil {
		t.Fatalf("write invalid zip: %v", err)
	}

	if _, err := ParseFile(path); err == nil {
		t.Fatal("expected invalid zip error")
	}
}

func TestParseRejectsMissingCSV(t *testing.T) {
	files := validFixture()
	delete(files, "projects.csv")

	if _, err := Parse(files); err == nil || !strings.Contains(err.Error(), "missing projects.csv") {
		t.Fatalf("expected missing projects.csv error, got %v", err)
	}
}

func TestParseRejectsUnexpectedHeaders(t *testing.T) {
	files := validFixture()
	files["clients.csv"] = []byte("wrong,name\nclient-1,Client\n")

	if _, err := Parse(files); err == nil || !strings.Contains(err.Error(), "unexpected clients.csv headers") {
		t.Fatalf("expected header error, got %v", err)
	}
}

func TestDryRunDoesNotWrite(t *testing.T) {
	ctx := context.Background()
	database := testDB(t, ctx)
	importer := New(database)

	summary, err := importer.ImportFile(ctx, Options{
		FilePath:  writeZipFixture(t, validFixture()),
		UserEmail: "admin@example.com",
		DryRun:    true,
	})
	if err != nil {
		t.Fatalf("dry-run import: %v", err)
	}

	if !summary.DryRun || summary.Clients.Created != 1 || summary.TimeEntries.Created != 1 {
		t.Fatalf("unexpected dry-run summary: %+v", summary)
	}
	assertCount(t, database, "clients", 0)
	assertCount(t, database, "external_mappings", 0)
}

func TestImportIsIdempotent(t *testing.T) {
	ctx := context.Background()
	database := testDB(t, ctx)
	importer := New(database)
	path := writeZipFixture(t, validFixture())

	first, err := importer.ImportFile(ctx, Options{
		FilePath:  path,
		UserEmail: "admin@example.com",
	})
	if err != nil {
		t.Fatalf("first import: %v", err)
	}
	if first.Clients.Created != 1 || first.Projects.Created != 1 || first.Tasks.Created != 1 || first.TimeEntries.Created != 1 {
		t.Fatalf("unexpected first import summary: %+v", first)
	}

	second, err := importer.ImportFile(ctx, Options{
		FilePath:  path,
		UserEmail: "admin@example.com",
	})
	if err != nil {
		t.Fatalf("second import: %v", err)
	}
	if second.Clients.Updated != 1 || second.Projects.Updated != 1 || second.Tasks.Updated != 1 || second.TimeEntries.Updated != 1 {
		t.Fatalf("unexpected second import summary: %+v", second)
	}

	assertCount(t, database, "clients", 1)
	assertCount(t, database, "projects", 1)
	assertCount(t, database, "tasks", 1)
	assertCount(t, database, "tags", 1)
	assertCount(t, database, "time_entries", 1)
}

func TestImportPersistsStillActiveEmailSentAt(t *testing.T) {
	ctx := context.Background()
	database := testDB(t, ctx)
	importer := New(database)

	files := validFixture()
	files["time_entries.csv"] = csvFile(requiredHeaders["time_entries.csv"], [][]string{
		{"entry-1", "Work", "2025-02-01T09:00:00Z", "", "", "true", "member-1", "user-1", "org-1", "client-1", "project-1", "task-1", `["tag-1"]`, "false", "2025-02-01T08:00:00Z", "2025-02-01T09:00:00Z", "2025-02-01T09:00:00Z"},
	})

	if _, err := importer.ImportFile(ctx, Options{
		FilePath:  writeZipFixture(t, files),
		UserEmail: "admin@example.com",
	}); err != nil {
		t.Fatalf("import with still_active_email_sent_at: %v", err)
	}

	var stillActive sql.NullString
	if err := database.QueryRowContext(ctx, `
		SELECT still_active_email_sent_at
		FROM time_entries
		LIMIT 1
	`).Scan(&stillActive); err != nil {
		t.Fatalf("query still_active_email_sent_at: %v", err)
	}
	if !stillActive.Valid || stillActive.String == "" {
		t.Fatal("expected still_active_email_sent_at to be persisted")
	}
	if !strings.HasPrefix(stillActive.String, "2025-02-01T08:00:00") {
		t.Fatalf("unexpected still_active_email_sent_at %q", stillActive.String)
	}
}

func TestImportRejectsInvalidStillActiveEmailSentAt(t *testing.T) {
	ctx := context.Background()
	database := testDB(t, ctx)
	files := validFixture()
	files["time_entries.csv"] = csvFile(requiredHeaders["time_entries.csv"], [][]string{
		{"entry-1", "Work", "2025-02-01T09:00:00Z", "2025-02-01T10:00:00Z", "", "true", "member-1", "user-1", "org-1", "client-1", "project-1", "task-1", `["tag-1"]`, "false", "not-a-timestamp", "2025-02-01T09:00:00Z", "2025-02-01T10:00:00Z"},
	})

	_, err := New(database).ImportFile(ctx, Options{
		FilePath:  writeZipFixture(t, files),
		UserEmail: "admin@example.com",
		DryRun:    true,
	})
	if err == nil || !strings.Contains(err.Error(), "invalid still_active_email_sent_at") {
		t.Fatalf("expected invalid still_active_email_sent_at error, got %v", err)
	}
}

func TestImportRejectsUnknownReference(t *testing.T) {
	ctx := context.Background()
	database := testDB(t, ctx)
	files := validFixture()
	files["time_entries.csv"] = csvFile(requiredHeaders["time_entries.csv"], [][]string{
		{"entry-1", "Work", "2025-02-01T09:00:00Z", "2025-02-01T10:00:00Z", "", "true", "member-1", "user-1", "org-1", "client-1", "missing-project", "task-1", "[]", "false", "", "2025-02-01T09:00:00Z", "2025-02-01T10:00:00Z"},
	})

	_, err := New(database).ImportFile(ctx, Options{
		FilePath:  writeZipFixture(t, files),
		UserEmail: "admin@example.com",
		DryRun:    true,
	})
	if err == nil || !strings.Contains(err.Error(), "unknown project") {
		t.Fatalf("expected unknown project error, got %v", err)
	}
}

func BenchmarkParseSyntheticSolidtimeExport(b *testing.B) {
	path := writeZipFixture(b, validFixture())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := ParseFile(path); err != nil {
			b.Fatalf("parse solidtime export: %v", err)
		}
	}
}

func testDB(t *testing.T, ctx context.Context) *sql.DB {
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
	if err := store.New(database).BootstrapAdmin(ctx, "admin@example.com", "change-me-now"); err != nil {
		t.Fatalf("bootstrap admin: %v", err)
	}
	return database
}

func assertCount(t *testing.T, database *sql.DB, table string, expected int) {
	t.Helper()

	var count int
	if err := database.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	if count != expected {
		t.Fatalf("expected %s count %d, got %d", table, expected, count)
	}
}

func writeZipFixture(t testing.TB, files map[string][]byte) string {
	t.Helper()

	path := t.TempDir() + "/solidtime-export.zip"
	output, err := os.Create(path)
	if err != nil {
		t.Fatalf("create zip: %v", err)
	}
	defer output.Close()

	writer := zip.NewWriter(output)
	for name, body := range files {
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

func validFixture() map[string][]byte {
	return map[string][]byte{
		"meta.json":                    []byte(`{"id":"export-1","version":"1.0","organizations":["org-1"],"exported_at":"2025-02-01T10:00:00Z"}`),
		"organizations.csv":            csvFile(requiredHeaders["organizations.csv"], [][]string{{"org-1", "Leonardo Org", "", "EUR", "2025-01-01T00:00:00Z", "2025-01-01T00:00:00Z"}}),
		"organization_invitations.csv": csvFile(requiredHeaders["organization_invitations.csv"], nil),
		"members.csv":                  csvFile(requiredHeaders["members.csv"], [][]string{{"member-1", "user-1", "Leonardo", "admin@example.com", "org-1", "", "owner", "2025-01-01T00:00:00Z", "2025-01-01T00:00:00Z"}}),
		"clients.csv":                  csvFile(requiredHeaders["clients.csv"], [][]string{{"client-1", "Client One", "org-1", "", "2025-01-01T00:00:00Z", "2025-01-01T00:00:00Z"}}),
		"projects.csv":                 csvFile(requiredHeaders["projects.csv"], [][]string{{"project-1", "Project One", "#42a5f5", "", "false", "client-1", "org-1", "true", "", "2025-01-01T00:00:00Z", "2025-01-01T00:00:00Z"}}),
		"project_members.csv":          csvFile(requiredHeaders["project_members.csv"], nil),
		"tasks.csv":                    csvFile(requiredHeaders["tasks.csv"], [][]string{{"task-1", "Task One", "project-1", "org-1", "", "2025-01-01T00:00:00Z", "2025-01-01T00:00:00Z"}}),
		"tags.csv":                     csvFile(requiredHeaders["tags.csv"], [][]string{{"tag-1", "Deep Work", "org-1", "2025-01-01T00:00:00Z", "2025-01-01T00:00:00Z"}}),
		"time_entries.csv":             csvFile(requiredHeaders["time_entries.csv"], [][]string{{"entry-1", "Work", "2025-02-01T09:00:00Z", "2025-02-01T10:00:00Z", "", "true", "member-1", "user-1", "org-1", "client-1", "project-1", "task-1", `["tag-1"]`, "false", "", "2025-02-01T09:00:00Z", "2025-02-01T10:00:00Z"}}),
	}
}

func csvFile(headers []string, rows [][]string) []byte {
	var buffer bytes.Buffer
	writer := csv.NewWriter(&buffer)
	_ = writer.Write(headers)
	for _, row := range rows {
		_ = writer.Write(row)
	}
	writer.Flush()
	return buffer.Bytes()
}
