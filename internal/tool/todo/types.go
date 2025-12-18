package todo

import (
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

// ReadTodosDTO is the wire format for ReadTodos operation
type ReadTodosDTO struct{}

// ReadTodosRequest is the validated domain entity for ReadTodos operation
type ReadTodosRequest struct{}

// NewReadTodosRequest creates a validated ReadTodosRequest from a DTO
func NewReadTodosRequest(dto ReadTodosDTO, cfg *config.Config) (*ReadTodosRequest, error) {
	// No validation needed
	return &ReadTodosRequest{}, nil
}

// WriteTodosDTO is the wire format for WriteTodos operation
type WriteTodosDTO struct {
	Todos []Todo `json:"todos"`
}

// WriteTodosRequest is the validated domain entity for WriteTodos operation
type WriteTodosRequest struct {
	todos []Todo
}

// NewWriteTodosRequest creates a validated WriteTodosRequest from a DTO
func NewWriteTodosRequest(dto WriteTodosDTO, cfg *config.Config) (*WriteTodosRequest, error) {
	// Constructor validation
	// Check for empty todos array (questionable but allow it - might be clearing all todos)
	// But validate each todo if present
	for i, todo := range dto.Todos {
		// Validate status
		switch todo.Status {
		case TodoStatusPending, TodoStatusInProgress, TodoStatusCompleted, TodoStatusCancelled:
			// Valid
		default:
			return nil, &InvalidStatusError{Index: i, Status: todo.Status}
		}

		// Validate description is not empty
		if todo.Description == "" {
			return nil, &EmptyDescriptionError{Index: i}
		}
	}

	return &WriteTodosRequest{
		todos: dto.Todos,
	}, nil
}

// Todos returns the list of todos
func (r *WriteTodosRequest) Todos() []Todo {
	return r.todos
}

// ReadTodosResponse contains the list of current todos.
type ReadTodosResponse struct {
	Todos []Todo `json:"todos"`
}

// WriteTodosResponse contains the result of a WriteTodos operation.
type WriteTodosResponse struct {
	Count int `json:"count"`
}
