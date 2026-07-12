package store

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode"
)

type DailySummaryOptions struct {
	Date           string `json:"date"`
	Timezone       string `json:"timezone,omitempty"`
	Locale         string `json:"locale,omitempty"`
	IncludeClient  bool   `json:"includeClient"`
	IncludeProject bool   `json:"includeProject"`
	IncludeClosing bool   `json:"includeClosing"`
	BillableOnly   bool   `json:"billableOnly"`
	ManualNote     string `json:"manualNote,omitempty"`
	ClientID       string `json:"clientId"`
	ProjectID      string `json:"projectId"`
}

type DailySummary struct {
	Date         string `json:"date"`
	Locale       string `json:"locale"`
	Timezone     string `json:"timezone"`
	EntryCount   int    `json:"entryCount"`
	TotalSeconds int    `json:"totalSeconds"`
	Text         string `json:"text"`
}

type dailySummaryPeriod int

const (
	dailySummaryMorning dailySummaryPeriod = iota
	dailySummaryAfternoon
	dailySummaryEvening
)

type dailySummaryMessages struct {
	header         string
	emptyBody      string
	closing        string
	periodOpeners  [3]string
	periodFollow   []string
	contextWorked  string
	contextOn      string
	activityJoiner string
}

func (s *Store) BuildDailySummary(ctx context.Context, userID string, options DailySummaryOptions) (*DailySummary, error) {
	loc, err := time.LoadLocation(strings.TrimSpace(options.Timezone))
	if err != nil {
		return nil, validationError(ErrInvalidProfileInput, "timezone", "invalid", "timezone is invalid")
	}

	day, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(options.Date), loc)
	if err != nil {
		return nil, validationError(ErrInvalidTimeEntryInput, "date", "invalid", "date must be YYYY-MM-DD")
	}

	from := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, loc).UTC().Format(time.RFC3339)
	to := time.Date(day.Year(), day.Month(), day.Day(), 23, 59, 59, 0, loc).UTC().Format(time.RFC3339)

	entries, err := s.listTimeEntriesForReport(ctx, userID, TimeEntryListOptions{
		From: from,
		To:   to,
	})
	if err != nil {
		return nil, err
	}

	filtered := make([]TimeEntry, 0, len(entries))
	clientID := strings.TrimSpace(options.ClientID)
	projectID := strings.TrimSpace(options.ProjectID)
	for _, entry := range entries {
		if options.BillableOnly && !entry.Billable {
			continue
		}
		if projectID != "" && entry.ProjectID != projectID {
			continue
		}
		if clientID != "" && entry.ClientID != clientID {
			continue
		}
		filtered = append(filtered, entry)
	}

	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].StartedAt < filtered[j].StartedAt
	})

	locale := normalizeDailySummaryLocale(options.Locale)
	messages := dailySummaryMessagesFor(locale)
	totalSeconds := 0
	for _, entry := range filtered {
		totalSeconds += entry.DurationSeconds
	}

	summary := &DailySummary{
		Date:         day.Format("2006-01-02"),
		Locale:       locale,
		Timezone:     loc.String(),
		EntryCount:   len(filtered),
		TotalSeconds: totalSeconds,
	}

	var bodyLines []string
	if len(filtered) == 0 {
		bodyLines = []string{messages.emptyBody}
	} else {
		bodyLines = buildDailySummaryBody(filtered, options, loc, messages)
	}

	parts := []string{
		formatDailySummaryDateHeader(day, locale),
		messages.header,
		strings.Join(bodyLines, "\n"),
	}
	if options.IncludeClosing {
		parts = append(parts, messages.closing)
	}
	summary.Text = weaveManualNoteIntoSummary(strings.Join(parts, "\n"), options.ManualNote)
	return summary, nil
}

func weaveManualNoteIntoSummary(text, note string) string {
	note = strings.TrimSpace(note)
	if note == "" || strings.Contains(text, note) {
		return text
	}

	lines := strings.Split(text, "\n")
	if len(lines) == 0 {
		return note
	}

	last := strings.TrimSpace(lines[len(lines)-1])
	lowerLast := strings.ToLower(last)
	if strings.HasPrefix(lowerLast, "hasta ") || strings.HasPrefix(lowerLast, "see you") {
		lines = append(lines[:len(lines)-1], note, lines[len(lines)-1])
		return strings.Join(lines, "\n")
	}

	return text + "\n" + note
}

func normalizeDailySummaryLocale(locale string) string {
	if strings.EqualFold(strings.TrimSpace(locale), "en") {
		return "en"
	}
	return "es"
}

func dailySummaryMessagesFor(locale string) dailySummaryMessages {
	if locale == "en" {
		return dailySummaryMessages{
			header:         "Summary for today:",
			emptyBody:      "No time entries recorded for this day.",
			closing:        "See you tomorrow team!",
			periodOpeners:  [3]string{"This morning", "This afternoon", "This evening"},
			periodFollow:   []string{"Also", "Then", "After that"},
			contextWorked:  "worked on",
			contextOn:      "focused on",
			activityJoiner: " and ",
		}
	}
	return dailySummaryMessages{
		header:         "Resumen de hoy:",
		emptyBody:      "Sin entradas registradas hoy.",
		closing:        "Hasta mañana team!",
		periodOpeners:  [3]string{"Por la mañana", "Por la tarde", "Por la noche"},
		periodFollow:   []string{"También", "Luego", "Después"},
		contextWorked:  "avancé con",
		contextOn:      "estuve en",
		activityJoiner: " y ",
	}
}

