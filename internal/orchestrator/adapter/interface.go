package adapter

import "context"

// Tool defines how the Agent interacts with a capability.
// All adapters must implement this interface.
type Tool interface {
	// Name returns the tool name for the LLM
	Name() string

	// Description returns a human-readable description
	Description() string

	// Schema returns the JSON schema for the tool arguments
	Schema() string

	// Execute runs the tool with JSON arguments and returns a JSON result
	Execute(ctx context.Context, args string) (string, error)
}
