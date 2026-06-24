package workflow

import (
	"context"

	"github.com/Cyclone1070/iav/internal/domain"
)

type DockerLinter struct {
	pipeline Pipeline
	stage    Stage
}

func NewDockerLinter(p Pipeline, s Stage) *DockerLinter {
	return &DockerLinter{
		pipeline: p,
		stage:    s,
	}
}

func (l *DockerLinter) Run(ctx context.Context, targetPath string) error {
	l.pipeline.Add(l.stage)

	ec := &domain.ExecutionContext{
		WorkspaceRoot: targetPath,
	}

	_, err := l.pipeline.Execute(ctx, ec)
	return err
}
