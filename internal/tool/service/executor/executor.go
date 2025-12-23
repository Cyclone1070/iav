package executor

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/Cyclone1070/iav/internal/config"
)

// Result represents the outcome of a command execution.
type Result struct {
	Stdout    string
	Stderr    string
	ExitCode  int
	Truncated bool
}

// OSCommandExecutor implements command execution using os/exec for real system commands.
type OSCommandExecutor struct {
	config *config.Config
}

// NewOSCommandExecutor creates a new OSCommandExecutor with injected config.
func NewOSCommandExecutor(cfg *config.Config) *OSCommandExecutor {
	if cfg == nil {
		panic("cfg is required")
	}
	return &OSCommandExecutor{config: cfg}
}

// Run executes a command and returns the result. It buffers output internally.
func (f *OSCommandExecutor) Run(ctx context.Context, command []string, dir string, env []string) (*Result, error) {
	if len(command) == 0 {
		return nil, os.ErrInvalid
	}

	cmd := exec.CommandContext(ctx, command[0], command[1:]...)
	cmd.Dir = dir
	cmd.Env = env
	cmd.Stdin = nil

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, &CommandError{Cmd: command[0], Cause: err, Stage: "start"}
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, &CommandError{Cmd: command[0], Cause: err, Stage: "start"}
	}

	if err := cmd.Start(); err != nil {
		return nil, &CommandError{Cmd: command[0], Cause: err, Stage: "start"}
	}

	stdoutStr, stderrStr, truncated := f.collectOutput(stdoutPipe, stderrPipe)

	err = cmd.Wait()
	exitCode := 0
	if err != nil {
		exitCode = f.getExitCode(err)
	}

	return &Result{
		Stdout:    stdoutStr,
		Stderr:    stderrStr,
		ExitCode:  exitCode,
		Truncated: truncated,
	}, err
}

// RunWithTimeout executes a command with a timeout and graceful shutdown.
func (f *OSCommandExecutor) RunWithTimeout(ctx context.Context, command []string, dir string, env []string, timeout time.Duration) (*Result, error) {
	if len(command) == 0 {
		return nil, os.ErrInvalid
	}

	// We don't use CommandContext's timeout here because we want to handle graceful shutdown
	cmd := exec.Command(command[0], command[1:]...)
	cmd.Dir = dir
	cmd.Env = env
	cmd.Stdin = nil

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, &CommandError{Cmd: command[0], Cause: err, Stage: "start"}
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, &CommandError{Cmd: command[0], Cause: err, Stage: "start"}
	}

	if err := cmd.Start(); err != nil {
		return nil, &CommandError{Cmd: command[0], Cause: err, Stage: "start"}
	}

	// Start output collection concurrently so it doesn't block the timeout select
	var stdoutStr, stderrStr string
	var truncated bool
	collectDone := make(chan struct{})
	go func() {
		stdoutStr, stderrStr, truncated = f.collectOutput(stdoutPipe, stderrPipe)
		close(collectDone)
	}()

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	var execErr error
	select {
	case err := <-done:
		execErr = err
	case <-ctx.Done():
		_ = cmd.Process.Kill()
		execErr = ctx.Err()
	case <-time.After(timeout):
		// Try graceful shutdown
		_ = cmd.Process.Signal(os.Interrupt)
		select {
		case <-done:
			execErr = ErrTimeout
		case <-time.After(time.Duration(f.config.Tools.DockerGracefulShutdownMs) * time.Millisecond):
			_ = cmd.Process.Kill()
			execErr = ErrTimeout
		}
	}

	// Wait for output collection to finish (it should when cmd.Wait/Kill happens)
	<-collectDone

	exitCode := 0
	if execErr != nil {
		exitCode = f.getExitCode(execErr)
		if errors.Is(execErr, ErrTimeout) {
			exitCode = -1
		}
	}

	return &Result{
		Stdout:    stdoutStr,
		Stderr:    stderrStr,
		ExitCode:  exitCode,
		Truncated: truncated,
	}, execErr
}

func (f *OSCommandExecutor) collectOutput(stdout, stderr io.Reader) (string, string, bool) {
	maxBytes := int(f.config.Tools.DefaultMaxCommandOutputSize)

	stdoutCollector := newCollector(maxBytes, 8000)
	stderrCollector := newCollector(maxBytes, 8000)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		_, _ = io.Copy(stdoutCollector, stdout)
	}()

	go func() {
		defer wg.Done()
		_, _ = io.Copy(stderrCollector, stderr)
	}()

	wg.Wait()

	truncated := stdoutCollector.Truncated() || stderrCollector.Truncated()
	return stdoutCollector.String(), stderrCollector.String(), truncated
}

func (f *OSCommandExecutor) getExitCode(err error) int {
	if err == nil {
		return 0
	}
	type exitCoder interface {
		ExitCode() int
	}
	if ec, ok := err.(exitCoder); ok {
		return ec.ExitCode()
	}
	return -1
}
