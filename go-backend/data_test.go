package main

import (
	"errors"
	"testing"
)

func TestDataStoreGetUsersReturnsCopy(t *testing.T) {
	dataStore := testDataStore()

	users := dataStore.GetUsers()
	users[0].Name = "Changed"

	storedUser := dataStore.GetUserByID(1)
	if storedUser == nil {
		t.Fatal("expected user")
	}
	if storedUser.Name != "John Doe" {
		t.Fatalf("expected stored user to remain unchanged, got %q", storedUser.Name)
	}
}

func TestDataStoreCreateUser(t *testing.T) {
	dataStore := testDataStore()

	user := dataStore.CreateUser(User{Name: "Grace Hopper", Email: "grace@example.com", Role: "developer"})

	if user.ID != 4 {
		t.Fatalf("expected ID 4, got %d", user.ID)
	}
	if len(dataStore.GetUsers()) != 4 {
		t.Fatalf("expected 4 users after create")
	}
}

func TestDataStoreGetTasksFilters(t *testing.T) {
	dataStore := testDataStore()

	statusTasks := dataStore.GetTasks("pending", "")
	if len(statusTasks) != 1 || statusTasks[0].ID != 1 {
		t.Fatalf("unexpected status filter result: %+v", statusTasks)
	}

	userTasks := dataStore.GetTasks("", "2")
	if len(userTasks) != 1 || userTasks[0].UserID != 2 {
		t.Fatalf("unexpected user filter result: %+v", userTasks)
	}

	invalidUserTasks := dataStore.GetTasks("", "invalid")
	if len(invalidUserTasks) != 0 {
		t.Fatalf("expected no tasks for invalid user filter, got %+v", invalidUserTasks)
	}
}

func TestDataStoreCreateTask(t *testing.T) {
	dataStore := testDataStore()

	task, err := dataStore.CreateTask(Task{Title: "Write tests", Status: "pending", UserID: 1})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	if task.ID != 4 {
		t.Fatalf("expected ID 4, got %d", task.ID)
	}

	_, err = dataStore.CreateTask(Task{Title: "Missing user", Status: "pending", UserID: 999})
	if !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}

func TestDataStoreUpdateTask(t *testing.T) {
	dataStore := testDataStore()
	title := "Updated title"
	status := "completed"
	userID := 2

	task, err := dataStore.UpdateTask(1, TaskUpdate{
		Title:  &title,
		Status: &status,
		UserID: &userID,
	})
	if err != nil {
		t.Fatalf("update task: %v", err)
	}
	if task.Title != title || task.Status != status || task.UserID != userID {
		t.Fatalf("unexpected updated task: %+v", task)
	}

	_, err = dataStore.UpdateTask(999, TaskUpdate{Title: &title})
	if !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf("expected ErrTaskNotFound, got %v", err)
	}

	missingUserID := 999
	_, err = dataStore.UpdateTask(1, TaskUpdate{UserID: &missingUserID})
	if !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}

func TestDataStoreGetStats(t *testing.T) {
	stats := testDataStore().GetStats()

	if stats.Users.Total != 3 {
		t.Fatalf("expected 3 users, got %d", stats.Users.Total)
	}
	if stats.Tasks.Total != 3 || stats.Tasks.Pending != 1 || stats.Tasks.InProgress != 1 || stats.Tasks.Completed != 1 {
		t.Fatalf("unexpected task stats: %+v", stats.Tasks)
	}
}
