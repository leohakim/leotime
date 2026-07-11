package solidtimeimport

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"strings"
)

const (
	maxSolidtimeZipEntries        = 16
	maxSolidtimeMetaJSONBytes     = 1 << 20
	maxSolidtimeCSVBytes          = 32 << 20
	maxSolidtimeUncompressedBytes = 128 << 20
)

var requiredHeaders = map[string][]string{
	"organizations.csv":            {"id", "name", "billable_rate", "currency", "created_at", "updated_at"},
	"organization_invitations.csv": {"id", "email", "organization_id", "role", "created_at", "updated_at"},
	"time_entries.csv":             {"id", "description", "start", "end", "billable_rate", "billable", "member_id", "user_id", "organization_id", "client_id", "project_id", "task_id", "tags", "is_imported", "still_active_email_sent_at", "created_at", "updated_at"},
	"clients.csv":                  {"id", "name", "organization_id", "archived_at", "created_at", "updated_at"},
	"projects.csv":                 {"id", "name", "color", "billable_rate", "is_public", "client_id", "organization_id", "is_billable", "archived_at", "created_at", "updated_at"},
	"project_members.csv":          {"id", "billable_rate", "project_id", "user_id", "member_id", "created_at", "updated_at"},
	"members.csv":                  {"id", "user_id", "name", "email", "organization_id", "billable_rate", "role", "created_at", "updated_at"},
	"tasks.csv":                    {"id", "name", "project_id", "organization_id", "done_at", "created_at", "updated_at"},
	"tags.csv":                     {"id", "name", "organization_id", "created_at", "updated_at"},
}

var allowedExportFiles = func() map[string]struct{} {
	files := make(map[string]struct{}, len(requiredHeaders)+1)
	files["meta.json"] = struct{}{}
	for name := range requiredHeaders {
		files[name] = struct{}{}
	}
	return files
}()

func ParseFile(path string) (*Export, error) {
	reader, err := zip.OpenReader(path)
	if err != nil {
		return nil, fmt.Errorf("open solidtime zip: %w", err)
	}
	defer reader.Close()

	files := make(map[string][]byte, len(allowedExportFiles))
	seen := make(map[string]struct{}, len(allowedExportFiles))
	var totalUncompressed int64
	entryCount := 0

	for _, file := range reader.File {
		if file.FileInfo().IsDir() {
			continue
		}
		entryCount++
		if entryCount > maxSolidtimeZipEntries {
			return nil, fmt.Errorf("solidtime export exceeds %d zip entries", maxSolidtimeZipEntries)
		}

		name, err := validateZipEntryName(file.Name)
		if err != nil {
			return nil, err
		}
		if _, ok := allowedExportFiles[name]; !ok {
			return nil, fmt.Errorf("solidtime export has unexpected file %q", name)
		}
		if _, ok := seen[name]; ok {
			return nil, fmt.Errorf("solidtime export has duplicate file %q", name)
		}
		seen[name] = struct{}{}

		limit := int64(maxSolidtimeCSVBytes)
		if name == "meta.json" {
			limit = maxSolidtimeMetaJSONBytes
		}

		body, err := readZipFileLimited(file, limit)
		if err != nil {
			return nil, err
		}
		totalUncompressed += int64(len(body))
		if totalUncompressed > maxSolidtimeUncompressedBytes {
			return nil, fmt.Errorf("solidtime export exceeds %d bytes uncompressed", maxSolidtimeUncompressedBytes)
		}
		files[name] = body
	}

	return Parse(files)
}

