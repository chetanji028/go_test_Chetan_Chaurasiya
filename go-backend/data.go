package main

import (
	"errors"
	"strconv"
	"sync"
)

var (
	ErrTaskNotFound = errors.New("task not found")
	ErrUserNotFound = errors.New("user not found")
)

// DataStore holds all application data
type DataStore struct {
	mu    sync.RWMutex
	users []User
	tasks []Task
}

var store = &DataStore{
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

func (ds *DataStore) GetUsers() []User {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	users := make([]User, len(ds.users))
	copy(users, ds.users)
	return users
}

func (ds *DataStore) GetUserByID(id int) *User {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	for i := range ds.users {
		if ds.users[i].ID == id {
			user := ds.users[i]
			return &user
		}
	}
	return nil
}

func (ds *DataStore) CreateUser(user User) User {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	maxID := 0
	for _, existingUser := range ds.users {
		if existingUser.ID > maxID {
			maxID = existingUser.ID
		}
	}

	user.ID = maxID + 1
	ds.users = append(ds.users, user)
	return user
}

func (ds *DataStore) GetTasks(status, userID string) []Task {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	var filtered []Task
	for _, task := range ds.tasks {
		matchStatus := status == "" || task.Status == status

		matchUserID := true
		if userID != "" {
			if id, err := strconv.Atoi(userID); err == nil {
				matchUserID = task.UserID == id
			} else {
				matchUserID = false
			}
		}

		if matchStatus && matchUserID {
			filtered = append(filtered, task)
		}
	}
	return filtered
}

func (ds *DataStore) CreateTask(task Task) (Task, error) {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	if !ds.userExistsLocked(task.UserID) {
		return Task{}, ErrUserNotFound
	}

	maxID := 0
	for _, existingTask := range ds.tasks {
		if existingTask.ID > maxID {
			maxID = existingTask.ID
		}
	}

	task.ID = maxID + 1
	ds.tasks = append(ds.tasks, task)
	return task, nil
}

func (ds *DataStore) UpdateTask(id int, updates TaskUpdate) (Task, error) {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	taskIndex := -1
	for i := range ds.tasks {
		if ds.tasks[i].ID == id {
			taskIndex = i
			break
		}
	}
	if taskIndex == -1 {
		return Task{}, ErrTaskNotFound
	}

	if updates.UserID != nil && !ds.userExistsLocked(*updates.UserID) {
		return Task{}, ErrUserNotFound
	}

	if updates.Title != nil {
		ds.tasks[taskIndex].Title = *updates.Title
	}
	if updates.Status != nil {
		ds.tasks[taskIndex].Status = *updates.Status
	}
	if updates.UserID != nil {
		ds.tasks[taskIndex].UserID = *updates.UserID
	}

	return ds.tasks[taskIndex], nil
}

func (ds *DataStore) userExistsLocked(id int) bool {
	for _, user := range ds.users {
		if user.ID == id {
			return true
		}
	}
	return false
}

func (ds *DataStore) GetStats() StatsResponse {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	var stats StatsResponse
	stats.Users.Total = len(ds.users)
	stats.Tasks.Total = len(ds.tasks)

	for _, task := range ds.tasks {
		switch task.Status {
		case "pending":
			stats.Tasks.Pending++
		case "in-progress":
			stats.Tasks.InProgress++
		case "completed":
			stats.Tasks.Completed++
		}
	}

	return stats
}
