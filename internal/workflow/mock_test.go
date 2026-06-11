package workflow

import (
	"context"

	"github.com/Cyclone1070/iav/internal/domain"
)

type mockDockerClient struct {
	composeDownFunc func(ctx context.Context, composeFiles []string, runID string) error
}

func (m *mockDockerClient) ComposeDown(ctx context.Context, composeFiles []string, runID string) error {
	if m.composeDownFunc != nil {
		return m.composeDownFunc(ctx, composeFiles, runID)
	}
	return nil
}

type mockStage struct {
	name string
}

func (m *mockStage) Name() string { return m.name }
func (m *mockStage) Run(ctx context.Context, ec *domain.ExecutionContext) (*domain.StageResult, error) {
	return &domain.StageResult{StageName: m.name, Success: true}, nil
}
func (m *mockStage) DependsOn() []string { return nil }

type mockPipeline struct {
	stages []Stage
}

func (m *mockPipeline) Add(s Stage) {
	m.stages = append(m.stages, s)
}

func (m *mockPipeline) Execute(ctx context.Context, ec *domain.ExecutionContext) (*domain.TestExecutionResult, error) {
	var stagesResults []domain.StageResult
	for _, s := range m.stages {
		res, _ := s.Run(ctx, ec)
		stagesResults = append(stagesResults, *res)
	}
	return &domain.TestExecutionResult{
		Success:  true,
		ExitCode: 0,
		Stages:   stagesResults,
		Summary:  "Succeeded",
	}, nil
}
