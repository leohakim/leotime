package httpapi

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/leotime/leotime/apps/api/internal/vcs"
)

func TestVCSConnectionLifecycleMasksToken(t *testing.T) {
	secretsKey := base64.StdEncoding.EncodeToString([]byte("01234567890123456789012345678901"))
	router := newTestRouterWithSecretsKey(t, secretsKey)
	cookies := loginCookies(t, router)

	createResponse := httptest.NewRecorder()
	createRequest := httptest.NewRequest(http.MethodPost, "/api/v1/vcs/connections", strings.NewReader("{\"provider\":\"gitea\",\"baseUrl\":\"https://gitea.osoigo.test\",\"ownerIdentity\":\"leo\",\"token\":\"read-only-token\"}"))
	createRequest.Header.Set("Content-Type", "application/json")
	for _, cookie := range cookies {
		createRequest.AddCookie(cookie)
	}
	router.ServeHTTP(createResponse, createRequest)
	if createResponse.Code != http.StatusCreated {
		t.Fatalf("expected create connection 201, got %d: %s", createResponse.Code, createResponse.Body.String())
	}

	var created map[string]any
	if err := json.Unmarshal(createResponse.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode connection: %v", err)
	}
	if created["provider"] != "gitea" || created["tokenConfigured"] != true {
		t.Fatalf("unexpected masked connection: %#v", created)
	}
	if strings.Contains(createResponse.Body.String(), "token") && strings.Contains(createResponse.Body.String(), "read-only-token") {
		t.Fatalf("connection response exposed token: %s", createResponse.Body.String())
	}

	connectionID, _ := created["id"].(string)
	updateResponse := httptest.NewRecorder()
	updateRequest := httptest.NewRequest(http.MethodPut, "/api/v1/vcs/connections/"+connectionID, strings.NewReader("{\"provider\":\"gitea\",\"baseUrl\":\"https://gitea.osoigo.test\",\"ownerIdentity\":\"leo@example.com\"}"))
	updateRequest.Header.Set("Content-Type", "application/json")
	for _, cookie := range cookies {
		updateRequest.AddCookie(cookie)
	}
	router.ServeHTTP(updateResponse, updateRequest)
	if updateResponse.Code != http.StatusOK {
		t.Fatalf("expected update connection 200, got %d: %s", updateResponse.Code, updateResponse.Body.String())
	}
	var updated map[string]any
	if err := json.Unmarshal(updateResponse.Body.Bytes(), &updated); err != nil {
		t.Fatalf("decode updated connection: %v", err)
	}
	if updated["ownerIdentity"] != "leo@example.com" || updated["tokenConfigured"] != true {
		t.Fatalf("unexpected updated connection: %#v", updated)
	}

	listResponse := httptest.NewRecorder()
	listRequest := httptest.NewRequest(http.MethodGet, "/api/v1/vcs/connections", nil)
	for _, cookie := range cookies {
		listRequest.AddCookie(cookie)
	}
	router.ServeHTTP(listResponse, listRequest)
	if listResponse.Code != http.StatusOK {
		t.Fatalf("expected list connections 200, got %d: %s", listResponse.Code, listResponse.Body.String())
	}
	if !strings.Contains(listResponse.Body.String(), "gitea.osoigo.test") {
		t.Fatalf("expected saved connection in list, got %s", listResponse.Body.String())
	}

	deleteResponse := httptest.NewRecorder()
	deleteRequest := httptest.NewRequest(http.MethodDelete, "/api/v1/vcs/connections/"+connectionID, nil)
	for _, cookie := range cookies {
		deleteRequest.AddCookie(cookie)
	}
	router.ServeHTTP(deleteResponse, deleteRequest)
	if deleteResponse.Code != http.StatusNoContent {
		t.Fatalf("expected delete connection 204, got %d: %s", deleteResponse.Code, deleteResponse.Body.String())
	}
}

