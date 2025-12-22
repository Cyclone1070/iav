package todo

import (
	"fmt"

	"github.com/Cyclone1070/iav/internal/config"
)

// -- Contract Types --

// TodoStatus represents the status of a todo item.
type TodoStatus string

const (
	TodoStatusPending    TodoStatus = "pending"
	TodoStatusInProgress TodoStatus = "in_progress"
	TodoStatusCompleted  TodoStatus = "completed"
	TodoStatusCancelled  TodoStatus = "cancelled"
)

// Todo represents a single todo item.
type Todo struct {
	Description string     `json:"description"`
	Status      TodoStatus `json:"status"`
}

// ReadTodosRequest represents a request to read all todos.
type ReadTodosRequest struct{}

func (r *ReadTodosRequest) Validate(cfg *config.Config) error {
	return nil
}

// ReadTodosResponse contains the list of current todos.
type ReadTodosResponse struct {
	Todos []Todo `json:"todos"`
}

// WriteTodosRequest represents a request to update the list of todos.
type WriteTodosRequest struct {
	Todos []Todo `json:"todos"`
}

func (r *WriteTodosRequest) Validate(cfg *config.Config) error {
	for i, t := range r.Todos {
		// Validate status
		switch t.Status {
		case TodoStatusPending, TodoStatusInProgress, TodoStatusCompleted, TodoStatusCancelled:
			// Valid
		default:
			return fmt.Errorf("todo[%d]: %w %q", i, ErrInvalidStatus, t.Status)
		}

		// Validate description is not empty
		if t.Description == "" {
			return fmt.Errorf("todo[%d]: %w", i, ErrEmptyDescription)
		}
	}
	return nil
}

// WriteTodosResponse contains the result of a WriteTodos operation.
type WriteTodosResponse struct {
	Count int `json:"count"`
}
