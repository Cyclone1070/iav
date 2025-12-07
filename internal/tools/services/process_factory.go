package services

import (
	"context"
	"io"
	"os"
	"os/exec"

	"github.com/Cyclone1070/iav/internal/tools/models"
)

// OSProcess wraps an os/exec.Cmd to implement the models.Process interface.
type OSProcess struct {
	Cmd *exec.Cmd
}

// Wait waits for the process to complete and returns any error.
func (p *OSProcess) Wait() error {
	return p.Cmd.Wait()
}

// Kill forcefully terminates the process.
func (p *OSProcess) Kill() error {
	if p.Cmd.Process != nil {
		return p.Cmd.Process.Kill()
	}
	return nil
}

// Signal sends a signal to the process.
func (p *OSProcess) Signal(sig os.Signal) error {
	if p.Cmd.Process != nil {
		return p.Cmd.Process.Signal(sig)
	}
	return nil
}

// OSCommandExecutor implements models.CommandExecutor using os/exec for real system commands.
type OSCommandExecutor struct {
}

// Run executes a command and returns the combined output (stdout + stderr).
// This is a convenience method for simple command execution.
func (f *OSCommandExecutor) Run(ctx context.Context, command []string) ([]byte, error) {
	if len(command) == 0 {
		return nil, os.ErrInvalid
	}

	cmd := exec.CommandContext(ctx, command[0], command[1:]...)
	cmd.Stdin = nil

	// CombinedOutput runs the command and returns stdout+stderr
	return cmd.CombinedOutput()
}

// Start starts a process and returns control immediately for streaming output or process management.
// Returns the process handle and separate readers for stdout and stderr.
func (f *OSCommandExecutor) Start(ctx context.Context, command []string, opts models.ProcessOptions) (models.Process, io.Reader, io.Reader, error) {
	if len(command) == 0 {
		return nil, nil, nil, os.ErrInvalid
	}

	cmd := exec.CommandContext(ctx, command[0], command[1:]...)
	cmd.Dir = opts.Dir
	cmd.Env = opts.Env

	// Explicitly close stdin to prevent interactive hangs
	cmd.Stdin = nil

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, nil, err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, nil, nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, nil, nil, err
	}

	return &OSProcess{Cmd: cmd}, stdout, stderr, nil
}
