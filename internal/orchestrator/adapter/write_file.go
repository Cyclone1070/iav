package adapter

import (
	"context"
	"encoding/json"
	"fmt"

	provider "github.com/Cyclone1070/deployforme/internal/provider/models"

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

// Definition returns the structured tool definition
func (w *WriteFile) Definition() provider.ToolDefinition {
	return provider.ToolDefinition{
		Name:        "write_file",
		Description: "Creates a new file in the workspace",
		Parameters: &provider.ParameterSchema{
			Type: "object",
			Properties: map[string]provider.PropertySchema{
				"path": {
					Type:        "string",
					Description: "Path to the file (relative to workspace root)",
				},
				"content": {
					Type:        "string",
					Description: "File content",
				},
				"perm": {
					Type:        "integer",
					Description: "File permissions (octal, e.g., 0644)",
				},
			},
			Required: []string{"path", "content"},
		},
	}
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
