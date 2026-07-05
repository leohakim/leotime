package store

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

type TimeReportOptions struct {
	From              string
	To                string
	GroupBy           string
	IncludeTimestamps bool
	BillableOnly      bool
}

type TimeReportGroup struct {
	Key          string `json:"key"`
	Label        string `json:"label"`
	ProjectColor string `json:"projectColor,omitempty"`
	TotalSeconds int    `json:"totalSeconds"`
	EntryCount   int    `json:"entryCount"`
}

type TimeReport struct {
	From              string            `json:"from"`
	To                string            `json:"to"`
	GroupBy           string            `json:"groupBy"`
	IncludeTimestamps bool              `json:"includeTimestamps"`
	BillableOnly      bool              `json:"billableOnly"`
	TotalSeconds      int               `json:"totalSeconds"`
	EntryCount        int               `json:"entryCount"`
	Groups            []TimeReportGroup `json:"groups,omitempty"`
	Entries           []TimeEntry       `json:"entries,omitempty"`
}

func (s *Store) BuildTimeReport(ctx context.Context, userID string, options TimeReportOptions) (*TimeReport, error) {
	entries, err := s.ListTimeEntries(ctx, userID, TimeEntryListOptions{
		From: strings.TrimSpace(options.From),
		To:   strings.TrimSpace(options.To),
	})
	if err != nil {
		return nil, err
	}

	filtered := make([]TimeEntry, 0, len(entries))
	for _, entry := range entries {
		if options.BillableOnly && !entry.Billable {
			continue
		}
		filtered = append(filtered, entry)
	}

	report := &TimeReport{
		From:              strings.TrimSpace(options.From),
		To:                strings.TrimSpace(options.To),
		GroupBy:           normalizeReportGroupBy(options.GroupBy),
		IncludeTimestamps: options.IncludeTimestamps,
		BillableOnly:      options.BillableOnly,
		EntryCount:        len(filtered),
	}
	for _, entry := range filtered {
		report.TotalSeconds += entry.DurationSeconds
	}

	if options.IncludeTimestamps {
		report.Entries = filtered
		return report, nil
	}

	report.Groups = groupTimeReportEntries(filtered, report.GroupBy)
	return report, nil
}

func normalizeReportGroupBy(groupBy string) string {
	switch strings.TrimSpace(groupBy) {
	case "day", "client", "project", "task":
		return strings.TrimSpace(groupBy)
	default:
		return "project"
	}
}

func groupTimeReportEntries(entries []TimeEntry, groupBy string) []TimeReportGroup {
	type aggregate struct {
		key          string
		label        string
		projectColor string
		total        int
		count        int
		latest       string
	}

	buckets := map[string]*aggregate{}
	order := make([]string, 0)

	for _, entry := range entries {
		key, label := reportGroupKey(entry, groupBy)
		current, ok := buckets[key]
		if !ok {
			current = &aggregate{key: key, label: label}
			buckets[key] = current
			order = append(order, key)
		}
		current.total += entry.DurationSeconds
		current.count++
		if groupBy == "project" && entry.ProjectColor != "" {
			current.projectColor = entry.ProjectColor
		}
		if entry.StartedAt > current.latest {
			current.latest = entry.StartedAt
		}
	}

	sort.Slice(order, func(i, j int) bool {
		left := buckets[order[i]]
		right := buckets[order[j]]
		if left.latest != right.latest {
			return left.latest > right.latest
		}
		return left.label < right.label
	})

	groups := make([]TimeReportGroup, 0, len(order))
	for _, key := range order {
		item := buckets[key]
		group := TimeReportGroup{
			Key:          item.key,
			Label:        item.label,
			TotalSeconds: item.total,
			EntryCount:   item.count,
		}
		if groupBy == "project" {
			group.ProjectColor = item.projectColor
		}
		groups = append(groups, group)
	}
	return groups
}

func reportGroupKey(entry TimeEntry, groupBy string) (string, string) {
	switch groupBy {
	case "day":
		if len(entry.StartedAt) >= 10 {
			return entry.StartedAt[:10], entry.StartedAt[:10]
		}
		return "unknown", "unknown"
	case "client":
		if entry.ClientID == "" {
			return "", "Unassigned client"
		}
		return entry.ClientID, entry.ClientName
	case "task":
		if entry.TaskID == "" {
			return "", "Unassigned task"
		}
		return entry.TaskID, entry.TaskName
	default:
		if entry.ProjectID == "" {
			return "", "Unassigned project"
		}
		return entry.ProjectID, entry.ProjectName
	}
}

func FormatReportDuration(totalSeconds int) string {
	hours := totalSeconds / 3600
	minutes := (totalSeconds % 3600) / 60
	return fmt.Sprintf("%dh %02dmin", hours, minutes)
}
