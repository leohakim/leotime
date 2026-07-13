package enrich

import "testing"

func TestBuildEnrichedTextMergesCommits(t *testing.T) {
	text := BuildEnrichedText(ContextBundle{
		TemplateText: "12/7:\nResumen de hoy:\n- RTVE:\n    - corrección de rutas API.\nHasta mañana team!",
		Commits: []CommitLine{{
			ProjectName: "leotime",
			Hash:        "abc1234",
			Subject:     "add daily summary workflow",
		}},
	})
	if text == "" {
		t.Fatal("expected enriched text")
	}
	if !containsAll(text, "12/7:", "Resumen de hoy:", "abc1234", "add daily summary workflow", "- leotime:", "    - add daily summary workflow") {
		t.Fatalf("unexpected text: %s", text)
	}
}

func TestBuildEnrichedTextIncludesManualNote(t *testing.T) {
	text := BuildEnrichedText(ContextBundle{
		TemplateText: "12/7:\nResumen de hoy:\n- RTVE:\n    - corrección de rutas API.\nHasta mañana team!",
		ManualNote:   "Quedó pendiente el deploy en staging.",
	})
	if !containsAll(text, "Quedó pendiente el deploy en staging.", "Hasta mañana team!") {
		t.Fatalf("unexpected text: %s", text)
	}
}

func containsAll(text string, parts ...string) bool {
	for _, part := range parts {
		if !contains(text, part) {
			return false
		}
	}
	return true
}

func contains(text, part string) bool {
	return len(part) == 0 || (len(text) >= len(part) && indexOf(text, part) >= 0)
}

func indexOf(text, part string) int {
	for i := 0; i+len(part) <= len(text); i++ {
		if text[i:i+len(part)] == part {
			return i
		}
	}
	return -1
}
