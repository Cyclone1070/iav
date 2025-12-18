package directory

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/Cyclone1070/iav/internal/config"
	"github.com/Cyclone1070/iav/internal/tool/pathutil"
)

// DirectoryEntry represents a single entry in a directory listing
type DirectoryEntry struct {
	RelativePath string
	IsDir        bool
}

// ListDirectoryDTO is the wire format for ListDirectory operation
type ListDirectoryDTO struct {
	Path           string `json:"path"`
	MaxDepth       int    `json:"max_depth,omitempty"`
	IncludeIgnored bool   `json:"include_ignored,omitempty"`
	Offset         int    `json:"offset,omitempty"`
	Limit          int    `json:"limit,omitempty"`
}

// ListDirectoryRequest is the validated domain entity for ListDirectory operation
type ListDirectoryRequest struct {
	absPath        string
	relPath        string
	maxDepth       int
	includeIgnored bool
	offset         int
	limit          int
}

// NewListDirectoryRequest creates a validated ListDirectoryRequest from a DTO
func NewListDirectoryRequest(
	dto ListDirectoryDTO,
	cfg *config.Config,
	workspaceRoot string,
	fs interface {
		Lstat(path string) (os.FileInfo, error)
		Readlink(path string) (string, error)
		UserHomeDir() (string, error)
	},
) (*ListDirectoryRequest, error) {
	// Constructor validation
	if dto.Offset < 0 {
		return nil, &NegativeOffsetError{Value: int64(dto.Offset)}
	}
	if dto.Limit < 0 {
		return nil, &NegativeLimitError{Value: int64(dto.Limit)}
	}
	if dto.Limit > cfg.Tools.MaxListDirectoryLimit {
		return nil, &LimitExceededError{Value: int64(dto.Limit), Max: int64(cfg.Tools.MaxListDirectoryLimit)}
	}

	// Path defaults to "." if empty
	path := dto.Path
	if path == "" {
		path = "."
	}

	// Path resolution
	abs, rel, err := resolvePathWithFS(workspaceRoot, fs, path)
	if err != nil {
		var ow interface{ OutsideWorkspace() bool }
		if errors.As(err, &ow) && ow.OutsideWorkspace() {
			return nil, &PathTraversalError{Path: path}
		}
		return nil, &ListDirError{Path: path, Cause: err}
	}

	return &ListDirectoryRequest{
		absPath:        abs,
		relPath:        rel,
		maxDepth:       dto.MaxDepth,
		includeIgnored: dto.IncludeIgnored,
		offset:         dto.Offset,
		limit:          dto.Limit,
	}, nil
}

// AbsPath returns the absolute path
func (r *ListDirectoryRequest) AbsPath() string {
	return r.absPath
}

// RelPath returns the relative path
func (r *ListDirectoryRequest) RelPath() string {
	return r.relPath
}

// MaxDepth returns the max depth
func (r *ListDirectoryRequest) MaxDepth() int {
	return r.maxDepth
}

// IncludeIgnored returns whether to include ignored files
func (r *ListDirectoryRequest) IncludeIgnored() bool {
	return r.includeIgnored
}

// Offset returns the offset
func (r *ListDirectoryRequest) Offset() int {
	return r.offset
}

// Limit returns the limit
func (r *ListDirectoryRequest) Limit() int {
	return r.limit
}

// ListDirectoryResponse contains the result of a ListDirectory operation
type ListDirectoryResponse struct {
	DirectoryPath    string
	Entries          []DirectoryEntry
	Offset           int
	Limit            int
	TotalCount       int    `json:"total_count"` // Total entries before pagination
	Truncated        bool   `json:"truncated"`   // True if more entries exist beyond offset+limit
	TruncationReason string `json:"truncation_reason,omitempty"`
}

// FindFileDTO is the wire format for FindFile operation
type FindFileDTO struct {
	Pattern        string `json:"pattern"`
	SearchPath     string `json:"search_path"`
	MaxDepth       int    `json:"max_depth,omitempty"`
	IncludeIgnored bool   `json:"include_ignored,omitempty"`
	Offset         int    `json:"offset,omitempty"`
	Limit          int    `json:"limit,omitempty"`
}

