package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDecodeJSONBodyRejectsUnknownField(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"email":"a@example.com","extra":true}`))

	var payload struct {
		Email string `json:"email"`
	}
	if decodeJSONBody(recorder, request, &payload) {
		t.Fatal("expected unknown field to be rejected")
	}
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
	assertErrorCode(t, recorder.Body.String(), "invalid_json")
}

func TestDecodeJSONBodyRejectsTrailingJSON(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"email":"a@example.com"}{"extra":true}`))

	var payload struct {
		Email string `json:"email"`
	}
	if decodeJSONBody(recorder, request, &payload) {
		t.Fatal("expected trailing json to be rejected")
	}
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
	assertErrorCode(t, recorder.Body.String(), "invalid_json")
}

func TestDecodeJSONBodyRejectsEmptyBody(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(""))

	var payload struct {
		Email string `json:"email"`
	}
	if decodeJSONBody(recorder, request, &payload) {
		t.Fatal("expected empty body to be rejected")
	}
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
	assertErrorCode(t, recorder.Body.String(), "invalid_json")
}

func TestDecodeJSONBodyAcceptsValidPayload(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"email":"a@example.com","password":"secret"}`))

	var payload struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if !decodeJSONBody(recorder, request, &payload) {
		t.Fatalf("expected valid payload, got %d: %s", recorder.Code, recorder.Body.String())
	}
	if payload.Email != "a@example.com" || payload.Password != "secret" {
		t.Fatalf("unexpected payload: %+v", payload)
	}
}

func assertErrorCode(t *testing.T, body string, code string) {
	t.Helper()
	var payload struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(body), &payload); err != nil {
		t.Fatalf("decode error body: %v", err)
	}
	if payload.Error.Code != code {
		t.Fatalf("expected error code %q, got %q (%s)", code, payload.Error.Code, body)
	}
}
