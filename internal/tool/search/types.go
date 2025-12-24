package search

import (
	"fmt"

	"github.com/Cyclone1070/iav/internal/config"
)

// -- Contract Types --

// SearchContentMatch represents a single match in a file
type SearchContentMatch struct {
	File        string `json:"file"`         // Relative path to the file
	LineNumber  int    `json:"line_number"`  // 1-based line number
	LineContent string `json:"line_content"` // Content of the matching line
}

// SearchContentRequest represents the parameters for a SearchContent operation
type SearchContentRequest struct {
	Query          string `json:"query"`
	SearchPath     string `json:"search_path"`
	CaseSensitive  bool   `json:"case_sensitive,omitempty"`
	IncludeIgnored bool   `json:"include_ignored,omitempty"`
	Offset         int    `json:"offset,omitempty"`
	Limit          int    `json:"limit,omitempty"`
}

func (r *SearchContentRequest) Validate(cfg *config.Config) error {
	if r.Query == "" {
		return ErrQueryRequired
	}
	if r.Offset < 0 {
		return fmt.Errorf("%w: %d", ErrInvalidOffset, r.Offset)
	}
	if r.Limit < 0 {
		return fmt.Errorf("%w: %d", ErrInvalidLimit, r.Limit)
	}
	if r.Limit != 0 && r.Limit > cfg.Tools.MaxSearchContentLimit {
		return fmt.Errorf("%w: %d (max %d)", ErrLimitExceeded, r.Limit, cfg.Tools.MaxSearchContentLimit)
	}
	if r.Limit == 0 {
		r.Limit = cfg.Tools.DefaultSearchContentLimit
	}
	return nil
}

// SearchContentResponse contains the result of a SearchContent operation
type SearchContentResponse struct {
	Matches    []SearchContentMatch `json:"matches"`
	Offset     int                  `json:"offset"`
	Limit      int                  `json:"limit"`
	TotalCount int                  `json:"total_count"` // Total matches found (may be capped for performance)
	Truncated  bool                 `json:"truncated"`   // True if more matches exist
}
