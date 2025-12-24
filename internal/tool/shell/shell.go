package shell

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/Cyclone1070/iav/internal/config"
	"github.com/Cyclone1070/iav/internal/tool/service/executor"
)

// ShellTool executes commands on the local machine.
type ShellTool struct {
	envFileOps      envFileReader
	commandExecutor commandExecutor
	config          *config.Config
	dockerConfig    DockerConfig
	pathResolver    pathResolver
}

// NewShellTool creates a new ShellTool with injected dependencies.
func NewShellTool(
	envFileOps envFileReader,
	commandExecutor commandExecutor,
	cfg *config.Config,
	dockerConfig DockerConfig,
	pathResolver pathResolver,
) *ShellTool {
	if commandExecutor == nil {
		panic("commandExecutor is required")
	}
	if cfg == nil {
		panic("cfg is required")
	}
	if pathResolver == nil {
		panic("pathResolver is required")
	}
	return &ShellTool{
		envFileOps:      envFileOps,
		commandExecutor: commandExecutor,
		config:          cfg,
		dockerConfig:    dockerConfig,
		pathResolver:    pathResolver,
	}
}

// Run executes a shell command with Docker readiness checks,
// environment variable support, timeout handling, and output collection.
// NOTE: This tool does NOT enforce policy - the caller is responsible for policy checks.
func (t *ShellTool) Run(ctx context.Context, req *ShellRequest) (*ShellResponse, error) {
	if err := req.Validate(t.config); err != nil {
		return nil, err
	}

	workingDir := req.WorkingDir
	if workingDir == "" {
		workingDir = "."
	}

	wdAbs, err := t.pathResolver.Abs(workingDir)
	if err != nil {
		return nil, err
	}
	wdRel, err := t.pathResolver.Rel(wdAbs)
	if err != nil {
		return nil, err
	}

	if IsDockerCommand(req.Command) {
		retryAttempts := t.config.Tools.DockerRetryAttempts
		retryIntervalMs := t.config.Tools.DockerRetryIntervalMs

		if err := EnsureDockerReady(ctx, t.commandExecutor, t.dockerConfig, retryAttempts, retryIntervalMs); err != nil {
			return nil, err
		}
	}

	env := os.Environ()

	for _, envFile := range req.EnvFiles {
		envFilePath, err := t.pathResolver.Abs(envFile)
		if err != nil {
			return nil, err
		}

		envVars, err := ParseEnvFile(t.envFileOps, envFilePath)
		if err != nil {
			return nil, err
		}

		// EnvFiles override system env
		for k, v := range envVars {
			env = append(env, k+"="+v)
		}
	}

	// Request.Env overrides everything
	for k, v := range req.Env {
		env = append(env, k+"="+v)
	}

	timeout := time.Duration(req.TimeoutSeconds) * time.Second

	result, execErr := t.commandExecutor.RunWithTimeout(ctx, req.Command, wdAbs, env, timeout)
	if result == nil {
		result = &executor.Result{ExitCode: -1}
	}

	resp := &ShellResponse{
		Stdout:     result.Stdout,
		Stderr:     result.Stderr,
		WorkingDir: wdRel,
		ExitCode:   result.ExitCode,
		Truncated:  result.Truncated,
	}

	if execErr != nil {
		if errors.Is(execErr, executor.ErrTimeout) {
			return resp, execErr
		}
		// Check for context cancellation
		if errors.Is(execErr, context.Canceled) || errors.Is(execErr, context.DeadlineExceeded) {
			return resp, execErr
		}
		// Command ran but failed - we already have the exit code in resp
		return resp, nil
	}

	if IsDockerComposeUpDetached(req.Command) {
		ids, err := CollectComposeContainers(ctx, t.commandExecutor, wdAbs)
		if err == nil {
			resp.Notes = append(resp.Notes, FormatContainerStartedNote(ids))
		} else {
			resp.Notes = append(resp.Notes, fmt.Sprintf("Warning: Could not list started containers: %v", err))
		}
	}

	return resp, nil
}
