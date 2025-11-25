package adapter

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Cyclone1070/deployforme/internal/tools"
	toolModels "github.com/Cyclone1070/deployforme/internal/tools/models"
)

// ReadFile adapts ReadFile function to the Tool interface
type ReadFile struct {
	wCtx *toolModels.WorkspaceContext
}

// NewReadFile creates a new ReadFile adapter
func NewReadFile(w *toolModels.WorkspaceContext) *ReadFile {
	return &ReadFile{wCtx: w}
}

// Name returns the tool name
func (r *ReadFile) Name() string {
	return "read_file"
}

// Description returns the tool description
func (r *ReadFile) Description() string {
	return "Reads a file from the workspace"
}

// Schema returns the JSON schema
func (r *ReadFile) Schema() string {
	return `{
		"type": "object",
		"properties": {
			"path": {
				"type": "string",
				"description": "Path to the file (relative to workspace root)"
			},
			"offset": {
				"type": "integer",
				"description": "Byte offset to start reading from"
			},
			"limit": {
				"type": "integer",
				"description": "Maximum number of bytes to read"
			}
		},
		"required": ["path"]
	}`
}

// Execute runs the tool
func (r *ReadFile) Execute(ctx context.Context, args string) (string, error) {
	var req toolModels.ReadFileRequest
	if err := json.Unmarshal([]byte(args), &req); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	resp, err := tools.ReadFile(r.wCtx, req)
	if err != nil {
		return "", err
	}

	bytes, err := json.Marshal(resp)
	if err != nil {
		return "", fmt.Errorf("failed to marshal response: %w", err)
	}
	return string(bytes), nil
}
