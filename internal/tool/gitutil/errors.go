package gitutil

import "fmt"

// GitignoreReadError is returned when .gitignore cannot be read.
type GitignoreReadError struct {
	Path  string
	Cause error
}

func (e *GitignoreReadError) Error() string {
	return fmt.Sprintf("failed to read .gitignore at %s: %v", e.Path, e.Cause)
}

func (e *GitignoreReadError) Unwrap() error {
	return e.Cause
}

func (e *GitignoreReadError) IOError() bool {
	return true
}
