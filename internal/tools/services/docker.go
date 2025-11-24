package services

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/Cyclone1070/deployforme/internal/tools/models"
)

// IsDockerCommand checks if the command is a docker command.
func IsDockerCommand(command []string) bool {
	if len(command) == 0 {
		return false
	}
	// Handle paths like /usr/bin/docker
	return filepath.Base(command[0]) == "docker"
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

// EnsureDockerReady checks if Docker is running and attempts to start it if not.
func EnsureDockerReady(ctx context.Context, runner models.CommandExecutor, config models.DockerConfig) error {
	// 1. Check if Docker is running
	if _, err := runner.Run(ctx, config.CheckCommand); err == nil {
		return nil
	}

	// 2. Attempt to start Docker
	if _, err := runner.Run(ctx, config.StartCommand); err != nil {
		return err
	}

	// 3. Wait for Docker to be ready
	// Retry up to 10 times with 1s delay (simplified for now)
	// In a real app, we might want this configurable or use a backoff.
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range 10 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if _, err := runner.Run(ctx, config.CheckCommand); err == nil {
				return nil
			}
		}
	}

	_, err := runner.Run(ctx, config.CheckCommand)
	return err // Return the last error
}

// CollectComposeContainers collects container IDs from a docker compose project.
func CollectComposeContainers(ctx context.Context, runner models.CommandExecutor, dir string) ([]string, error) {
	cmd := []string{"docker", "compose", "--project-directory", dir, "ps", "-q"}
	output, err := runner.Run(ctx, cmd)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var ids []string
	for _, line := range lines {
		if line != "" {
			ids = append(ids, strings.TrimSpace(line))
		}
	}
	return ids, nil
}

// FormatContainerStartedNote returns a human-readable note about started containers.
func FormatContainerStartedNote(ids []string) string {
	if len(ids) == 0 {
		return ""
	}
	count := len(ids)
	if count == 1 {
		return fmt.Sprintf("Started 1 Docker container: %s", ids[0])
	}
	return fmt.Sprintf("Started %d Docker containers", count)
}
