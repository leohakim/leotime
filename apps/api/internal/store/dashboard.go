package store

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

const dashboardRecentLimit = 5

var ErrInvalidDashboardInput = errors.New("invalid dashboard input")

type DashboardRecentEntry struct {
	ID              string `json:"id"`
	ClientID        string `json:"clientId"`
	ProjectID       string `json:"projectId"`
	ProjectName     string `json:"projectName"`
	ProjectColor    string `json:"projectColor"`
	TaskID          string `json:"taskId"`
	Description     string `json:"description"`
	StartedAt       string `json:"startedAt"`
	DurationSeconds int    `json:"durationSeconds"`
	Billable        bool   `json:"billable"`
}

type DashboardDaySummary struct {
	Date         string `json:"date"`
	Label        string `json:"label"`
	TotalSeconds int    `json:"totalSeconds"`
}

type DashboardHeatmapDay struct {
	Date         string `json:"date"`
	TotalSeconds int    `json:"totalSeconds"`
	Level        int    `json:"level"`
	InMonth      bool   `json:"inMonth"`
}

type DashboardWeekDay struct {
	Date         string `json:"date"`
	Weekday      string `json:"weekday"`
	TotalSeconds int    `json:"totalSeconds"`
}

type DashboardProjectShare struct {
	ProjectID    string `json:"projectId"`
	ProjectName  string `json:"projectName"`
	ProjectColor string `json:"projectColor"`
	TotalSeconds int    `json:"totalSeconds"`
}

type DashboardStats struct {
	ActivityMonth       string                  `json:"activityMonth"`
	RecentEntries       []DashboardRecentEntry  `json:"recentEntries"`
	LastSevenDays       []DashboardDaySummary   `json:"lastSevenDays"`
	ActivityHeatmap     []DashboardHeatmapDay   `json:"activityHeatmap"`
	WeekDays            []DashboardWeekDay      `json:"weekDays"`
	WeekSpentSeconds    int                     `json:"weekSpentSeconds"`
	WeekBillableSeconds int                     `json:"weekBillableSeconds"`
	WeekBillableMinor   int64                   `json:"weekBillableMinor"`
	WeekCurrency        string                  `json:"weekCurrency"`
	ProjectBreakdown    []DashboardProjectShare `json:"projectBreakdown"`
}

type dashboardEntryRow struct {
	ID               string
	ClientID         string
	ClientRateMinor  int64
	ClientCurrency   string
	ProjectID        string
	ProjectName      string
	ProjectColor     string
	ProjectRateMinor *int64
	TaskID           string
	Description      string
	StartedAt        string
	DurationSeconds  int
	Billable         bool
}

