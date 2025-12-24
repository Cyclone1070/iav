package file

import (
	"errors"
	"fmt"
)

// -- Errors --

type StatError struct {
	Path  string
	Cause error
}

func (e *StatError) Error() string { return fmt.Sprintf("failed to stat %s: %v", e.Path, e.Cause) }
func (e *StatError) Unwrap() error { return e.Cause }

type ReadError struct {
	Path  string
	Cause error
}

func (e *ReadError) Error() string { return fmt.Sprintf("failed to read file %s: %v", e.Path, e.Cause) }
func (e *ReadError) Unwrap() error { return e.Cause }

type WriteError struct {
	Path  string
	Cause error
}

func (e *WriteError) Error() string {
	return fmt.Sprintf("failed to write file %s: %v", e.Path, e.Cause)
}
func (e *WriteError) Unwrap() error { return e.Cause }

type RevalidateError struct {
	Path  string
	Cause error
}

func (e *RevalidateError) Error() string {
	return fmt.Sprintf("failed to revalidate file %s: %v", e.Path, e.Cause)
}
func (e *RevalidateError) Unwrap() error { return e.Cause }

type EnsureDirsError struct {
	Path  string
	Cause error
}

func (e *EnsureDirsError) Error() string {
	return fmt.Sprintf("failed to ensure directories for %s: %v", e.Path, e.Cause)
}
func (e *EnsureDirsError) Unwrap() error { return e.Cause }

// -- Sentinels --

var (
	ErrFileMissing              = errors.New("file or path does not exist")
	ErrFileExists               = errors.New("file already exists")
	ErrBinaryFile               = errors.New("file is binary")
	ErrFileTooLarge             = errors.New("file too large")
	ErrContentRequired          = errors.New("content is required")
	ErrOperationsRequired       = errors.New("operations cannot be empty")
	ErrSnippetNotFound          = errors.New("snippet not found")
	ErrReplacementCountMismatch = errors.New("replacement count mismatch")
	ErrEditConflict             = errors.New("edit conflict")
	ErrIsDirectory              = errors.New("path is a directory")
	ErrPathRequired             = errors.New("path is required")
	ErrInvalidOffset            = errors.New("invalid offset")
	ErrInvalidLimit             = errors.New("invalid limit")
	ErrInvalidPermissions       = errors.New("invalid permissions: must be between 0000 and 0777")
	ErrContentRequiredForWrite  = errors.New("content is required for write operation")
)
