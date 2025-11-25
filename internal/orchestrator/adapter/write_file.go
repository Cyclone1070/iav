package adapter

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Cyclone1070/deployforme/internal/tools"
	toolModels "github.com/Cyclone1070/deployforme/internal/tools/models"
)

// WriteFile adapts WriteFile function to the Tool interface
type WriteFile struct {
	wCtx *toolModels.WorkspaceContext
}

// NewWriteFile creates a new WriteFile adapter
func NewWriteFile(w *toolModels.WorkspaceContext) *WriteFile {
	return &WriteFile{wCtx: w}
}

// Name returns the tool name
func (w *WriteFile) Name() string {
	return "write_file"
}

// Description returns the tool description
func (w *WriteFile) Description() string {
	return "Creates a new file in the workspace"
}

// Schema returns the JSON schema
func (w *WriteFile) Schema() string {
	return `{
		"type": "object",
		"properties": {
			"path": {
				"type": "string",
				"description": "Path to the file (relative to workspace root)"
			},
			"content": {
				"type": "string",
				"description": "File content"
			},
			"perm": {
				"type": "integer",
				"description": "File permissions (octal, e.g., 0644)"
			}
		},
		"required": ["path", "content"]
	}`
}

// Execute runs the tool
func (w *WriteFile) Execute(ctx context.Context, args string) (string, error) {
	var req toolModels.WriteFileRequest
	if err := json.Unmarshal([]byte(args), &req); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	resp, err := tools.WriteFile(w.wCtx, req)
	if err != nil {
		return "", err
	}

	bytes, err := json.Marshal(resp)
	if err != nil {
		return "", fmt.Errorf("failed to marshal response: %w", err)
	}
	return string(bytes), nil
}
