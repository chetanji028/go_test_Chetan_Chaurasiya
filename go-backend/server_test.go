package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreateUser(t *testing.T) {
	server := NewServer(testDataStore())

	response := performRequest(server, http.MethodPost, "/api/users", `{
		"name": "Ada Lovelace",
		"email": "ada@example.com",
		"role": "developer"
	}`)

	if response.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, response.Code, response.Body.String())
	}

	var user User
	if err := json.NewDecoder(response.Body).Decode(&user); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if user.ID != 4 || user.Name != "Ada Lovelace" || user.Email != "ada@example.com" || user.Role != "developer" {
		t.Fatalf("unexpected user: %+v", user)
	}

	users := server.dataStore.GetUsers()
	if len(users) != 4 {
		t.Fatalf("expected created user to be stored, got %d users", len(users))
	}
}

func TestGetUserByID(t *testing.T) {
	server := NewServer(testDataStore())

	response := performRequest(server, http.MethodGet, "/api/users/1", "")

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, response.Code, response.Body.String())
	}

	var user User
	if err := json.NewDecoder(response.Body).Decode(&user); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if user.ID != 1 || user.Name != "John Doe" {
		t.Fatalf("unexpected user: %+v", user)
	}
}

func TestGetUserByIDErrors(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		statusCode int
		message    string
	}{
		{name: "invalid ID", path: "/api/users/nope", statusCode: http.StatusBadRequest, message: "invalid"},
		{name: "missing user", path: "/api/users/999", statusCode: http.StatusNotFound, message: "not found"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := NewServer(testDataStore())
			response := performRequest(server, http.MethodGet, tt.path, "")
			assertError(t, response, tt.statusCode, tt.message)
		})
	}
}

func TestCreateUserValidation(t *testing.T) {
	server := NewServer(testDataStore())

	response := performRequest(server, http.MethodPost, "/api/users", `{
		"name": "Ada Lovelace",
		"email": "not-an-email",
		"role": "developer"
	}`)

	assertError(t, response, http.StatusBadRequest, "email")
}

func TestHealthAndStats(t *testing.T) {
	server := NewServer(testDataStore())

	healthResponse := performRequest(server, http.MethodGet, "/health", "")
	if healthResponse.Code != http.StatusOK {
		t.Fatalf("expected health status %d, got %d: %s", http.StatusOK, healthResponse.Code, healthResponse.Body.String())
	}

	statsResponse := performRequest(server, http.MethodGet, "/api/stats", "")
	if statsResponse.Code != http.StatusOK {
		t.Fatalf("expected stats status %d, got %d: %s", http.StatusOK, statsResponse.Code, statsResponse.Body.String())
	}

	var stats StatsResponse
	if err := json.NewDecoder(statsResponse.Body).Decode(&stats); err != nil {
		t.Fatalf("decode stats: %v", err)
	}
	if stats.Users.Total != 3 || stats.Tasks.Total != 3 {
		t.Fatalf("unexpected stats: %+v", stats)
	}
}

func TestCreateTask(t *testing.T) {
	server := NewServer(testDataStore())

	response := performRequest(server, http.MethodPost, "/api/tasks", `{
		"title": "Ship API",
		"status": "pending",
		"userId": 1
	}`)

	if response.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, response.Code, response.Body.String())
	}

	var task Task
	if err := json.NewDecoder(response.Body).Decode(&task); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if task.ID != 4 || task.Title != "Ship API" || task.Status != "pending" || task.UserID != 1 {
		t.Fatalf("unexpected task: %+v", task)
	}
}

func TestCreateTaskValidation(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		message string
	}{
		{
			name:    "invalid status",
			body:    `{"title":"Ship API","status":"blocked","userId":1}`,
			message: "status",
		},
		{
			name:    "missing user",
			body:    `{"title":"Ship API","status":"pending","userId":999}`,
			message: "userId",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := NewServer(testDataStore())
			response := performRequest(server, http.MethodPost, "/api/tasks", tt.body)
			assertError(t, response, http.StatusBadRequest, tt.message)
		})
	}
}

func TestUpdateTaskPartial(t *testing.T) {
	server := NewServer(testDataStore())

	response := performRequest(server, http.MethodPut, "/api/tasks/1", `{"status":"completed"}`)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, response.Code, response.Body.String())
	}

	var task Task
	if err := json.NewDecoder(response.Body).Decode(&task); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if task.ID != 1 || task.Title != "Implement authentication" || task.Status != "completed" || task.UserID != 1 {
		t.Fatalf("unexpected task: %+v", task)
	}
}

func TestUpdateTaskValidation(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		body       string
		statusCode int
		message    string
	}{
		{
			name:       "missing task",
			path:       "/api/tasks/999",
			body:       `{"status":"completed"}`,
			statusCode: http.StatusNotFound,
			message:    "not found",
		},
		{
			name:       "invalid status",
			path:       "/api/tasks/1",
			body:       `{"status":"blocked"}`,
			statusCode: http.StatusBadRequest,
			message:    "status",
		},
		{
			name:       "missing user",
			path:       "/api/tasks/1",
			body:       `{"userId":999}`,
			statusCode: http.StatusBadRequest,
			message:    "userId",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := NewServer(testDataStore())
			response := performRequest(server, http.MethodPut, tt.path, tt.body)
			assertError(t, response, tt.statusCode, tt.message)
		})
	}
}

func performRequest(server *Server, method, path, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	server.routes().ServeHTTP(response, request)
	return response
}

func assertError(t *testing.T, response *httptest.ResponseRecorder, statusCode int, messagePart string) {
	t.Helper()

	if response.Code != statusCode {
		t.Fatalf("expected status %d, got %d: %s", statusCode, response.Code, response.Body.String())
	}

	var errorResponse errorResponse
	if err := json.NewDecoder(response.Body).Decode(&errorResponse); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if !strings.Contains(strings.ToLower(errorResponse.Error), strings.ToLower(messagePart)) {
		t.Fatalf("expected error containing %q, got %q", messagePart, errorResponse.Error)
	}
}

func testDataStore() *DataStore {
	return &DataStore{
		users: []User{
			{ID: 1, Name: "John Doe", Email: "john@example.com", Role: "developer"},
			{ID: 2, Name: "Jane Smith", Email: "jane@example.com", Role: "designer"},
			{ID: 3, Name: "Bob Johnson", Email: "bob@example.com", Role: "manager"},
		},
		tasks: []Task{
			{ID: 1, Title: "Implement authentication", Status: "pending", UserID: 1},
			{ID: 2, Title: "Design user interface", Status: "in-progress", UserID: 2},
			{ID: 3, Title: "Review code changes", Status: "completed", UserID: 3},
		},
	}
}
