package models

import (
	"context"
)

// PolicyService decides if an action is allowed.
// It encapsulates both the static rules and the user interaction.
type PolicyService interface {
	// CheckShell validates if a shell command is allowed to execute
	CheckShell(ctx context.Context, command []string) error

	// CheckTool validates if a tool is allowed to be used
	CheckTool(ctx context.Context, toolName string) error
}
