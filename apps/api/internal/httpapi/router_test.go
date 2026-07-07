package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/leotime/leotime/apps/api/internal/config"
	"github.com/leotime/leotime/apps/api/internal/db"
	"github.com/leotime/leotime/apps/api/internal/notify"
	"github.com/leotime/leotime/apps/api/internal/outbox"
	"github.com/leotime/leotime/apps/api/internal/store"
)

func TestHealth(t *testing.T) {
	router := newTestRouter(t)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", response.Code)
	}
}

func TestLoginSessionAndOverview(t *testing.T) {
	router := newTestRouter(t)

	loginBody := bytes.NewBufferString(`{"email":"admin@example.com","password":"change-me-now"}`)
	loginResponse := httptest.NewRecorder()
	loginRequest := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", loginBody)
	router.ServeHTTP(loginResponse, loginRequest)

	if loginResponse.Code != http.StatusOK {
		t.Fatalf("expected login 200, got %d: %s", loginResponse.Code, loginResponse.Body.String())
	}

	var loginPayload sessionResponse
	if err := json.Unmarshal(loginResponse.Body.Bytes(), &loginPayload); err != nil {
		t.Fatalf("decode login payload: %v", err)
	}
	if !loginPayload.Authenticated || loginPayload.User == nil {
		t.Fatal("expected authenticated login response")
	}

	cookies := loginResponse.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected session cookie")
	}

	overviewResponse := httptest.NewRecorder()
	overviewRequest := httptest.NewRequest(http.MethodGet, "/api/v1/overview", nil)
	for _, cookie := range cookies {
		overviewRequest.AddCookie(cookie)
	}
	router.ServeHTTP(overviewResponse, overviewRequest)

	if overviewResponse.Code != http.StatusOK {
		t.Fatalf("expected overview 200, got %d: %s", overviewResponse.Code, overviewResponse.Body.String())
	}
}

func TestClientsRequireAuthentication(t *testing.T) {
	router := newTestRouter(t)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/clients", nil)
	router.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", response.Code)
	}
}

func TestProjectsRequireAuthentication(t *testing.T) {
	router := newTestRouter(t)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/projects", nil)
	router.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", response.Code)
	}
}

func TestTasksRequireAuthentication(t *testing.T) {
	router := newTestRouter(t)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/tasks", nil)
	router.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", response.Code)
	}
}

func TestTagsRequireAuthentication(t *testing.T) {
	router := newTestRouter(t)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/tags", nil)
	router.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", response.Code)
	}
}

func TestProfileHTTPLifecycle(t *testing.T) {
	router := newTestRouter(t)
	cookies := loginCookies(t, router)

	getResponse := httptest.NewRecorder()
	getRequest := httptest.NewRequest(http.MethodGet, "/api/v1/profile", nil)
	for _, cookie := range cookies {
		getRequest.AddCookie(cookie)
	}
	router.ServeHTTP(getResponse, getRequest)
	if getResponse.Code != http.StatusOK {
		t.Fatalf("expected profile 200, got %d: %s", getResponse.Code, getResponse.Body.String())
	}

	patchResponse := httptest.NewRecorder()
	patchRequest := httptest.NewRequest(http.MethodPatch, "/api/v1/profile", bytes.NewBufferString(`{
		"name":"Leo",
		"email":"leo@example.com",
		"locale":"en",
		"layoutMode":"compact",
		"taskProjectRequired":true,
		"defaultCurrency":"USD",
		"timezone":"America/New_York",
		"themeMode":"dark",
		"timerStillRunningEnabled":true,
		"timerStillRunningHours":6
	}`))
	for _, cookie := range cookies {
		patchRequest.AddCookie(cookie)
	}
	router.ServeHTTP(patchResponse, patchRequest)
	if patchResponse.Code != http.StatusOK {
		t.Fatalf("expected profile patch 200, got %d: %s", patchResponse.Code, patchResponse.Body.String())
	}

	passwordResponse := httptest.NewRecorder()
	passwordRequest := httptest.NewRequest(http.MethodPost, "/api/v1/profile/change-password", bytes.NewBufferString(`{
		"currentPassword":"change-me-now",
		"newPassword":"new-password-123"
	}`))
	for _, cookie := range cookies {
		passwordRequest.AddCookie(cookie)
	}
	router.ServeHTTP(passwordResponse, passwordRequest)
	if passwordResponse.Code != http.StatusNoContent {
		t.Fatalf("expected password change 204, got %d: %s", passwordResponse.Code, passwordResponse.Body.String())
	}
}

