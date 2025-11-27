package models

import (
	"errors"
	"fmt"
	"os"
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

// Request structs for tool operations

// ReadFileRequest contains parameters for ReadFile operation
type ReadFileRequest struct {
	Path   string `json:"path"`
	Offset *int64 `json:"offset,omitempty"`
	Limit  *int64 `json:"limit,omitempty"`
}

// WriteFileRequest contains parameters for WriteFile operation
type WriteFileRequest struct {
	Path    string       `json:"path"`
	Content string       `json:"content"`
	Perm    *os.FileMode `json:"perm,omitempty"`
}

// EditFileRequest contains parameters for EditFile operation
type EditFileRequest struct {
	Path       string      `json:"path"`
	Operations []Operation `json:"operations"`
}

// ListDirectoryRequest contains parameters for ListDirectory operation
type ListDirectoryRequest struct {
	Path           string `json:"path"`
	MaxDepth       int    `json:"max_depth,omitempty"`
	IncludeIgnored bool   `json:"include_ignored,omitempty"`
	Offset         int    `json:"offset,omitempty"`
	Limit          int    `json:"limit,omitempty"`
}

// FindFileRequest contains parameters for FindFile operation
type FindFileRequest struct {
	Pattern        string `json:"pattern"`
	SearchPath     string `json:"search_path"`
	MaxDepth       int    `json:"max_depth,omitempty"`
	IncludeIgnored bool   `json:"include_ignored,omitempty"`
	Offset         int    `json:"offset,omitempty"`
	Limit          int    `json:"limit,omitempty"`
}

// SearchContentRequest contains parameters for SearchContent operation
type SearchContentRequest struct {
	Query          string `json:"query"`
	SearchPath     string `json:"search_path"`
	CaseSensitive  bool   `json:"case_sensitive,omitempty"`
	IncludeIgnored bool   `json:"include_ignored,omitempty"`
	Offset         int    `json:"offset,omitempty"`
	Limit          int    `json:"limit,omitempty"`
}

// ReadTodosRequest contains parameters for ReadTodos operation
type ReadTodosRequest struct{}

// WriteTodosRequest contains parameters for WriteTodos operation
type WriteTodosRequest struct {
	Todos []Todo `json:"todos"`
}

// Validate methods for request structs

func (r ReadFileRequest) Validate() error {
	if r.Path == "" {
		return fmt.Errorf("path is required")
	}
	return nil
}

func (r WriteFileRequest) Validate() error {
	if r.Path == "" {
		return fmt.Errorf("path is required")
	}
	if r.Content == "" {
		return fmt.Errorf("content is required")
	}
	return nil
}

func (r EditFileRequest) Validate() error {
	if r.Path == "" {
		return fmt.Errorf("path is required")
	}
	if len(r.Operations) == 0 {
		return fmt.Errorf("operations cannot be empty")
	}
	return nil
}

func (r ShellRequest) Validate() error {
	if len(r.Command) == 0 {
		return fmt.Errorf("command cannot be empty")
	}
	return nil
}

func (r FindFileRequest) Validate() error {
	if r.Pattern == "" {
		return fmt.Errorf("pattern is required")
	}
	return nil
}

func (r SearchContentRequest) Validate() error {
	if r.Query == "" {
		return fmt.Errorf("query is required")
	}
	return nil
}

func (r ListDirectoryRequest) Validate() error {
	if r.Path == "" {
		return fmt.Errorf("path is required")
	}
	return nil
}

// Response structs for tool operations

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

// FindFileResponse contains the result of a FindFile operation
type FindFileResponse struct {
	Matches    []string // List of relative paths matching the pattern
	Offset     int
	Limit      int
	TotalCount int  // Total matches found (may be capped for performance)
	Truncated  bool // True if more matches exist
}

// SearchContentMatch represents a single match in a file
type SearchContentMatch struct {
	File        string // Relative path to the file
	LineNumber  int    // 1-based line number
	LineContent string // Content of the matching line
}

// SearchContentResponse contains the result of a SearchContent operation
type SearchContentResponse struct {
	Matches    []SearchContentMatch
	Offset     int
	Limit      int
	TotalCount int  // Total matches found
	Truncated  bool // True if more matches exist
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
	Command        []string          `json:"command"`
	WorkingDir     string            `json:"working_dir,omitempty"`
	TimeoutSeconds int               `json:"timeout_seconds,omitempty"`
	Env            map[string]string `json:"env,omitempty"`
	EnvFiles       []string          `json:"env_files,omitempty"` // Paths to .env files to load (relative to workspace root)
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

// DockerConfig contains configuration for Docker readiness checks.
type DockerConfig struct {
	CheckCommand []string // e.g., ["docker", "info"]
	StartCommand []string // e.g., ["docker", "desktop", "start"]
}

// TodoStatus represents the status of a todo item.
type TodoStatus string

const (
	TodoStatusPending    TodoStatus = "pending"
	TodoStatusInProgress TodoStatus = "in_progress"
	TodoStatusCompleted  TodoStatus = "completed"
	TodoStatusCancelled  TodoStatus = "cancelled"
)

// Todo represents a single task item.
type Todo struct {
	Description string     `json:"description"`
	Status      TodoStatus `json:"status"`
}

// ReadTodosResponse contains the list of current todos.
type ReadTodosResponse struct {
	Todos []Todo `json:"todos"`
}

// WriteTodosResponse contains the result of a WriteTodos operation.
type WriteTodosResponse struct {
	Count int `json:"count"`
}
