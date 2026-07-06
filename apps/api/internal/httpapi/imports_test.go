package httpapi

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestImportSolidtimeRequiresAuthentication(t *testing.T) {
	router := newTestRouter(t)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/imports/solidtime", nil)
	router.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", response.Code)
	}
}

func TestImportSolidtimeDryRun(t *testing.T) {
	router := newTestRouter(t)
	cookies := loginCookies(t, router)
	zipBytes := buildTestSolidtimeZip(t)

	body, contentType := multipartBody(t, "file", "solidtime-export.zip", zipBytes)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/imports/solidtime?dryRun=true", body)
	request.Header.Set("Content-Type", contentType)
	for _, cookie := range cookies {
		request.AddCookie(cookie)
	}
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}

	var payload solidtimeImportResponse
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode import response: %v", err)
	}
	if !payload.Summary.DryRun || payload.Summary.Clients.Created != 1 || payload.Summary.TimeEntries.Created != 1 {
		t.Fatalf("unexpected import summary: %+v", payload.Summary)
	}
}

func multipartBody(t *testing.T, fieldName, filename string, content []byte) (*bytes.Buffer, string) {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile(fieldName, filename)
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := io.Copy(part, bytes.NewReader(content)); err != nil {
		t.Fatalf("write form file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}
	return &body, writer.FormDataContentType()
}

func buildTestSolidtimeZip(t *testing.T) []byte {
	t.Helper()

	files := map[string][]byte{
		"meta.json":                    []byte(`{"id":"export-1","version":"1.0","organizations":["org-1"],"exported_at":"2025-02-01T10:00:00Z"}`),
		"organizations.csv":            testCSV([]string{"id", "name", "billable_rate", "currency", "created_at", "updated_at"}, [][]string{{"org-1", "Leonardo Org", "", "EUR", "2025-01-01T00:00:00Z", "2025-01-01T00:00:00Z"}}),
		"organization_invitations.csv": testCSV([]string{"id", "email", "organization_id", "role", "created_at", "updated_at"}, nil),
		"members.csv":                  testCSV([]string{"id", "user_id", "name", "email", "organization_id", "billable_rate", "role", "created_at", "updated_at"}, [][]string{{"member-1", "user-1", "Leonardo", "admin@example.com", "org-1", "", "owner", "2025-01-01T00:00:00Z", "2025-01-01T00:00:00Z"}}),
		"clients.csv":                  testCSV([]string{"id", "name", "organization_id", "archived_at", "created_at", "updated_at"}, [][]string{{"client-1", "Client One", "org-1", "", "2025-01-01T00:00:00Z", "2025-01-01T00:00:00Z"}}),
		"projects.csv":                 testCSV([]string{"id", "name", "color", "billable_rate", "is_public", "client_id", "organization_id", "is_billable", "archived_at", "created_at", "updated_at"}, [][]string{{"project-1", "Project One", "#42a5f5", "", "false", "client-1", "org-1", "true", "", "2025-01-01T00:00:00Z", "2025-01-01T00:00:00Z"}}),
		"project_members.csv":          testCSV([]string{"id", "billable_rate", "project_id", "user_id", "member_id", "created_at", "updated_at"}, nil),
		"tasks.csv":                    testCSV([]string{"id", "name", "project_id", "organization_id", "done_at", "created_at", "updated_at"}, [][]string{{"task-1", "Task One", "project-1", "org-1", "", "2025-01-01T00:00:00Z", "2025-01-01T00:00:00Z"}}),
		"tags.csv":                     testCSV([]string{"id", "name", "organization_id", "created_at", "updated_at"}, [][]string{{"tag-1", "Deep Work", "org-1", "2025-01-01T00:00:00Z", "2025-01-01T00:00:00Z"}}),
		"time_entries.csv":             testCSV([]string{"id", "description", "start", "end", "billable_rate", "billable", "member_id", "user_id", "organization_id", "client_id", "project_id", "task_id", "tags", "is_imported", "still_active_email_sent_at", "created_at", "updated_at"}, [][]string{{"entry-1", "Work", "2025-02-01T09:00:00Z", "2025-02-01T10:00:00Z", "", "true", "member-1", "user-1", "org-1", "client-1", "project-1", "task-1", `["tag-1"]`, "false", "", "2025-02-01T09:00:00Z", "2025-02-01T10:00:00Z"}}),
	}

	path := t.TempDir() + "/solidtime-export.zip"
	output, err := os.Create(path)
	if err != nil {
		t.Fatalf("create zip: %v", err)
	}
	writer := zip.NewWriter(output)
	for name, body := range files {
		part, err := writer.Create(name)
		if err != nil {
			t.Fatalf("create zip entry: %v", err)
		}
		if _, err := part.Write(body); err != nil {
			t.Fatalf("write zip entry: %v", err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close zip writer: %v", err)
	}
	if err := output.Close(); err != nil {
		t.Fatalf("close zip file: %v", err)
	}

	zipBytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read zip: %v", err)
	}
	return zipBytes
}

func testCSV(headers []string, rows [][]string) []byte {
	var buffer bytes.Buffer
	writer := csv.NewWriter(&buffer)
	_ = writer.Write(headers)
	for _, row := range rows {
		_ = writer.Write(row)
	}
	writer.Flush()
	return buffer.Bytes()
}