func Parse(files map[string][]byte) (*Export, error) {
	if err := validateExportFileSet(files); err != nil {
		return nil, err
	}

	metaBytes := files["meta.json"]

	var meta Meta
	if err := json.Unmarshal(metaBytes, &meta); err != nil {
		return nil, fmt.Errorf("parse meta.json: %w", err)
	}
	if meta.Version != "1.0" {
		return nil, fmt.Errorf("unsupported solidtime export version %q", meta.Version)
	}

	organizations, err := readCSV(files, "organizations.csv", func(row map[string]string) (Organization, error) {
		return Organization{
			ID:           row["id"],
			Name:         row["name"],
			BillableRate: row["billable_rate"],
			Currency:     row["currency"],
			CreatedAt:    row["created_at"],
			UpdatedAt:    row["updated_at"],
		}, nil
	})
	if err != nil {
		return nil, err
	}

	members, err := readCSV(files, "members.csv", func(row map[string]string) (Member, error) {
		return Member{
			ID:             row["id"],
			UserID:         row["user_id"],
			Name:           row["name"],
			Email:          row["email"],
			OrganizationID: row["organization_id"],
			BillableRate:   row["billable_rate"],
			Role:           row["role"],
			CreatedAt:      row["created_at"],
			UpdatedAt:      row["updated_at"],
		}, nil
	})
	if err != nil {
		return nil, err
	}

	clients, err := readCSV(files, "clients.csv", func(row map[string]string) (Client, error) {
		return Client{
			ID:             row["id"],
			Name:           row["name"],
			OrganizationID: row["organization_id"],
			ArchivedAt:     row["archived_at"],
			CreatedAt:      row["created_at"],
			UpdatedAt:      row["updated_at"],
		}, nil
	})
	if err != nil {
		return nil, err
	}

	projects, err := readCSV(files, "projects.csv", func(row map[string]string) (Project, error) {
		return Project{
			ID:             row["id"],
			Name:           row["name"],
			Color:          row["color"],
			BillableRate:   row["billable_rate"],
			IsPublic:       row["is_public"],
			ClientID:       row["client_id"],
			OrganizationID: row["organization_id"],
			IsBillable:     row["is_billable"],
			ArchivedAt:     row["archived_at"],
			CreatedAt:      row["created_at"],
			UpdatedAt:      row["updated_at"],
		}, nil
	})
	if err != nil {
		return nil, err
	}

	tasks, err := readCSV(files, "tasks.csv", func(row map[string]string) (Task, error) {
		return Task{
			ID:             row["id"],
			Name:           row["name"],
			ProjectID:      row["project_id"],
			OrganizationID: row["organization_id"],
			DoneAt:         row["done_at"],
			CreatedAt:      row["created_at"],
			UpdatedAt:      row["updated_at"],
		}, nil
	})
	if err != nil {
		return nil, err
	}

	tags, err := readCSV(files, "tags.csv", func(row map[string]string) (Tag, error) {
		return Tag{
			ID:             row["id"],
			Name:           row["name"],
			OrganizationID: row["organization_id"],
			CreatedAt:      row["created_at"],
			UpdatedAt:      row["updated_at"],
		}, nil
	})
	if err != nil {
		return nil, err
	}

	timeEntries, err := readCSV(files, "time_entries.csv", func(row map[string]string) (TimeEntry, error) {
		return TimeEntry{
			ID:                     row["id"],
			Description:            row["description"],
			Start:                  row["start"],
			End:                    row["end"],
			BillableRate:           row["billable_rate"],
			Billable:               row["billable"],
			MemberID:               row["member_id"],
			UserID:                 row["user_id"],
			OrganizationID:         row["organization_id"],
			ClientID:               row["client_id"],
			ProjectID:              row["project_id"],
			TaskID:                 row["task_id"],
			Tags:                   row["tags"],
			IsImported:             row["is_imported"],
			StillActiveEmailSentAt: row["still_active_email_sent_at"],
			CreatedAt:              row["created_at"],
			UpdatedAt:              row["updated_at"],
		}, nil
	})
	if err != nil {
		return nil, err
	}

	for _, name := range []string{"organization_invitations.csv", "project_members.csv"} {
		if err := validateCSVHeaders(files, name); err != nil {
			return nil, err
		}
	}

	return &Export{
		Meta:          meta,
		Organizations: organizations,
		Members:       members,
		Clients:       clients,
		Projects:      projects,
		Tasks:         tasks,
		Tags:          tags,
		TimeEntries:   timeEntries,
	}, nil
}

func validateExportFileSet(files map[string][]byte) error {
	for name := range files {
		if _, ok := allowedExportFiles[name]; !ok {
			return fmt.Errorf("solidtime export has unexpected file %q", name)
		}
	}
	for name := range allowedExportFiles {
		if _, ok := files[name]; !ok {
			return fmt.Errorf("solidtime export missing %s", name)
		}
	}
	return nil
}

func validateZipEntryName(name string) (string, error) {
	clean := strings.TrimSpace(strings.ReplaceAll(name, "\\", "/"))
	if clean == "" {
		return "", fmt.Errorf("solidtime export has empty zip entry name")
	}
	if strings.Contains(clean, "\x00") {
		return "", fmt.Errorf("solidtime export has invalid zip entry name %q", name)
	}
	if path.IsAbs(clean) || strings.HasPrefix(clean, "/") {
		return "", fmt.Errorf("solidtime export has absolute zip entry %q", name)
	}
	base := path.Base(clean)
	if base != clean || strings.Contains(clean, "..") {
		return "", fmt.Errorf("solidtime export has traversal-like zip entry %q", name)
	}
	return base, nil
}

func readZipFileLimited(file *zip.File, limit int64) ([]byte, error) {
	reader, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("open %s in solidtime zip: %w", file.Name, err)
	}
	defer reader.Close()

	limited := io.LimitReader(reader, limit+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("read %s in solidtime zip: %w", file.Name, err)
	}
	if int64(len(body)) > limit {
		return nil, fmt.Errorf("solidtime export file %q exceeds %d bytes", path.Base(file.Name), limit)
	}
	return body, nil
}

func validateCSVHeaders(files map[string][]byte, name string) error {
	_, err := readCSV(files, name, func(row map[string]string) (struct{}, error) {
		return struct{}{}, nil
	})
	return err
}

func readCSV[T any](files map[string][]byte, name string, decode func(map[string]string) (T, error)) ([]T, error) {
	body := bytes.TrimPrefix(files[name], []byte{0xEF, 0xBB, 0xBF})
	reader := csv.NewReader(bytes.NewReader(body))
	reader.FieldsPerRecord = -1

	headers, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("read %s headers: %w", name, err)
	}
	if !sameStrings(headers, requiredHeaders[name]) {
		return nil, fmt.Errorf("unexpected %s headers: got %s", name, strings.Join(headers, ","))
	}

	var rows []T
	for line := 2; ; line++ {
		values, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read %s line %d: %w", name, line, err)
		}
		if len(values) != len(headers) {
			return nil, fmt.Errorf("read %s line %d: expected %d fields, got %d", name, line, len(headers), len(values))
		}

		row := make(map[string]string, len(headers))
		for index, header := range headers {
			row[header] = values[index]
		}

		decoded, err := decode(row)
		if err != nil {
			return nil, fmt.Errorf("decode %s line %d: %w", name, line, err)
		}
		rows = append(rows, decoded)
	}

	return rows, nil
}

func sameStrings(first []string, second []string) bool {
	if len(first) != len(second) {
		return false
	}
	for index := range first {
		if first[index] != second[index] {
			return false
		}
	}
	return true
}

// SourcePathBasename returns only the ZIP filename for import audit rows.
func SourcePathBasename(filePath string) string {
	clean := strings.TrimSpace(strings.ReplaceAll(filePath, "\\", "/"))
	base := path.Base(clean)
	if base == "." || base == "/" || base == "" {
		return "solidtime-export.zip"
	}
	return base
}
