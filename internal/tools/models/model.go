package models

import (
	"errors"
)

const (
	// DefaultMaxFileSize is the default maximum file size (5 MB)
	DefaultMaxFileSize = 5 * 1024 * 1024
	// BinaryDetectionSampleSize is how many bytes to sample for binary detection
	BinaryDetectionSampleSize = 4096
	// DefaultListDirectoryLimit is the default limit for directory listing pagination
	DefaultListDirectoryLimit = 1000
	// MaxListDirectoryLimit is the maximum allowed limit for directory listing pagination
	MaxListDirectoryLimit = 10000
	// DefaultMaxCommandOutputSize is the default maximum size for command output (10 MB)
	DefaultMaxCommandOutputSize = 10 * 1024 * 1024
)

// Operation represents a single edit operation for EditFile.
// Before must be a non-empty literal snippet that exists in the file.
// ExpectedReplacements must match the exact number of occurrences of Before in the file.
type Operation struct {
	Before               string // required, non-empty literal snippet
	After                string // required
	ExpectedReplacements int    // required, >=1
}

// ReadFileResponse contains the result of a ReadFile operation
type ReadFileResponse struct {
	AbsolutePath string
	RelativePath string
	Size         int64
	Content      string
}

// WriteFileResponse contains the result of a WriteFile operation
type WriteFileResponse struct {
	AbsolutePath string
	RelativePath string
	BytesWritten int
	FileMode     uint32
}

// EditFileResponse contains the result of an EditFile operation
type EditFileResponse struct {
	AbsolutePath      string
	RelativePath      string
	OperationsApplied int
	FileSize          int64
}

// DirectoryEntry represents a single entry in a directory listing
type DirectoryEntry struct {
	RelativePath string
	IsDir        bool
	Size         int64
}

// ListDirectoryResponse contains the result of a ListDirectory operation
type ListDirectoryResponse struct {
	DirectoryPath string
	Entries       []DirectoryEntry
	Offset        int
	Limit         int
	TotalCount    int  // Total entries before pagination
	Truncated     bool // True if more entries exist beyond offset+limit
}

// Sentinel errors for consistent error handling
var (
	ErrOutsideWorkspace        = errors.New("path is outside workspace root")
	ErrFileExists              = errors.New("file already exists, use EditFile instead")
	ErrBinaryFile              = errors.New("binary files are not supported")
	ErrEditConflict            = errors.New("file was modified since last read, please re-read first")
	ErrSnippetNotFound         = errors.New("snippet not found in file")
	ErrSnippetAmbiguous        = errors.New("snippet occurrence count does not match expected")
	ErrTooLarge                = errors.New("file or content exceeds size limit")
	ErrFileMissing             = errors.New("file does not exist")
	ErrInvalidOffset           = errors.New("offset must be >= 0")
	ErrInvalidLimit            = errors.New("limit must be >= 0")
	ErrInvalidPaginationOffset = errors.New("offset must be >= 0")
	ErrInvalidPaginationLimit  = errors.New("limit must be between 1 and MaxListDirectoryLimit")

	// Shell Sentinel Errors
	ErrShellTimeout                    = errors.New("shell command timed out")
	ErrShellRejected                   = errors.New("shell command rejected by policy")
	ErrShellWorkingDirOutsideWorkspace = errors.New("working directory is outside workspace")
	ErrShellApprovalRequired           = errors.New("shell command requires approval")
)

// ShellRequest represents a request to execute a command on the local machine.
type ShellRequest struct {
	Command        []string
	WorkingDir     string
	TimeoutSeconds int
	Env            map[string]string
}

// ShellResponse represents the result of a local command execution.
type ShellResponse struct {
	Stdout         string
	Stderr         string
	ExitCode       int
	Truncated      bool
	DurationMs     int64
	WorkingDir     string
	Notes          []string
	BackgroundPIDs []int
}

// CommandPolicy defines allowed and denied commands.
type CommandPolicy struct {
	Allow        []string
	Ask          []string
	SessionAllow map[string]bool // command roots approved for this session
}

// DockerConfig contains configuration for Docker readiness checks.
type DockerConfig struct {
	CheckCommand []string // e.g., ["docker", "info"]
	StartCommand []string // e.g., ["docker", "desktop", "start"]
}
