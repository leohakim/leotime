package enrich

import (
	"fmt"
	"strings"
)

func BuildEnrichedText(bundle ContextBundle) string {
	if strings.TrimSpace(bundle.CurrentDraft) != "" && strings.TrimSpace(bundle.Feedback) != "" {
		return weaveManualNote(mergeFeedback(bundle), bundle.ManualNote)
	}

	lines := []string{}
	if strings.TrimSpace(bundle.TemplateText) != "" {
		lines = append(lines, strings.TrimSpace(bundle.TemplateText))
	} else if strings.TrimSpace(bundle.CurrentDraft) != "" {
		lines = append(lines, strings.TrimSpace(bundle.CurrentDraft))
	}

	extra := make([]string, 0)
	if note := strings.TrimSpace(bundle.ManualNote); note != "" {
		base := strings.Join(lines, "\n")
		if base == "" || !strings.Contains(base, note) {
			extra = append(extra, note)
		}
	}
	for _, commit := range bundle.Commits {
		heading := strings.TrimSpace(commit.ProjectName)
		if heading == "" {
			heading = "Commits"
		}
		extra = append(extra, fmt.Sprintf("- %s:", heading))
		extra = append(extra, fmt.Sprintf("    - %s (%s)", commit.Subject, commit.Hash))
	}
	for _, activity := range bundle.CursorActivity {
		if len(activity.UserQueries) > 0 {
			limit := minInt(3, len(activity.UserQueries))
			extra = append(extra, fmt.Sprintf("- %s:", activity.WorkspaceSlug))
			extra = append(extra, "    - En Cursor consulté: "+strings.Join(activity.UserQueries[:limit], "; ")+".")
		}
		if len(activity.FilesTouched) > 0 {
			limit := minInt(6, len(activity.FilesTouched))
			if len(activity.UserQueries) == 0 {
				extra = append(extra, fmt.Sprintf("- %s:", activity.WorkspaceSlug))
			}
			extra = append(extra, "    - Archivos tocados: "+strings.Join(activity.FilesTouched[:limit], ", ")+".")
		}
	}
	if len(extra) == 0 {
		return strings.Join(lines, "\n")
	}
	if len(lines) == 0 {
		return strings.Join(extra, "\n")
	}

	return insertDailySummaryExtras(lines[0], extra)
}

func insertDailySummaryExtras(body string, extra []string) string {
	lines := strings.Split(body, "\n")
	insertAt := len(lines)
	for i := len(lines) - 1; i >= 0; i-- {
		lower := strings.ToLower(strings.TrimSpace(lines[i]))
		if strings.HasPrefix(lower, "hasta ") || strings.HasPrefix(lower, "see you") {
			insertAt = i
			continue
		}
		break
	}
	prefix := append([]string{}, lines[:insertAt]...)
	suffix := append([]string{}, lines[insertAt:]...)
	prefix = append(prefix, extra...)
	return strings.Join(append(prefix, suffix...), "\n")
}

func mergeFeedback(bundle ContextBundle) string {
	feedback := strings.TrimSpace(bundle.Feedback)
	draft := strings.TrimSpace(bundle.CurrentDraft)
	if feedback == "" {
		return draft
	}
	if strings.Contains(strings.ToLower(draft), strings.ToLower(feedback)) {
		return draft
	}
	lines := strings.Split(draft, "\n")
	if len(lines) >= 2 && strings.HasPrefix(strings.ToLower(lines[len(lines)-1]), "hasta ") {
		lines = append(lines[:len(lines)-1], feedback, lines[len(lines)-1])
		return strings.Join(lines, "\n")
	}
	return draft + "\n" + feedback
}

func weaveManualNote(text, note string) string {
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

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