func (s *Store) BuildDashboardStats(ctx context.Context, userID string, activityMonth string) (*DashboardStats, error) {
	monthStart, err := parseActivityMonth(activityMonth)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	today := dateOnlyUTC(now)
	weekStart := startOfWeekUTC(today, time.Monday)
	weekEnd := weekStart.AddDate(0, 0, 6)
	lastSevenStart := today.AddDate(0, 0, -6)
	gridStart := startOfWeekUTC(monthStart, time.Monday)

	queryStart := weekStart
	if gridStart.Before(queryStart) {
		queryStart = gridStart
	}
	if lastSevenStart.Before(queryStart) {
		queryStart = lastSevenStart
	}

	rows, err := s.listDashboardEntries(ctx, userID, queryStart.Format(time.RFC3339Nano))
	if err != nil {
		return nil, err
	}

	stats := &DashboardStats{
		ActivityMonth:    monthStart.Format("2006-01"),
		RecentEntries:    make([]DashboardRecentEntry, 0, dashboardRecentLimit),
		LastSevenDays:    buildLastSevenDays(lastSevenStart, today),
		ActivityHeatmap:  buildActivityMonthGrid(monthStart, map[string]int{}),
		WeekDays:         buildWeekDays(weekStart),
		WeekCurrency:     "EUR",
		ProjectBreakdown: []DashboardProjectShare{},
	}

	dayTotals := map[string]int{}
	projectTotals := map[string]*DashboardProjectShare{}

	for _, row := range rows {
		dayKey := entryDayKey(row.StartedAt)
		dayTotals[dayKey] += row.DurationSeconds

		if dayKey >= weekStart.Format("2006-01-02") && dayKey <= weekEnd.Format("2006-01-02") {
			stats.WeekSpentSeconds += row.DurationSeconds
			if row.Billable {
				stats.WeekBillableSeconds += row.DurationSeconds
				stats.WeekBillableMinor += billableEntryMinor(row)
				if row.ClientCurrency != "" {
					stats.WeekCurrency = strings.ToUpper(row.ClientCurrency)
				}
			}

			projectKey := row.ProjectID
			if projectKey == "" {
				projectKey = "_none"
			}
			share, ok := projectTotals[projectKey]
			if !ok {
				label := row.ProjectName
				color := row.ProjectColor
				if projectKey == "_none" {
					label = ""
					color = "#64748b"
				}
				share = &DashboardProjectShare{
					ProjectID:    row.ProjectID,
					ProjectName:  label,
					ProjectColor: color,
				}
				projectTotals[projectKey] = share
			}
			share.TotalSeconds += row.DurationSeconds
		}
	}

	for index, day := range stats.LastSevenDays {
		stats.LastSevenDays[index].TotalSeconds = dayTotals[day.Date]
	}

	stats.ActivityHeatmap = buildActivityMonthGrid(monthStart, dayTotals)

	for index, day := range stats.WeekDays {
		stats.WeekDays[index].TotalSeconds = dayTotals[day.Date]
	}

	for index, row := range rows {
		if index >= dashboardRecentLimit {
			break
		}
		stats.RecentEntries = append(stats.RecentEntries, DashboardRecentEntry{
			ID:              row.ID,
			ClientID:        row.ClientID,
			ProjectID:       row.ProjectID,
			ProjectName:     row.ProjectName,
			ProjectColor:    row.ProjectColor,
			TaskID:          row.TaskID,
			Description:     row.Description,
			StartedAt:       row.StartedAt,
			DurationSeconds: row.DurationSeconds,
			Billable:        row.Billable,
		})
	}

	stats.ProjectBreakdown = sortedProjectShares(projectTotals)
	return stats, nil
}

func (s *Store) listDashboardEntries(ctx context.Context, userID string, from string) ([]dashboardEntryRow, error) {
	query := `
		SELECT te.id, COALESCE(te.client_id, ''), COALESCE(c.default_hourly_rate_minor, 0),
			COALESCE(c.default_currency, 'EUR'), COALESCE(te.project_id, ''), COALESCE(p.name, ''),
			COALESCE(p.color, ''), p.default_hourly_rate_minor, COALESCE(te.task_id, ''), te.description,
			te.started_at, te.duration_seconds, te.billable
		FROM time_entries te
		LEFT JOIN clients c ON c.id = te.client_id AND c.user_id = te.user_id
		LEFT JOIN projects p ON p.id = te.project_id AND p.user_id = te.user_id
		WHERE te.user_id = ? AND te.ended_at IS NOT NULL AND te.started_at >= ?
		ORDER BY te.started_at DESC
	`

	result, err := s.db.QueryContext(ctx, query, userID, from)
	if err != nil {
		return nil, fmt.Errorf("list dashboard entries: %w", err)
	}
	defer result.Close()

	rows := make([]dashboardEntryRow, 0)
	for result.Next() {
		row := dashboardEntryRow{}
		var billable int
		if err := result.Scan(
			&row.ID, &row.ClientID, &row.ClientRateMinor, &row.ClientCurrency, &row.ProjectID,
			&row.ProjectName, &row.ProjectColor, &row.ProjectRateMinor, &row.TaskID, &row.Description,
			&row.StartedAt, &row.DurationSeconds, &billable,
		); err != nil {
			return nil, fmt.Errorf("scan dashboard entry: %w", err)
		}
		row.Billable = billable == 1
		rows = append(rows, row)
	}
	if err := result.Err(); err != nil {
		return nil, fmt.Errorf("iterate dashboard entries: %w", err)
	}
	return rows, nil
}

func billableEntryMinor(row dashboardEntryRow) int64 {
	minutes := row.DurationSeconds / 60
	if minutes <= 0 {
		return 0
	}
	rate := row.ClientRateMinor
	if row.ProjectRateMinor != nil {
		rate = *row.ProjectRateMinor
	}
	return lineSubtotalMinor(minutes, rate)
}

