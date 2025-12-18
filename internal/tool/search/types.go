package search

import (
	"os"

	"github.com/Cyclone1070/iav/internal/config"
	"github.com/Cyclone1070/iav/internal/tool/pathutil"
)

// SearchContentMatch represents a single match in a file
type SearchContentMatch struct {
	File        string // Relative path to the file
	LineNumber  int    // 1-based line number
	LineContent string // Content of the matching line
}

// SearchContentDTO is the wire format for SearchContent operation
type SearchContentDTO struct {
	Query          string `json:"query"`
	SearchPath     string `json:"search_path"`
	CaseSensitive  bool   `json:"case_sensitive,omitempty"`
	IncludeIgnored bool   `json:"include_ignored,omitempty"`
	Offset         int    `json:"offset,omitempty"`
	Limit          int    `json:"limit,omitempty"`
}

// SearchContentRequest is the validated domain entity for SearchContent operation
type SearchContentRequest struct {
	query          string
	searchAbsPath  string
	searchRelPath  string
	caseSensitive  bool
	includeIgnored bool
	offset         int
	limit          int
}

// NewSearchContentRequest creates a validated SearchContentRequest from a DTO
func NewSearchContentRequest(
	dto SearchContentDTO,
	cfg *config.Config,
	workspaceRoot string,
	fs interface {
		Lstat(path string) (os.FileInfo, error)
		Readlink(path string) (string, error)
		UserHomeDir() (string, error)
	},
) (*SearchContentRequest, error) {
	// Constructor validation
	if dto.Query == "" {
		return nil, &QueryRequiredError{}
	}
	if dto.Offset < 0 {
		return nil, &NegativeOffsetError{Value: int64(dto.Offset)}
	}
	if dto.Limit < 0 {
		return nil, &NegativeLimitError{Value: int64(dto.Limit)}
	}
	if dto.Limit > cfg.Tools.MaxSearchContentLimit {
		return nil, &LimitExceededError{Value: int64(dto.Limit), Max: int64(cfg.Tools.MaxSearchContentLimit)}
	}

	// SearchPath defaults to "." if empty
	searchPath := dto.SearchPath
	if searchPath == "" {
		searchPath = "."
	}

	// Path resolution for search path
	searchAbs, searchRel, err := resolvePathWithFS(workspaceRoot, fs, searchPath)
	if err != nil {
		return nil, err
	}

	return &SearchContentRequest{
		query:          dto.Query,
		searchAbsPath:  searchAbs,
		searchRelPath:  searchRel,
		caseSensitive:  dto.CaseSensitive,
		includeIgnored: dto.IncludeIgnored,
		offset:         dto.Offset,
		limit:          dto.Limit,
	}, nil
}

// Query returns the search query
func (r *SearchContentRequest) Query() string {
	return r.query
}

// SearchAbsPath returns the absolute search path
func (r *SearchContentRequest) SearchAbsPath() string {
	return r.searchAbsPath
}

// SearchRelPath returns the relative search path
func (r *SearchContentRequest) SearchRelPath() string {
	return r.searchRelPath
}

// CaseSensitive returns whether the search is case sensitive
func (r *SearchContentRequest) CaseSensitive() bool {
	return r.caseSensitive
}

// IncludeIgnored returns whether to include ignored files
func (r *SearchContentRequest) IncludeIgnored() bool {
	return r.includeIgnored
}

// Offset returns the offset
func (r *SearchContentRequest) Offset() int {
	return r.offset
}

// Limit returns the limit
func (r *SearchContentRequest) Limit() int {
	return r.limit
}

// SearchContentResponse contains the result of a SearchContent operation
type SearchContentResponse struct {
	Matches    []SearchContentMatch
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
