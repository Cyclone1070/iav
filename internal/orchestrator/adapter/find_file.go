package adapter

import (
	provider "github.com/Cyclone1070/deployforme/internal/provider/models"
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

// Definition returns the structured tool definition
func (f *FindFile) Definition() provider.ToolDefinition {
	return provider.ToolDefinition{
		Name:        "find_file",
		Description: "Finds files in the workspace matching a pattern",
		Parameters: &provider.ParameterSchema{
			Type: "object",
			Properties: map[string]provider.PropertySchema{
				"pattern": {
					Type:        "string",
					Description: "Glob pattern to match files",
				},
				"max_depth": {
					Type:        "integer",
					Description: "Maximum directory depth to search",
				},
				"include_ignored": {
					Type:        "boolean",
					Description: "Include gitignored files",
				},
			},
			Required: []string{"pattern"},
		},
	}
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
