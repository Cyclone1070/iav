package adapter

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Cyclone1070/deployforme/internal/tools"
	toolModels "github.com/Cyclone1070/deployforme/internal/tools/models"
)

// FindFile adapts FindFile function to the Tool interface
type FindFile struct {
	wCtx *toolModels.WorkspaceContext
}

// NewFindFile creates a new FindFile adapter
func NewFindFile(w *toolModels.WorkspaceContext) *FindFile {
	return &FindFile{wCtx: w}
}

// Name returns the tool name
func (f *FindFile) Name() string {
	return "find_file"
}

// Description returns the tool description
func (f *FindFile) Description() string {
	return "Searches for files matching a glob pattern"
}

// Schema returns the JSON schema
func (f *FindFile) Schema() string {
	return `{
		"type": "object",
		"properties": {
			"pattern": {
				"type": "string",
				"description": "Glob pattern to match"
			},
			"search_path": {
				"type": "string",
				"description": "Directory to search in"
			},
			"max_depth": {
				"type": "integer",
				"description": "Maximum search depth"
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
		"required": ["pattern", "search_path"]
	}`
}

// Execute runs the tool
func (f *FindFile) Execute(ctx context.Context, args string) (string, error) {
	var req toolModels.FindFileRequest
	if err := json.Unmarshal([]byte(args), &req); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	// Set defaults
	if req.Limit == 0 {
		req.Limit = 100
	}

	resp, err := tools.FindFile(f.wCtx, req)
	if err != nil {
		return "", err
	}

	bytes, err := json.Marshal(resp)
	if err != nil {
		return "", fmt.Errorf("failed to marshal response: %w", err)
	}
	return string(bytes), nil
}
