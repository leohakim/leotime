package httpapi

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAISettingsGetAndPut(t *testing.T) {
	secretsKey := base64.StdEncoding.EncodeToString([]byte("01234567890123456789012345678901"))
	router := newTestRouterWithSecretsKey(t, secretsKey)
	cookies := loginCookies(t, router)

	getResponse := httptest.NewRecorder()
	getRequest := httptest.NewRequest(http.MethodGet, "/api/v1/settings/ai", nil)
	for _, cookie := range cookies {
		getRequest.AddCookie(cookie)
	}
	router.ServeHTTP(getResponse, getRequest)
	if getResponse.Code != http.StatusOK {
		t.Fatalf("expected get ai settings 200, got %d: %s", getResponse.Code, getResponse.Body.String())
	}

	putBody := `{"enabled":true,"gitAuthorEmail":"dev@example.com","cursorApiKey":"cursor-test-key"}`
	putResponse := httptest.NewRecorder()
	putRequest := httptest.NewRequest(http.MethodPut, "/api/v1/settings/ai", strings.NewReader(putBody))
	putRequest.Header.Set("Content-Type", "application/json")
	for _, cookie := range cookies {
		putRequest.AddCookie(cookie)
	}
	router.ServeHTTP(putResponse, putRequest)
	if putResponse.Code != http.StatusOK {
		t.Fatalf("expected put ai settings 200, got %d: %s", putResponse.Code, putResponse.Body.String())
	}

	var payload struct {
		Enabled                bool   `json:"enabled"`
		GitAuthorEmail         string `json:"gitAuthorEmail"`
		CursorAPIKeyConfigured bool   `json:"cursorApiKeyConfigured"`
	}
	if err := json.Unmarshal(putResponse.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode ai settings: %v", err)
	}
	if !payload.Enabled || payload.GitAuthorEmail != "dev@example.com" || !payload.CursorAPIKeyConfigured {
		t.Fatalf("unexpected ai settings payload: %+v", payload)
	}
}

func TestPutAISettingsRequiresSecretsKeyForCursorKey(t *testing.T) {
	router := newTestRouter(t)
	cookies := loginCookies(t, router)

	putBody := `{"enabled":true,"gitAuthorEmail":"dev@example.com","cursorApiKey":"cursor-test-key"}`
	putResponse := httptest.NewRecorder()
	putRequest := httptest.NewRequest(http.MethodPut, "/api/v1/settings/ai", strings.NewReader(putBody))
	putRequest.Header.Set("Content-Type", "application/json")
	for _, cookie := range cookies {
		putRequest.AddCookie(cookie)
	}
	router.ServeHTTP(putResponse, putRequest)
	if putResponse.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 without secrets key, got %d: %s", putResponse.Code, putResponse.Body.String())
	}
}
