package docker

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"

	"github.com/Cyclone1070/iav/internal/domain"
)

type lintDockerClient interface {
	RunContainerInfo(ctx context.Context, opts domain.RunOptions) (stdout string, stderr string, exitCode int, err error)
}

type LintStage struct {
	client     lintDockerClient
	lookPath   func(string) (string, error)
	hostRunner func(ctx context.Context, name string, arg string) (stdout string, stderr string, exitCode int, err error)
}

var defaultHostRunner = func(ctx context.Context, name string, arg string) (stdout string, stderr string, exitCode int, err error) {
	//nolint:gosec // G204: Subprocess command is parameterized by design
	cmd := exec.CommandContext(ctx, name, arg)
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	runErr := cmd.Run()
	if runErr != nil {
		var exitErr *exec.ExitError
		if errors.As(runErr, &exitErr) {
			return outBuf.String(), errBuf.String(), exitErr.ExitCode(), nil
		}
		return outBuf.String(), errBuf.String(), -1, runErr
	}
	return outBuf.String(), errBuf.String(), 0, nil
}

func NewLintStage(client lintDockerClient) *LintStage {
	return &LintStage{
		client:     client,
		lookPath:   exec.LookPath,
		hostRunner: defaultHostRunner,
	}
}

func (s *LintStage) Name() string {
	return "Hadolint"
}

func (s *LintStage) DependsOn() []string {
	return []string{}
}

func (s *LintStage) Run(ctx context.Context, ec *domain.ExecutionContext) (*domain.StageResult, error) {
	hadolintPath, err := s.lookPath("hadolint")
	if err == nil {
		filePath := filepath.Join(ec.WorkspaceRoot, "Dockerfile")
		stdout, stderr, exitCode, runErr := s.hostRunner(ctx, hadolintPath, filePath)
		if runErr != nil {
			return &domain.StageResult{
				StageName: s.Name(),
				Success:   false,
				Stdout:    stdout,
				Stderr:    runErr.Error(),
			}, runErr
		}

		success, stderr, runErr := processLintResult(exitCode, stderr)
		return &domain.StageResult{
			StageName: s.Name(),
			Success:   success,
			Stdout:    stdout,
			Stderr:    stderr,
		}, runErr
	}

	// The path to the Dockerfile is assumed to be inside ec.WorkspaceRoot.
	// We mount ec.WorkspaceRoot into /workspace in the hadolint container.
	binds := []string{
		ec.WorkspaceRoot + ":/workspace:ro",
	}

	opts := domain.RunOptions{
		Image: "hadolint/hadolint:latest-alpine",
		Cmd:   []string{"hadolint", "/workspace/Dockerfile"},
		Binds: binds,
	}

	stdout, stderr, exitCode, err := s.client.RunContainerInfo(ctx, opts)
	if err != nil {
		return &domain.StageResult{
			StageName: s.Name(),
			Success:   false,
			Stdout:    stdout,
			Stderr:    err.Error(),
		}, err
	}

	success, stderr, runErr := processLintResult(exitCode, stderr)
	return &domain.StageResult{
		StageName: s.Name(),
		Success:   success,
		Stdout:    stdout,
		Stderr:    stderr,
	}, runErr
}

func processLintResult(exitCode int, stderr string) (bool, string, error) {
	success := exitCode == 0
	var runErr error
	if !success {
		if stderr == "" {
			stderr = fmt.Sprintf("Hadolint failed with exit code %d", exitCode)
		}
		runErr = fmt.Errorf("hadolint lint errors: %s", stderr)
	}
	return success, stderr, runErr
}
