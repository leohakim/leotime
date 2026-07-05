package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/leotime/leotime/apps/api/internal/config"
	"github.com/leotime/leotime/apps/api/internal/db"
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

	return NewRouter(config.Config{
		HTTPAddr:          ":0",
		DBPath:            "unused",
		BootstrapEmail:    "admin@example.com",
		BootstrapPassword: "change-me-now",
		SessionCookieName: "leotime_session",
		SessionTTL:        time.Hour,
	}, st)
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
