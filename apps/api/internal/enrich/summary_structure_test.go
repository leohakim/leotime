package enrich

import (
	"strings"
	"testing"
)

const sampleTemplate = `12/7:
Resumen de hoy:
- RTVE:
    - corrección de rutas API
    - custom forms
- ENACT:
    - backend y kubernetes
- Meet Tech
Hasta mañana team!`

func TestEnforceDailySummaryStructureKeepsCompatibleAIOutput(t *testing.T) {
	enriched := `12/7:
Resumen de hoy:
- RTVE:
    - Corregimos las rutas absolutas de la API para el despliegue nuevo
    - Avanzamos con custom forms y campos nuevos de la implementación
- ENACT:
    - Afinamos el backend y el manifiesto de Kubernetes; mejoramos cache y procesamiento
- Meet Tech
Hasta mañana team!`

	got := enforceDailySummaryStructure(sampleTemplate, enriched)
	if got != enriched {
		t.Fatalf("expected compatible output unchanged, got:\n%s", got)
	}
}

func TestEnforceDailySummaryStructureRecoversBulletsFromProse(t *testing.T) {
	prose := `12/7:
Resumen de hoy:
Por la mañana avancé con RTVE corrigiendo rutas API y custom forms. Por la tarde seguí con ENACT en backend y Kubernetes.
Hasta mañana team!`

	got := enforceDailySummaryStructure(sampleTemplate, prose)
	if !strings.Contains(got, "- RTVE:") {
		t.Fatalf("expected RTVE group preserved, got:\n%s", got)
	}
	if !strings.Contains(got, "    - ") {
		t.Fatalf("expected child bullets preserved, got:\n%s", got)
	}
	if strings.Contains(got, "Por la mañana avancé") {
		t.Fatalf("did not expect prose paragraph, got:\n%s", got)
	}
	if !strings.Contains(got, "corrección de rutas API") {
		t.Fatalf("expected template bullets when AI returned prose, got:\n%s", got)
	}
}

func TestEnforceDailySummaryStructureMapsRecoveredChildBullets(t *testing.T) {
	enriched := `12/7:
Resumen de hoy:
- RTVE:
    - Corregimos las rutas absolutas de la API para el despliegue nuevo
    - Avanzamos con custom forms y campos nuevos
- ENACT:
    - Afinamos backend y manifiesto de Kubernetes
Hasta mañana team!`

	got := enforceDailySummaryStructure(sampleTemplate, enriched)
	if !containsAll(got, "Corregimos las rutas absolutas", "Avanzamos con custom forms", "Afinamos backend", "- Meet Tech") {
		t.Fatalf("unexpected mapped output:\n%s", got)
	}
}

func TestBuildCursorPromptPreservesDocumentFirst(t *testing.T) {
	prompt := BuildCursorPrompt(ContextBundle{
		Date:         "2026-07-12",
		Locale:       "es",
		TemplateText: sampleTemplate,
		ManualNote:   "Quedó pendiente el deploy.",
		EntryFacts: []TimeEntryFact{{
			ClientName:  "RTVE",
			ProjectName: "Participa",
			TaskName:    "Cropper Imagenes + Reunion Nico",
			Topics:      []string{"Cropper Imagenes", "Reunion Nico"},
			Description: "ADR cropper y sync con Nico",
		}},
		Commits: []CommitLine{{
			ProjectName: "leotime",
			Hash:        "abc1234",
			Subject:     "add daily summary workflow",
		}},
	})
	if !containsAll(
		prompt,
		"Documento a enriquecer",
		sampleTemplate,
		"primera persona",
		"Quedó pendiente el deploy.",
		"abc1234",
		"Hasta mañana team!",
		"Cropper Imagenes",
	) {
		t.Fatalf("unexpected prompt:\n%s", prompt)
	}
	if strings.Contains(prompt, "Reorganiza") {
		t.Fatalf("prompt should not ask to reorganize:\n%s", prompt)
	}
}
