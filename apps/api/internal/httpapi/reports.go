package httpapi

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/leotime/leotime/apps/api/internal/store"
)

func (s *Server) getTimeReport(w http.ResponseWriter, r *http.Request, user *store.User) {
	options, ok := parseTimeReportOptions(w, r)
	if !ok {
		return
	}

	report, err := s.store.BuildTimeReport(r.Context(), user.ID, options)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "build report failed")
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
		writeError(w, http.StatusInternalServerError, "build report failed")
		return
	}

	switch format {
	case "json":
		w.Header().Set("Content-Disposition", `attachment; filename="leotime-report.json"`)
		writeJSON(w, http.StatusOK, report)
	case "csv":
		payload, err := renderTimeReportCSV(report)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "render csv failed")
			return
		}
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="leotime-report.csv"`)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(payload)
	default:
		writeError(w, http.StatusBadRequest, "format must be csv or json")
	}
}

func parseTimeReportOptions(w http.ResponseWriter, r *http.Request) (store.TimeReportOptions, bool) {
	from := strings.TrimSpace(r.URL.Query().Get("from"))
	to := strings.TrimSpace(r.URL.Query().Get("to"))
	if from == "" || to == "" {
		writeError(w, http.StatusBadRequest, "from and to are required")
		return store.TimeReportOptions{}, false
	}

	includeTimestamps := strings.EqualFold(r.URL.Query().Get("includeTimestamps"), "true")
	billableOnly := strings.EqualFold(r.URL.Query().Get("billableOnly"), "true")

	return store.TimeReportOptions{
		From:              from,
		To:                to,
		GroupBy:           r.URL.Query().Get("groupBy"),
		IncludeTimestamps: includeTimestamps,
		BillableOnly:      billableOnly,
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
