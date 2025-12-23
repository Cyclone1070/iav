package path

import (
	"errors"
	"fmt"
)

// -- Error Types --

// WorkspaceRootError is returned when the workspace root is invalid.
type WorkspaceRootError struct {
	Root  string
	Cause error
}

func (e *WorkspaceRootError) Error() string {
	return fmt.Sprintf("invalid workspace root %s: %v", e.Root, e.Cause)
}
func (e *WorkspaceRootError) Unwrap() error { return e.Cause }

// -- Sentinels --

var (
	ErrOutsideWorkspace    = errors.New("path is outside workspace root")
	ErrWorkspaceRootNotSet = errors.New("workspace root not set")
	ErrNotADirectory       = errors.New("not a directory")
)
