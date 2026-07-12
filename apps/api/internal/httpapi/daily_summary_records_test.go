package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestApplyDailySummaryEnrichmentAcceptsScopedOptions(t *testing.T) {
	router := newTestRouter(t)
	cookies := loginCookies(t, router)

	body := `{
		"text": "12/3:\nResumen de hoy:\nPor la mañana avancé con Osoigo SL — RTVE: Agregar auditoria de cambios a los procesos.\nHasta mañana team!\nReunion con Huesca y despliegue de ramstein",
		"manualNote": "Reunion con Huesca y despliegue de ramstein",
		"generationSource": "context",
		"options": {
			"date": "2026-03-12",
			"clientId": "cli_scope_test",
			"projectId": "",
			"includeClient": true,
			"includeProject": true,
			"includeClosing": true,
			"billableOnly": false
		}
	}`

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/daily-summaries/2026-03-12/enrich", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	for _, cookie := range cookies {
		request.AddCookie(cookie)
	}

	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected enrich apply 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestDecodeDailySummaryEnrichmentBody(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{
		"text": "12/3:\nResumen de hoy:",
		"manualNote": "Reunion con Huesca",
		"generationSource": "context",
		"options": {
			"date": "2026-03-12",
			"clientId": "cli_d8251f3fa8f86df0ec74d16215b29529",
			"projectId": "",
			"includeClient": true,
			"includeProject": true,
			"includeClosing": true,
			"billableOnly": false
		}
	}`))

	var body struct {
		Text             string                     `json:"text"`
		ManualNote       string                     `json:"manualNote"`
		GenerationSource string                     `json:"generationSource"`
		ContextJSON      string                     `json:"contextJson"`
		Options          dailySummaryOptionsPayload `json:"options"`
	}
	if !decodeJSONBody(recorder, request, &body) {
		t.Fatalf("expected valid enrichment body, got %d: %s", recorder.Code, recorder.Body.String())
	}
	if body.Options.ClientID != "cli_d8251f3fa8f86df0ec74d16215b29529" {
		t.Fatalf("unexpected client id: %+v", body.Options)
	}
}
