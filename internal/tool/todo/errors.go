package todo

import "fmt"

// InvalidStatusError is returned when a todo has an invalid status.
type InvalidStatusError struct {
	Index  int
	Status TodoStatus
}

func (e *InvalidStatusError) Error() string {
	return fmt.Sprintf("todo[%d]: invalid status %q", e.Index, e.Status)
}

func (e *InvalidStatusError) InvalidInput() bool {
	return true
}

// EmptyDescriptionError is returned when a todo has an empty description.
type EmptyDescriptionError struct {
	Index int
}

func (e *EmptyDescriptionError) Error() string {
	return fmt.Sprintf("todo[%d]: description cannot be empty", e.Index)
}

func (e *EmptyDescriptionError) InvalidInput() bool {
	return true
}

// StoreNotConfiguredError is returned when the todo store is nil.
type StoreNotConfiguredError struct{}

func (e *StoreNotConfiguredError) Error() string {
	return "todo store not configured"
}

func (e *StoreNotConfiguredError) StoreNotConfigured() bool {
	return true
}

// StoreReadError is returned when reading from the todo store fails.
type StoreReadError struct {
	Cause error
}

func (e *StoreReadError) Error() string {
	return fmt.Sprintf("failed to read todos: %v", e.Cause)
}

func (e *StoreReadError) Unwrap() error {
	return e.Cause
}

func (e *StoreReadError) IOError() bool {
	return true
}

// StoreWriteError is returned when writing to the todo store fails.
type StoreWriteError struct {
	Cause error
}

func (e *StoreWriteError) Error() string {
	return fmt.Sprintf("failed to write todos: %v", e.Cause)
}

func (e *StoreWriteError) Unwrap() error {
	return e.Cause
}

func (e *StoreWriteError) IOError() bool {
	return true
}
