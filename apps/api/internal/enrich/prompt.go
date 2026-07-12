package enrich

import (
	"fmt"
	"strings"
	"time"
)

func BuildCursorPrompt(bundle ContextBundle) string {
	spanish := isSpanishLocale(bundle.Locale)
	intro := cursorPromptIntro(spanish)
	format := cursorPromptFormat(spanish, bundle.Date)

	var facts []string
	if text := strings.TrimSpace(bundle.TemplateText); text != "" {
		facts = append(facts, cursorLabel(spanish, "time_entries", "Time entries summary")+":\n"+text)
	} else if draft := strings.TrimSpace(bundle.CurrentDraft); draft != "" {
		facts = append(facts, cursorLabel(spanish, "current_draft", "Current draft")+":\n"+draft)
	}
	if note := strings.TrimSpace(bundle.ManualNote); note != "" {
		facts = append(facts, cursorLabel(spanish, "manual_note", "Owner note")+": "+note)
	}
	if feedback := strings.TrimSpace(bundle.Feedback); feedback != "" {
		facts = append(facts, cursorLabel(spanish, "feedback", "Revision feedback")+": "+feedback)
	}
	if len(bundle.Commits) > 0 {
		lines := make([]string, 0, len(bundle.Commits))
		for _, commit := range bundle.Commits {
			lines = append(lines, fmt.Sprintf("- %s: %s (%s)", commit.ProjectName, commit.Subject, commit.Hash))
		}
		facts = append(facts, cursorLabel(spanish, "commits", "Git commits")+":\n"+strings.Join(lines, "\n"))
	}
	for _, activity := range bundle.CursorActivity {
		if len(activity.UserQueries) == 0 && len(activity.FilesTouched) == 0 {
			continue
		}
		block := cursorLabel(spanish, "cursor_activity", "Cursor activity") + " (" + activity.WorkspaceSlug + "):"
		if len(activity.UserQueries) > 0 {
			block += "\n" + cursorLabel(spanish, "queries", "Queries") + ": " + strings.Join(activity.UserQueries, "; ")
		}
		if len(activity.FilesTouched) > 0 {
			block += "\n" + cursorLabel(spanish, "files", "Files touched") + ": " + strings.Join(activity.FilesTouched, ", ")
		}
		facts = append(facts, block)
	}

	rules := cursorPromptRules(spanish)
	return intro + "\n\n" + format + "\n\n" + cursorLabel(spanish, "facts", "Facts for the day") + ":\n" + strings.Join(facts, "\n\n") + "\n\n" + rules
}

func isSpanishLocale(locale string) bool {
	locale = strings.ToLower(strings.TrimSpace(locale))
	return locale == "" || strings.HasPrefix(locale, "es")
}

func cursorLabel(spanish bool, _, english string) string {
	if !spanish {
		return english
	}
	switch english {
	case "Time entries summary":
		return "Resumen de entradas de tiempo"
	case "Current draft":
		return "Borrador actual"
	case "Owner note":
		return "Nota del propietario"
	case "Revision feedback":
		return "Feedback de revisión"
	case "Git commits":
		return "Commits de git"
	case "Cursor activity":
		return "Actividad en Cursor"
	case "Queries":
		return "Consultas"
	case "Files touched":
		return "Archivos tocados"
	case "Facts for the day":
		return "Datos del día"
	default:
		return english
	}
}

func cursorPromptIntro(spanish bool) string {
	if spanish {
		return "Eres mi asistente de bitácora diaria para Slack. Escribe UN párrafo fluido por bloque (mañana/tarde/noche) en estilo equipo remoto, sin inventar reuniones ni personas. Usa solo los hechos proporcionados."
	}
	return "You are my daily Slack standup assistant. Write one fluent paragraph per block (morning/afternoon/evening) in a remote-team tone. Do not invent meetings or people. Use only the provided facts."
}

func cursorPromptFormat(spanish bool, date string) string {
	if spanish {
		return fmt.Sprintf(`Formato exacto:
%s:
Resumen de hoy:
…
Hasta mañana team!`, formatSummaryDate(date, true))
	}
	return fmt.Sprintf(`Exact format:
%s:
Today's summary:
…
See you tomorrow team!`, formatSummaryDate(date, false))
}

func cursorPromptRules(spanish bool) string {
	if spanish {
		return "Reglas: no inventes datos; integra commits y actividad de Cursor solo si aparecen en los hechos; conserva nombres reales de proyectos/clientes; devuelve solo el texto final del standup, sin explicaciones."
	}
	return "Rules: do not invent facts; weave commits and Cursor activity only when present in the facts; keep real project/client names; return only the final standup text with no commentary."
}

func formatSummaryDate(date string, spanish bool) string {
	day, err := parseSummaryDate(date)
	if err != nil {
		return date
	}
	if spanish {
		return fmt.Sprintf("%d/%d", day.Day(), int(day.Month()))
	}
	return fmt.Sprintf("%d/%d", day.Month(), day.Day())
}

func parseSummaryDate(date string) (time.Time, error) {
	return time.Parse("2006-01-02", strings.TrimSpace(date))
}
