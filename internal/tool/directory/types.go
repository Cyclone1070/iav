package directory

import (
	"github.com/Cyclone1070/iav/internal/config"
)

// -- Directory Tool Contract Types --

// DirectoryEntry represents a single entry in a directory listing
type DirectoryEntry struct {
	RelativePath string
	IsDir        bool
}

// ListDirectoryRequest represents the parameters for a ListDirectory operation
type ListDirectoryRequest struct {
	Path           string `json:"path"`
	MaxDepth       int    `json:"max_depth,omitempty"`
	IncludeIgnored bool   `json:"include_ignored,omitempty"`
	Offset         int    `json:"offset,omitempty"`
	Limit          int    `json:"limit,omitempty"`
}

func (r *ListDirectoryRequest) Validate(cfg *config.Config) error {
	if r.Offset < 0 {
		return ErrInvalidOffset
	}
	if r.Limit < 0 {
		return ErrInvalidLimit
	}
	if r.Limit != 0 && r.Limit > cfg.Tools.MaxListDirectoryLimit {
		return ErrLimitExceeded
	}
	if r.Limit == 0 {
		r.Limit = cfg.Tools.DefaultListDirectoryLimit
	}
	if r.MaxDepth < 0 {
		r.MaxDepth = -1 // unlimited
	}
	return nil
}

// ListDirectoryResponse contains the result of a ListDirectory operation
type ListDirectoryResponse struct {
	DirectoryPath    string
	Entries          []DirectoryEntry
	Offset           int
	Limit            int
	TotalCount       int
	Truncated        bool
	TruncationReason string
}

// FindFileRequest represents the parameters for a FindFile operation
type FindFileRequest struct {
	Pattern        string `json:"pattern"`
	SearchPath     string `json:"search_path"`
	MaxDepth       int    `json:"max_depth,omitempty"`
	IncludeIgnored bool   `json:"include_ignored,omitempty"`
	Offset         int    `json:"offset,omitempty"`
	Limit          int    `json:"limit,omitempty"`
}

func (r *FindFileRequest) Validate(cfg *config.Config) error {
	if r.Pattern == "" {
		return ErrPatternRequired
	}
	if r.Offset < 0 {
		return ErrInvalidOffset
	}
	if r.Limit < 0 {
		return ErrInvalidLimit
	}
	if r.Limit != 0 && r.Limit > cfg.Tools.MaxFindFileLimit {
		return ErrLimitExceeded
	}
	if r.Limit == 0 {
		r.Limit = cfg.Tools.DefaultFindFileLimit
	}
	return nil
}

// FindFileResponse contains the result of a FindFile operation
type FindFileResponse struct {
	Matches    []string
	Offset     int
	Limit      int
	TotalCount int
	Truncated  bool
}
