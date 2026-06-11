package workflow

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/Cyclone1070/iav/internal/domain"
)

type runComposeFileSystem interface {
	Abs(path string) (string, error)
	EvalSymlinks(path string) (string, error)
}

type runComposeDockerClient interface {
	ComposeDown(ctx context.Context, composeFiles []string, runID string) error
}

type Stage interface {
	Name() string
	Run(ctx context.Context, ec *domain.ExecutionContext) (*domain.StageResult, error)
	DependsOn() []string
}

type Pipeline interface {
	Add(s Stage)
	Execute(ctx context.Context, ec *domain.ExecutionContext) (*domain.TestExecutionResult, error)
}

type RunCompose struct {
	fs              runComposeFileSystem
	docker          runComposeDockerClient
	stages          []Stage
	pipelineFactory func() Pipeline
}

func NewRunCompose(fs runComposeFileSystem, docker runComposeDockerClient, stages []Stage, factory func() Pipeline) *RunCompose {
	return &RunCompose{
		fs:              fs,
		docker:          docker,
		stages:          stages,
		pipelineFactory: factory,
	}
}

func (w *RunCompose) Run(ctx context.Context, testComposePath string, runID string, timeoutMs int) (*domain.TestExecutionResult, error) {
	testComposePath, err := w.fs.Abs(testComposePath)
	if err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}
	testComposePath, err = w.fs.EvalSymlinks(testComposePath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve symlinks: %w", err)
	}

	parent := filepath.Dir(testComposePath)
	var composeFiles []string

	// Auto-detect base compose file
	baseYml := filepath.Join(parent, "docker-compose.yml")
	if _, statErr := w.fs.EvalSymlinks(baseYml); statErr == nil {
		composeFiles = append(composeFiles, baseYml)
	} else {
		baseYaml := filepath.Join(parent, "docker-compose.yaml")
		if _, statErr := w.fs.EvalSymlinks(baseYaml); statErr == nil {
			composeFiles = append(composeFiles, baseYaml)
		}
	}

	composeFiles = append(composeFiles, testComposePath)

	ec := &domain.ExecutionContext{
		WorkspaceRoot:        parent,
		ComposeFiles:         composeFiles,
		TimeoutMs:            timeoutMs,
		RunID:                runID,
		EnvironmentVariables: make(map[string]string),
	}

	p := w.pipelineFactory()
	for _, s := range w.stages {
		p.Add(s)
	}

	result, pipelineErr := p.Execute(ctx, ec)

	// Cleanup
	cleanupCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_ = w.docker.ComposeDown(cleanupCtx, composeFiles, runID)

	return result, pipelineErr
}
