package tools

import (
	"errors"
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

// ReadTodos returns the current list of todos from the context's store.
func ReadTodos(ctx *models.WorkspaceContext) (*models.ReadTodosResponse, error) {
	if ctx.TodoStore == nil {
		return nil, errors.New("todo store not initialized in context")
	}

	todos, err := ctx.TodoStore.Read()
	if err != nil {
		return nil, err
	}

	return &models.ReadTodosResponse{
		Todos: todos,
	}, nil
}

// WriteTodos replaces the current list of todos in the context's store.
func WriteTodos(ctx *models.WorkspaceContext, todos []models.Todo) (*models.WriteTodosResponse, error) {
	if ctx.TodoStore == nil {
		return nil, errors.New("todo store not initialized in context")
	}

	if err := ctx.TodoStore.Write(todos); err != nil {
		return nil, err
	}

	return &models.WriteTodosResponse{
		Count: len(todos),
	}, nil
}
