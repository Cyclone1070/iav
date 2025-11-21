package services

import (
	"context"
	"strings"
	"time"

	"github.com/Cyclone1070/deployforme/internal/tools/models"
)

// EnsureDockerReady checks if Docker is running and attempts to start it if not.
func EnsureDockerReady(ctx context.Context, runner models.CommandRunner, config models.DockerConfig) error {
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

	for i := 0; i < 10; i++ {
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

func CollectComposeContainers(ctx context.Context, runner models.CommandRunner, dir string) ([]string, error) {
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
