package adapter

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Cyclone1070/deployforme/internal/tools"
	toolModels "github.com/Cyclone1070/deployforme/internal/tools/models"
)

// EditFile adapts EditFile function to the Tool interface
type EditFile struct {
	wCtx *toolModels.WorkspaceContext
}

// NewEditFile creates a new EditFile adapter
func NewEditFile(w *toolModels.WorkspaceContext) *EditFile {
	return &EditFile{wCtx: w}
}

// Name returns the tool name
func (e *EditFile) Name() string {
	return "edit_file"
}

// Description returns the tool description
func (e *EditFile) Description() string {
	return "Applies edit operations to an existing file"
}

// Schema returns the JSON schema
func (e *EditFile) Schema() string {
	return `{
		"type": "object",
		"properties": {
			"path": {
				"type": "string",
				"description": "Path to the file (relative to workspace root)"
			},
			"operations": {
				"type": "array",
				"items": {
					"type": "object",
					"properties": {
						"before": {"type": "string"},
						"after": {"type": "string"},
						"expected_replacements": {"type": "integer"}
					},
					"required": ["before", "after", "expected_replacements"]
				},
				"description": "List of edit operations"
			}
		},
		"required": ["path", "operations"]
	}`
}

// Execute runs the tool
func (e *EditFile) Execute(ctx context.Context, args string) (string, error) {
	var req toolModels.EditFileRequest
	if err := json.Unmarshal([]byte(args), &req); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	resp, err := tools.EditFile(e.wCtx, req)
	if err != nil {
		return "", err
	}

	bytes, err := json.Marshal(resp)
	if err != nil {
		return "", fmt.Errorf("failed to marshal response: %w", err)
	}
	return string(bytes), nil
}
