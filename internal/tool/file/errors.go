package file

import "fmt"

// FileMissingError indicates a file does not exist.
type FileMissingError struct {
	Path string
}

func (e *FileMissingError) Error() string {
	if e.Path != "" {
		return "file does not exist: " + e.Path
	}
	return "file does not exist"
}

func (e *FileMissingError) FileMissing() bool {
	return true
}

// FileExistsError indicates a file already exists.
type FileExistsError struct {
	Path string
}

func (e *FileExistsError) Error() string {
	return fmt.Sprintf("file already exists: %s", e.Path)
}

func (e *FileExistsError) FileExists() bool {
	return true
}

// BinaryFileError indicates a file is binary.
type BinaryFileError struct {
	Path string
}

func (e *BinaryFileError) Error() string {
	return fmt.Sprintf("binary file not supported: %s", e.Path)
}

func (e *BinaryFileError) BinaryFile() bool {
	return true
}

// EditConflictError indicates a file was modified concurrently.
type EditConflictError struct {
	Path string
}

func (e *EditConflictError) Error() string {
	return fmt.Sprintf("edit conflict: file %s was modified since last read", e.Path)
}

func (e *EditConflictError) EditConflict() bool {
	return true
}

// SnippetNotFoundError indicates the snippet to replace was not found.
type SnippetNotFoundError struct {
	Path    string
	Snippet string
}

func (e *SnippetNotFoundError) Error() string {
	return fmt.Sprintf("snippet not found in file %s", e.Path)
}

func (e *SnippetNotFoundError) SnippetNotFound() bool {
	return true
}

// ReplacementMismatchError indicates expected replacements count mismatch.
type ReplacementMismatchError struct {
	Path     string
	Expected int
	Actual   int
}

func (e *ReplacementMismatchError) Error() string {
	return fmt.Sprintf("expected %d replacements in %s, found %d", e.Expected, e.Path, e.Actual)
}

func (e *ReplacementMismatchError) ReplacementMismatch() bool {
	return true
}

// TooLargeError indicates file size exceeds limit.
type TooLargeError struct {
	Path  string
	Size  int64
	Limit int64
}

func (e *TooLargeError) Error() string {
	return fmt.Sprintf("file %s size %d exceeds limit %d", e.Path, e.Size, e.Limit)
}

func (e *TooLargeError) TooLarge() bool {
	return true
}

// PathRequiredError is returned when path is empty.
type PathRequiredError struct{}

func (e *PathRequiredError) Error() string { return "path is required" }

func (e *PathRequiredError) InvalidInput() bool { return true }

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

// ContentRequiredError is returned when content is empty.
type ContentRequiredError struct{}

func (e *ContentRequiredError) Error() string { return "content is required" }

func (e *ContentRequiredError) InvalidInput() bool { return true }

// InvalidPermissionError is returned when permission is invalid.
type InvalidPermissionError struct {
	Perm uint32
}

func (e *InvalidPermissionError) Error() string {
	return fmt.Sprintf("invalid permissions: %o", e.Perm)
}

func (e *InvalidPermissionError) InvalidInput() bool { return true }

// OperationsRequiredError is returned when operations list is empty.
type OperationsRequiredError struct{}

func (e *OperationsRequiredError) Error() string { return "operations cannot be empty" }

func (e *OperationsRequiredError) InvalidInput() bool { return true }

// BeforeRequiredError is returned when Before field is empty.
type BeforeRequiredError struct {
	Index int
}

func (e *BeforeRequiredError) Error() string {
	return fmt.Sprintf("operation %d: Before must be non-empty", e.Index)
}

func (e *BeforeRequiredError) InvalidInput() bool { return true }

// NegativeExpectedReplacementsError is returned when ExpectedReplacements is negative.
type NegativeExpectedReplacementsError struct {
	Index int
	Value int
}

func (e *NegativeExpectedReplacementsError) Error() string {
	return fmt.Sprintf("operation %d: ExpectedReplacements cannot be negative: %d", e.Index, e.Value)
}

func (e *NegativeExpectedReplacementsError) InvalidInput() bool { return true }

// StatError is returned when stat fails.
type StatError struct {
	Path  string
	Cause error
}

func (e *StatError) Error() string { return fmt.Sprintf("failed to stat file %s: %v", e.Path, e.Cause) }
func (e *StatError) Unwrap() error { return e.Cause }
func (e *StatError) IOError() bool { return true }

// IsDirectoryError is returned when a path is a directory but file is expected.
type IsDirectoryError struct {
	Path string
}

func (e *IsDirectoryError) Error() string     { return fmt.Sprintf("path is a directory: %s", e.Path) }
func (e *IsDirectoryError) IsDirectory() bool { return true }

// ReadError is returned when read fails.
type ReadError struct {
	Path  string
	Cause error
}

func (e *ReadError) Error() string { return fmt.Sprintf("failed to read file %s: %v", e.Path, e.Cause) }
func (e *ReadError) Unwrap() error { return e.Cause }
func (e *ReadError) IOError() bool { return true }

// EnsureDirsError is returned when mkdir fails.
type EnsureDirsError struct {
	Path  string
	Cause error
}

func (e *EnsureDirsError) Error() string {
	return fmt.Sprintf("failed to ensure directories for %s: %v", e.Path, e.Cause)
}
func (e *EnsureDirsError) Unwrap() error { return e.Cause }
func (e *EnsureDirsError) IOError() bool { return true }

// WriteError is returned when write fails.
type WriteError struct {
	Path  string
	Cause error
}

func (e *WriteError) Error() string {
	return fmt.Sprintf("failed to write file %s: %v", e.Path, e.Cause)
}
func (e *WriteError) Unwrap() error { return e.Cause }
func (e *WriteError) IOError() bool { return true }

// RevalidateError is returned when revalidation fails.
type RevalidateError struct {
	Path  string
	Cause error
}

func (e *RevalidateError) Error() string {
	return fmt.Sprintf("failed to revalidate file %s: %v", e.Path, e.Cause)
}
func (e *RevalidateError) Unwrap() error { return e.Cause }
func (e *RevalidateError) IOError() bool { return true }
