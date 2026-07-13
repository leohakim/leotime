package store

import (
	"regexp"
	"strings"
)

var dailySummaryCompoundTopicPattern = regexp.MustCompile(`\s*(?:\+|;|\||/)\s*`)

type DailySummaryEntryFact struct {
	ClientName  string   `json:"clientName"`
	ProjectName string   `json:"projectName"`
	TaskName    string   `json:"taskName"`
	Topics      []string `json:"topics"`
	Description string   `json:"description"`
}

func BuildDailySummaryEntryFacts(entries []TimeEntry) []DailySummaryEntryFact {
	facts := make([]DailySummaryEntryFact, 0, len(entries))
	for _, entry := range entries {
		taskName := strings.TrimSpace(entry.TaskName)
		topics := splitDailySummaryTopics(taskName)
		if len(topics) == 0 && taskName != "" {
			topics = []string{taskName}
		}
		facts = append(facts, DailySummaryEntryFact{
			ClientName:  strings.TrimSpace(entry.ClientName),
			ProjectName: strings.TrimSpace(entry.ProjectName),
			TaskName:    taskName,
			Topics:      topics,
			Description: strings.TrimSpace(entry.Description),
		})
	}
	return facts
}

func splitDailySummaryTopics(text string) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	raw := dailySummaryCompoundTopicPattern.Split(text, -1)
	topics := make([]string, 0, len(raw))
	for _, part := range raw {
		part = strings.TrimSpace(part)
		if part != "" {
			topics = append(topics, part)
		}
	}
	if len(topics) == 0 {
		return []string{text}
	}
	return topics
}

func dailySummaryEntryBullets(entry TimeEntry) []string {
	description := strings.TrimSpace(entry.Description)
	taskName := strings.TrimSpace(entry.TaskName)
	topics := splitDailySummaryTopics(taskName)

	if len(topics) > 1 {
		return appendUniquePhrases(nil, topics)
	}

	if description != "" {
		return []string{description}
	}
	if len(topics) == 1 {
		return []string{topics[0]}
	}
	if taskName != "" {
		return []string{taskName}
	}
	return nil
}

func appendUniquePhrases(items []string, phrases []string) []string {
	for _, phrase := range phrases {
		items = appendUniquePhrase(items, phrase)
	}
	return items
}

func dailySummaryTopicLooksLikeMeeting(topic string) bool {
	haystack := strings.ToLower(strings.TrimSpace(topic))
	keywords := []string{
		"reunión", "reunion", "meeting", "meet ", "weekly", "standup", "stand-up",
		"sync", "llamada", "videollamada", "call with",
	}
	for _, keyword := range keywords {
		if strings.Contains(haystack, keyword) {
			return true
		}
	}
	return strings.HasPrefix(haystack, "reunión ") || strings.HasPrefix(haystack, "reunion ")
}
