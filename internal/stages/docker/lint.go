package docker

import (
	"context"
	"fmt"

	"github.com/Cyclone1070/iav/internal/domain"
)

type lintDockerClient interface {
	RunContainerInfo(ctx context.Context, opts domain.RunOptions) (stdout string, stderr string, exitCode int, err error)
}

type LintStage struct {
	client lintDockerClient
}

func NewLintStage(client lintDockerClient) *LintStage {
	return &LintStage{client: client}
}

func (s *LintStage) Name() string {
	return "Hadolint"
}

func (s *LintStage) DependsOn() []string {
	return []string{}
}

func (s *LintStage) Run(ctx context.Context, ec *domain.ExecutionContext) (*domain.StageResult, error) {
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

	success := exitCode == 0
	var runErr error
	if !success {
		if stderr == "" {
			stderr = fmt.Sprintf("Hadolint failed with exit code %d", exitCode)
		}
		runErr = fmt.Errorf("hadolint lint errors: %s", stderr)
	}

	return &domain.StageResult{
		StageName: s.Name(),
		Success:   success,
		Stdout:    stdout,
		Stderr:    stderr,
	}, runErr
}
