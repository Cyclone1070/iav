package shell

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// IsDockerCommand checks if the command is a docker command by examining the base name.
// It handles both simple commands ("docker") and full paths ("/usr/bin/docker").
func IsDockerCommand(command []string) bool {
	if len(command) == 0 {
		return false
	}
	// Handle paths like /usr/bin/docker
	return filepath.Base(command[0]) == "docker"
}

// IsDockerComposeUpDetached checks if the command is 'docker compose up' with detached mode (-d or --detach).
// It parses the command arguments to find the compose, up, and detach flags in order.
func IsDockerComposeUpDetached(command []string) bool {
	if !IsDockerCommand(command) {
		return false
	}

	if len(command) < 3 {
		return false
	}

	foundCompose := false
	foundUp := false
	foundDetach := false

	for _, arg := range command[1:] {
		if !foundCompose {
			if arg == "compose" {
				foundCompose = true
			}
			continue
		}

		if !foundUp {
			if arg == "up" {
				foundUp = true
			}
			continue
		}

		if arg == "-d" || arg == "--detach" {
			foundDetach = true
			break
		}
	}

	return foundCompose && foundUp && foundDetach
}

// EnsureDockerReady checks if Docker is running and attempts to start it if not.
// It retries the check up to retryAttempts times with retryIntervalMs formatted as milliseconds after starting Docker.
func EnsureDockerReady(ctx context.Context, runner commandExecutor, config DockerConfig, retryAttempts int, retryIntervalMs int) error {
	if res, err := runner.Run(ctx, config.CheckCommand, "", nil); err == nil && res.ExitCode == 0 {
		return nil
	}

	if _, err := runner.Run(ctx, config.StartCommand, "", nil); err != nil {
		return err
	}

	ticker := time.NewTicker(time.Duration(retryIntervalMs) * time.Millisecond)
	defer ticker.Stop()

	for range retryAttempts {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if res, err := runner.Run(ctx, config.CheckCommand, "", nil); err == nil && res.ExitCode == 0 {
				return nil
			}
		}
	}

	res, err := runner.Run(ctx, config.CheckCommand, "", nil)
	if err != nil {
		return err
	}
	if res.ExitCode != 0 {
		return fmt.Errorf("docker failed to start after retries: exit code %d", res.ExitCode)
	}
	return nil
}

// CollectComposeContainers collects container IDs from a docker compose project in the specified directory.
// It uses 'docker compose ps -q' to get the list of container IDs.
func CollectComposeContainers(ctx context.Context, runner commandExecutor, dir string) ([]string, error) {
	cmd := []string{"docker", "compose", "--project-directory", dir, "ps", "-q"}
	res, err := runner.Run(ctx, cmd, dir, nil)
	if err != nil {
		return nil, err
	}

	stdout := res.Stdout

	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	var ids []string
	for _, line := range lines {
		if line != "" {
			ids = append(ids, strings.TrimSpace(line))
		}
	}
	return ids, nil
}

// FormatContainerStartedNote returns a human-readable note about started containers.
// It formats the message based on the number of containers started.
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
