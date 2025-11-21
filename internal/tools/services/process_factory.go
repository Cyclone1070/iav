package services

import (
	"context"
	"io"
	"os"
	"os/exec"

	"github.com/Cyclone1070/deployforme/internal/tools/models"
)

// OSProcess implements models.Process for real OS processes.
type OSProcess struct {
	Cmd *exec.Cmd
}

func (p *OSProcess) Wait() error {
	return p.Cmd.Wait()
}

func (p *OSProcess) Kill() error {
	if p.Cmd.Process != nil {
		return p.Cmd.Process.Kill()
	}
	return nil
}

func (p *OSProcess) Signal(sig os.Signal) error {
	if p.Cmd.Process != nil {
		return p.Cmd.Process.Signal(sig)
	}
	return nil
}

// OSProcessFactory implements models.ProcessFactory using os/exec.
type OSProcessFactory struct{}

func (f *OSProcessFactory) Start(ctx context.Context, command []string, opts models.ProcessOptions) (models.Process, io.Reader, io.Reader, error) {
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
