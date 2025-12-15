package todo

import (
	"fmt"

	"github.com/Cyclone1070/iav/internal/config"
)

// TodoStatus represents the status of a todo item.
type TodoStatus string

const (
	TodoStatusPending    TodoStatus = "pending"
	TodoStatusInProgress TodoStatus = "in_progress"
	TodoStatusCompleted  TodoStatus = "completed"
	TodoStatusCancelled  TodoStatus = "cancelled"
)

// Todo represents a single task item.
type Todo struct {
	Description string     `json:"description"`
	Status      TodoStatus `json:"status"`
}

// ReadTodosRequest contains parameters for ReadTodos operation
type ReadTodosRequest struct{}

// Validate validates the ReadTodosRequest
func (r ReadTodosRequest) Validate(cfg *config.Config) error {
	return nil // No fields to validate
}

// WriteTodosRequest contains parameters for WriteTodos operation
type WriteTodosRequest struct {
	Todos []Todo `json:"todos"`
}

// Validate validates the WriteTodosRequest
func (r WriteTodosRequest) Validate(cfg *config.Config) error {
	// Check for empty todos array (questionable but allow it - might be clearing all todos)
	// But validate each todo if present
	for i, todo := range r.Todos {
		// Validate status
		switch todo.Status {
		case TodoStatusPending, TodoStatusInProgress, TodoStatusCompleted, TodoStatusCancelled:
			// Valid
		default:
			return fmt.Errorf("todo[%d]: invalid status %q", i, todo.Status)
		}

		// Validate description is not empty
		if todo.Description == "" {
			return fmt.Errorf("todo[%d]: description cannot be empty", i)
		}
	}
	return nil
}

// ReadTodosResponse contains the list of current todos.
type ReadTodosResponse struct {
	Todos []Todo `json:"todos"`
}

// WriteTodosResponse contains the result of a WriteTodos operation.
type WriteTodosResponse struct {
	Count int `json:"count"`
}
