package workflow

import (
	"context"
	"fmt"
	"io"

	"github.com/Cyclone1070/iav/internal/domain"
)

type DockerLinter struct {
	pipeline Pipeline
	stage    Stage
	out      io.Writer
}

func NewDockerLinter(p Pipeline, s Stage, out io.Writer) *DockerLinter {
	return &DockerLinter{
		pipeline: p,
		stage:    s,
		out:      out,
	}
}

func (l *DockerLinter) Run(ctx context.Context, targetPath string) error {
	_, _ = fmt.Fprintln(l.out, "Linting Dockerfile")
	l.pipeline.Add(l.stage)

	ec := &domain.ExecutionContext{
		WorkspaceRoot: targetPath,
	}

	_, err := l.pipeline.Execute(ctx, ec)
	return err
}