func TestClientHTTPLifecycle(t *testing.T) {
	router := newTestRouter(t)
	cookies := loginCookies(t, router)

	createResponse := httptest.NewRecorder()
	createRequest := httptest.NewRequest(http.MethodPost, "/api/v1/clients", bytes.NewBufferString(`{
		"name": "Client One",
		"email": "billing@example.com",
		"taxId": "B12345678",
		"billingAddress": "Madrid",
		"defaultCurrency": "eur",
		"defaultHourlyRateMinor": 7500
	}`))
	for _, cookie := range cookies {
		createRequest.AddCookie(cookie)
	}
	router.ServeHTTP(createResponse, createRequest)

	if createResponse.Code != http.StatusCreated {
		t.Fatalf("expected create 201, got %d: %s", createResponse.Code, createResponse.Body.String())
	}

	var created struct {
		ID              string `json:"id"`
		DefaultCurrency string `json:"defaultCurrency"`
	}
	if err := json.Unmarshal(createResponse.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created client: %v", err)
	}
	if created.ID == "" || created.DefaultCurrency != "EUR" {
		t.Fatalf("unexpected created client: %+v", created)
	}

	updateResponse := httptest.NewRecorder()
	updateRequest := httptest.NewRequest(http.MethodPatch, "/api/v1/clients/"+created.ID, bytes.NewBufferString(`{
		"name": "Client One Updated",
		"defaultCurrency": "USD",
		"defaultHourlyRateMinor": 9000
	}`))
	for _, cookie := range cookies {
		updateRequest.AddCookie(cookie)
	}
	router.ServeHTTP(updateResponse, updateRequest)

	if updateResponse.Code != http.StatusOK {
		t.Fatalf("expected update 200, got %d: %s", updateResponse.Code, updateResponse.Body.String())
	}

	listResponse := httptest.NewRecorder()
	listRequest := httptest.NewRequest(http.MethodGet, "/api/v1/clients", nil)
	for _, cookie := range cookies {
		listRequest.AddCookie(cookie)
	}
	router.ServeHTTP(listResponse, listRequest)

	if listResponse.Code != http.StatusOK {
		t.Fatalf("expected list 200, got %d: %s", listResponse.Code, listResponse.Body.String())
	}

	var listPayload clientsResponse
	if err := json.Unmarshal(listResponse.Body.Bytes(), &listPayload); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	if len(listPayload.Clients) != 1 || listPayload.Clients[0].Name != "Client One Updated" {
		t.Fatalf("unexpected client list: %+v", listPayload)
	}

	deleteResponse := httptest.NewRecorder()
	deleteRequest := httptest.NewRequest(http.MethodDelete, "/api/v1/clients/"+created.ID, nil)
	for _, cookie := range cookies {
		deleteRequest.AddCookie(cookie)
	}
	router.ServeHTTP(deleteResponse, deleteRequest)

	if deleteResponse.Code != http.StatusNoContent {
		t.Fatalf("expected delete 204, got %d: %s", deleteResponse.Code, deleteResponse.Body.String())
	}

	emptyListResponse := httptest.NewRecorder()
	emptyListRequest := httptest.NewRequest(http.MethodGet, "/api/v1/clients", nil)
	for _, cookie := range cookies {
		emptyListRequest.AddCookie(cookie)
	}
	router.ServeHTTP(emptyListResponse, emptyListRequest)

	if emptyListResponse.Code != http.StatusOK {
		t.Fatalf("expected empty list 200, got %d: %s", emptyListResponse.Code, emptyListResponse.Body.String())
	}
	var emptyListPayload clientsResponse
	if err := json.Unmarshal(emptyListResponse.Body.Bytes(), &emptyListPayload); err != nil {
		t.Fatalf("decode empty list response: %v", err)
	}
	if len(emptyListPayload.Clients) != 0 {
		t.Fatalf("expected no active clients after archive, got %+v", emptyListPayload)
	}

	restoreResponse := httptest.NewRecorder()
	restoreRequest := httptest.NewRequest(http.MethodPost, "/api/v1/clients/"+created.ID+"/restore", nil)
	for _, cookie := range cookies {
		restoreRequest.AddCookie(cookie)
	}
	router.ServeHTTP(restoreResponse, restoreRequest)

	if restoreResponse.Code != http.StatusOK {
		t.Fatalf("expected restore 200, got %d: %s", restoreResponse.Code, restoreResponse.Body.String())
	}

	var restored struct {
		ID         string `json:"id"`
		ArchivedAt string `json:"archivedAt"`
	}
	if err := json.Unmarshal(restoreResponse.Body.Bytes(), &restored); err != nil {
		t.Fatalf("decode restored client: %v", err)
	}
	if restored.ID != created.ID || restored.ArchivedAt != "" {
		t.Fatalf("unexpected restored client: %+v", restored)
	}

	activeListResponse := httptest.NewRecorder()
	activeListRequest := httptest.NewRequest(http.MethodGet, "/api/v1/clients", nil)
	for _, cookie := range cookies {
		activeListRequest.AddCookie(cookie)
	}
	router.ServeHTTP(activeListResponse, activeListRequest)

	if activeListResponse.Code != http.StatusOK {
		t.Fatalf("expected active list 200, got %d: %s", activeListResponse.Code, activeListResponse.Body.String())
	}
	var activeListPayload clientsResponse
	if err := json.Unmarshal(activeListResponse.Body.Bytes(), &activeListPayload); err != nil {
		t.Fatalf("decode active list response: %v", err)
	}
	if len(activeListPayload.Clients) != 1 || activeListPayload.Clients[0].Name != "Client One Updated" {
		t.Fatalf("unexpected active client list after restore: %+v", activeListPayload)
	}
}