func formatDailySummaryDateHeader(day time.Time, locale string) string {
	if locale == "en" {
		return fmt.Sprintf("%d/%d:", day.Month(), day.Day())
	}
	return fmt.Sprintf("%d/%d:", day.Day(), int(day.Month()))
}

func buildDailySummaryBody(entries []TimeEntry, options DailySummaryOptions, loc *time.Location, messages dailySummaryMessages) []string {
	type groupKey struct {
		period  dailySummaryPeriod
		context string
	}

	groups := make([]struct {
		key        groupKey
		activities []string
	}, 0)
	indexByKey := map[groupKey]int{}

	for _, entry := range entries {
		period := dailySummaryPeriodFor(entry.StartedAt, loc)
		contextLabel := dailySummaryContextLabel(entry, options)
		key := groupKey{period: period, context: contextLabel}
		activity := dailySummaryActivityText(entry)
		if idx, ok := indexByKey[key]; ok {
			groups[idx].activities = appendUniquePhrase(groups[idx].activities, activity)
			continue
		}
		indexByKey[key] = len(groups)
		activities := []string{}
		if activity != "" {
			activities = append(activities, activity)
		}
		groups = append(groups, struct {
			key        groupKey
			activities []string
		}{key: key, activities: activities})
	}

	lines := make([]string, 0, len(groups))
	periodUsed := map[dailySummaryPeriod]bool{}
	followIndex := 0

	for _, group := range groups {
		connector := messages.periodOpeners[group.key.period]
		if periodUsed[group.key.period] {
			if followIndex < len(messages.periodFollow) {
				connector = messages.periodFollow[followIndex]
				followIndex++
			} else {
				connector = messages.periodFollow[len(messages.periodFollow)-1]
			}
		} else {
			periodUsed[group.key.period] = true
		}

		activities := joinDailySummaryActivities(group.activities, messages.activityJoiner)
		line := dailySummarySentence(connector, group.key.context, activities, messages)
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

func dailySummaryPeriodFor(startedAt string, loc *time.Location) dailySummaryPeriod {
	parsed, err := time.Parse(time.RFC3339, startedAt)
	if err != nil {
		return dailySummaryMorning
	}
	hour := parsed.In(loc).Hour()
	switch {
	case hour < 14:
		return dailySummaryMorning
	case hour < 20:
		return dailySummaryAfternoon
	default:
		return dailySummaryEvening
	}
}

func dailySummaryContextLabel(entry TimeEntry, options DailySummaryOptions) string {
	parts := make([]string, 0, 2)
	if options.IncludeClient {
		if name := strings.TrimSpace(entry.ClientName); name != "" {
			parts = append(parts, name)
		}
	}
	if options.IncludeProject {
		if name := strings.TrimSpace(entry.ProjectName); name != "" {
			parts = append(parts, name)
		}
	}
	return strings.Join(parts, " — ")
}

func dailySummaryActivityText(entry TimeEntry) string {
	description := strings.TrimSpace(entry.Description)
	taskName := strings.TrimSpace(entry.TaskName)
	switch {
	case description != "" && taskName != "" && !strings.EqualFold(description, taskName):
		return taskName + ": " + description
	case description != "":
		return description
	case taskName != "":
		return taskName
	default:
		return ""
	}
}

func appendUniquePhrase(items []string, phrase string) []string {
	phrase = strings.TrimSpace(phrase)
	if phrase == "" {
		return items
	}
	for _, existing := range items {
		if strings.EqualFold(existing, phrase) {
			return items
		}
	}
	return append(items, phrase)
}

func joinDailySummaryActivities(items []string, joiner string) string {
	clean := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item != "" {
			clean = append(clean, item)
		}
	}
	if len(clean) == 0 {
		return ""
	}
	if len(clean) == 1 {
		return clean[0]
	}
	if strings.Contains(joiner, " y ") {
		return strings.Join(clean[:len(clean)-1], ", ") + joiner + clean[len(clean)-1]
	}
	return strings.Join(clean[:len(clean)-1], ", ") + joiner + clean[len(clean)-1]
}

func dailySummarySentence(connector, contextLabel, activities string, messages dailySummaryMessages) string {
	connector = strings.TrimSpace(connector)
	contextLabel = strings.TrimSpace(contextLabel)
	activities = strings.TrimSpace(activities)

	switch {
	case contextLabel != "" && activities != "":
		return fmt.Sprintf("%s %s %s: %s.", connector, messages.contextWorked, contextLabel, activities)
	case contextLabel != "":
		return fmt.Sprintf("%s %s %s.", connector, messages.contextOn, contextLabel)
	case activities != "":
		return fmt.Sprintf("%s %s.", connector, lowercaseFirst(activities))
	default:
		return ""
	}
}

func lowercaseFirst(value string) string {
	if value == "" {
		return value
	}
	runes := []rune(value)
	runes[0] = unicode.ToLower(runes[0])
	return string(runes)
}
