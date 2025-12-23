package fs

import (
	"errors"
	"fmt"
	"os"
)

// -- Errors --

type TempFileError struct {
	Dir   string
	Cause error
}

func (e *TempFileError) Error() string {
	return fmt.Sprintf("failed to create temp file in %s: %v", e.Dir, e.Cause)
}
func (e *TempFileError) Unwrap() error { return e.Cause }

type TempWriteError struct {
	Path  string
	Cause error
}

func (e *TempWriteError) Error() string {
	return fmt.Sprintf("failed to write to temp file %s: %v", e.Path, e.Cause)
}
func (e *TempWriteError) Unwrap() error { return e.Cause }

type TempSyncError struct {
	Path  string
	Cause error
}

func (e *TempSyncError) Error() string {
	return fmt.Sprintf("failed to sync temp file %s: %v", e.Path, e.Cause)
}
func (e *TempSyncError) Unwrap() error { return e.Cause }

type TempCloseError struct {
	Path  string
	Cause error
}

func (e *TempCloseError) Error() string {
	return fmt.Sprintf("failed to close temp file %s: %v", e.Path, e.Cause)
}
func (e *TempCloseError) Unwrap() error { return e.Cause }

type RenameError struct {
	Old   string
	New   string
	Cause error
}

func (e *RenameError) Error() string {
	return fmt.Sprintf("failed to rename %s to %s: %v", e.Old, e.New, e.Cause)
}
func (e *RenameError) Unwrap() error { return e.Cause }

type ChmodError struct {
	Path  string
	Mode  os.FileMode
	Cause error
}

func (e *ChmodError) Error() string {
	return fmt.Sprintf("failed to set permissions for %s to %v: %v", e.Path, e.Mode, e.Cause)
}
func (e *ChmodError) Unwrap() error { return e.Cause }

// -- Sentinels --

var (
	ErrInvalidOffset = errors.New("invalid offset")
)