func TestProjectHTTPLifecycle(t *testing.T) {
	router := newTestRouter(t)
	cookies := loginCookies(t, router)
	clientID := createClientForHTTPTest(t, router, cookies)

	createResponse := httptest.NewRecorder()
	createRequest := httptest.NewRequest(http.MethodPost, "/api/v1/projects", bytes.NewBufferString(`{
		"clientId": "`+clientID+`",
		"name": "Project One",
		"color": "#0f7a5b",
		"defaultHourlyRateMinor": 7500
	}`))
	for _, cookie := range cookies {
		createRequest.AddCookie(cookie)
	}
	router.ServeHTTP(createResponse, createRequest)

	if createResponse.Code != http.StatusCreated {
		t.Fatalf("expected create 201, got %d: %s", createResponse.Code, createResponse.Body.String())
	}

	var created struct {
		ID         string `json:"id"`
		ClientID   string `json:"clientId"`
		ClientName string `json:"clientName"`
		Color      string `json:"color"`
	}
	if err := json.Unmarshal(createResponse.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created project: %v", err)
	}
	if created.ID == "" || created.ClientID != clientID || created.ClientName != "Client One" || created.Color != "#0f7a5b" {
		t.Fatalf("unexpected created project: %+v", created)
	}

	updateResponse := httptest.NewRecorder()
	updateRequest := httptest.NewRequest(http.MethodPatch, "/api/v1/projects/"+created.ID, bytes.NewBufferString(`{
		"name": "Project One Updated",
		"color": "#2563eb",
		"defaultHourlyRateMinor": null
	}`))
	for _, cookie := range cookies {
		updateRequest.AddCookie(cookie)
	}
	router.ServeHTTP(updateResponse, updateRequest)

	if updateResponse.Code != http.StatusOK {
		t.Fatalf("expected update 200, got %d: %s", updateResponse.Code, updateResponse.Body.String())
	}

	listResponse := httptest.NewRecorder()
	listRequest := httptest.NewRequest(http.MethodGet, "/api/v1/projects", nil)
	for _, cookie := range cookies {
		listRequest.AddCookie(cookie)
	}
	router.ServeHTTP(listResponse, listRequest)

	if listResponse.Code != http.StatusOK {
		t.Fatalf("expected list 200, got %d: %s", listResponse.Code, listResponse.Body.String())
	}

	var listPayload projectsResponse
	if err := json.Unmarshal(listResponse.Body.Bytes(), &listPayload); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	if len(listPayload.Projects) != 1 || listPayload.Projects[0].Name != "Project One Updated" {
		t.Fatalf("unexpected project list: %+v", listPayload)
	}

	deleteResponse := httptest.NewRecorder()
	deleteRequest := httptest.NewRequest(http.MethodDelete, "/api/v1/projects/"+created.ID, nil)
	for _, cookie := range cookies {
		deleteRequest.AddCookie(cookie)
	}
	router.ServeHTTP(deleteResponse, deleteRequest)

	if deleteResponse.Code != http.StatusNoContent {
		t.Fatalf("expected delete 204, got %d: %s", deleteResponse.Code, deleteResponse.Body.String())
	}

	restoreResponse := httptest.NewRecorder()
	restoreRequest := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+created.ID+"/restore", nil)
	for _, cookie := range cookies {
		restoreRequest.AddCookie(cookie)
	}
	router.ServeHTTP(restoreResponse, restoreRequest)

	if restoreResponse.Code != http.StatusOK {
		t.Fatalf("expected restore 200, got %d: %s", restoreResponse.Code, restoreResponse.Body.String())
	}

	var restored struct {
		ID         string `json:"id"`
		ArchivedAt string `json:"archivedAt"`
	}
	if err := json.Unmarshal(restoreResponse.Body.Bytes(), &restored); err != nil {
		t.Fatalf("decode restored project: %v", err)
	}
	if restored.ID != created.ID || restored.ArchivedAt != "" {
		t.Fatalf("unexpected restored project: %+v", restored)
	}
}

