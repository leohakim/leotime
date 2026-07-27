package httpapi

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/leotime/leotime/apps/api/internal/vcs"
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

func TestDailySummaryEnrichContextIncludesScopedVerifiedVCSFacts(t *testing.T) {
	secretsKey := base64.StdEncoding.EncodeToString([]byte("01234567890123456789012345678901"))
	router := newTestRouterWithSecretsKey(t, secretsKey)
	cookies := loginCookies(t, router)
	clientID := createClientForHTTPTest(t, router, cookies)

	originalFactory := newVCSProvider
	newVCSProvider = func(string) vcs.Provider {
		return fakeVCSCollector{}
	}
	t.Cleanup(func() {
		newVCSProvider = originalFactory
	})

	connectionID := createVCSConnectionForHTTPTest(t, router, cookies, clientID)
	createVCSRepositoryForHTTPTest(t, router, cookies, connectionID, clientID)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/daily-summaries/2026-07-27/enrich-context?clientId="+clientID, nil)
	for _, cookie := range cookies {
		request.AddCookie(cookie)
	}
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected enrichment context 200, got %d: %s", response.Code, response.Body.String())
	}

	var payload struct {
		VCS struct {
			Commits []vcs.Commit `json:"commits"`
		} `json:"vcs"`
		VCSStatus string `json:"vcsStatus"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode enrichment context: %v", err)
	}
	if payload.VCSStatus != "ready" || len(payload.VCS.Commits) != 1 {
		t.Fatalf("unexpected VCS payload: %+v", payload)
	}
	if payload.VCS.Commits[0].Subject != "Ajustar integración del cliente" {
		t.Fatalf("unexpected VCS commit: %+v", payload.VCS.Commits[0])
	}
}

type fakeVCSCollector struct{}

func (fakeVCSCollector) TestConnection(_ context.Context, _ string) error {
	return nil
}

func (fakeVCSCollector) CollectDayContext(_ context.Context, request vcs.CollectionRequest) (vcs.Context, error) {
	if request.Date != "2026-07-27" {
		return vcs.Context{}, &vcs.ProviderError{Kind: vcs.ErrorInvalid}
	}
	return vcs.Context{Commits: []vcs.Commit{{
		Repository: "osoigo/backend",
		Hash:       "abcdef1",
		Subject:    "Ajustar integración del cliente",
	}}}, nil
}

func createVCSConnectionForHTTPTest(t *testing.T, router http.Handler, cookies []*http.Cookie, clientID string) string {
	t.Helper()
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/vcs/connections", strings.NewReader(`{"provider":"gitea","baseUrl":"https://gitea.osoigo.test","ownerIdentity":"leo","defaultClientId":"`+clientID+`","token":"read-only-token"}`))
	request.Header.Set("Content-Type", "application/json")
	for _, cookie := range cookies {
		request.AddCookie(cookie)
	}
	router.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("create VCS connection: %d: %s", response.Code, response.Body.String())
	}
	var payload struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil || payload.ID == "" {
		t.Fatalf("decode VCS connection: %v, body=%s", err, response.Body.String())
	}
	return payload.ID
}

func createVCSRepositoryForHTTPTest(t *testing.T, router http.Handler, cookies []*http.Cookie, connectionID, clientID string) {
	t.Helper()
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/vcs/repositories", strings.NewReader(`{"connectionId":"`+connectionID+`","owner":"osoigo","name":"backend","clientId":"`+clientID+`"}`))
	request.Header.Set("Content-Type", "application/json")
	for _, cookie := range cookies {
		request.AddCookie(cookie)
	}
	router.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("create VCS repository: %d: %s", response.Code, response.Body.String())
	}
}
