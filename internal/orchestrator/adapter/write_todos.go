package adapter

import (
	provider "github.com/Cyclone1070/deployforme/internal/provider/models"
	"context"
	"encoding/json"
	"fmt"

	"github.com/Cyclone1070/deployforme/internal/tools"
	toolModels "github.com/Cyclone1070/deployforme/internal/tools/models"
)

// WriteTodos adapts WriteTodos function to the Tool interface
type WriteTodos struct {
	wCtx *toolModels.WorkspaceContext
}

// NewWriteTodos creates a new WriteTodos adapter
func NewWriteTodos(w *toolModels.WorkspaceContext) *WriteTodos {
	return &WriteTodos{wCtx: w}
}

// Name returns the tool name
func (w *WriteTodos) Name() string {
	return "write_todos"
}

// Description returns the tool description
func (w *WriteTodos) Description() string {
	return "Replaces the current list of todos"
}

// Definition returns the structured tool definition
func (w *WriteTodos) Definition() provider.ToolDefinition {
	return provider.ToolDefinition{
		Name:        "write_todos",
		Description: "Writes TODO items",
		Parameters: &provider.ParameterSchema{
			Type: "object",
			Properties: map[string]provider.PropertySchema{
				"todos": {
					Type:        "array",
					Description: "List of TODO items",
				},
			},
			Required: []string{"todos"},
		},
	}
}

// Execute runs the tool
func (w *WriteTodos) Execute(ctx context.Context, args string) (string, error) {
	var req toolModels.WriteTodosRequest
	if err := json.Unmarshal([]byte(args), &req); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	resp, err := tools.WriteTodos(w.wCtx, req)
	if err != nil {
		return "", err
	}

	bytes, err := json.Marshal(resp)
	if err != nil {
		return "", fmt.Errorf("failed to marshal response: %w", err)
	}
	return string(bytes), nil
}
