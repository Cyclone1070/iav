package search

import (
	"fmt"

	"github.com/Cyclone1070/iav/internal/config"
)

// SearchContentMatch represents a single match in a file
type SearchContentMatch struct {
	File        string // Relative path to the file
	LineNumber  int    // 1-based line number
	LineContent string // Content of the matching line
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

// Validate validates the SearchContentRequest
func (r SearchContentRequest) Validate(cfg *config.Config) error {
	if r.Query == "" {
		return fmt.Errorf("query is required")
	}
	if r.Offset < 0 {
		return fmt.Errorf("offset cannot be negative")
	}
	if r.Limit < 0 {
		return fmt.Errorf("limit cannot be negative")
	}
	if r.Limit > cfg.Tools.MaxSearchContentLimit {
		return fmt.Errorf("limit %d exceeds maximum %d", r.Limit, cfg.Tools.MaxSearchContentLimit)
	}
	return nil
}

// SearchContentResponse contains the result of a SearchContent operation
type SearchContentResponse struct {
	Matches    []SearchContentMatch
	Offset     int
	Limit      int
	TotalCount int  // Total matches found (may be capped for performance)
	Truncated  bool // True if more matches exist
}
