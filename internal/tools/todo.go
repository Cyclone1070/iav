package tools

import (
	"fmt"
	"sync"

	"github.com/Cyclone1070/deployforme/internal/tools/models"
)

// InMemoryTodoStore implements models.TodoStore using an in-memory slice.
type InMemoryTodoStore struct {
	todos []models.Todo
	mu    sync.RWMutex
}

// NewInMemoryTodoStore creates a new instance of InMemoryTodoStore.
func NewInMemoryTodoStore() *InMemoryTodoStore {
	return &InMemoryTodoStore{
		todos: make([]models.Todo, 0),
	}
}

// Read returns the current list of todos.
func (s *InMemoryTodoStore) Read() ([]models.Todo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Return a copy to ensure isolation
	result := make([]models.Todo, len(s.todos))
	copy(result, s.todos)
	return result, nil
}

// Write replaces the current list of todos with the provided list.
func (s *InMemoryTodoStore) Write(todos []models.Todo) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Store a copy to ensure isolation
	s.todos = make([]models.Todo, len(todos))
	copy(s.todos, todos)
	return nil
}

// ReadTodos retrieves all todos from the in-memory store.
// Returns an empty list if no todos exist.
func ReadTodos(ctx *models.WorkspaceContext, req models.ReadTodosRequest) (*models.ReadTodosResponse, error) {
	if ctx.TodoStore == nil {
		return nil, fmt.Errorf("todo store not configured")
	}

	todos, err := ctx.TodoStore.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read todos: %w", err)
	}

	return &models.ReadTodosResponse{
		Todos: todos,
	}, nil
}

// WriteTodos replaces all todos in the in-memory store.
// This is an atomic operation that completely replaces the todo list.
func WriteTodos(ctx *models.WorkspaceContext, req models.WriteTodosRequest) (*models.WriteTodosResponse, error) {
	if ctx.TodoStore == nil {
		return nil, fmt.Errorf("todo store not configured")
	}

	if err := ctx.TodoStore.Write(req.Todos); err != nil {
		return nil, fmt.Errorf("failed to write todos: %w", err)
	}

	return &models.WriteTodosResponse{
		Count: len(req.Todos),
	}, nil
}
