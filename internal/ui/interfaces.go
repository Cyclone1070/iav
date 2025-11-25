package ui

import "context"

// PermissionDecision represents the user's choice for a permission request
type PermissionDecision string

const (
	DecisionAllow       PermissionDecision = "allow"
	DecisionDeny        PermissionDecision = "deny"
	DecisionAllowAlways PermissionDecision = "allow_always"
)

// UserInterface defines the contract for all user interactions.
// It follows a Read/Write pattern for clarity.
//
// Context Usage:
// All methods accept context.Context for cancellation support.
// If the user cancels (Ctrl+C), the context will be cancelled,
// and implementations should return immediately with context.Canceled error.
type UserInterface interface {
	// ReadInput prompts the user for general text input
	ReadInput(ctx context.Context, prompt string) (string, error)

	// ReadPermission prompts the user for a yes/no/always permission decision
	ReadPermission(ctx context.Context, prompt string) (PermissionDecision, error)

	// WriteStatus displays ephemeral status updates (e.g., "Thinking...")
	WriteStatus(phase string, message string)

	// WriteMessage displays the agent's actual text responses
	WriteMessage(content string)
}