func TestTaskHTTPLifecycle(t *testing.T) {
	router := newTestRouter(t)
	cookies := loginCookies(t, router)
	projectID := createProjectForHTTPTest(t, router, cookies)

	createResponse := httptest.NewRecorder()
	createRequest := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewBufferString(`{
		"projectId": "`+projectID+`",
		"name": "Task One",
		"billable": true
	}`))
	for _, cookie := range cookies {
		createRequest.AddCookie(cookie)
	}
	router.ServeHTTP(createResponse, createRequest)

	if createResponse.Code != http.StatusCreated {
		t.Fatalf("expected create 201, got %d: %s", createResponse.Code, createResponse.Body.String())
	}

	var created struct {
		ID          string `json:"id"`
		ProjectID   string `json:"projectId"`
		ProjectName string `json:"projectName"`
		Billable    bool   `json:"billable"`
	}
	if err := json.Unmarshal(createResponse.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created task: %v", err)
	}
	if created.ID == "" || created.ProjectID != projectID || created.ProjectName != "Project One" || !created.Billable {
		t.Fatalf("unexpected created task: %+v", created)
	}

	updateResponse := httptest.NewRecorder()
	updateRequest := httptest.NewRequest(http.MethodPatch, "/api/v1/tasks/"+created.ID, bytes.NewBufferString(`{
		"name": "Task One Updated",
		"billable": false
	}`))
	for _, cookie := range cookies {
		updateRequest.AddCookie(cookie)
	}
	router.ServeHTTP(updateResponse, updateRequest)

	if updateResponse.Code != http.StatusOK {
		t.Fatalf("expected update 200, got %d: %s", updateResponse.Code, updateResponse.Body.String())
	}

	listResponse := httptest.NewRecorder()
	listRequest := httptest.NewRequest(http.MethodGet, "/api/v1/tasks", nil)
	for _, cookie := range cookies {
		listRequest.AddCookie(cookie)
	}
	router.ServeHTTP(listResponse, listRequest)

	if listResponse.Code != http.StatusOK {
		t.Fatalf("expected list 200, got %d: %s", listResponse.Code, listResponse.Body.String())
	}

	var listPayload tasksResponse
	if err := json.Unmarshal(listResponse.Body.Bytes(), &listPayload); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	if len(listPayload.Tasks) != 1 || listPayload.Tasks[0].Name != "Task One Updated" || listPayload.Tasks[0].Billable {
		t.Fatalf("unexpected task list: %+v", listPayload)
	}

	deleteResponse := httptest.NewRecorder()
	deleteRequest := httptest.NewRequest(http.MethodDelete, "/api/v1/tasks/"+created.ID, nil)
	for _, cookie := range cookies {
		deleteRequest.AddCookie(cookie)
	}
	router.ServeHTTP(deleteResponse, deleteRequest)

	if deleteResponse.Code != http.StatusNoContent {
		t.Fatalf("expected delete 204, got %d: %s", deleteResponse.Code, deleteResponse.Body.String())
	}

	restoreResponse := httptest.NewRecorder()
	restoreRequest := httptest.NewRequest(http.MethodPost, "/api/v1/tasks/"+created.ID+"/restore", nil)
	for _, cookie := range cookies {
		restoreRequest.AddCookie(cookie)
	}
	router.ServeHTTP(restoreResponse, restoreRequest)

	if restoreResponse.Code != http.StatusOK {
		t.Fatalf("expected restore 200, got %d: %s", restoreResponse.Code, restoreResponse.Body.String())
	}

	var restored struct {
		ID         string `json:"id"`
		ArchivedAt string `json:"archivedAt"`
	}
	if err := json.Unmarshal(restoreResponse.Body.Bytes(), &restored); err != nil {
		t.Fatalf("decode restored task: %v", err)
	}
	if restored.ID != created.ID || restored.ArchivedAt != "" {
		t.Fatalf("unexpected restored task: %+v", restored)
	}
}

func TestTagHTTPLifecycle(t *testing.T) {
	router := newTestRouter(t)
	cookies := loginCookies(t, router)

	createResponse := httptest.NewRecorder()
	createRequest := httptest.NewRequest(http.MethodPost, "/api/v1/tags", bytes.NewBufferString(`{
		"name": "Deep Work",
		"color": "#2563eb"
	}`))
	for _, cookie := range cookies {
		createRequest.AddCookie(cookie)
	}
	router.ServeHTTP(createResponse, createRequest)

	if createResponse.Code != http.StatusCreated {
		t.Fatalf("expected create 201, got %d: %s", createResponse.Code, createResponse.Body.String())
	}

	var created struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Color string `json:"color"`
	}
	if err := json.Unmarshal(createResponse.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created tag: %v", err)
	}
	if created.ID == "" || created.Name != "Deep Work" || created.Color != "#2563eb" {
		t.Fatalf("unexpected created tag: %+v", created)
	}

	updateResponse := httptest.NewRecorder()
	updateRequest := httptest.NewRequest(http.MethodPatch, "/api/v1/tags/"+created.ID, bytes.NewBufferString(`{
		"name": "Focus",
		"color": "#0f7a5b"
	}`))
	for _, cookie := range cookies {
		updateRequest.AddCookie(cookie)
	}
	router.ServeHTTP(updateResponse, updateRequest)

	if updateResponse.Code != http.StatusOK {
		t.Fatalf("expected update 200, got %d: %s", updateResponse.Code, updateResponse.Body.String())
	}

	listResponse := httptest.NewRecorder()
	listRequest := httptest.NewRequest(http.MethodGet, "/api/v1/tags", nil)
	for _, cookie := range cookies {
		listRequest.AddCookie(cookie)
	}
	router.ServeHTTP(listResponse, listRequest)

	if listResponse.Code != http.StatusOK {
		t.Fatalf("expected list 200, got %d: %s", listResponse.Code, listResponse.Body.String())
	}

	var listPayload tagsResponse
	if err := json.Unmarshal(listResponse.Body.Bytes(), &listPayload); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	if len(listPayload.Tags) != 1 || listPayload.Tags[0].Name != "Focus" {
		t.Fatalf("unexpected tag list: %+v", listPayload)
	}

	deleteResponse := httptest.NewRecorder()
	deleteRequest := httptest.NewRequest(http.MethodDelete, "/api/v1/tags/"+created.ID, nil)
	for _, cookie := range cookies {
		deleteRequest.AddCookie(cookie)
	}
	router.ServeHTTP(deleteResponse, deleteRequest)

	if deleteResponse.Code != http.StatusNoContent {
		t.Fatalf("expected delete 204, got %d: %s", deleteResponse.Code, deleteResponse.Body.String())
	}

	restoreResponse := httptest.NewRecorder()
	restoreRequest := httptest.NewRequest(http.MethodPost, "/api/v1/tags/"+created.ID+"/restore", nil)
	for _, cookie := range cookies {
		restoreRequest.AddCookie(cookie)
	}
	router.ServeHTTP(restoreResponse, restoreRequest)

	if restoreResponse.Code != http.StatusOK {
		t.Fatalf("expected restore 200, got %d: %s", restoreResponse.Code, restoreResponse.Body.String())
	}

	var restored struct {
		ID         string `json:"id"`
		ArchivedAt string `json:"archivedAt"`
	}
	if err := json.Unmarshal(restoreResponse.Body.Bytes(), &restored); err != nil {
		t.Fatalf("decode restored tag: %v", err)
	}
	if restored.ID != created.ID || restored.ArchivedAt != "" {
		t.Fatalf("unexpected restored tag: %+v", restored)
	}
}

