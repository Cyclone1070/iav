package todo

import (
	"context"

	"github.com/Cyclone1070/iav/internal/config"
)

// todoStore defines the interface for todo storage.
// This is a consumer-defined interface per architecture guidelines §2.
type todoStore interface {
	Read() ([]Todo, error)
	Write(todos []Todo) error
}

// ReadTodosTool handles reading todos.
type ReadTodosTool struct {
	store  todoStore
	config *config.Config
}

// NewReadTodosTool creates a new ReadTodosTool with injected dependencies.
func NewReadTodosTool(store todoStore, cfg *config.Config) *ReadTodosTool {
	return &ReadTodosTool{
		store:  store,
		config: cfg,
	}
}

// Run retrieves all todos from the store.
// Returns an empty list if no todos exist.
func (t *ReadTodosTool) Run(ctx context.Context, req *ReadTodosRequest) (*ReadTodosResponse, error) {
	if err := req.Validate(t.config); err != nil {
		return nil, err
	}
	if t.store == nil {
		return nil, ErrStoreNotConfigured
	}

	todos, err := t.store.Read()
	if err != nil {
		return nil, &StoreReadError{Cause: err}
	}

	return &ReadTodosResponse{
		Todos: todos,
	}, nil
}

// WriteTodosTool handles writing todos.
type WriteTodosTool struct {
	store  todoStore
	config *config.Config
}

// NewWriteTodosTool creates a new WriteTodosTool with injected dependencies.
func NewWriteTodosTool(store todoStore, cfg *config.Config) *WriteTodosTool {
	return &WriteTodosTool{
		store:  store,
		config: cfg,
	}
}

// Run replaces all todos in the store.
// This is an atomic operation that completely replaces the todo list.
func (t *WriteTodosTool) Run(ctx context.Context, req *WriteTodosRequest) (*WriteTodosResponse, error) {
	if err := req.Validate(t.config); err != nil {
		return nil, err
	}
	if t.store == nil {
		return nil, ErrStoreNotConfigured
	}

	todos := req.Todos
	if err := t.store.Write(todos); err != nil {
		return nil, &StoreWriteError{Cause: err}
	}

	return &WriteTodosResponse{
		Count: len(todos),
	}, nil
}
