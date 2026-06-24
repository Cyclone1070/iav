package workflow

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Cyclone1070/iav/internal/domain"
)

type mockLinterPipeline struct {
	added   []Stage
	execute func(ctx context.Context, ec *domain.ExecutionContext) (*domain.TestExecutionResult, error)
}

func (m *mockLinterPipeline) Add(s Stage) {
	m.added = append(m.added, s)
}

func (m *mockLinterPipeline) Execute(ctx context.Context, ec *domain.ExecutionContext) (*domain.TestExecutionResult, error) {
	if m.execute != nil {
		return m.execute(ctx, ec)
	}
	return &domain.TestExecutionResult{Success: true}, nil
}

type mockLinterStage struct {
	name      string
	run       func(ctx context.Context, ec *domain.ExecutionContext) (*domain.StageResult, error)
	dependsOn []string
}

func (m *mockLinterStage) Name() string {
	return m.name
}

func (m *mockLinterStage) Run(ctx context.Context, ec *domain.ExecutionContext) (*domain.StageResult, error) {
	if m.run != nil {
		return m.run(ctx, ec)
	}
	return &domain.StageResult{StageName: m.name, Success: true}, nil
}

func (m *mockLinterStage) DependsOn() []string {
	return m.dependsOn
}

func TestDockerLinter_Success(t *testing.T) {
	stage := &mockLinterStage{name: "Hadolint"}
	pipeline := &mockLinterPipeline{}
	var buf bytes.Buffer

	linter := NewDockerLinter(pipeline, stage, &buf)
	err := linter.Run(context.Background(), "/workspace")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(pipeline.added) != 1 || pipeline.added[0] != stage {
		t.Errorf("expected Hadolint stage to be added to pipeline")
	}

	output := buf.String()
	if !strings.Contains(output, "Linting Dockerfile") {
		t.Errorf("expected output to contain static analysis header, got: %q", output)
	}
}

func TestDockerLinter_Failure(t *testing.T) {
	stage := &mockLinterStage{name: "Hadolint"}
	pipeline := &mockLinterPipeline{
		execute: func(ctx context.Context, ec *domain.ExecutionContext) (*domain.TestExecutionResult, error) {
			return nil, errors.New("hadolint run failed")
		},
	}
	var buf bytes.Buffer

	linter := NewDockerLinter(pipeline, stage, &buf)
	err := linter.Run(context.Background(), "/workspace")
	if err == nil {
		t.Fatal("expected linter execution error, got nil")
	}
}