func TestTimeEntryHTTPLifecycle(t *testing.T) {
	router := newTestRouter(t)
	cookies := loginCookies(t, router)

	createResponse := httptest.NewRecorder()
	createRequest := httptest.NewRequest(http.MethodPost, "/api/v1/time-entries", bytes.NewBufferString(`{
		"description": "Manual work",
		"startedAt": "2026-06-29T08:04:00Z",
		"endedAt": "2026-06-29T10:55:00Z",
		"billable": true
	}`))
	for _, cookie := range cookies {
		createRequest.AddCookie(cookie)
	}
	router.ServeHTTP(createResponse, createRequest)

	if createResponse.Code != http.StatusCreated {
		t.Fatalf("expected create 201, got %d: %s", createResponse.Code, createResponse.Body.String())
	}

	var created struct {
		ID              string `json:"id"`
		DurationSeconds int    `json:"durationSeconds"`
		Source          string `json:"source"`
	}
	if err := json.Unmarshal(createResponse.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created time entry: %v", err)
	}
	if created.ID == "" || created.DurationSeconds != 10260 || created.Source != "manual" {
		t.Fatalf("unexpected created time entry: %+v", created)
	}

	listResponse := httptest.NewRecorder()
	listRequest := httptest.NewRequest(http.MethodGet, "/api/v1/time-entries", nil)
	for _, cookie := range cookies {
		listRequest.AddCookie(cookie)
	}
	router.ServeHTTP(listResponse, listRequest)

	if listResponse.Code != http.StatusOK {
		t.Fatalf("expected list 200, got %d: %s", listResponse.Code, listResponse.Body.String())
	}

	deleteResponse := httptest.NewRecorder()
	deleteRequest := httptest.NewRequest(http.MethodDelete, "/api/v1/time-entries/"+created.ID, nil)
	for _, cookie := range cookies {
		deleteRequest.AddCookie(cookie)
	}
	router.ServeHTTP(deleteResponse, deleteRequest)

	if deleteResponse.Code != http.StatusNoContent {
		t.Fatalf("expected delete 204, got %d: %s", deleteResponse.Code, deleteResponse.Body.String())
	}
}

