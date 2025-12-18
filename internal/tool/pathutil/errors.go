package pathutil

import "fmt"

// OutsideWorkspaceError indicates a path is outside the workspace boundary.
type OutsideWorkspaceError struct{}

func (e *OutsideWorkspaceError) Error() string {
	return "path is outside workspace root"
}

// OutsideWorkspace implements the behavioral interface for cross-package error checking.
func (e *OutsideWorkspaceError) OutsideWorkspace() bool {
	return true
}

// ErrOutsideWorkspace is returned when a path escapes the workspace boundary.
var ErrOutsideWorkspace = &OutsideWorkspaceError{}

// WorkspaceRootError is returned when the workspace root is invalid.
type WorkspaceRootError struct {
	Root  string
	Cause error
}

func (e *WorkspaceRootError) Error() string {
	return fmt.Sprintf("invalid workspace root %s: %v", e.Root, e.Cause)
}

func (e *WorkspaceRootError) Unwrap() error {
	return e.Cause
}

func (e *WorkspaceRootError) InvalidWorkspace() bool {
	return true
}

// WorkspaceRootNotSetError is returned when the workspace root is empty.
type WorkspaceRootNotSetError struct{}

func (e *WorkspaceRootNotSetError) Error() string {
	return "workspace root not set"
}

func (e *WorkspaceRootNotSetError) InvalidWorkspace() bool {
	return true
}

// TildeExpansionError is returned when tilde expansion fails.
type TildeExpansionError struct {
	Cause error
}

func (e *TildeExpansionError) Error() string {
	return fmt.Sprintf("failed to expand tilde: %v", e.Cause)
}

func (e *TildeExpansionError) Unwrap() error {
	return e.Cause
}

func (e *TildeExpansionError) PathResolutionFailed() bool {
	return true
}

// SymlinkLoopError is returned when a symlink loop is detected.
type SymlinkLoopError struct {
	Path string
}

func (e *SymlinkLoopError) Error() string {
	return fmt.Sprintf("symlink loop detected: %s", e.Path)
}

func (e *SymlinkLoopError) PathResolutionFailed() bool {
	return true
}

// LstatError is returned when lstat fails.
type LstatError struct {
	Path  string
	Cause error
}

func (e *LstatError) Error() string {
	return fmt.Sprintf("failed to lstat path %s: %v", e.Path, e.Cause)
}

func (e *LstatError) Unwrap() error {
	return e.Cause
}

func (e *LstatError) IOError() bool {
	return true
}

// ReadlinkError is returned when readlink fails.
type ReadlinkError struct {
	Path  string
	Cause error
}

func (e *ReadlinkError) Error() string {
	return fmt.Sprintf("failed to read symlink %s: %v", e.Path, e.Cause)
}

func (e *ReadlinkError) Unwrap() error {
	return e.Cause
}

func (e *ReadlinkError) IOError() bool {
	return true
}

// SymlinkChainTooLongError is returned when valid symlink chain is exceeded.
type SymlinkChainTooLongError struct {
	MaxHops int
}

func (e *SymlinkChainTooLongError) Error() string {
	return fmt.Sprintf("symlink chain too long (max %d hops)", e.MaxHops)
}

func (e *SymlinkChainTooLongError) PathResolutionFailed() bool {
	return true
}

// NotADirectoryError is returned when a path is expected to be a directory but isn't.
type NotADirectoryError struct {
	Path string
}

func (e *NotADirectoryError) Error() string {
	return fmt.Sprintf("not a directory: %s", e.Path)
}

func (e *NotADirectoryError) InvalidWorkspace() bool {
	return true
}