func buildLastSevenDays(start time.Time, end time.Time) []DashboardDaySummary {
	days := make([]DashboardDaySummary, 0, 7)
	for current := end; !current.Before(start); current = current.AddDate(0, 0, -1) {
		days = append(days, DashboardDaySummary{
			Date:  current.Format("2006-01-02"),
			Label: relativeDayLabel(current, end),
		})
	}
	return days
}

func buildActivityMonthGrid(monthStart time.Time, dayTotals map[string]int) []DashboardHeatmapDay {
	monthEnd := endOfMonthUTC(monthStart)
	gridStart := startOfWeekUTC(monthStart, time.Monday)
	gridEnd := startOfWeekUTC(monthEnd, time.Monday).AddDate(0, 0, 6)

	days := make([]DashboardHeatmapDay, 0)
	for current := gridStart; !current.After(gridEnd); current = current.AddDate(0, 0, 1) {
		key := current.Format("2006-01-02")
		inMonth := !current.Before(monthStart) && !current.After(monthEnd)
		seconds := 0
		level := 0
		if inMonth {
			seconds = dayTotals[key]
			level = heatmapLevel(seconds)
		}
		days = append(days, DashboardHeatmapDay{
			Date:         key,
			TotalSeconds: seconds,
			Level:        level,
			InMonth:      inMonth,
		})
	}
	return days
}

func parseActivityMonth(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		now := time.Now().UTC()
		return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC), nil
	}
	parsed, err := time.Parse("2006-01", value)
	if err != nil {
		return time.Time{}, fmt.Errorf("%w: activityMonth must be YYYY-MM", ErrInvalidDashboardInput)
	}
	return time.Date(parsed.Year(), parsed.Month(), 1, 0, 0, 0, 0, time.UTC), nil
}

func endOfMonthUTC(monthStart time.Time) time.Time {
	return monthStart.AddDate(0, 1, 0).AddDate(0, 0, -1)
}

func buildWeekDays(weekStart time.Time) []DashboardWeekDay {
	labels := []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}
	days := make([]DashboardWeekDay, 0, 7)
	for index := 0; index < 7; index += 1 {
		current := weekStart.AddDate(0, 0, index)
		days = append(days, DashboardWeekDay{
			Date:    current.Format("2006-01-02"),
			Weekday: labels[index],
		})
	}
	return days
}

func sortedProjectShares(shares map[string]*DashboardProjectShare) []DashboardProjectShare {
	items := make([]DashboardProjectShare, 0, len(shares))
	for _, share := range shares {
		if share.TotalSeconds <= 0 {
			continue
		}
		items = append(items, *share)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].TotalSeconds == items[j].TotalSeconds {
			return items[i].ProjectName < items[j].ProjectName
		}
		return items[i].TotalSeconds > items[j].TotalSeconds
	})
	return items
}

func heatmapLevel(totalSeconds int) int {
	switch {
	case totalSeconds <= 0:
		return 0
	case totalSeconds < 2*3600:
		return 1
	case totalSeconds < 4*3600:
		return 2
	case totalSeconds < 6*3600:
		return 3
	default:
		return 4
	}
}

func relativeDayLabel(day time.Time, today time.Time) string {
	switch day.Format("2006-01-02") {
	case today.Format("2006-01-02"):
		return "today"
	case today.AddDate(0, 0, -1).Format("2006-01-02"):
		return "yesterday"
	default:
		offset := int(today.Sub(day).Hours() / 24)
		if offset < 0 {
			offset = 0
		}
		return fmt.Sprintf("%dd", offset)
	}
}

func entryDayKey(startedAt string) string {
	parsed, err := time.Parse(time.RFC3339Nano, startedAt)
	if err != nil {
		parsed, err = time.Parse(time.RFC3339, startedAt)
		if err != nil {
			return strings.Split(startedAt, "T")[0]
		}
	}
	return parsed.UTC().Format("2006-01-02")
}

func dateOnlyUTC(value time.Time) time.Time {
	year, month, day := value.UTC().Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func startOfWeekUTC(day time.Time, weekday time.Weekday) time.Time {
	normalized := dateOnlyUTC(day)
	offset := (int(normalized.Weekday()) - int(weekday) + 7) % 7
	return normalized.AddDate(0, 0, -offset)
}
