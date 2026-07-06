package solidtimeimport

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

const provider = "solidtime"

type queryer interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type Importer struct {
	db *sql.DB
}

type importState struct {
	db       queryer
	summary  Summary
	userID   string
	currency string
	now      string
	maps     map[string]map[string]mappedID
	tagLinks map[string][]string
}

type mappedID struct {
	internalID string
	exists     bool
}

func New(db *sql.DB) *Importer {
	return &Importer{db: db}
}

func (i *Importer) ImportFile(ctx context.Context, opts Options) (Summary, error) {
	if strings.TrimSpace(opts.FilePath) == "" {
		return Summary{}, errors.New("solidtime import file is required")
	}
	if strings.TrimSpace(opts.UserEmail) == "" {
		return Summary{}, errors.New("solidtime import user email is required")
	}

	export, err := ParseFile(opts.FilePath)
	if err != nil {
		return Summary{}, err
	}

	return i.Import(ctx, export, opts)
}

func (i *Importer) Import(ctx context.Context, export *Export, opts Options) (Summary, error) {
	userID, err := i.userIDByEmail(ctx, opts.UserEmail)
	if err != nil {
		return Summary{}, err
	}

	state := &importState{
		summary: Summary{
			Provider: provider,
			ExportID: export.Meta.ID,
			Version:  export.Meta.Version,
			DryRun:   opts.DryRun,
			Warnings: []string{},
			Errors:   []string{},
		},
		userID:   userID,
		currency: exportCurrency(export),
		now:      nowString(),
		maps:     map[string]map[string]mappedID{},
		tagLinks: map[string][]string{},
	}

	if opts.DryRun {
		state.db = i.db
		if err := state.prepareMappings(ctx, export); err != nil {
			state.summary.Errors = append(state.summary.Errors, err.Error())
			return state.summary, err
		}
		if err := state.validateReferences(export); err != nil {
			state.summary.Errors = append(state.summary.Errors, err.Error())
			return state.summary, err
		}
		state.countDryRun(export)
		return state.summary, nil
	}

	tx, err := i.db.BeginTx(ctx, nil)
	if err != nil {
		return Summary{}, fmt.Errorf("begin solidtime import: %w", err)
	}
	defer tx.Rollback()
	state.db = tx

	runID, err := newID("imp")
	if err != nil {
		return Summary{}, err
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO import_runs (id, provider, source_path, dry_run, status, started_at)
		VALUES (?, ?, ?, 0, 'running', ?)
	`, runID, provider, opts.FilePath, state.now); err != nil {
		return Summary{}, fmt.Errorf("create import run: %w", err)
	}

	if err := state.prepareMappings(ctx, export); err != nil {
		return failRun(ctx, tx, runID, state.summary, err)
	}
	if err := state.validateReferences(export); err != nil {
		return failRun(ctx, tx, runID, state.summary, err)
	}
	if err := state.write(ctx, export); err != nil {
		return failRun(ctx, tx, runID, state.summary, err)
	}

	summaryJSON, err := json.Marshal(state.summary)
	if err != nil {
		return failRun(ctx, tx, runID, state.summary, fmt.Errorf("marshal import summary: %w", err))
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE import_runs
		SET status = 'completed', summary_json = ?, finished_at = ?
		WHERE id = ?
	`, string(summaryJSON), nowString(), runID); err != nil {
		return Summary{}, fmt.Errorf("complete import run: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return Summary{}, fmt.Errorf("commit solidtime import: %w", err)
	}

	return state.summary, nil
}

func (i *Importer) userIDByEmail(ctx context.Context, email string) (string, error) {
	var userID string
	if err := i.db.QueryRowContext(ctx, `
		SELECT id
		FROM users
		WHERE lower(email) = lower(?)
	`, strings.TrimSpace(email)).Scan(&userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", fmt.Errorf("user %q not found", email)
		}
		return "", fmt.Errorf("query import user: %w", err)
	}
	return userID, nil
}

