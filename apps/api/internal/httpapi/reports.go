package httpapi

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/leotime/leotime/apps/api/internal/apierr"
	"github.com/leotime/leotime/apps/api/internal/store"
)

func (s *Server) getTimeReport(w http.ResponseWriter, r *http.Request, user *store.User) {
	options, ok := parseTimeReportOptions(w, r)
	if !ok {
		return
	}

	report, err := s.store.BuildTimeReport(r.Context(), user.ID, options)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "report_build_failed", "build report failed")
		return
	}
	writeJSON(w, http.StatusOK, report)
}

func (s *Server) exportTimeReport(w http.ResponseWriter, r *http.Request, user *store.User) {
	options, ok := parseTimeReportOptions(w, r)
	if !ok {
		return
	}

	format := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("format")))
	if format == "" {
		format = "csv"
	}

	report, err := s.store.BuildTimeReport(r.Context(), user.ID, options)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "report_build_failed", "build report failed")
		return
	}

	switch format {
	case "json":
		w.Header().Set("Content-Disposition", `attachment; filename="leotime-report.json"`)
		writeJSON(w, http.StatusOK, report)
	case "csv":
		payload, err := renderTimeReportCSV(report)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "report_render_failed", "render csv failed")
			return
		}
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="leotime-report.csv"`)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(payload)
	default:
		writeError(w, http.StatusBadRequest, "invalid_format", "format must be csv or json")
	}
}

func (s *Server) getDailySummary(w http.ResponseWriter, r *http.Request, user *store.User) {
	options, ok := s.parseDailySummaryOptions(w, r, user)
	if !ok {
		return
	}

	summary, err := s.store.BuildDailySummary(r.Context(), user.ID, options)
	if err != nil {
		if store.IsValidation(err, store.ErrInvalidProfileInput) || store.IsValidation(err, store.ErrInvalidTimeEntryInput) {
			writeValidationStoreError(w, err)
			return
		}
		writeError(w, http.StatusInternalServerError, "daily_summary_failed", "build daily summary failed")
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

func parseTimeReportOptions(w http.ResponseWriter, r *http.Request) (store.TimeReportOptions, bool) {
	from := strings.TrimSpace(r.URL.Query().Get("from"))
	to := strings.TrimSpace(r.URL.Query().Get("to"))
	if from == "" || to == "" {
		writeError(w, http.StatusBadRequest, "date_range_required", "from and to are required")
		return store.TimeReportOptions{}, false
	}

	fromTime, err := time.Parse(time.RFC3339, from)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, apierr.Validation("from", "invalid", "from must be RFC3339"))
		return store.TimeReportOptions{}, false
	}
	toTime, err := time.Parse(time.RFC3339, to)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, apierr.Validation("to", "invalid", "to must be RFC3339"))
		return store.TimeReportOptions{}, false
	}
	if toTime.Before(fromTime) {
		writeAPIError(w, http.StatusBadRequest, apierr.Validation("to", "invalid", "to must be on or after from"))
		return store.TimeReportOptions{}, false
	}

	includeTimestamps := strings.EqualFold(r.URL.Query().Get("includeTimestamps"), "true")
	billableOnly := strings.EqualFold(r.URL.Query().Get("billableOnly"), "true")

	return store.TimeReportOptions{
		From:              fromTime.UTC().Format(time.RFC3339),
		To:                toTime.UTC().Format(time.RFC3339),
		GroupBy:           r.URL.Query().Get("groupBy"),
		IncludeTimestamps: includeTimestamps,
		BillableOnly:      billableOnly,
	}, true
}

func (s *Server) parseDailySummaryOptions(w http.ResponseWriter, r *http.Request, user *store.User) (store.DailySummaryOptions, bool) {
	date := strings.TrimSpace(r.URL.Query().Get("date"))
	if date == "" {
		writeAPIError(w, http.StatusBadRequest, apierr.Validation("date", "required", "date is required"))
		return store.DailySummaryOptions{}, false
	}

	profile, err := s.store.ProfileByUserID(r.Context(), user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "profile_load_failed", "load profile failed")
		return store.DailySummaryOptions{}, false
	}

	includeClient := !strings.EqualFold(r.URL.Query().Get("includeClient"), "false")
	includeProject := !strings.EqualFold(r.URL.Query().Get("includeProject"), "false")
	includeClosing := !strings.EqualFold(r.URL.Query().Get("includeClosing"), "false")
	billableOnly := strings.EqualFold(r.URL.Query().Get("billableOnly"), "true")
	clientID, projectID := dailySummaryScopeFromQuery(r)
	manualNote := strings.TrimSpace(r.URL.Query().Get("manualNote"))

	return store.DailySummaryOptions{
		Date:           date,
		Timezone:       profile.Settings.Timezone,
		Locale:         profile.Locale,
		IncludeClient:  includeClient,
		IncludeProject: includeProject,
		IncludeClosing: includeClosing,
		BillableOnly:   billableOnly,
		ClientID:       clientID,
		ProjectID:      projectID,
		ManualNote:     manualNote,
	}, true
}

func renderTimeReportCSV(report *store.TimeReport) ([]byte, error) {
	buffer := &bytes.Buffer{}
	writer := csv.NewWriter(buffer)

	if report.IncludeTimestamps {
		if err := writer.Write([]string{
			"description", "client", "project", "task", "started_at", "ended_at", "duration_seconds", "billable", "tags",
		}); err != nil {
			return nil, err
		}
		for _, entry := range report.Entries {
			tagNames := make([]string, 0, len(entry.Tags))
			for _, tag := range entry.Tags {
				tagNames = append(tagNames, tag.Name)
			}
			if err := writer.Write([]string{
				entry.Description,
				entry.ClientName,
				entry.ProjectName,
				entry.TaskName,
				entry.StartedAt,
				entry.EndedAt,
				strconv.Itoa(entry.DurationSeconds),
				strconv.FormatBool(entry.Billable),
				strings.Join(tagNames, "; "),
			}); err != nil {
				return nil, err
			}
		}
	} else {
		if err := writer.Write([]string{"group", "label", "entry_count", "total_seconds", "total_duration"}); err != nil {
			return nil, err
		}
		for _, group := range report.Groups {
			if err := writer.Write([]string{
				report.GroupBy,
				group.Label,
				strconv.Itoa(group.EntryCount),
				strconv.Itoa(group.TotalSeconds),
				store.FormatReportDuration(group.TotalSeconds),
			}); err != nil {
				return nil, err
			}
		}
		if err := writer.Write([]string{
			"total",
			"All",
			strconv.Itoa(report.EntryCount),
			strconv.Itoa(report.TotalSeconds),
			store.FormatReportDuration(report.TotalSeconds),
		}); err != nil {
			return nil, err
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, fmt.Errorf("flush csv: %w", err)
	}
	return buffer.Bytes(), nil
}
