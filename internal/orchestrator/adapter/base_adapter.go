package adapter

import (
	"context"
	"encoding/json"
	"fmt"

	provider "github.com/Cyclone1070/deployforme/internal/provider/models"
	toolModels "github.com/Cyclone1070/deployforme/internal/tools/models"
	"github.com/mitchellh/mapstructure"
)

// Validator is an interface for request types that support validation
type Validator interface {
	Validate() error
}

// ToolExecutor is a function that executes a tool with typed request/response.
// The function signature must match: func(*WorkspaceContext, RequestType) (ResponseType, error)
type ToolExecutor[Req, Resp any] func(*toolModels.WorkspaceContext, Req) (Resp, error)

// BaseAdapter provides common adapter functionality using generics.
// This eliminates duplication across all tool adapters by centralizing:
// - Argument decoding (mapstructure)
// - Tool execution
// - Response marshaling
// - Error handling
//
// Type Parameters:
//   - Req: The request type (e.g., toolModels.ReadFileRequest)
//   - Resp: The response type (e.g., toolModels.ReadFileResponse)
type BaseAdapter[Req, Resp any] struct {
	name        string
	description string
	definition  provider.ToolDefinition
	wCtx        *toolModels.WorkspaceContext
	executor    ToolExecutor[Req, Resp]
}

// NewBaseAdapter creates a new base adapter with the given configuration.
//
// Example usage:
//
//	adapter := NewBaseAdapter(
//	    "read_file",
//	    "Reads a file from the workspace",
//	    &provider.ParameterSchema{...},
//	    workspaceCtx,
//	    tools.ReadFile,  // Direct function reference
//	)
func NewBaseAdapter[Req, Resp any](
	name string,
	description string,
	paramSchema *provider.ParameterSchema,
	wCtx *toolModels.WorkspaceContext,
	executor ToolExecutor[Req, Resp],
) *BaseAdapter[Req, Resp] {
	return &BaseAdapter[Req, Resp]{
		name:        name,
		description: description,
		definition: provider.ToolDefinition{
			Name:        name,
			Description: description,
			Parameters:  paramSchema,
		},
		wCtx:     wCtx,
		executor: executor,
	}
}

// Name implements adapter.Tool
func (b *BaseAdapter[Req, Resp]) Name() string {
	return b.name
}

// Description implements adapter.Tool
func (b *BaseAdapter[Req, Resp]) Description() string {
	return b.description
}

// Definition implements adapter.Tool
func (b *BaseAdapter[Req, Resp]) Definition() provider.ToolDefinition {
	return b.definition
}

// Execute implements adapter.Tool
//
// This method:
// 1. Decodes the args map into a typed request using mapstructure
// 2. Validates the request if it implements Validator interface
// 3. Calls the tool executor function with the typed request
// 4. Marshals the response back to JSON
//
// All error handling is centralized here, eliminating duplication.
func (b *BaseAdapter[Req, Resp]) Execute(ctx context.Context, args map[string]any) (string, error) {
	var req Req

	// Decode map to typed request using mapstructure
	if err := mapstructure.Decode(args, &req); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	// Validate request if it implements Validator interface
	if v, ok := any(req).(Validator); ok {
		if err := v.Validate(); err != nil {
			return "", fmt.Errorf("validation failed: %w", err)
		}
	}

	// Execute the tool function with typed request
	resp, err := b.executor(b.wCtx, req)
	if err != nil {
		return "", err
	}

	// Marshal response to JSON
	bytes, err := json.Marshal(resp)
	if err != nil {
		return "", fmt.Errorf("failed to marshal response: %w", err)
	}

	return string(bytes), nil
}
