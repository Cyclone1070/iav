package tools

import (
	"sync"

	"github.com/Cyclone1070/deployforme/internal/tools/models"
)

var (
	sessionTodos []models.Todo
	todoMutex    sync.RWMutex
)

// ReadTodos returns the current list of todos.
func ReadTodos(ctx *models.WorkspaceContext) (*models.ReadTodosResponse, error) {
	todoMutex.RLock()
	defer todoMutex.RUnlock()

	// Return a copy to ensure isolation
	todos := make([]models.Todo, len(sessionTodos))
	copy(todos, sessionTodos)

	return &models.ReadTodosResponse{
		Todos: todos,
	}, nil
}

// WriteTodos replaces the current list of todos with the provided list.
func WriteTodos(ctx *models.WorkspaceContext, todos []models.Todo) (*models.WriteTodosResponse, error) {
	todoMutex.Lock()
	defer todoMutex.Unlock()

	// Store a copy to ensure isolation
	sessionTodos = make([]models.Todo, len(todos))
	copy(sessionTodos, todos)

	return &models.WriteTodosResponse{
		Count: len(sessionTodos),
	}, nil
}
