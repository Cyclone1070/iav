package services

import (
	"context"
	"errors"
	"testing"

	"github.com/Cyclone1070/iav/internal/tools/models"
)

func TestEnsureDockerReady(t *testing.T) {
	config := models.DockerConfig{
		CheckCommand: []string{"docker", "info"},
		StartCommand: []string{"open", "-a", "Docker"},
	}

	t.Run("Success immediately", func(t *testing.T) {
		runner := &MockCommandExecutor{
			RunFunc: func(ctx context.Context, cmd []string) ([]byte, error) {
				if cmd[0] == "docker" && cmd[1] == "info" {
					return nil, nil
				}
				return nil, errors.New("unexpected command")
			},
		}
		err := EnsureDockerReady(context.Background(), runner, config)
		if err != nil {
			t.Errorf("EnsureDockerReady failed: %v", err)
		}
	})

	t.Run("Start required and succeeds", func(t *testing.T) {
		checkCalls := 0
		runner := &MockCommandExecutor{
			RunFunc: func(ctx context.Context, cmd []string) ([]byte, error) {
				if cmd[0] == "docker" && cmd[1] == "info" {
					checkCalls++
					if checkCalls == 1 {
						return nil, errors.New("docker not running")
					}
					return nil, nil // Success on second call
				}
				if cmd[0] == "open" {
					return nil, nil // Start command succeeds
				}
				return nil, errors.New("unexpected command")
			},
		}
		// We need to inject a shorter retry delay/count for testing to avoid slow tests
		// For now, we'll assume the implementation allows configuring this or we accept a small delay.
		// Ideally, we'd inject a "Sleeper" interface, but for simplicity we might just rely on the loop.

		err := EnsureDockerReady(context.Background(), runner, config)
		if err != nil {
			t.Errorf("EnsureDockerReady failed: %v", err)
		}
		if checkCalls != 2 {
			t.Errorf("Expected 2 check calls, got %d", checkCalls)
		}
	})

	t.Run("Start fails", func(t *testing.T) {
		runner := &MockCommandExecutor{
			RunFunc: func(ctx context.Context, cmd []string) ([]byte, error) {
				return nil, errors.New("command failed")
			},
		}
		err := EnsureDockerReady(context.Background(), runner, config)
		if err == nil {
			t.Error("EnsureDockerReady succeeded, want error")
		}
	})
}
