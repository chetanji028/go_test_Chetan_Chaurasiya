package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Server struct {
	dataStore *DataStore
}

func NewServer(dataStore *DataStore) *Server {
	return &Server{
		dataStore: dataStore,
	}
}

func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/api/users", s.handleUsers)
	mux.HandleFunc("/api/users/", s.handleUserByID)
	mux.HandleFunc("/api/tasks", s.handleTasks)
	mux.HandleFunc("/api/tasks/", s.handleTaskByID)
	mux.HandleFunc("/api/stats", s.handleStats)
	return loggingMiddleware(mux)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	response := HealthResponse{
		Status:  "ok",
		Message: "Go backend is running",
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleUsers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	switch r.Method {
	case http.MethodGet:
		users := s.dataStore.GetUsers()
		response := UsersResponse{
			Users: users,
			Count: len(users),
		}
		json.NewEncoder(w).Encode(response)
	case http.MethodPost:
		var request User
		if err := decodeJSON(r, &request); err != nil {
			log.Printf("request_error method=%s path=%s error=%q", r.Method, r.URL.Path, err)
			writeError(w, http.StatusBadRequest, "Invalid JSON request body")
			return
		}
		request.Name = strings.TrimSpace(request.Name)
		request.Email = strings.TrimSpace(request.Email)
		request.Role = strings.TrimSpace(request.Role)
		if request.Name == "" || request.Email == "" || request.Role == "" {
			writeError(w, http.StatusBadRequest, "name, email, and role are required")
			return
		}
		if !isValidEmail(request.Email) {
			writeError(w, http.StatusBadRequest, "email must be a valid email address")
			return
		}

		user := s.dataStore.CreateUser(User{
			Name:  request.Name,
			Email: request.Email,
			Role:  request.Role,
		})
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(user)
	default:
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func (s *Server) handleUserByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Extract ID from path
	path := strings.TrimPrefix(r.URL.Path, "/api/users/")
	id, err := strconv.Atoi(path)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	user := s.dataStore.GetUserByID(id)
	if user == nil {
		writeError(w, http.StatusNotFound, "User not found")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(user)
}

func (s *Server) handleTasks(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	switch r.Method {
	case http.MethodGet:
		status := r.URL.Query().Get("status")
		userID := r.URL.Query().Get("userId")

		tasks := s.dataStore.GetTasks(status, userID)
		response := TasksResponse{
			Tasks: tasks,
			Count: len(tasks),
		}

		json.NewEncoder(w).Encode(response)
	case http.MethodPost:
		var request Task
		if err := decodeJSON(r, &request); err != nil {
			log.Printf("request_error method=%s path=%s error=%q", r.Method, r.URL.Path, err)
			writeError(w, http.StatusBadRequest, "Invalid JSON request body")
			return
		}
		request.Title = strings.TrimSpace(request.Title)
		request.Status = strings.TrimSpace(request.Status)
		if request.Title == "" || request.Status == "" || request.UserID == 0 {
			writeError(w, http.StatusBadRequest, "title, status, and userId are required")
			return
		}
		if !isValidTaskStatus(request.Status) {
			writeError(w, http.StatusBadRequest, "status must be one of: pending, in-progress, completed")
			return
		}

		task, err := s.dataStore.CreateTask(Task{
			Title:  request.Title,
			Status: request.Status,
			UserID: request.UserID,
		})
		if err != nil {
			if errors.Is(err, ErrUserNotFound) {
				writeError(w, http.StatusBadRequest, "userId does not reference an existing user")
				return
			}
			log.Printf("server_error method=%s path=%s error=%q", r.Method, r.URL.Path, err)
			writeError(w, http.StatusInternalServerError, "Unable to create task")
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(task)
	default:
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func (s *Server) handleTaskByID(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != http.MethodPut {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/tasks/")
	id, err := strconv.Atoi(path)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid task ID")
		return
	}

	var request TaskUpdate
	if err := decodeJSON(r, &request); err != nil {
		log.Printf("request_error method=%s path=%s error=%q", r.Method, r.URL.Path, err)
		writeError(w, http.StatusBadRequest, "Invalid JSON request body")
		return
	}

	if request.Title != nil {
		trimmedTitle := strings.TrimSpace(*request.Title)
		if trimmedTitle == "" {
			writeError(w, http.StatusBadRequest, "title cannot be empty")
			return
		}
		request.Title = &trimmedTitle
	}
	if request.Status != nil {
		trimmedStatus := strings.TrimSpace(*request.Status)
		if !isValidTaskStatus(trimmedStatus) {
			writeError(w, http.StatusBadRequest, "status must be one of: pending, in-progress, completed")
			return
		}
		request.Status = &trimmedStatus
	}
	if request.UserID != nil && *request.UserID == 0 {
		writeError(w, http.StatusBadRequest, "userId must reference an existing user")
		return
	}

	task, err := s.dataStore.UpdateTask(id, request)
	if err != nil {
		switch {
		case errors.Is(err, ErrTaskNotFound):
			writeError(w, http.StatusNotFound, "Task not found")
		case errors.Is(err, ErrUserNotFound):
			writeError(w, http.StatusBadRequest, "userId does not reference an existing user")
		default:
			log.Printf("server_error method=%s path=%s error=%q", r.Method, r.URL.Path, err)
			writeError(w, http.StatusInternalServerError, "Unable to update task")
		}
		return
	}

	json.NewEncoder(w).Encode(task)
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	stats := s.dataStore.GetStats()

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(stats)
}

func (s *Server) Start(port string) {
	if port == "" {
		port = defaultPort
	}

	log.Printf("Go backend server starting on http://localhost:%s", port)
	log.Printf("Serving data directly from Go backend")

	if err := http.ListenAndServe(":"+port, s.routes()); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

type errorResponse struct {
	Error string `json:"error"`
}

type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (sr *statusRecorder) WriteHeader(statusCode int) {
	sr.statusCode = statusCode
	sr.ResponseWriter.WriteHeader(statusCode)
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		recorder := &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(recorder, r)
		log.Printf(
			"request method=%s path=%s status=%d duration=%s",
			r.Method,
			r.URL.RequestURI(),
			recorder.statusCode,
			time.Since(start),
		)
	})
}

func decodeJSON(r *http.Request, target any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(target)
}

func writeError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(errorResponse{Error: message})
}

func isValidEmail(email string) bool {
	at := strings.Index(email, "@")
	dot := strings.LastIndex(email, ".")
	return at > 0 && dot > at+1 && dot < len(email)-1
}

func isValidTaskStatus(status string) bool {
	switch status {
	case "pending", "in-progress", "completed":
		return true
	default:
		return false
	}
}
