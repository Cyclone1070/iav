package shell

import (
	"context"
	"io"
	"os"
	"sync"
	"time"

	toolserrors "github.com/Cyclone1070/iav/internal/tools/errors"
)

// ExecuteWithTimeout runs a process with a timeout, handling graceful shutdown.
// It waits for the process to complete, or kills it if the timeout is reached or context is cancelled.
// Returns ErrShellTimeout if the timeout is reached.
func ExecuteWithTimeout(ctx context.Context, timeout time.Duration, gracefulShutdownMs int, proc Process) error {
	done := make(chan error, 1)
	go func() {
		done <- proc.Wait()
	}()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		// Context cancelled (e.g. user cancellation)
		_ = proc.Kill()
		return ctx.Err()
	case <-time.After(timeout):
		// Timeout reached
		// Try graceful shutdown first
		_ = proc.Signal(os.Interrupt) // SIGINT/SIGTERM equivalent

		// Wait a bit for graceful shutdown
		select {
		case <-done:
			return toolserrors.ErrShellTimeout
		case <-time.After(time.Duration(gracefulShutdownMs) * time.Millisecond):
			_ = proc.Kill()
			return toolserrors.ErrShellTimeout
		}
	}
}

// CollectProcessOutput reads stdout and stderr concurrently and returns them as strings.
// It enforces a maximum size limit for the collected output and detects binary content.
// Returns the stdout string, stderr string, whether output was truncated, and any error.
func CollectProcessOutput(stdout, stderr io.Reader, maxBytes int, sampleSize int) (string, string, bool, error) {
	stdoutCollector := NewCollector(maxBytes, sampleSize)
	stderrCollector := NewCollector(maxBytes, sampleSize)

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

	wg.Wait()

	truncated := stdoutCollector.Truncated || stderrCollector.Truncated
	return stdoutCollector.String(), stderrCollector.String(), truncated, nil
}

// GetExitCode extracts the exit code from an error returned by a process.
// Returns 0 if err is nil, the exit code if it's an ExitError, or -1 for unknown error types.
func GetExitCode(err error) int {
	if err == nil {
		return 0
	}

	// Check for exec.ExitError (real processes)
	type exitCoder interface {
		ExitCode() int
	}
	if ec, ok := err.(exitCoder); ok {
		return ec.ExitCode()
	}

	// Unknown error type
	return -1
}
