package directory

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Cyclone1070/iav/internal/config"
)

// DirectoryEntry represents a single entry in a directory listing
type DirectoryEntry struct {
	RelativePath string
	IsDir        bool
}

// ListDirectoryRequest contains parameters for ListDirectory operation
type ListDirectoryRequest struct {
	Path           string `json:"path"`
	MaxDepth       int    `json:"max_depth,omitempty"`
	IncludeIgnored bool   `json:"include_ignored,omitempty"`
	Offset         int    `json:"offset,omitempty"`
	Limit          int    `json:"limit,omitempty"`
}

// Validate validates the ListDirectoryRequest
func (r ListDirectoryRequest) Validate(cfg *config.Config) error {
	// Path is optional - defaults to "." (workspace root) in tool
	if r.Offset < 0 {
		return fmt.Errorf("offset cannot be negative")
	}
	if r.Limit < 0 {
		return fmt.Errorf("limit cannot be negative")
	}
	if r.Limit > cfg.Tools.MaxListDirectoryLimit {
		return fmt.Errorf("limit %d exceeds maximum %d", r.Limit, cfg.Tools.MaxListDirectoryLimit)
	}
	return nil
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

// FindFileRequest contains parameters for FindFile operation
type FindFileRequest struct {
	Pattern        string `json:"pattern"`
	SearchPath     string `json:"search_path"`
	MaxDepth       int    `json:"max_depth,omitempty"`
	IncludeIgnored bool   `json:"include_ignored,omitempty"`
	Offset         int    `json:"offset,omitempty"`
	Limit          int    `json:"limit,omitempty"`
}

// Validate validates the FindFileRequest
func (r FindFileRequest) Validate(cfg *config.Config) error {
	if r.Pattern == "" {
		return fmt.Errorf("pattern is required")
	}
	// Check for path traversal or absolute path in pattern
	if strings.Contains(r.Pattern, "..") || filepath.IsAbs(r.Pattern) {
		return fmt.Errorf("invalid pattern: traversal or absolute path not allowed")
	}

	// Simple check for path traversal in search path if provided
	if r.SearchPath != "" && (r.SearchPath == ".." || r.SearchPath == "/" || r.SearchPath == "\\") {
		return fmt.Errorf("invalid search path: traversal not allowed")
	}
	if r.Offset < 0 {
		return fmt.Errorf("offset cannot be negative")
	}
	if r.Limit < 0 {
		return fmt.Errorf("limit cannot be negative")
	}
	if r.Limit > cfg.Tools.MaxFindFileLimit {
		return fmt.Errorf("limit %d exceeds maximum %d", r.Limit, cfg.Tools.MaxFindFileLimit)
	}
	return nil
}

// FindFileResponse contains the result of a FindFile operation
type FindFileResponse struct {
	Matches    []string // List of relative paths matching the pattern
	Offset     int
	Limit      int
	TotalCount int  // Total matches found (may be capped for performance)
	Truncated  bool // True if more matches exist
}
