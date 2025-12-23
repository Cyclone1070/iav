package shell

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Cyclone1070/iav/internal/tool/service/executor"
)

// mockCommandExecutorForDocker is a local mock for testing docker functions
type mockCommandExecutorForDocker struct {
	runFunc func(ctx context.Context, cmd []string, dir string, env []string) (*executor.Result, error)
}

func (m *mockCommandExecutorForDocker) Run(ctx context.Context, cmd []string, dir string, env []string) (*executor.Result, error) {
	if m.runFunc != nil {
		return m.runFunc(ctx, cmd, dir, env)
	}
	return nil, errors.New("not implemented")
}

func (m *mockCommandExecutorForDocker) RunWithTimeout(ctx context.Context, cmd []string, dir string, env []string, timeout time.Duration) (*executor.Result, error) {
	return nil, errors.New("not implemented")
}

func TestEnsureDockerReady(t *testing.T) {
	config := DockerConfig{
		CheckCommand: []string{"docker", "info"},
		StartCommand: []string{"open", "-a", "Docker"},
	}

	t.Run("Success immediately", func(t *testing.T) {
		runner := &mockCommandExecutorForDocker{}
		runner.runFunc = func(ctx context.Context, cmd []string, dir string, env []string) (*executor.Result, error) {
			if cmd[0] == "docker" && cmd[1] == "info" {
				return &executor.Result{Stdout: "", ExitCode: 0}, nil
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
		runner.runFunc = func(ctx context.Context, cmd []string, dir string, env []string) (*executor.Result, error) {
			if cmd[0] == "docker" && cmd[1] == "info" {
				checkCalls++
				if checkCalls == 1 {
					return &executor.Result{Stdout: "", ExitCode: 1}, errors.New("docker not running")
				}
				return &executor.Result{Stdout: "", ExitCode: 0}, nil // Success on second call
			}
			if cmd[0] == "open" {
				return &executor.Result{Stdout: "", ExitCode: 0}, nil // Start command succeeds
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
		runner.runFunc = func(ctx context.Context, cmd []string, dir string, env []string) (*executor.Result, error) {
			return &executor.Result{Stdout: "", ExitCode: 1}, errors.New("command failed")
		}
		err := EnsureDockerReady(context.Background(), runner, config, 5, 10)
		if err == nil {
			t.Error("EnsureDockerReady succeeded, want error")
		}
	})
}
