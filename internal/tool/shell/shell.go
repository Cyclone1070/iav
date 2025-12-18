package shell

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/Cyclone1070/iav/internal/config"
)

// ShellTool executes commands on the local machine.
type ShellTool struct {
	fs              fileSystem
	commandExecutor commandExecutor
	config          *config.Config
	dockerConfig    DockerConfig
	workspaceRoot   string
}

// NewShellTool creates a new ShellTool with injected dependencies.
func NewShellTool(
	fs fileSystem,
	commandExecutor commandExecutor,
	cfg *config.Config,
	dockerConfig DockerConfig,
	workspaceRoot string,
) *ShellTool {
	return &ShellTool{
		fs:              fs,
		commandExecutor: commandExecutor,
		config:          cfg,
		dockerConfig:    dockerConfig,
		workspaceRoot:   workspaceRoot,
	}
}

// Run executes a shell command with Docker readiness checks,
// environment variable support, timeout handling, and output collection.
// NOTE: This tool does NOT enforce policy - the caller is responsible for policy checks.
func (t *ShellTool) Run(ctx context.Context, req *ShellRequest) (*ShellResponse, error) {
	// Runtime Validation
	wd := req.WorkingDirAbs()

	// Policy check removed - caller is responsible for enforcement

	if IsDockerCommand(req.Command()) {
		retryAttempts := t.config.Tools.DockerRetryAttempts
		retryIntervalMs := t.config.Tools.DockerRetryIntervalMs

		if err := EnsureDockerReady(ctx, t.commandExecutor, t.dockerConfig, retryAttempts, retryIntervalMs); err != nil {
			return nil, err
		}
	}

	env := os.Environ()

	for _, envFilePath := range req.EnvFilesAbsPaths() {
		envVars, err := ParseEnvFile(t.fs, envFilePath)
		if err != nil {
			return nil, err
		}

		// EnvFiles override system env
		for k, v := range envVars {
			env = append(env, k+"="+v)
		}
	}

	// Request.Env overrides everything
	for k, v := range req.Env() {
		env = append(env, k+"="+v)
	}

	opts := ProcessOptions{
		Dir: wd,
		Env: env,
	}

	proc, stdout, stderr, err := t.commandExecutor.Start(ctx, req.Command(), opts)
	if err != nil {
		return nil, err
	}

	// Use configured binary detection sample size
	sampleSize := t.config.Tools.BinaryDetectionSampleSize
	// Use configured max output size
	maxOutputSize := t.config.Tools.DefaultMaxCommandOutputSize

	stdoutStr, stderrStr, truncated, _ := CollectProcessOutput(stdout, stderr, int(maxOutputSize), sampleSize)

	timeout := time.Duration(req.TimeoutSeconds()) * time.Second
	if timeout == 0 {
		timeout = time.Duration(t.config.Tools.DefaultShellTimeout) * time.Second
	}

	// Use configured graceful shutdown
	gracefulShutdownMs := t.config.Tools.DockerGracefulShutdownMs

	execErr := ExecuteWithTimeout(ctx, req.Command(), timeout, gracefulShutdownMs, proc)

	resp := &ShellResponse{
		Stdout:     stdoutStr,
		Stderr:     stderrStr,
		WorkingDir: wd,
		Truncated:  truncated,
	}

	if execErr != nil {
		var t interface{ Timeout() bool }
		if errors.As(execErr, &t) && t.Timeout() {
			resp.ExitCode = -1
			return resp, execErr
		}
		// Check for context cancellation
		if errors.Is(execErr, context.Canceled) || errors.Is(execErr, context.DeadlineExceeded) {
			resp.ExitCode = -1
			return resp, execErr
		}
		// Command ran but failed - extract exit code and return success
		resp.ExitCode = GetExitCode(execErr)
		return resp, nil
	}

	resp.ExitCode = 0

	if IsDockerComposeUpDetached(req.Command()) {
		ids, err := CollectComposeContainers(ctx, t.commandExecutor, wd)
		if err == nil {
			resp.Notes = append(resp.Notes, FormatContainerStartedNote(ids))
		} else {
			resp.Notes = append(resp.Notes, fmt.Sprintf("Warning: Could not list started containers: %v", err))
		}
	}

	return resp, nil
}
