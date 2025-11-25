package services

import (
	"path/filepath"
	"slices"

	"github.com/Cyclone1070/deployforme/internal/tools/models"
)

// GetCommandRoot extracts the root command (basename) from a command slice.
// It handles full paths by extracting just the command name.
// Example: ["/usr/bin/docker", "run"] returns "docker".
func GetCommandRoot(command []string) string {
	if len(command) == 0 {
		return ""
	}
	// Handle paths like /usr/bin/docker
	return filepath.Base(command[0])
}

// EvaluatePolicy checks if a command is allowed by the given policy.
// It checks session-allowed commands first, then the allow list, then the deny list.
// Returns ErrShellApprovalRequired if the command needs approval, or ErrShellRejected if denied.
// Default behavior: commands not in Allow or Deny lists require approval (ask).
func EvaluatePolicy(policy models.CommandPolicy, command []string) error {
	root := GetCommandRoot(command)
	if root == "" {
		return models.ErrShellRejected
	}

	// 1. Check SessionAllow (Override)
	if policy.SessionAllow != nil && policy.SessionAllow[root] {
		return nil
	}

	// 2. Check Allow List
	if slices.Contains(policy.Allow, root) {
		return nil
	}

	// 3. Check Deny List
	if slices.Contains(policy.Deny, root) {
		return models.ErrShellRejected
	}

	// 4. Default Ask (commands not in any list require approval)
	return models.ErrShellApprovalRequired
}
