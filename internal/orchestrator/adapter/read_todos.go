package adapter

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Cyclone1070/deployforme/internal/tools"
	toolModels "github.com/Cyclone1070/deployforme/internal/tools/models"
)

// ReadTodos adapts ReadTodos function to the Tool interface
type ReadTodos struct {
	wCtx *toolModels.WorkspaceContext
}

// NewReadTodos creates a new ReadTodos adapter
func NewReadTodos(w *toolModels.WorkspaceContext) *ReadTodos {
	return &ReadTodos{wCtx: w}
}

// Name returns the tool name
func (r *ReadTodos) Name() string {
	return "read_todos"
}

// Description returns the tool description
func (r *ReadTodos) Description() string {
	return "Reads the current list of todos"
}

// Schema returns the JSON schema
func (r *ReadTodos) Schema() string {
	return `{
		"type": "object",
		"properties": {}
	}`
}

// Execute runs the tool
func (r *ReadTodos) Execute(ctx context.Context, args string) (string, error) {
	resp, err := tools.ReadTodos(r.wCtx, toolModels.ReadTodosRequest{})
	if err != nil {
		return "", err
	}

	bytes, err := json.Marshal(resp)
	if err != nil {
		return "", fmt.Errorf("failed to marshal response: %w", err)
	}
	return string(bytes), nil
}
