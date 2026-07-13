package enrich

import (
	"fmt"
	"strings"
	"time"
)

func BuildCursorPrompt(bundle ContextBundle) string {
	spanish := isSpanishLocale(bundle.Locale)
	baseText := dailySummaryBaseText(bundle)
	intro := cursorPromptIntro(spanish)
	format := cursorPromptFormat(spanish, bundle.Date)
	document := cursorPromptDocument(spanish, baseText)

	var facts []string
	if len(bundle.EntryFacts) > 0 {
		facts = append(facts, formatTimeEntryFacts(spanish, bundle.EntryFacts))
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
	parts := []string{intro, format, document}
	if len(facts) > 0 {
		label := cursorLabel(spanish, "extra_facts", "Additional facts")
		parts = append(parts, label+":\n"+strings.Join(facts, "\n\n"))
	}
	parts = append(parts, rules)
	return strings.Join(parts, "\n\n")
}

func formatTimeEntryFacts(spanish bool, facts []TimeEntryFact) string {
	lines := make([]string, 0, len(facts))
	for _, fact := range facts {
		scope := dailySummaryFactScope(fact)
		topics := strings.Join(fact.Topics, ", ")
		if topics == "" {
			topics = strings.TrimSpace(fact.TaskName)
		}
		line := fmt.Sprintf("- %s | temas: %s", scope, topics)
		if description := strings.TrimSpace(fact.Description); description != "" {
			if spanish {
				line += " | notas: " + description
			} else {
				line += " | notes: " + description
			}
		}
		lines = append(lines, line)
	}
	label := cursorLabel(spanish, "time_entries", "Time entry detail")
	if spanish {
		return label + " (cada tema debe acabar en su propia viñeta, en primera persona):\n" + strings.Join(lines, "\n")
	}
	return label + " (each topic must become its own bullet, in first person):\n" + strings.Join(lines, "\n")
}

func dailySummaryFactScope(fact TimeEntryFact) string {
	client := strings.TrimSpace(fact.ClientName)
	project := strings.TrimSpace(fact.ProjectName)
	switch {
	case client != "" && project != "":
		return client + " — " + project
	case client != "":
		return client
	case project != "":
		return project
	default:
		return "General"
	}
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
	case "Document to enrich":
		return "Documento a enriquecer"
	case "Time entry detail":
		return "Detalle de entradas de tiempo"
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
	case "Additional facts":
		return "Hechos adicionales"
	default:
		return english
	}
}

func cursorPromptDocument(spanish bool, baseText string) string {
	label := cursorLabel(spanish, "document", "Document to enrich")
	if strings.TrimSpace(baseText) == "" {
		if spanish {
			return label + ":\n(sin borrador; genera el formato de viñetas desde los hechos adicionales)"
		}
		return label + ":\n(no draft; build the bullet format from additional facts)"
	}
	if spanish {
		return label + " (mantén grupos y cantidad de viñetas; solo enriquece el texto de cada viñeta hija):\n" + strings.TrimSpace(baseText)
	}
	return label + " (keep groups and bullet count; only enrich each child bullet text):\n" + strings.TrimSpace(baseText)
}

