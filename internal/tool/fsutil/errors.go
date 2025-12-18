package fsutil

import (
	"fmt"
	"os"
)

// InvalidOffsetError is returned when an offset is negative.
type InvalidOffsetError struct {
	Value int64
}

func (e *InvalidOffsetError) Error() string {
	return fmt.Sprintf("offset cannot be negative: %d", e.Value)
}

func (e *InvalidOffsetError) InvalidInput() bool {
	return true
}

// TempFileError is returned when creating a temp file fails.
type TempFileError struct {
	Dir   string
	Cause error
}

func (e *TempFileError) Error() string {
	return fmt.Sprintf("failed to create temp file in %s: %v", e.Dir, e.Cause)
}

func (e *TempFileError) Unwrap() error {
	return e.Cause
}

func (e *TempFileError) IOError() bool {
	return true
}

// TempWriteError is returned when writing to a temp file fails.
type TempWriteError struct {
	Path  string
	Cause error
}

func (e *TempWriteError) Error() string {
	return fmt.Sprintf("failed to write to temp file %s: %v", e.Path, e.Cause)
}

func (e *TempWriteError) Unwrap() error {
	return e.Cause
}

func (e *TempWriteError) IOError() bool {
	return true
}

// TempSyncError is returned when syncing a temp file fails.
type TempSyncError struct {
	Path  string
	Cause error
}

func (e *TempSyncError) Error() string {
	return fmt.Sprintf("failed to sync temp file %s: %v", e.Path, e.Cause)
}

func (e *TempSyncError) Unwrap() error {
	return e.Cause
}

func (e *TempSyncError) IOError() bool {
	return true
}

// TempCloseError is returned when closing a temp file fails.
type TempCloseError struct {
	Path  string
	Cause error
}

func (e *TempCloseError) Error() string {
	return fmt.Sprintf("failed to close temp file %s: %v", e.Path, e.Cause)
}

func (e *TempCloseError) Unwrap() error {
	return e.Cause
}

func (e *TempCloseError) IOError() bool {
	return true
}

// RenameError is returned when renaming a file fails.
type RenameError struct {
	Old   string
	New   string
	Cause error
}

func (e *RenameError) Error() string {
	return fmt.Sprintf("failed to rename %s to %s: %v", e.Old, e.New, e.Cause)
}

func (e *RenameError) Unwrap() error {
	return e.Cause
}

func (e *RenameError) IOError() bool {
	return true
}

// ChmodError is returned when changing file permissions fails.
type ChmodError struct {
	Path  string
	Mode  os.FileMode
	Cause error
}

func (e *ChmodError) Error() string {
	return fmt.Sprintf("failed to set permissions for %s to %v: %v", e.Path, e.Mode, e.Cause)
}

func (e *ChmodError) Unwrap() error {
	return e.Cause
}

func (e *ChmodError) IOError() bool {
	return true
}
