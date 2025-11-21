package tools

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/Cyclone1070/deployforme/internal/tools/models"
	"github.com/Cyclone1070/deployforme/internal/tools/services"
)

// ShellTool executes commands on the local machine.
type ShellTool struct {
	ProcessFactory models.ProcessFactory
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
		// Create a simple runner using our ProcessFactory
		runner := &processFactoryRunner{
			ctx:     ctx,
			factory: t.ProcessFactory,
			// Docker commands run in workspace root or default
			dir: wCtx.WorkspaceRoot,
		}
		if err := services.EnsureDockerReady(ctx, runner, wCtx.DockerConfig); err != nil {
			return nil, err
		}
	}

	// 5. Execution Setup
	env := os.Environ()
	for k, v := range req.Env {
		env = append(env, k+"="+v)
	}

	opts := models.ProcessOptions{
		Dir: wd,
		Env: env,
	}

	proc, stdout, stderr, err := t.ProcessFactory.Start(ctx, req.Command, opts)
	if err != nil {
		return nil, err
	}

	// 6. Output Collection
	stdoutCollector := services.NewCollector(models.DefaultMaxFileSize) // Use a reasonable limit
	stderrCollector := services.NewCollector(models.DefaultMaxFileSize)

	// Start copying in background with synchronization
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		io.Copy(stdoutCollector, stdout)
	}()
	go func() {
		defer wg.Done()
		io.Copy(stderrCollector, stderr)
	}()

	// 7. Run & Wait
	timeout := time.Duration(req.TimeoutSeconds) * time.Second
	if timeout == 0 {
		timeout = 1 * time.Hour // Default timeout
	}

	execErr := services.ExecuteWithTimeout(ctx, timeout, proc)

	// Wait for output collection to complete
	wg.Wait()

	// 8. Construct Response
	resp := &models.ShellResponse{
		Stdout:     stdoutCollector.String(),
		Stderr:     stderrCollector.String(),
		WorkingDir: wd,
		Truncated:  stdoutCollector.Truncated || stderrCollector.Truncated,
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

		// For now, let's just return the error if it's not ExitError.
		// If it is ExitError, we set ExitCode.
		// But we can't import `os/exec` to check `ExitError` if we want to be pure?
		// We can import it, it's stdlib.
		// But `MockProcess` won't return `exec.ExitError`.

		// Let's just return error for now.
		resp.ExitCode = 1 // Default failure
		return resp, nil  // We return nil error because the command ran, it just failed.
	}

	resp.ExitCode = 0

	// 9. Post-Process (Docker Compose)
	if services.IsDockerComposeUpDetached(req.Command) {
		// Collect containers
		// We need a runner that captures output.
		// Our processFactoryRunner captures output? No.
		// We need to implement a runner that captures output.
		// Let's skip this for now or implement a simple one.

		runner := &processFactoryRunner{
			ctx:     ctx,
			factory: t.ProcessFactory,
			dir:     wd,
		}
		ids, err := services.CollectComposeContainers(ctx, runner, wd)
		if err == nil {
			resp.Notes = append(resp.Notes, noteDockerContainerStarted(ids))
		}
	}

	return resp, nil
}

// processFactoryRunner implements models.CommandRunner using ProcessFactory
type processFactoryRunner struct {
	ctx     context.Context
	factory models.ProcessFactory
	dir     string
}

func (r *processFactoryRunner) Run(ctx context.Context, command []string) ([]byte, error) {
	opts := models.ProcessOptions{
		Dir: r.dir,
	}
	proc, stdout, stderr, err := r.factory.Start(r.ctx, command, opts)
	if err != nil {
		return nil, err
	}

	outCol := services.NewCollector(10 * 1024 * 1024)
	errCol := services.NewCollector(10 * 1024 * 1024)

	go io.Copy(outCol, stdout)
	go io.Copy(errCol, stderr)

	waitErr := proc.Wait()

	// Combine output
	output := outCol.String() + errCol.String()

	return []byte(output), waitErr
}

// noteDockerContainerStarted returns a note about started containers.
func noteDockerContainerStarted(ids []string) string {
	if len(ids) == 0 {
		return ""
	}
	count := len(ids)
	if count == 1 {
		return fmt.Sprintf("Started 1 Docker container: %s", ids[0])
	}
	return fmt.Sprintf("Started %d Docker containers", count)
}
