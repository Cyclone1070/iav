package services

import (
	"context"
	"io"
	"os"
	"sync"
	"time"

	"github.com/Cyclone1070/deployforme/internal/tools/models"
)

// ExecuteWithTimeout runs a process with a timeout.
// It assumes the process has already been started.
func ExecuteWithTimeout(ctx context.Context, timeout time.Duration, proc models.Process) error {
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
			return models.ErrShellTimeout
		case <-time.After(2 * time.Second):
			_ = proc.Kill()
			return models.ErrShellTimeout
		}
	}
}

// CollectProcessOutput reads stdout and stderr concurrently and returns them as strings.
// It enforces a maximum size limit for the collected output.
func CollectProcessOutput(stdout, stderr io.Reader, maxBytes int) (string, string, bool, error) {
	stdoutCollector := NewCollector(maxBytes)
	stderrCollector := NewCollector(maxBytes)

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