func cursorPromptIntro(spanish bool) string {
	if spanish {
		return `Eres mi asistente de bitácora diaria para Slack. Recibes un resumen YA FORMATEADO con viñetas.

Tu tarea: enriquecer SOLO el texto de cada viñeta hija (líneas con 4 espacios que empiezan por "- ").

Estilo obligatorio:
- siempre en primera persona del singular: Avancé, Corregí, Desplegué, Cerré, Reuní con..., Investigué...
- una viñeta = un solo tema o logro concreto; nunca juntes varios temas en la misma viñeta
- si una entrada tiene varios temas (p. ej. "Cropper Imágenes + Reunión Nico + Visibilidad de procesos"), cada tema va en su propia viñeta bajo el mismo proyecto
- convierte cada tema en una frase profesional que explique qué hiciste de verdad, no un relato genérico ni un listado plano del título

PROHIBIDO:
- tercera persona o plural ("cerramos", "avanzamos")
- convertir viñetas en párrafos o prosa continua
- fusionar temas distintos en una sola viñeta
- cambiar grupos, fecha, encabezado o cierre
- inventar reuniones, personas o tareas`
	}
	return `You are my daily Slack standup assistant. You receive an ALREADY FORMATTED bullet summary.

Your job: enrich ONLY the text of each child bullet (lines with 4 leading spaces starting with "- ").

Mandatory style:
- always first-person singular: I advanced, I fixed, I deployed, I closed, I met with..., I investigated...
- one bullet = one concrete topic or accomplishment; never combine multiple topics in one bullet
- if an entry has several topics (e.g. "Cropper Images + Nico meeting + Process visibility"), each topic gets its own bullet under the same project
- turn each topic into a professional sentence explaining what you actually did, not a generic recap or bare title

FORBIDDEN:
- third person or plural voice
- turning bullets into paragraphs or continuous prose
- merging different topics into one bullet
- changing groups, date line, header, or closing
- inventing meetings, people, or tasks`
}

func cursorPromptFormat(spanish bool, date string) string {
	dateLabel := formatSummaryDate(date, spanish)
	if spanish {
		return fmt.Sprintf(`Formato obligatorio (misma estructura; solo cambia el texto tras "    - "):

%s:
Resumen de hoy:
- RTVE:
    - Cerré la rama de /recommendations/ y la integré en master
    - Avancé con custom forms y los campos nuevos de la implementación
    - Avancé con el ADR de Cropper y la optimización de imágenes
- ENACT:
    - Afiné el backend y el manifiesto de Kubernetes; mejoré cache y procesamiento
    - Reuní con el equipo de sistemas para revisar el despliegue
- Meet Tech
Hasta mañana team!

Ejemplo con título compuesto partido en 3 viñetas del mismo proyecto:
Título de tarea: "Cropper Imágenes + Reunión Nico + Visibilidad de procesos"
→   - Avancé con el ADR de Cropper y la optimización de imágenes para la carga en formularios
→   - Reuní con Nico para revisar la visibilidad de procesos destacados y acordar siguientes pasos
→   - Ajusté la visibilidad de procesos según lo revisado con Nico

Ejemplo inválido (NO hagas esto):
    - Avancé con Cropper, reunión con Nico y visibilidad de procesos durante el día`, dateLabel)
	}
	return fmt.Sprintf(`Mandatory format (same structure; only change text after "    - "):

%s:
Today's summary:
- ACME:
    - Merged the recommendations branch into master
    - Advanced custom forms and the new field types
    - Advanced the Cropper ADR and image optimization work
- OTHER:
    - Tuned the backend manifest and improved cache and processing
    - Met with the systems team to review deployment
- Meet Tech
See you tomorrow team!

Compound title split into 3 bullets under the same project:
Task title: "Cropper Images + Nico meeting + Process visibility"
→   - Advanced the Cropper ADR and image optimization for form uploads
→   - Met with Nico to review featured process visibility and agree next steps
→   - Adjusted process visibility based on what Nico and I reviewed

Invalid example (DO NOT do this):
    - Advanced Cropper, Nico meeting, and process visibility during the day`, dateLabel)
}

func cursorPromptRules(spanish bool) string {
	if spanish {
		return `Reglas finales:
- Devuelve el documento completo con la misma cantidad de viñetas hijas por grupo que el borrador
- Cada viñeta hija en primera persona y con un solo tema
- Los títulos de grupo quedan igual; las reuniones generales siguen al final como viñeta de primer nivel
- Usa commits, Cursor y notas solo para afinar la viñeta correcta
- Devuelve solo el texto final del standup, sin explicaciones ni markdown extra`
	}
	return `Final rules:
- Return the full document with the same number of child bullets per group as the draft
- Each child bullet must be first person and cover a single topic
- Group titles stay unchanged; general meetings remain as a final top-level bullet
- Use commits, Cursor, and notes only to refine the matching bullet
- Return only the final standup text with no commentary or extra markdown`
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
