package adapter

import (
	"context"

	provider "github.com/Cyclone1070/deployforme/internal/provider/models"
)

// Tool defines how the Agent interacts with a capability.
// All adapters must implement this interface.
type Tool interface {
	// Name returns the tool name for the LLM
	Name() string

	// Description returns a human-readable description
	Description() string

	// Definition returns the structured tool definition for native tool calling
	Definition() provider.ToolDefinition

	// Execute runs the tool with JSON arguments and returns a JSON result
	Execute(ctx context.Context, args string) (string, error)
}
