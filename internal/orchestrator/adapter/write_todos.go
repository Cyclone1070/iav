package adapter

import (
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

// Schema returns the JSON schema
func (w *WriteTodos) Schema() string {
	return `{
		"type": "object",
		"properties": {
			"todos": {
				"type": "array",
				"items": {
					"type": "object",
					"properties": {
						"description": {"type": "string"},
						"status": {"type": "string"}
					},
					"required": ["description", "status"]
				},
				"description": "List of todos"
			}
		},
		"required": ["todos"]
	}`
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
