package services

import (
	"path/filepath"
	"slices"

	"github.com/Cyclone1070/deployforme/internal/tools/models"
)

// GetCommandRoot extracts the root command (basename) from a command slice.
// e.g., ["/usr/bin/docker", "run"] -> "docker"
func GetCommandRoot(command []string) string {
	if len(command) == 0 {
		return ""
	}
	// Handle paths like /usr/bin/docker
	return filepath.Base(command[0])
}

// EvaluatePolicy checks if a command is allowed by the given policy.
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

	// 3. Check Ask List
	if slices.Contains(policy.Ask, root) {
		return models.ErrShellApprovalRequired
	}

	// 4. Default Deny
	return models.ErrShellRejected
}