// FindFileRequest is the validated domain entity for FindFile operation
type FindFileRequest struct {
	pattern        string
	searchAbsPath  string
	searchRelPath  string
	maxDepth       int
	includeIgnored bool
	offset         int
	limit          int
}

// NewFindFileRequest creates a validated FindFileRequest from a DTO
func NewFindFileRequest(
	dto FindFileDTO,
	cfg *config.Config,
	workspaceRoot string,
	fs interface {
		Lstat(path string) (os.FileInfo, error)
		Readlink(path string) (string, error)
		UserHomeDir() (string, error)
	},
) (*FindFileRequest, error) {
	// Constructor validation
	if dto.Pattern == "" {
		return nil, &PatternRequiredError{}
	}

	// Check for path traversal or absolute path in pattern
	if strings.Contains(dto.Pattern, "..") || filepath.IsAbs(dto.Pattern) {
		return nil, &PathTraversalError{Path: dto.Pattern}
	}

	// Simple check for path traversal in search path if provided
	if dto.SearchPath != "" && (dto.SearchPath == ".." || dto.SearchPath == "/" || dto.SearchPath == "\\") {
		return nil, &PathTraversalError{Path: dto.SearchPath}
	}

	if dto.Offset < 0 {
		return nil, &NegativeOffsetError{Value: int64(dto.Offset)}
	}
	if dto.Limit < 0 {
		return nil, &NegativeLimitError{Value: int64(dto.Limit)}
	}
	if dto.Limit > cfg.Tools.MaxFindFileLimit {
		return nil, &LimitExceededError{Value: int64(dto.Limit), Max: int64(cfg.Tools.MaxFindFileLimit)}
	}

	// SearchPath defaults to "." if empty
	searchPath := dto.SearchPath
	if searchPath == "" {
		searchPath = "."
	}

	// Path resolution for search path
	searchAbs, searchRel, err := resolvePathWithFS(workspaceRoot, fs, searchPath)
	if err != nil {
		var ow interface{ OutsideWorkspace() bool }
		if errors.As(err, &ow) && ow.OutsideWorkspace() {
			return nil, &PathTraversalError{Path: searchPath}
		}
		return nil, &FindFileError{Path: searchPath, Cause: err}
	}

	return &FindFileRequest{
		pattern:        dto.Pattern,
		searchAbsPath:  searchAbs,
		searchRelPath:  searchRel,
		maxDepth:       dto.MaxDepth,
		includeIgnored: dto.IncludeIgnored,
		offset:         dto.Offset,
		limit:          dto.Limit,
	}, nil
}

// Pattern returns the search pattern
func (r *FindFileRequest) Pattern() string {
	return r.pattern
}

// SearchAbsPath returns the absolute search path
func (r *FindFileRequest) SearchAbsPath() string {
	return r.searchAbsPath
}

// SearchRelPath returns the relative search path
func (r *FindFileRequest) SearchRelPath() string {
	return r.searchRelPath
}

// MaxDepth returns the max depth
func (r *FindFileRequest) MaxDepth() int {
	return r.maxDepth
}

// IncludeIgnored returns whether to include ignored files
func (r *FindFileRequest) IncludeIgnored() bool {
	return r.includeIgnored
}

// Offset returns the offset
func (r *FindFileRequest) Offset() int {
	return r.offset
}

// Limit returns the limit
func (r *FindFileRequest) Limit() int {
	return r.limit
}

// FindFileResponse contains the result of a FindFile operation
type FindFileResponse struct {
	Matches    []string // List of relative paths matching the pattern
	Offset     int
	Limit      int
	TotalCount int  // Total matches found (may be capped for performance)
	Truncated  bool // True if more matches exist
}

// resolvePathWithFS is a helper that calls pathutil.Resolve with the given filesystem
func resolvePathWithFS(
	workspaceRoot string,
	fs interface {
		Lstat(path string) (os.FileInfo, error)
		Readlink(path string) (string, error)
		UserHomeDir() (string, error)
	},
	path string,
) (string, string, error) {
	// Cast to pathutil.FileSystem (the interface is identical)
	fsImpl := fs.(pathutil.FileSystem)
	return pathutil.Resolve(workspaceRoot, fsImpl, path)
}
