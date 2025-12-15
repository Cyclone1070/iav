package errutil

import "errors"

// Sentinel errors for consistent error handling across tools
var (
	// File operation errors
	ErrOutsideWorkspace             = errors.New("path is outside workspace root")
	ErrFileExists                   = errors.New("file already exists, use EditFile instead")
	ErrBinaryFile                   = errors.New("binary files are not supported")
	ErrEditConflict                 = errors.New("file was modified since last read, please re-read first")
	ErrSnippetNotFound              = errors.New("snippet not found in file")
	ErrExpectedReplacementsMismatch = errors.New("expected replacements count does not match actual occurrences")
	ErrTooLarge                     = errors.New("file or content exceeds size limit")
	ErrFileMissing                  = errors.New("file does not exist")
	ErrInvalidOffset                = errors.New("offset must be >= 0")
	ErrInvalidLimit                 = errors.New("limit must be >= 0")
	ErrInvalidPaginationOffset      = errors.New("offset must be >= 0")
	ErrInvalidPaginationLimit       = errors.New("limit must be between 1 and the configured maximum")

	// Shell operation errors
	ErrShellTimeout                    = errors.New("shell command timed out")
	ErrShellRejected                   = errors.New("shell command rejected by policy")
	ErrShellWorkingDirOutsideWorkspace = errors.New("working directory is outside workspace")
	ErrShellApprovalRequired           = errors.New("shell command requires approval")
)