func TestTimerHTTPLifecycle(t *testing.T) {
	router := newTestRouter(t)
	cookies := loginCookies(t, router)

	startResponse := httptest.NewRecorder()
	startRequest := httptest.NewRequest(http.MethodPost, "/api/v1/timers", bytes.NewBufferString(`{
		"description": "Timer work",
		"billable": true
	}`))
	for _, cookie := range cookies {
		startRequest.AddCookie(cookie)
	}
	router.ServeHTTP(startResponse, startRequest)

	if startResponse.Code != http.StatusCreated {
		t.Fatalf("expected start 201, got %d: %s", startResponse.Code, startResponse.Body.String())
	}

	var started struct {
		ID      string `json:"id"`
		Source  string `json:"source"`
		EndedAt string `json:"endedAt"`
	}
	if err := json.Unmarshal(startResponse.Body.Bytes(), &started); err != nil {
		t.Fatalf("decode started timer: %v", err)
	}
	if started.ID == "" || started.Source != "timer" || started.EndedAt != "" {
		t.Fatalf("unexpected started timer: %+v", started)
	}

	listResponse := httptest.NewRecorder()
	listRequest := httptest.NewRequest(http.MethodGet, "/api/v1/timers", nil)
	for _, cookie := range cookies {
		listRequest.AddCookie(cookie)
	}
	router.ServeHTTP(listResponse, listRequest)

	if listResponse.Code != http.StatusOK {
		t.Fatalf("expected list 200, got %d: %s", listResponse.Code, listResponse.Body.String())
	}

	updatedStart := time.Now().UTC().Add(-30 * time.Minute).Truncate(time.Minute)
	updateResponse := httptest.NewRecorder()
	updateRequest := httptest.NewRequest(http.MethodPatch, "/api/v1/timers/"+started.ID, bytes.NewBufferString(`{
		"description": "Timer work",
		"startedAt": "`+updatedStart.Format(time.RFC3339Nano)+`",
		"billable": true
	}`))
	for _, cookie := range cookies {
		updateRequest.AddCookie(cookie)
	}
	router.ServeHTTP(updateResponse, updateRequest)

	if updateResponse.Code != http.StatusOK {
		t.Fatalf("expected update 200, got %d: %s", updateResponse.Code, updateResponse.Body.String())
	}

	var updated struct {
		StartedAt string `json:"startedAt"`
	}
	if err := json.Unmarshal(updateResponse.Body.Bytes(), &updated); err != nil {
		t.Fatalf("decode updated timer: %v", err)
	}
	if updated.StartedAt == "" {
		t.Fatalf("expected updated startedAt")
	}

	stopResponse := httptest.NewRecorder()
	stopRequest := httptest.NewRequest(http.MethodPost, "/api/v1/timers/"+started.ID+"/stop", nil)
	for _, cookie := range cookies {
		stopRequest.AddCookie(cookie)
	}
	router.ServeHTTP(stopResponse, stopRequest)

	if stopResponse.Code != http.StatusOK {
		t.Fatalf("expected stop 200, got %d: %s", stopResponse.Code, stopResponse.Body.String())
	}

	var stopped struct {
		EndedAt         string `json:"endedAt"`
		DurationSeconds int    `json:"durationSeconds"`
	}
	if err := json.Unmarshal(stopResponse.Body.Bytes(), &stopped); err != nil {
		t.Fatalf("decode stopped timer: %v", err)
	}
	if stopped.EndedAt == "" || stopped.DurationSeconds < 60 {
		t.Fatalf("unexpected stopped timer: %+v", stopped)
	}
}

func TestTimeReportExport(t *testing.T) {
	router := newTestRouter(t)
	cookies := loginCookies(t, router)

	createResponse := httptest.NewRecorder()
	createRequest := httptest.NewRequest(http.MethodPost, "/api/v1/time-entries", bytes.NewBufferString(`{
		"description": "Report work",
		"startedAt": "2026-07-01T08:00:00Z",
		"endedAt": "2026-07-01T09:00:00Z",
		"billable": true
	}`))
	for _, cookie := range cookies {
		createRequest.AddCookie(cookie)
	}
	router.ServeHTTP(createResponse, createRequest)
	if createResponse.Code != http.StatusCreated {
		t.Fatalf("expected create 201, got %d: %s", createResponse.Code, createResponse.Body.String())
	}

	summaryResponse := httptest.NewRecorder()
	summaryRequest := httptest.NewRequest(http.MethodGet, "/api/v1/reports/time?from=2026-07-01T00:00:00Z&to=2026-07-31T23:59:59Z&groupBy=day", nil)
	for _, cookie := range cookies {
		summaryRequest.AddCookie(cookie)
	}
	router.ServeHTTP(summaryResponse, summaryRequest)
	if summaryResponse.Code != http.StatusOK {
		t.Fatalf("expected summary 200, got %d: %s", summaryResponse.Code, summaryResponse.Body.String())
	}

	csvResponse := httptest.NewRecorder()
	csvRequest := httptest.NewRequest(http.MethodGet, "/api/v1/reports/time/export?format=csv&from=2026-07-01T00:00:00Z&to=2026-07-31T23:59:59Z&includeTimestamps=true", nil)
	for _, cookie := range cookies {
		csvRequest.AddCookie(cookie)
	}
	router.ServeHTTP(csvResponse, csvRequest)
	if csvResponse.Code != http.StatusOK {
		t.Fatalf("expected csv 200, got %d: %s", csvResponse.Code, csvResponse.Body.String())
	}
	if !strings.Contains(csvResponse.Body.String(), "started_at") || !strings.Contains(csvResponse.Body.String(), "Report work") {
		t.Fatalf("unexpected csv body: %s", csvResponse.Body.String())
	}
}