func (s *importState) prepareMappings(ctx context.Context, export *Export) error {
	for _, organization := range export.Organizations {
		if _, err := s.prepareID(ctx, "organization", organization.ID, "user", s.userID, "usr"); err != nil {
			return err
		}
	}
	for _, member := range export.Members {
		if _, err := s.prepareID(ctx, "member", member.ID, "user", s.userID, "usr"); err != nil {
			return err
		}
		if _, err := s.prepareID(ctx, "solidtime_user", member.UserID, "user", s.userID, "usr"); err != nil {
			return err
		}
	}
	for _, client := range export.Clients {
		if _, err := s.prepareID(ctx, "client", client.ID, "client", "", "cli"); err != nil {
			return err
		}
	}
	for _, project := range export.Projects {
		if _, err := s.prepareID(ctx, "project", project.ID, "project", "", "prj"); err != nil {
			return err
		}
	}
	for _, task := range export.Tasks {
		if _, err := s.prepareID(ctx, "task", task.ID, "task", "", "tsk"); err != nil {
			return err
		}
	}
	for _, tag := range export.Tags {
		if _, err := s.prepareID(ctx, "tag", tag.ID, "tag", "", "tag"); err != nil {
			return err
		}
	}
	for _, entry := range export.TimeEntries {
		if _, err := s.prepareID(ctx, "time_entry", entry.ID, "time_entry", "", "ten"); err != nil {
			return err
		}
		tags, err := parseTags(entry.Tags)
		if err != nil {
			return fmt.Errorf("parse tags for time entry %s: %w", entry.ID, err)
		}
		s.tagLinks[entry.ID] = tags
	}
	return nil
}

func (s *importState) prepareID(ctx context.Context, externalType string, externalID string, internalType string, fixedInternalID string, prefix string) (mappedID, error) {
	if strings.TrimSpace(externalID) == "" {
		return mappedID{}, fmt.Errorf("%s external id is required", externalType)
	}
	if s.maps[externalType] == nil {
		s.maps[externalType] = map[string]mappedID{}
	}

	var internalID string
	err := s.db.QueryRowContext(ctx, `
		SELECT internal_id
		FROM external_mappings
		WHERE provider = ? AND external_type = ? AND external_id = ?
	`, provider, externalType, externalID).Scan(&internalID)
	if err == nil {
		result := mappedID{internalID: internalID, exists: true}
		s.maps[externalType][externalID] = result
		return result, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return mappedID{}, fmt.Errorf("query external mapping %s/%s: %w", externalType, externalID, err)
	}

	if fixedInternalID != "" {
		internalID = fixedInternalID
	} else {
		generated, err := newID(prefix)
		if err != nil {
			return mappedID{}, err
		}
		internalID = generated
	}

	result := mappedID{internalID: internalID, exists: false}
	s.maps[externalType][externalID] = result
	return result, nil
}

func (s *importState) validateReferences(export *Export) error {
	organizations := stringSet{}
	for _, organization := range export.Organizations {
		organizations.add(organization.ID)
	}

	for _, client := range export.Clients {
		if !organizations.has(client.OrganizationID) {
			return fmt.Errorf("client %s references unknown organization %s", client.ID, client.OrganizationID)
		}
	}
	for _, project := range export.Projects {
		if !organizations.has(project.OrganizationID) {
			return fmt.Errorf("project %s references unknown organization %s", project.ID, project.OrganizationID)
		}
		if _, ok := s.maps["client"][project.ClientID]; !ok {
			return fmt.Errorf("project %s references unknown client %s", project.ID, project.ClientID)
		}
	}
	for _, task := range export.Tasks {
		if !organizations.has(task.OrganizationID) {
			return fmt.Errorf("task %s references unknown organization %s", task.ID, task.OrganizationID)
		}
		if _, ok := s.maps["project"][task.ProjectID]; !ok {
			return fmt.Errorf("task %s references unknown project %s", task.ID, task.ProjectID)
		}
	}
	for _, tag := range export.Tags {
		if !organizations.has(tag.OrganizationID) {
			return fmt.Errorf("tag %s references unknown organization %s", tag.ID, tag.OrganizationID)
		}
	}
	for _, entry := range export.TimeEntries {
		if !organizations.has(entry.OrganizationID) {
			return fmt.Errorf("time entry %s references unknown organization %s", entry.ID, entry.OrganizationID)
		}
		if strings.TrimSpace(entry.ClientID) != "" {
			if _, ok := s.maps["client"][entry.ClientID]; !ok {
				return fmt.Errorf("time entry %s references unknown client %s", entry.ID, entry.ClientID)
			}
		}
		if strings.TrimSpace(entry.ProjectID) != "" {
			if _, ok := s.maps["project"][entry.ProjectID]; !ok {
				return fmt.Errorf("time entry %s references unknown project %s", entry.ID, entry.ProjectID)
			}
		}
		if strings.TrimSpace(entry.TaskID) != "" {
			if _, ok := s.maps["task"][entry.TaskID]; !ok {
				return fmt.Errorf("time entry %s references unknown task %s", entry.ID, entry.TaskID)
			}
		}
		for _, tagID := range s.tagLinks[entry.ID] {
			if _, ok := s.maps["tag"][tagID]; !ok {
				return fmt.Errorf("time entry %s references unknown tag %s", entry.ID, tagID)
			}
		}
		if _, err := parseTime(entry.Start); err != nil {
			return fmt.Errorf("time entry %s has invalid start: %w", entry.ID, err)
		}
		if strings.TrimSpace(entry.End) != "" {
			if _, err := parseTime(entry.End); err != nil {
				return fmt.Errorf("time entry %s has invalid end: %w", entry.ID, err)
			}
		}
	}
	return nil
}

