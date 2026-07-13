package store

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
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
	Date         string                  `json:"date"`
	Locale       string                  `json:"locale"`
	Timezone     string                  `json:"timezone"`
	EntryCount   int                     `json:"entryCount"`
	TotalSeconds int                     `json:"totalSeconds"`
	Text         string                  `json:"text"`
	EntryFacts   []DailySummaryEntryFact `json:"entryFacts,omitempty"`
}

type dailySummaryMessages struct {
	header    string
	emptyBody string
	closing   string
}

type dailySummaryGroup struct {
	bullets []string
	firstAt string
	heading string
}

type dailySummaryStandaloneMeeting struct {
	at    string
	label string
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
		EntryFacts:   BuildDailySummaryEntryFacts(filtered),
	}

	var bodyLines []string
	if len(filtered) == 0 {
		bodyLines = []string{messages.emptyBody}
	} else {
		bodyLines = buildDailySummaryBody(filtered, options)
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
			header:    "Summary for today:",
			emptyBody: "No time entries recorded for this day.",
			closing:   "See you tomorrow team!",
		}
	}
	return dailySummaryMessages{
		header:    "Resumen de hoy:",
		emptyBody: "Sin entradas registradas hoy.",
		closing:   "Hasta mañana team!",
	}
}

func formatDailySummaryDateHeader(day time.Time, locale string) string {
	if locale == "en" {
		return fmt.Sprintf("%d/%d:", day.Month(), day.Day())
	}
	return fmt.Sprintf("%d/%d:", day.Day(), int(day.Month()))
}

func buildDailySummaryBody(entries []TimeEntry, options DailySummaryOptions) []string {
	groups := make([]dailySummaryGroup, 0)
	groupIndex := map[string]int{}
	standaloneMeetings := make([]dailySummaryStandaloneMeeting, 0)
	orphanBullets := make([]string, 0)

	for _, entry := range entries {
		if dailySummaryIsMeeting(entry) && dailySummaryIsStandaloneMeeting(entry, options) {
			standaloneMeetings = append(standaloneMeetings, dailySummaryStandaloneMeeting{
				at:    entry.StartedAt,
				label: dailySummaryMeetingLine(entry),
			})
			continue
		}

		bullets := dailySummaryEntryBullets(entry)
		if len(bullets) == 0 {
			continue
		}

		heading := dailySummaryGroupHeading(entry, options)
		if heading == "" {
			orphanBullets = appendUniquePhrases(orphanBullets, bullets)
			continue
		}

		if idx, ok := groupIndex[heading]; ok {
			groups[idx].bullets = appendUniquePhrases(groups[idx].bullets, bullets)
			continue
		}

		groupIndex[heading] = len(groups)
		groups = append(groups, dailySummaryGroup{
			bullets: append([]string(nil), bullets...),
			firstAt: entry.StartedAt,
			heading: heading,
		})
	}

	sort.Slice(groups, func(i, j int) bool {
		return groups[i].firstAt < groups[j].firstAt
	})
	sort.Slice(standaloneMeetings, func(i, j int) bool {
		return standaloneMeetings[i].at < standaloneMeetings[j].at
	})

	lines := make([]string, 0, len(groups)+len(standaloneMeetings)+len(orphanBullets))
	for _, bullet := range orphanBullets {
		lines = append(lines, "- "+bullet)
	}
	for _, group := range groups {
		lines = append(lines, "- "+group.heading+":")
		for _, bullet := range group.bullets {
			lines = append(lines, "    - "+bullet)
		}
	}
	for _, meeting := range standaloneMeetings {
		lines = append(lines, "- "+meeting.label)
	}
	return lines
}

func dailySummaryGroupHeading(entry TimeEntry, options DailySummaryOptions) string {
	client := strings.TrimSpace(entry.ClientName)
	project := strings.TrimSpace(entry.ProjectName)
	if options.IncludeClient && client != "" {
		return client
	}
	if options.IncludeProject && project != "" {
		return project
	}
	if client != "" {
		return client
	}
	if project != "" {
		return project
	}
	return ""
}

func dailySummaryIsMeeting(entry TimeEntry) bool {
	taskName := strings.TrimSpace(entry.TaskName)
	if len(splitDailySummaryTopics(taskName)) > 1 {
		return false
	}
	haystack := strings.ToLower(strings.Join([]string{
		entry.TaskName,
		entry.Description,
		entry.ProjectName,
	}, " "))
	keywords := []string{
		"reunión", "reunion", "meeting", "meet ", " weekly", "weekly ", "standup", "stand-up",
		"sync ", "sync.", "llamada", "videollamada", "call with", "team meet", "all hands",
	}
	for _, keyword := range keywords {
		if strings.Contains(haystack, keyword) {
			return true
		}
	}
	return false
}

func dailySummaryIsStandaloneMeeting(entry TimeEntry, options DailySummaryOptions) bool {
	if !dailySummaryIsMeeting(entry) {
		return false
	}
	if dailySummaryLooksLikeGeneralMeetingLabel(dailySummaryMeetingLine(entry)) {
		return true
	}
	if strings.TrimSpace(entry.ProjectID) == "" && strings.TrimSpace(entry.ClientID) == "" {
		return true
	}
	return dailySummaryGroupHeading(entry, options) == ""
}

func dailySummaryLooksLikeGeneralMeetingLabel(label string) bool {
	lower := strings.ToLower(strings.TrimSpace(label))
	if lower == "" {
		return false
	}
	generalPatterns := []string{
		"meet tech", "tech meet", "weekly", "reunión de tech", "reunion de tech",
		"team meet", "all hands", "reunión general", "reunion general",
	}
	for _, pattern := range generalPatterns {
		if strings.Contains(lower, pattern) {
			return true
		}
	}
	return strings.HasPrefix(lower, "meet ")
}

func dailySummaryMeetingLine(entry TimeEntry) string {
	project := strings.TrimSpace(entry.ProjectName)
	if dailySummaryLooksLikeGeneralMeetingLabel(project) {
		return project
	}
	if bullets := dailySummaryEntryBullets(entry); len(bullets) > 0 {
		return bullets[0]
	}
	if project != "" {
		return project
	}
	if client := strings.TrimSpace(entry.ClientName); client != "" {
		return client
	}
	return "Reunión"
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
