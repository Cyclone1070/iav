package search

import (
	"fmt"

	"github.com/Cyclone1070/iav/internal/config"
)

// -- Contract Types --

// searchContentMatch represents a single match in a file
type searchContentMatch struct {
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
		return fmt.Errorf("query is required")
	}
	if r.Offset < 0 {
		r.Offset = 0
	}
	if r.Limit <= 0 {
		r.Limit = cfg.Tools.DefaultSearchContentLimit
	}
	if r.Limit > cfg.Tools.MaxSearchContentLimit {
		r.Limit = cfg.Tools.MaxSearchContentLimit
	}
	return nil
}

// SearchContentResponse contains the result of a SearchContent operation
type SearchContentResponse struct {
	FormattedMatches string `json:"formatted_matches"` // grep-style for LLM
	Offset           int    `json:"offset"`
	Limit            int    `json:"limit"`
	TotalCount       int    `json:"total_count"` // Total matches found (may be capped for performance)
	HitMaxResults    bool   `json:"hit_max_results"`
}