func (s *importState) countDryRun(export *Export) {
	s.summary.Organization = countStats(export.Organizations, s.maps["organization"])
	s.summary.Members = countStats(export.Members, s.maps["member"])
	s.summary.Clients = countStats(export.Clients, s.maps["client"])
	s.summary.Projects = countStats(export.Projects, s.maps["project"])
	s.summary.Tasks = countStats(export.Tasks, s.maps["task"])
	s.summary.Tags = countStats(export.Tags, s.maps["tag"])
	s.summary.TimeEntries = countStats(export.TimeEntries, s.maps["time_entry"])
}

func countStats[T interface{}](items []T, mappings map[string]mappedID) EntityStats {
	stats := EntityStats{Seen: len(items)}
	for _, mapping := range mappings {
		if mapping.exists {
			stats.Updated++
		} else {
			stats.Created++
		}
	}
	return stats
}

func (s *importState) write(ctx context.Context, export *Export) error {
	s.summary.Organization = countStats(export.Organizations, s.maps["organization"])
	s.summary.Members = countStats(export.Members, s.maps["member"])

	if err := s.writeIdentityMappings(ctx); err != nil {
		return err
	}
	if err := s.writeClients(ctx, export.Clients); err != nil {
		return err
	}
	if err := s.writeProjects(ctx, export.Projects); err != nil {
		return err
	}
	if err := s.writeTasks(ctx, export.Tasks); err != nil {
		return err
	}
	if err := s.writeTags(ctx, export.Tags); err != nil {
		return err
	}
	if err := s.writeTimeEntries(ctx, export.TimeEntries); err != nil {
		return err
	}
	return nil
}