func TestVCSConnectionTestAndPreviewContext(t *testing.T) {
	secretsKey := base64.StdEncoding.EncodeToString([]byte("01234567890123456789012345678901"))
	router := newTestRouterWithSecretsKey(t, secretsKey)
	cookies := loginCookies(t, router)

	original := newVCSProvider
	newVCSProvider = func(string) vcs.Provider {
		return fakeVCSCollector{}
	}
	t.Cleanup(func() { newVCSProvider = original })

	createResponse := httptest.NewRecorder()
	createRequest := httptest.NewRequest(http.MethodPost, "/api/v1/vcs/connections", strings.NewReader(`{"provider":"gitea","baseUrl":"https://gitea.osoigo.test","ownerIdentity":"leo","token":"read-only-token"}`))
	createRequest.Header.Set("Content-Type", "application/json")
	for _, cookie := range cookies {
		createRequest.AddCookie(cookie)
	}
	router.ServeHTTP(createResponse, createRequest)
	if createResponse.Code != http.StatusCreated {
		t.Fatalf("create connection: %d %s", createResponse.Code, createResponse.Body.String())
	}
	var created map[string]any
	if err := json.Unmarshal(createResponse.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode: %v", err)
	}
	connectionID, _ := created["id"].(string)

	clientID := createClientForHTTPTest(t, router, cookies)
	repoResponse := httptest.NewRecorder()
	repoRequest := httptest.NewRequest(http.MethodPost, "/api/v1/vcs/repositories", strings.NewReader(`{"connectionId":"`+connectionID+`","owner":"osoigo","name":"backend","clientId":"`+clientID+`"}`))
	repoRequest.Header.Set("Content-Type", "application/json")
	for _, cookie := range cookies {
		repoRequest.AddCookie(cookie)
	}
	router.ServeHTTP(repoResponse, repoRequest)
	if repoResponse.Code != http.StatusCreated {
		t.Fatalf("create repository: %d %s", repoResponse.Code, repoResponse.Body.String())
	}
	var repository map[string]any
	if err := json.Unmarshal(repoResponse.Body.Bytes(), &repository); err != nil {
		t.Fatalf("decode repository: %v", err)
	}
	repositoryID, _ := repository["id"].(string)

	testResponse := httptest.NewRecorder()
	testRequest := httptest.NewRequest(http.MethodPost, "/api/v1/vcs/connections/"+connectionID+"/test", nil)
	for _, cookie := range cookies {
		testRequest.AddCookie(cookie)
	}
	router.ServeHTTP(testResponse, testRequest)
	if testResponse.Code != http.StatusOK || !strings.Contains(testResponse.Body.String(), `"ok":true`) {
		t.Fatalf("expected connection test ok, got %d %s", testResponse.Code, testResponse.Body.String())
	}

	previewConnection := httptest.NewRecorder()
	previewConnectionRequest := httptest.NewRequest(http.MethodPost, "/api/v1/vcs/connections/"+connectionID+"/preview-context?date=2026-07-27", nil)
	for _, cookie := range cookies {
		previewConnectionRequest.AddCookie(cookie)
	}
	router.ServeHTTP(previewConnection, previewConnectionRequest)
	if previewConnection.Code != http.StatusOK || !strings.Contains(previewConnection.Body.String(), "Ajustar integración del cliente") {
		t.Fatalf("expected connection preview facts, got %d %s", previewConnection.Code, previewConnection.Body.String())
	}

	previewRepo := httptest.NewRecorder()
	previewRepoRequest := httptest.NewRequest(http.MethodPost, "/api/v1/vcs/repositories/"+repositoryID+"/preview-context?date=2026-07-27", nil)
	for _, cookie := range cookies {
		previewRepoRequest.AddCookie(cookie)
	}
	router.ServeHTTP(previewRepo, previewRepoRequest)
	if previewRepo.Code != http.StatusOK || !strings.Contains(previewRepo.Body.String(), `"repositoryCount":1`) {
		t.Fatalf("expected repository preview, got %d %s", previewRepo.Code, previewRepo.Body.String())
	}
}
