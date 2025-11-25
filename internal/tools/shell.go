package tools

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/Cyclone1070/deployforme/internal/tools/models"
	"github.com/Cyclone1070/deployforme/internal/tools/services"
)

// ShellTool executes commands on the local machine.
type ShellTool struct {
	CommandExecutor models.CommandExecutor
}

// Run executes a shell command with Docker readiness checks,
// environment variable support, timeout handling, and output collection.
// NOTE: This tool does NOT enforce policy - the caller is responsible for policy checks.
func (t *ShellTool) Run(ctx context.Context, wCtx *models.WorkspaceContext, req models.ShellRequest) (*models.ShellResponse, error) {
	if len(req.Command) == 0 {
		return nil, fmt.Errorf("command cannot be empty")
	}

	workingDir := req.WorkingDir
	if workingDir == "" {
		workingDir = "."
	}

	wd, _, err := services.Resolve(wCtx, workingDir)
	if err != nil {
		return nil, models.ErrShellWorkingDirOutsideWorkspace
	}

	// Policy check removed - caller is responsible for enforcement

	if services.IsDockerCommand(req.Command) {
		if err := services.EnsureDockerReady(ctx, t.CommandExecutor, wCtx.DockerConfig); err != nil {
			return nil, err
		}
	}

	env := os.Environ()

	for _, envFile := range req.EnvFiles {
		envFilePath, _, err := services.Resolve(wCtx, envFile)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve env file %s: %w", envFile, err)
		}

		envVars, err := services.ParseEnvFile(wCtx.FS, envFilePath)
		if err != nil {
			return nil, fmt.Errorf("failed to parse env file %s: %w", envFile, err)
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

	opts := models.ProcessOptions{
		Dir: wd,
		Env: env,
	}

	proc, stdout, stderr, err := t.CommandExecutor.Start(ctx, req.Command, opts)
	if err != nil {
		return nil, err
	}

	stdoutStr, stderrStr, truncated, _ := services.CollectProcessOutput(stdout, stderr, int(models.DefaultMaxCommandOutputSize))

	timeout := time.Duration(req.TimeoutSeconds) * time.Second
	if timeout == 0 {
		timeout = 1 * time.Hour // Default timeout
	}

	execErr := services.ExecuteWithTimeout(ctx, timeout, proc)

	resp := &models.ShellResponse{
		Stdout:     stdoutStr,
		Stderr:     stderrStr,
		WorkingDir: wd,
		Truncated:  truncated,
	}

	if execErr != nil {
		if execErr == models.ErrShellTimeout {
			resp.ExitCode = -1
			return resp, models.ErrShellTimeout
		}
		// Command ran but failed - extract exit code and return success
		resp.ExitCode = services.GetExitCode(execErr)
		return resp, nil
	}

	resp.ExitCode = 0

	if services.IsDockerComposeUpDetached(req.Command) {
		ids, err := services.CollectComposeContainers(ctx, t.CommandExecutor, wd)
		if err == nil {
			resp.Notes = append(resp.Notes, services.FormatContainerStartedNote(ids))
		}
	}

	return resp, nil
}
