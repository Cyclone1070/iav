package adapter

import (
	"context"

	provider "github.com/Cyclone1070/iav/internal/provider/models"
)

// Tool represents a capability the agent can use.
// Each tool must be stateless and safe for concurrent use.
type Tool interface {
	// Name returns the unique identifier for this tool
	Name() string

	// Description returns a human-readable description
	Description() string

	// Definition returns the structured tool definition for the provider
	Definition() provider.ToolDefinition

	// Execute runs the tool with the given arguments
	// Args is a map of argument names to values, as provided by the LLM
	Execute(ctx context.Context, args map[string]any) (string, error)
}
