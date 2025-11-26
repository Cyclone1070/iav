package adapter

import (
	"context"
	"encoding/json"
	"fmt"

	provider "github.com/Cyclone1070/deployforme/internal/provider/models"

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

// Definition returns the structured tool definition
func (e *EditFile) Definition() provider.ToolDefinition {
	return provider.ToolDefinition{
		Name:        "edit_file",
		Description: "Applies edit operations to an existing file",
		Parameters: &provider.ParameterSchema{
			Type: "object",
			Properties: map[string]provider.PropertySchema{
				"path": {
					Type:        "string",
					Description: "Path to the file (relative to workspace root)",
				},
				"operations": {
					Type:        "array",
					Description: "List of edit operations",
					Items: &provider.PropertySchema{
						Type: "object",
					},
				},
			},
			Required: []string{"path", "operations"},
		},
	}
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
