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

// Run executes a shell command.
func (t *ShellTool) Run(ctx context.Context, wCtx *models.WorkspaceContext, req models.ShellRequest) (*models.ShellResponse, error) {
	// 1. Input Validation
	if len(req.Command) == 0 {
		return nil, fmt.Errorf("command cannot be empty")
	}

	// 2. Resolve Working Directory
	// Working directory: default to workspace root if not specified
	workingDir := req.WorkingDir
	if workingDir == "" {
		workingDir = "."
	}

	wd, _, err := services.Resolve(wCtx, workingDir)
	if err != nil {
		return nil, models.ErrShellWorkingDirOutsideWorkspace
	}

	// 3. Policy Check
	if err := services.EvaluatePolicy(wCtx.CommandPolicy, req.Command); err != nil {
		return nil, err
	}

	// 4. Docker Readiness
	if services.IsDockerCommand(req.Command) {
		if err := services.EnsureDockerReady(ctx, t.CommandExecutor, wCtx.DockerConfig); err != nil {
			return nil, err
		}
	}

	// 5. Execution Setup
	// We get the current environment variables and append any custom environment variables from the request.
	// This allows the command to have access to both the system's environment and any custom environment variables.
	env := os.Environ()

	// Load environment variables from .env files (if specified)
	for _, envFile := range req.EnvFiles {
		// Resolve the env file path relative to workspace root
		envFilePath, _, err := services.Resolve(wCtx, envFile)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve env file %s: %w", envFile, err)
		}

		// Parse the env file
		envVars, err := services.ParseEnvFile(envFilePath)
		if err != nil {
			return nil, fmt.Errorf("failed to parse env file %s: %w", envFile, err)
		}

		// Merge into environment (EnvFiles override system env)
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

	// 6. Output Collection
	stdoutStr, stderrStr, truncated, _ := services.CollectProcessOutput(stdout, stderr, int(models.DefaultMaxCommandOutputSize))

	// 7. Run & Wait
	timeout := time.Duration(req.TimeoutSeconds) * time.Second
	if timeout == 0 {
		timeout = 1 * time.Hour // Default timeout
	}

	execErr := services.ExecuteWithTimeout(ctx, timeout, proc)

	// 8. Construct Response
	resp := &models.ShellResponse{
		Stdout:     stdoutStr,
		Stderr:     stderrStr,
		WorkingDir: wd,
		Truncated:  truncated,
	}

	if execErr != nil {
		if execErr == models.ErrShellTimeout {
			resp.ExitCode = -1 // Timeout
			// We still return the response with captured output
			return resp, models.ErrShellTimeout
		}
		// Other execution errors (e.g. non-zero exit code)
		// Wait, ExecuteWithTimeout returns error from Wait().
		// If Wait() returns ExitError, we should parse it.
		// But Process interface Wait() returns error.
		// We need to extract ExitCode.
		// Since we are using an interface, we can't easily cast to exec.ExitError without leaking impl.
		// However, for now, let's assume non-zero exit is returned as error.
		// We should ideally parse it.
		// But `services.ExecuteWithTimeout` returns the error from `proc.Wait()`.
		// If we use `os/exec`, `Wait()` returns `*exec.ExitError`.
		// We can try to cast it here?
		// Or `Process` interface should have `ExitCode() int`?
		// That would be better.

		// Use the helper to extract the specific exit code (e.g. 127, 2, etc.)
		resp.ExitCode = services.GetExitCode(execErr)
		return resp, nil // We return nil error because the command ran, it just failed.
	}

	resp.ExitCode = 0

	// 9. Post-Process (Docker Compose)
	if services.IsDockerComposeUpDetached(req.Command) {
		// Collect containers
		// We need a runner that captures output.
		// Our processFactoryRunner captures output? No.
		// We need to implement a runner that captures output.
		// Let's skip this for now or implement a simple one.

		ids, err := services.CollectComposeContainers(ctx, t.CommandExecutor, wd)
		if err == nil {
			resp.Notes = append(resp.Notes, services.FormatContainerStartedNote(ids))
		}
	}

	return resp, nil
}