func TestInvoiceDraftFromTimeAndExport(t *testing.T) {
	router := newTestRouter(t)
	cookies := loginCookies(t, router)

	clientResponse := httptest.NewRecorder()
	clientRequest := httptest.NewRequest(http.MethodPost, "/api/v1/clients", bytes.NewBufferString(`{
		"name": "Invoice Client",
		"defaultCurrency": "EUR",
		"defaultHourlyRateMinor": 12000
	}`))
	for _, cookie := range cookies {
		clientRequest.AddCookie(cookie)
	}
	router.ServeHTTP(clientResponse, clientRequest)
	if clientResponse.Code != http.StatusCreated {
		t.Fatalf("expected client 201, got %d: %s", clientResponse.Code, clientResponse.Body.String())
	}

	var client struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(clientResponse.Body.Bytes(), &client); err != nil {
		t.Fatalf("decode client: %v", err)
	}

	entryResponse := httptest.NewRecorder()
	entryRequest := httptest.NewRequest(http.MethodPost, "/api/v1/time-entries", bytes.NewBufferString(`{
		"clientId": "`+client.ID+`",
		"description": "Invoiceable work",
		"startedAt": "2026-07-04T09:00:00Z",
		"endedAt": "2026-07-04T11:00:00Z",
		"billable": true
	}`))
	for _, cookie := range cookies {
		entryRequest.AddCookie(cookie)
	}
	router.ServeHTTP(entryResponse, entryRequest)
	if entryResponse.Code != http.StatusCreated {
		t.Fatalf("expected entry 201, got %d: %s", entryResponse.Code, entryResponse.Body.String())
	}

	draftResponse := httptest.NewRecorder()
	draftRequest := httptest.NewRequest(http.MethodPost, "/api/v1/invoices/draft-from-time", bytes.NewBufferString(`{
		"clientId": "`+client.ID+`",
		"from": "2026-07-01T00:00:00Z",
		"to": "2026-07-31T23:59:59Z",
		"taxRateBasisPoints": 2100
	}`))
	for _, cookie := range cookies {
		draftRequest.AddCookie(cookie)
	}
	router.ServeHTTP(draftResponse, draftRequest)
	if draftResponse.Code != http.StatusCreated {
		t.Fatalf("expected draft 201, got %d: %s", draftResponse.Code, draftResponse.Body.String())
	}

	var invoice struct {
		ID            string `json:"id"`
		InvoiceNumber string `json:"invoiceNumber"`
		Lines         []struct {
			QuantityMinutes int `json:"quantityMinutes"`
		} `json:"lines"`
	}
	if err := json.Unmarshal(draftResponse.Body.Bytes(), &invoice); err != nil {
		t.Fatalf("decode invoice: %v", err)
	}
	if invoice.ID == "" || invoice.InvoiceNumber == "" || len(invoice.Lines) != 1 || invoice.Lines[0].QuantityMinutes != 120 {
		t.Fatalf("unexpected invoice payload: %+v", invoice)
	}

	listResponse := httptest.NewRecorder()
	listRequest := httptest.NewRequest(http.MethodGet, "/api/v1/invoices", nil)
	for _, cookie := range cookies {
		listRequest.AddCookie(cookie)
	}
	router.ServeHTTP(listResponse, listRequest)
	if listResponse.Code != http.StatusOK {
		t.Fatalf("expected list 200, got %d: %s", listResponse.Code, listResponse.Body.String())
	}

	htmlResponse := httptest.NewRecorder()
	htmlRequest := httptest.NewRequest(http.MethodGet, "/api/v1/invoices/"+invoice.ID+"/export?format=html", nil)
	for _, cookie := range cookies {
		htmlRequest.AddCookie(cookie)
	}
	router.ServeHTTP(htmlResponse, htmlRequest)
	if htmlResponse.Code != http.StatusOK {
		t.Fatalf("expected html export 200, got %d: %s", htmlResponse.Code, htmlResponse.Body.String())
	}
	if !strings.Contains(htmlResponse.Body.String(), invoice.InvoiceNumber) {
		t.Fatalf("expected html export to contain invoice number")
	}

	statusResponse := httptest.NewRecorder()
	statusRequest := httptest.NewRequest(http.MethodPost, "/api/v1/invoices/"+invoice.ID+"/status", bytes.NewBufferString(`{"status":"issued"}`))
	for _, cookie := range cookies {
		statusRequest.AddCookie(cookie)
	}
	router.ServeHTTP(statusResponse, statusRequest)
	if statusResponse.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", statusResponse.Code, statusResponse.Body.String())
	}
}

