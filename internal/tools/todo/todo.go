package todo

import (
	"context"
	"fmt"
)

// todoStore defines the interface for todo storage.
// This is a consumer-defined interface per architecture guidelines §2.
type todoStore interface {
	Read() ([]Todo, error)
	Write(todos []Todo) error
}

// ReadTodosTool handles reading todos.
type ReadTodosTool struct {
	store todoStore
}

// NewReadTodosTool creates a new ReadTodosTool with injected dependencies.
func NewReadTodosTool(store todoStore) *ReadTodosTool {
	return &ReadTodosTool{
		store: store,
	}
}

// Run retrieves all todos from the store.
// Returns an empty list if no todos exist.
func (t *ReadTodosTool) Run(ctx context.Context, req ReadTodosRequest) (*ReadTodosResponse, error) {
	if t.store == nil {
		return nil, fmt.Errorf("todo store not configured")
	}

	todos, err := t.store.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read todos: %w", err)
	}

	return &ReadTodosResponse{
		Todos: todos,
	}, nil
}

// WriteTodosTool handles writing todos.
type WriteTodosTool struct {
	store todoStore
}

// NewWriteTodosTool creates a new WriteTodosTool with injected dependencies.
func NewWriteTodosTool(store todoStore) *WriteTodosTool {
	return &WriteTodosTool{
		store: store,
	}
}

// Run replaces all todos in the store.
// This is an atomic operation that completely replaces the todo list.
func (t *WriteTodosTool) Run(ctx context.Context, req WriteTodosRequest) (*WriteTodosResponse, error) {
	if t.store == nil {
		return nil, fmt.Errorf("todo store not configured")
	}

	if err := t.store.Write(req.Todos); err != nil {
		return nil, fmt.Errorf("failed to write todos: %w", err)
	}

	return &WriteTodosResponse{
		Count: len(req.Todos),
	}, nil
}
