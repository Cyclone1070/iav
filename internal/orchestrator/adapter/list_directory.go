package adapter

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Cyclone1070/deployforme/internal/tools"
	toolModels "github.com/Cyclone1070/deployforme/internal/tools/models"
)

// ListDirectory adapts ListDirectory function to the Tool interface
type ListDirectory struct {
	wCtx *toolModels.WorkspaceContext
}

// NewListDirectory creates a new ListDirectory adapter
func NewListDirectory(w *toolModels.WorkspaceContext) *ListDirectory {
	return &ListDirectory{wCtx: w}
}

// Name returns the tool name
func (l *ListDirectory) Name() string {
	return "list_directory"
}

// Description returns the tool description
func (l *ListDirectory) Description() string {
	return "Lists the contents of a directory"
}

// Schema returns the JSON schema
func (l *ListDirectory) Schema() string {
	return `{
		"type": "object",
		"properties": {
			"path": {
				"type": "string",
				"description": "Path to the directory (relative to workspace root)"
			},
			"max_depth": {
				"type": "integer",
				"description": "Maximum recursion depth (-1 for unlimited)"
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
		"required": ["path"]
	}`
}

// Execute runs the tool
func (l *ListDirectory) Execute(ctx context.Context, args string) (string, error) {
	var req toolModels.ListDirectoryRequest
	if err := json.Unmarshal([]byte(args), &req); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	// Set defaults
	if req.Limit == 0 {
		req.Limit = 100
	}
	if req.MaxDepth < 0 {
		req.MaxDepth = -1 // unlimited
	}

	resp, err := tools.ListDirectory(l.wCtx, req)
	if err != nil {
		return "", err
	}

	bytes, err := json.Marshal(resp)
	if err != nil {
		return "", fmt.Errorf("failed to marshal response: %w", err)
	}
	return string(bytes), nil
}
