package adapter

import (
	provider "github.com/Cyclone1070/deployforme/internal/provider/models"
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

// Definition returns the structured tool definition
func (s *SearchContent) Definition() provider.ToolDefinition {
	return provider.ToolDefinition{
		Name:        "search_content",
		Description: "Searches for content within files",
		Parameters: &provider.ParameterSchema{
			Type: "object",
			Properties: map[string]provider.PropertySchema{
				"query": {
					Type:        "string",
					Description: "Search query",
				},
				"path": {
					Type:        "string",
					Description: "Path to search in",
				},
				"case_sensitive": {
					Type:        "boolean",
					Description: "Case sensitive search",
				},
				"include_ignored": {
					Type:        "boolean",
					Description: "Include gitignored files",
				},
			},
			Required: []string{"query"},
		},
	}
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