func TestDashboardStatsHTTP(t *testing.T) {
	router := newTestRouter(t)
	cookies := loginCookies(t, router)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/stats", nil)
	for _, cookie := range cookies {
		request.AddCookie(cookie)
	}
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected dashboard 200, got %d: %s", response.Code, response.Body.String())
	}

	var stats struct {
		RecentEntries []any `json:"recentEntries"`
		LastSevenDays []any `json:"lastSevenDays"`
		WeekDays      []any `json:"weekDays"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &stats); err != nil {
		t.Fatalf("decode dashboard stats: %v", err)
	}
	if len(stats.LastSevenDays) != 7 || len(stats.WeekDays) != 7 {
		t.Fatalf("unexpected dashboard payload: %+v", stats)
	}
}

func newTestRouter(t *testing.T) http.Handler {
	t.Helper()

	ctx := context.Background()
	database, err := db.Open(ctx, t.TempDir()+"/leotime.db")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() {
		database.Close()
	})

	if err := db.Migrate(ctx, database); err != nil {
		t.Fatalf("migrate db: %v", err)
	}

	st := store.New(database)
	if err := st.BootstrapAdmin(ctx, "admin@example.com", "change-me-now"); err != nil {
		t.Fatalf("bootstrap admin: %v", err)
	}

	cfg := config.Config{
		HTTPAddr:          ":0",
		DBPath:            "unused",
		BootstrapEmail:    "admin@example.com",
		BootstrapPassword: "change-me-now",
		SessionCookieName: "leotime_session",
		SessionTTL:        time.Hour,
		PublicBaseURL:     "http://127.0.0.1:8080",
		PasswordResetTTL:  time.Hour,
		MailMaxAttempts:   5,
	}

	outboxStore := outbox.NewStore(database)
	passwordReset := notify.NewPasswordResetService(st, outboxStore, cfg)
	return NewRouter(cfg, st, passwordReset)
}

func loginCookies(t *testing.T, router http.Handler) []*http.Cookie {
	t.Helper()

	loginBody := bytes.NewBufferString(`{"email":"admin@example.com","password":"change-me-now"}`)
	loginResponse := httptest.NewRecorder()
	loginRequest := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", loginBody)
	router.ServeHTTP(loginResponse, loginRequest)

	if loginResponse.Code != http.StatusOK {
		t.Fatalf("expected login 200, got %d: %s", loginResponse.Code, loginResponse.Body.String())
	}
	return loginResponse.Result().Cookies()
}

func createClientForHTTPTest(t *testing.T, router http.Handler, cookies []*http.Cookie) string {
	t.Helper()

	createResponse := httptest.NewRecorder()
	createRequest := httptest.NewRequest(http.MethodPost, "/api/v1/clients", bytes.NewBufferString(`{
		"name": "Client One",
		"defaultCurrency": "EUR"
	}`))
	for _, cookie := range cookies {
		createRequest.AddCookie(cookie)
	}
	router.ServeHTTP(createResponse, createRequest)

	if createResponse.Code != http.StatusCreated {
		t.Fatalf("expected client create 201, got %d: %s", createResponse.Code, createResponse.Body.String())
	}

	var created struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(createResponse.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created client: %v", err)
	}
	return created.ID
}

func createProjectForHTTPTest(t *testing.T, router http.Handler, cookies []*http.Cookie) string {
	t.Helper()

	createResponse := httptest.NewRecorder()
	createRequest := httptest.NewRequest(http.MethodPost, "/api/v1/projects", bytes.NewBufferString(`{
		"name": "Project One",
		"color": "#2563eb"
	}`))
	for _, cookie := range cookies {
		createRequest.AddCookie(cookie)
	}
	router.ServeHTTP(createResponse, createRequest)

	if createResponse.Code != http.StatusCreated {
		t.Fatalf("expected project create 201, got %d: %s", createResponse.Code, createResponse.Body.String())
	}

	var created struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(createResponse.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created project: %v", err)
	}
	return created.ID
}

func TestForgotPasswordAlwaysReturnsNoContent(t *testing.T) {
	router := newTestRouter(t)

	for _, body := range []string{
		`{"email":"admin@example.com"}`,
		`{"email":"missing@example.com"}`,
	} {
		response := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/forgot-password", bytes.NewBufferString(body))
		router.ServeHTTP(response, request)
		if response.Code != http.StatusNoContent {
			t.Fatalf("expected 204 for %s, got %d: %s", body, response.Code, response.Body.String())
		}
	}
}

func TestResetPasswordWithToken(t *testing.T) {
	ctx := context.Background()
	database, err := db.Open(ctx, t.TempDir()+"/reset.db")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	if err := db.Migrate(ctx, database); err != nil {
		t.Fatalf("migrate db: %v", err)
	}
	st := store.New(database)
	if err := st.BootstrapAdmin(ctx, "admin@example.com", "change-me-now"); err != nil {
		t.Fatalf("bootstrap admin: %v", err)
	}
	user, err := st.Authenticate(ctx, "admin@example.com", "change-me-now")
	if err != nil {
		t.Fatalf("authenticate: %v", err)
	}
	rawToken, err := st.CreatePasswordResetToken(ctx, user.ID, time.Hour)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}

	cfg := config.Config{
		SessionCookieName: "leotime_session",
		SessionTTL:        time.Hour,
		PublicBaseURL:     "http://127.0.0.1:8080",
		PasswordResetTTL:  time.Hour,
		MailMaxAttempts:   5,
	}
	router := NewRouter(cfg, st, notify.NewPasswordResetService(st, outbox.NewStore(database), cfg))

	resetResponse := httptest.NewRecorder()
	resetRequest := httptest.NewRequest(http.MethodPost, "/api/v1/auth/reset-password", bytes.NewBufferString(`{"token":"`+rawToken+`","newPassword":"brand-new-password"}`))
	router.ServeHTTP(resetResponse, resetRequest)
	if resetResponse.Code != http.StatusNoContent {
		t.Fatalf("expected reset 204, got %d: %s", resetResponse.Code, resetResponse.Body.String())
	}

	if _, err := st.Authenticate(ctx, "admin@example.com", "brand-new-password"); err != nil {
		t.Fatalf("authenticate with reset password: %v", err)
	}
}
