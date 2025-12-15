package shell

import (
	"context"
	"errors"
	"io"
	"testing"
)

// mockCommandExecutorForDocker is a local mock for testing docker functions
type mockCommandExecutorForDocker struct {
	runFunc func(ctx context.Context, cmd []string) ([]byte, error)
}

func (m *mockCommandExecutorForDocker) Run(ctx context.Context, cmd []string) ([]byte, error) {
	if m.runFunc != nil {
		return m.runFunc(ctx, cmd)
	}
	return nil, errors.New("not implemented")
}

func (m *mockCommandExecutorForDocker) Start(ctx context.Context, cmd []string, opts ProcessOptions) (Process, io.Reader, io.Reader, error) {
	return nil, nil, nil, errors.New("not implemented")
}

func TestEnsureDockerReady(t *testing.T) {
	config := DockerConfig{
		CheckCommand: []string{"docker", "info"},
		StartCommand: []string{"open", "-a", "Docker"},
	}

	t.Run("Success immediately", func(t *testing.T) {
		runner := &mockCommandExecutorForDocker{}
		runner.runFunc = func(ctx context.Context, cmd []string) ([]byte, error) {
			if cmd[0] == "docker" && cmd[1] == "info" {
				return nil, nil
			}
			return nil, errors.New("unexpected command")
		}
		err := EnsureDockerReady(context.Background(), runner, config, 5, 10)
		if err != nil {
			t.Errorf("EnsureDockerReady failed: %v", err)
		}
	})

	t.Run("Start required and succeeds", func(t *testing.T) {
		checkCalls := 0
		runner := &mockCommandExecutorForDocker{}
		runner.runFunc = func(ctx context.Context, cmd []string) ([]byte, error) {
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
		}

		err := EnsureDockerReady(context.Background(), runner, config, 5, 10)
		if err != nil {
			t.Errorf("EnsureDockerReady failed: %v", err)
		}
		if checkCalls != 2 {
			t.Errorf("Expected 2 check calls, got %d", checkCalls)
		}
	})

	t.Run("Start fails", func(t *testing.T) {
		runner := &mockCommandExecutorForDocker{}
		runner.runFunc = func(ctx context.Context, cmd []string) ([]byte, error) {
			return nil, errors.New("command failed")
		}
		err := EnsureDockerReady(context.Background(), runner, config, 5, 10)
		if err == nil {
			t.Error("EnsureDockerReady succeeded, want error")
		}
	})
}
