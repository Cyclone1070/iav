package adapter

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Cyclone1070/deployforme/internal/tools"
	toolModels "github.com/Cyclone1070/deployforme/internal/tools/models"
)

// SearchContent adapts SearchContent function to the Tool interface
type SearchContent struct {
	wCtx *toolModels.WorkspaceContext
}

// NewSearchContent creates a new SearchContent adapter
func NewSearchContent(w *toolModels.WorkspaceContext) *SearchContent {
	return &SearchContent{wCtx: w}
}

// Name returns the tool name
func (s *SearchContent) Name() string {
	return "search_content"
}

// Description returns the tool description
func (s *SearchContent) Description() string {
	return "Searches for content matching a regex pattern within files"
}

// Schema returns the JSON schema
func (s *SearchContent) Schema() string {
	return `{
		"type": "object",
		"properties": {
			"query": {
				"type": "string",
				"description": "Regex pattern to search for"
			},
			"search_path": {
				"type": "string",
				"description": "Directory to search in"
			},
			"case_sensitive": {
				"type": "boolean",
				"description": "Case-sensitive search"
			},
			"include_ignored": {
				"type": "boolean",
				"description": "Include gitignored files"
			},
			"offset": {
				"type": "integer",
				"description": "Pagination offset"
			},
			"limit": {
				"type": "integer",
				"description": "Pagination limit"
			}
		},
		"required": ["query", "search_path"]
	}`
}

// Execute runs the tool
func (s *SearchContent) Execute(ctx context.Context, args string) (string, error) {
	var req toolModels.SearchContentRequest
	if err := json.Unmarshal([]byte(args), &req); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	// Set defaults
	if req.Limit == 0 {
		req.Limit = 100
	}

	resp, err := tools.SearchContent(s.wCtx, req)
	if err != nil {
		return "", err
	}

	bytes, err := json.Marshal(resp)
	if err != nil {
		return "", fmt.Errorf("failed to marshal response: %w", err)
	}
	return string(bytes), nil
}
