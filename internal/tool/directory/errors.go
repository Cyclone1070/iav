package directory

import "fmt"

// FileMissingError implements the behavioral interface for missing files.
type FileMissingError struct {
	Path string
}

func (e *FileMissingError) Error() string {
	return "path does not exist: " + e.Path
}

func (e *FileMissingError) FileMissing() bool {
	return true
}

// NotDirectoryError implements the behavioral interface for non-directory paths.
type NotDirectoryError struct {
	Path string
}

func (e *NotDirectoryError) Error() string {
	return "path is not a directory: " + e.Path
}

func (e *NotDirectoryError) NotDirectory() bool {
	return true
}

// NegativeOffsetError is returned when offset is negative.
type NegativeOffsetError struct {
	Value int64
}

func (e *NegativeOffsetError) Error() string {
	return fmt.Sprintf("offset cannot be negative: %d", e.Value)
}

func (e *NegativeOffsetError) InvalidInput() bool { return true }

// NegativeLimitError is returned when limit is negative.
type NegativeLimitError struct {
	Value int64
}

func (e *NegativeLimitError) Error() string {
	return fmt.Sprintf("limit cannot be negative: %d", e.Value)
}

func (e *NegativeLimitError) InvalidInput() bool { return true }

// LimitExceededError is returned when limit exceeds maximum.
type LimitExceededError struct {
	Value int64
	Max   int64
}

func (e *LimitExceededError) Error() string {
	return fmt.Sprintf("limit %d exceeds maximum %d", e.Value, e.Max)
}

func (e *LimitExceededError) InvalidInput() bool { return true }

// PatternRequiredError is returned when pattern is empty.
type PatternRequiredError struct{}

func (e *PatternRequiredError) Error() string { return "pattern is required" }

func (e *PatternRequiredError) InvalidInput() bool { return true }

// PathTraversalError is returned when path traversal is detected.
type PathTraversalError struct {
	Path string
}

func (e *PathTraversalError) Error() string {
	return fmt.Sprintf("path traversal detected: %s", e.Path)
}

func (e *PathTraversalError) InvalidInput() bool { return true }

func (e *PathTraversalError) OutsideWorkspace() bool { return true }

// StatError is returned when stat fails.
type StatError struct {
	Path  string
	Cause error
}

func (e *StatError) Error() string { return fmt.Sprintf("failed to stat path %s: %v", e.Path, e.Cause) }
func (e *StatError) Unwrap() error { return e.Cause }
func (e *StatError) IOError() bool { return true }

// ListDirError is returned when Readdir fails.
type ListDirError struct {
	Path  string
	Cause error
}

func (e *ListDirError) Error() string {
	return fmt.Sprintf("failed to list directory %s: %v", e.Path, e.Cause)
}
func (e *ListDirError) Unwrap() error { return e.Cause }
func (e *ListDirError) IOError() bool { return true }

// FindFileError is returned when find operation fails.
type FindFileError struct {
	Path  string
	Cause error
}

func (e *FindFileError) Error() string {
	return fmt.Sprintf("failed to find files in %s: %v", e.Path, e.Cause)
}
func (e *FindFileError) Unwrap() error { return e.Cause }
func (e *FindFileError) IOError() bool { return true }

// RelPathError is returned when Rel fails.
type RelPathError struct {
	Path  string
	Cause error
}

func (e *RelPathError) Error() string {
	return fmt.Sprintf("failed to calculate relative path for %s: %v", e.Path, e.Cause)
}
func (e *RelPathError) Unwrap() error { return e.Cause }
func (e *RelPathError) IOError() bool { return true }

// InvalidPatternError is returned when glob pattern is invalid.
type InvalidPatternError struct {
	Pattern string
	Cause   error
}

func (e *InvalidPatternError) Error() string {
	return fmt.Sprintf("invalid glob pattern %s: %v", e.Pattern, e.Cause)
}
func (e *InvalidPatternError) Unwrap() error      { return e.Cause }
func (e *InvalidPatternError) InvalidInput() bool { return true }

// CommandStartError is returned when command fails to start.
type CommandStartError struct {
	Cmd   string
	Cause error
}

func (e *CommandStartError) Error() string {
	return fmt.Sprintf("failed to start command %s: %v", e.Cmd, e.Cause)
}
func (e *CommandStartError) Unwrap() error { return e.Cause }
func (e *CommandStartError) IOError() bool { return true }

// CommandOutputError is returned when command output cannot be read.
type CommandOutputError struct {
	Cmd   string
	Cause error
}

func (e *CommandOutputError) Error() string {
	return fmt.Sprintf("failed to read command output for %s: %v", e.Cmd, e.Cause)
}
func (e *CommandOutputError) Unwrap() error { return e.Cause }
func (e *CommandOutputError) IOError() bool { return true }

// CommandFailedError is returned when command execution fails.
type CommandFailedError struct {
	Cmd   string
	Cause error
}

func (e *CommandFailedError) Error() string {
	return fmt.Sprintf("command %s failed: %v", e.Cmd, e.Cause)
}
func (e *CommandFailedError) Unwrap() error       { return e.Cause }
func (e *CommandFailedError) CommandFailed() bool { return true }