func (s *importState) writeIdentityMappings(ctx context.Context) error {
	for _, externalType := range []string{"organization", "member", "solidtime_user"} {
		for externalID, mapping := range s.maps[externalType] {
			if err := s.writeMapping(ctx, externalType, externalID, "user", mapping); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *importState) writeClients(ctx context.Context, clients []Client) error {
	s.summary.Clients.Seen = len(clients)
	for _, client := range clients {
		mapping := s.maps["client"][client.ID]
		if mapping.exists {
			s.summary.Clients.Updated++
			if _, err := s.db.ExecContext(ctx, `
				UPDATE clients
				SET name = ?, default_currency = ?, archived_at = ?, updated_at = ?
				WHERE id = ? AND user_id = ?
			`, client.Name, s.currency, nullString(client.ArchivedAt), normalizeTimeOrNow(client.UpdatedAt), mapping.internalID, s.userID); err != nil {
				return fmt.Errorf("update client %s: %w", client.ID, err)
			}
			continue
		}

		s.summary.Clients.Created++
		if _, err := s.db.ExecContext(ctx, `
			INSERT INTO clients (id, user_id, name, default_currency, archived_at, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, mapping.internalID, s.userID, client.Name, s.currency, nullString(client.ArchivedAt), normalizeTimeOrNow(client.CreatedAt), normalizeTimeOrNow(client.UpdatedAt)); err != nil {
			return fmt.Errorf("insert client %s: %w", client.ID, err)
		}
		if err := s.writeMapping(ctx, "client", client.ID, "client", mapping); err != nil {
			return err
		}
	}
	return nil
}

func (s *importState) writeProjects(ctx context.Context, projects []Project) error {
	s.summary.Projects.Seen = len(projects)
	for _, project := range projects {
		mapping := s.maps["project"][project.ID]
		clientID := s.maps["client"][project.ClientID].internalID
		rate, err := parseRateMinor(project.BillableRate)
		if err != nil {
			return fmt.Errorf("project %s billable rate: %w", project.ID, err)
		}

		if mapping.exists {
			s.summary.Projects.Updated++
			if _, err := s.db.ExecContext(ctx, `
				UPDATE projects
				SET client_id = ?, name = ?, color = ?, default_hourly_rate_minor = ?, archived_at = ?, updated_at = ?
				WHERE id = ? AND user_id = ?
			`, clientID, project.Name, defaultString(project.Color, "#2563eb"), rate, nullString(project.ArchivedAt), normalizeTimeOrNow(project.UpdatedAt), mapping.internalID, s.userID); err != nil {
				return fmt.Errorf("update project %s: %w", project.ID, err)
			}
			continue
		}

		s.summary.Projects.Created++
		if _, err := s.db.ExecContext(ctx, `
			INSERT INTO projects (id, user_id, client_id, name, color, default_hourly_rate_minor, archived_at, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, mapping.internalID, s.userID, clientID, project.Name, defaultString(project.Color, "#2563eb"), rate, nullString(project.ArchivedAt), normalizeTimeOrNow(project.CreatedAt), normalizeTimeOrNow(project.UpdatedAt)); err != nil {
			return fmt.Errorf("insert project %s: %w", project.ID, err)
		}
		if err := s.writeMapping(ctx, "project", project.ID, "project", mapping); err != nil {
			return err
		}
	}
	return nil
}

func (s *importState) writeTasks(ctx context.Context, tasks []Task) error {
	s.summary.Tasks.Seen = len(tasks)
	for _, task := range tasks {
		mapping := s.maps["task"][task.ID]
		projectID := s.maps["project"][task.ProjectID].internalID

		if mapping.exists {
			s.summary.Tasks.Updated++
			if _, err := s.db.ExecContext(ctx, `
				UPDATE tasks
				SET project_id = ?, name = ?, archived_at = ?, updated_at = ?
				WHERE id = ? AND user_id = ?
			`, projectID, task.Name, nullString(task.DoneAt), normalizeTimeOrNow(task.UpdatedAt), mapping.internalID, s.userID); err != nil {
				return fmt.Errorf("update task %s: %w", task.ID, err)
			}
			continue
		}

		s.summary.Tasks.Created++
		if _, err := s.db.ExecContext(ctx, `
			INSERT INTO tasks (id, user_id, project_id, name, archived_at, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, mapping.internalID, s.userID, projectID, task.Name, nullString(task.DoneAt), normalizeTimeOrNow(task.CreatedAt), normalizeTimeOrNow(task.UpdatedAt)); err != nil {
			return fmt.Errorf("insert task %s: %w", task.ID, err)
		}
		if err := s.writeMapping(ctx, "task", task.ID, "task", mapping); err != nil {
			return err
		}
	}
	return nil
}

func (s *importState) writeTags(ctx context.Context, tags []Tag) error {
	s.summary.Tags.Seen = len(tags)
	for _, tag := range tags {
		mapping := s.maps["tag"][tag.ID]

		if mapping.exists {
			s.summary.Tags.Updated++
			if _, err := s.db.ExecContext(ctx, `
				UPDATE tags
				SET name = ?, updated_at = ?
				WHERE id = ? AND user_id = ?
			`, tag.Name, normalizeTimeOrNow(tag.UpdatedAt), mapping.internalID, s.userID); err != nil {
				return fmt.Errorf("update tag %s: %w", tag.ID, err)
			}
			continue
		}

		s.summary.Tags.Created++
		if _, err := s.db.ExecContext(ctx, `
			INSERT INTO tags (id, user_id, name, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?)
		`, mapping.internalID, s.userID, tag.Name, normalizeTimeOrNow(tag.CreatedAt), normalizeTimeOrNow(tag.UpdatedAt)); err != nil {
			return fmt.Errorf("insert tag %s: %w", tag.ID, err)
		}
		if err := s.writeMapping(ctx, "tag", tag.ID, "tag", mapping); err != nil {
			return err
		}
	}
	return nil
}

func (s *importState) writeTimeEntries(ctx context.Context, entries []TimeEntry) error {
	s.summary.TimeEntries.Seen = len(entries)
	for _, entry := range entries {
		mapping := s.maps["time_entry"][entry.ID]
		startedAt, err := parseTime(entry.Start)
		if err != nil {
			return fmt.Errorf("time entry %s start: %w", entry.ID, err)
		}
		endedAt, err := parseOptionalTime(entry.End)
		if err != nil {
			return fmt.Errorf("time entry %s end: %w", entry.ID, err)
		}

		durationSeconds := 0
		if endedAt.Valid {
			durationSeconds = int(endedAt.Time.Sub(startedAt).Seconds())
			if durationSeconds < 0 {
				return fmt.Errorf("time entry %s end is before start", entry.ID)
			}
		}

		overlap, err := s.hasOverlap(ctx, mapping.internalID, startedAt, endedAt)
		if err != nil {
			return err
		}
		if overlap {
			s.summary.Warnings = append(s.summary.Warnings, fmt.Sprintf("time entry %s overlaps another entry", entry.ID))
		}

		clientID := optionalMappedID(s.maps["client"], entry.ClientID)
		projectID := optionalMappedID(s.maps["project"], entry.ProjectID)
		taskID := optionalMappedID(s.maps["task"], entry.TaskID)
		billable, err := parseBool(entry.Billable)
		if err != nil {
			return fmt.Errorf("time entry %s billable: %w", entry.ID, err)
		}

		if mapping.exists {
			s.summary.TimeEntries.Updated++
			if _, err := s.db.ExecContext(ctx, `
				UPDATE time_entries
				SET client_id = ?, project_id = ?, task_id = ?, description = ?, started_at = ?, ended_at = ?,
					duration_seconds = ?, billable = ?, overlap_warning = ?, source = 'import', sync_state = 'synced', updated_at = ?
				WHERE id = ? AND user_id = ?
			`, clientID, projectID, taskID, entry.Description, formatTime(startedAt), optionalTimeString(endedAt), durationSeconds, boolInt(billable), boolInt(overlap), normalizeTimeOrNow(entry.UpdatedAt), mapping.internalID, s.userID); err != nil {
				return fmt.Errorf("update time entry %s: %w", entry.ID, err)
			}
		} else {
			s.summary.TimeEntries.Created++
			if _, err := s.db.ExecContext(ctx, `
				INSERT INTO time_entries (
					id, user_id, client_id, project_id, task_id, description, started_at, ended_at,
					duration_seconds, billable, overlap_warning, source, sync_state, created_at, updated_at
				)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'import', 'synced', ?, ?)
			`, mapping.internalID, s.userID, clientID, projectID, taskID, entry.Description, formatTime(startedAt), optionalTimeString(endedAt), durationSeconds, boolInt(billable), boolInt(overlap), normalizeTimeOrNow(entry.CreatedAt), normalizeTimeOrNow(entry.UpdatedAt)); err != nil {
				return fmt.Errorf("insert time entry %s: %w", entry.ID, err)
			}
			if err := s.writeMapping(ctx, "time_entry", entry.ID, "time_entry", mapping); err != nil {
				return err
			}
		}

		if _, err := s.db.ExecContext(ctx, "DELETE FROM time_entry_tags WHERE time_entry_id = ?", mapping.internalID); err != nil {
			return fmt.Errorf("clear tags for time entry %s: %w", entry.ID, err)
		}
		for _, tagID := range s.tagLinks[entry.ID] {
			if _, err := s.db.ExecContext(ctx, `
				INSERT INTO time_entry_tags (time_entry_id, tag_id)
				VALUES (?, ?)
			`, mapping.internalID, s.maps["tag"][tagID].internalID); err != nil {
				return fmt.Errorf("link tag %s to time entry %s: %w", tagID, entry.ID, err)
			}
		}
	}
	return nil
}

func (s *importState) hasOverlap(ctx context.Context, entryID string, startedAt time.Time, endedAt sql.NullTime) (bool, error) {
	endForQuery := "9999-12-31T23:59:59Z"
	if endedAt.Valid {
		endForQuery = formatTime(endedAt.Time)
	}

	var count int
	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM time_entries
		WHERE user_id = ?
			AND id != ?
			AND started_at < ?
			AND COALESCE(ended_at, '9999-12-31T23:59:59Z') > ?
	`, s.userID, entryID, endForQuery, formatTime(startedAt)).Scan(&count); err != nil {
		return false, fmt.Errorf("check overlap for time entry %s: %w", entryID, err)
	}
	return count > 0, nil
}

func (s *importState) writeMapping(ctx context.Context, externalType string, externalID string, internalType string, mapping mappedID) error {
	if mapping.exists {
		if _, err := s.db.ExecContext(ctx, `
			UPDATE external_mappings
			SET internal_type = ?, internal_id = ?, updated_at = ?
			WHERE provider = ? AND external_type = ? AND external_id = ?
		`, internalType, mapping.internalID, s.now, provider, externalType, externalID); err != nil {
			return fmt.Errorf("update external mapping %s/%s: %w", externalType, externalID, err)
		}
		return nil
	}

	mappingID, err := newID("map")
	if err != nil {
		return err
	}
	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO external_mappings (id, provider, external_type, external_id, internal_type, internal_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, mappingID, provider, externalType, externalID, internalType, mapping.internalID, s.now, s.now); err != nil {
		return fmt.Errorf("insert external mapping %s/%s: %w", externalType, externalID, err)
	}
	return nil
}

func failRun(ctx context.Context, tx *sql.Tx, runID string, summary Summary, err error) (Summary, error) {
	summary.Errors = append(summary.Errors, err.Error())
	body, marshalErr := json.Marshal(summary)
	if marshalErr == nil {
		_, _ = tx.ExecContext(ctx, `
			UPDATE import_runs
			SET status = 'failed', summary_json = ?, finished_at = ?, error = ?
			WHERE id = ?
		`, string(body), nowString(), err.Error(), runID)
	}
	return summary, err
}

type stringSet map[string]struct{}

func (s stringSet) add(value string) {
	s[value] = struct{}{}
}

func (s stringSet) has(value string) bool {
	_, ok := s[value]
	return ok
}

func optionalMappedID(mappings map[string]mappedID, externalID string) sql.NullString {
	externalID = strings.TrimSpace(externalID)
	if externalID == "" {
		return sql.NullString{}
	}
	mapping, ok := mappings[externalID]
	if !ok {
		return sql.NullString{}
	}
	return sql.NullString{String: mapping.internalID, Valid: true}
}

func parseTags(raw string) ([]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}

	var ids []string
	if err := json.Unmarshal([]byte(raw), &ids); err == nil {
		return ids, nil
	}

	var objects []struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(raw), &objects); err != nil {
		return nil, err
	}
	for _, object := range objects {
		if object.ID != "" {
			ids = append(ids, object.ID)
		}
	}
	return ids, nil
}

func parseBool(value string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "true", "1":
		return true, nil
	case "false", "0":
		return false, nil
	default:
		return false, fmt.Errorf("invalid boolean %q", value)
	}
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func parseRateMinor(value string) (sql.NullInt64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return sql.NullInt64{}, nil
	}
	if strings.Contains(value, ".") {
		floatValue, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return sql.NullInt64{}, err
		}
		return sql.NullInt64{Int64: int64(math.Round(floatValue * 100)), Valid: true}, nil
	}
	intValue, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return sql.NullInt64{}, err
	}
	return sql.NullInt64{Int64: intValue, Valid: true}, nil
}

