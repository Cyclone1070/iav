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

// IsDockerCommand checks if the command is a docker command.
func IsDockerCommand(command []string) bool {
	return GetCommandRoot(command) == "docker"
}

// IsDockerComposeUpDetached checks if the command is 'docker compose up -d'.
func IsDockerComposeUpDetached(command []string) bool {
	if !IsDockerCommand(command) {
		return false
	}

	// Need at least "docker", "compose", "up"
	if len(command) < 3 {
		return false
	}

	// We need to find "compose" and "up" as subcommands, and "-d" or "--detach" as a flag for "up".
	// Note: "compose" is a subcommand of "docker". "up" is a subcommand of "compose".
	// Flags can appear:
	// 1. After "docker" (global flags)
	// 2. After "compose" (compose flags)
	// 3. After "up" (up flags)
	//
	// We are looking for the presence of "compose" then "up" then "-d"/"--detach".
	// However, flags for "compose" can be anywhere between "compose" and "up".
	// And flags for "up" can be anywhere after "up".
	// Also, "docker-compose" (hyphenated) is legacy but we might want to support it?
	// The current IsDockerCommand only checks for "docker".
	// Let's stick to "docker compose" for now as per the original code.

	foundCompose := false
	foundUp := false
	foundDetach := false

	// Skip the first element (docker)
	for _, arg := range command[1:] {
		if !foundCompose {
			if arg == "compose" {
				foundCompose = true
			}
			continue
		}

		// We found compose, now looking for up
		if !foundUp {
			if arg == "up" {
				foundUp = true
			}
			continue
		}

		// We found up, now looking for detach
		if arg == "-d" || arg == "--detach" {
			foundDetach = true
			break // We found everything we need
		}
	}

	return foundCompose && foundUp && foundDetach
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