func parseTime(value string) (time.Time, error) {
	parsed, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(value))
	if err != nil {
		return time.Time{}, err
	}
	return parsed.UTC(), nil
}

func parseOptionalTime(value string) (sql.NullTime, error) {
	if strings.TrimSpace(value) == "" {
		return sql.NullTime{}, nil
	}
	parsed, err := parseTime(value)
	if err != nil {
		return sql.NullTime{}, err
	}
	return sql.NullTime{Time: parsed, Valid: true}, nil
}

func formatTime(value time.Time) string {
	return value.UTC().Format(time.RFC3339Nano)
}

func normalizeTimeOrNow(value string) string {
	if parsed, err := parseTime(value); err == nil {
		return formatTime(parsed)
	}
	return nowString()
}

func nullString(value string) sql.NullString {
	value = strings.TrimSpace(value)
	if value == "" {
		return sql.NullString{}
	}
	if parsed, err := parseTime(value); err == nil {
		return sql.NullString{String: formatTime(parsed), Valid: true}
	}
	return sql.NullString{String: value, Valid: true}
}

func optionalTimeString(value sql.NullTime) sql.NullString {
	if !value.Valid {
		return sql.NullString{}
	}
	return sql.NullString{String: formatTime(value.Time), Valid: true}
}

func defaultString(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func exportCurrency(export *Export) string {
	for _, organization := range export.Organizations {
		if strings.TrimSpace(organization.Currency) != "" {
			return strings.ToUpper(strings.TrimSpace(organization.Currency))
		}
	}
	return "EUR"
}

func nowString() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}

func newID(prefix string) (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate id: %w", err)
	}
	return prefix + "_" + hex.EncodeToString(bytes), nil
}
