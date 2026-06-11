package workflow

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Cyclone1070/iav/internal/domain"
)

type mockRunner struct {
	runFunc func(ctx context.Context, testComposePath string, runID string, timeoutMs int) (*domain.TestExecutionResult, error)
}

func (m *mockRunner) Run(ctx context.Context, testComposePath string, runID string, timeoutMs int) (*domain.TestExecutionResult, error) {
	if m.runFunc != nil {
		return m.runFunc(ctx, testComposePath, runID, timeoutMs)
	}
	return &domain.TestExecutionResult{Success: true, Summary: "Compose succeeded"}, nil
}

type mockDiscoveryService struct {
	runFunc func(targetPath string) ([]string, error)
}

func (m *mockDiscoveryService) Run(targetPath string) ([]string, error) {
	if m.runFunc != nil {
		return m.runFunc(targetPath)
	}
	return []string{targetPath}, nil
}

type mockPruner struct {
	pruneFunc func(ctx context.Context) error
}

func (m *mockPruner) PruneExpiredResources(ctx context.Context) error {
	if m.pruneFunc != nil {
		return m.pruneFunc(ctx)
	}
	return nil
}

func TestEngine_Success(t *testing.T) {
	discover := &mockDiscoveryService{
		runFunc: func(targetPath string) ([]string, error) {
			return []string{"/workspace/docker-compose.test.yml"}, nil
		},
	}
	pruner := &mockPruner{}
	runner := &mockRunner{
		runFunc: func(ctx context.Context, testComposePath string, runID string, timeoutMs int) (*domain.TestExecutionResult, error) {
			return &domain.TestExecutionResult{
				Success: true,
				Summary: "Succeeded",
				Stages: []domain.StageResult{
					{StageName: "Compose Up", Success: true, DurationMs: 20},
				},
			}, nil
		},
	}

	var buf bytes.Buffer
	engine := NewEngine(discover, pruner, runner, &buf)

	err := engine.Run(context.Background(), "/workspace/docker-compose.test.yml", 30000)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Running test compose: docker-compose.test.yml") {
		t.Errorf("expected output to contain compose name, got: %s", output)
	}
	if !strings.Contains(output, "### Test Execution Summary for `docker-compose.test.yml`") {
		t.Errorf("expected output to contain summary table, got: %s", output)
	}
}

func TestEngine_DiscoveryFailure(t *testing.T) {
	discover := &mockDiscoveryService{
		runFunc: func(targetPath string) ([]string, error) {
			return nil, errors.New("discovery failed")
		},
	}
	pruner := &mockPruner{}
	runner := &mockRunner{}

	var buf bytes.Buffer
	engine := NewEngine(discover, pruner, runner, &buf)

	err := engine.Run(context.Background(), "/workspace/nonexistent", 30000)
	if err == nil {
		t.Fatal("expected discovery error, got nil")
	}
}

func TestEngine_ScriptFailure(t *testing.T) {
	discover := &mockDiscoveryService{
		runFunc: func(targetPath string) ([]string, error) {
			return []string{"/workspace/docker-compose.test.yml"}, nil
		},
	}
	pruner := &mockPruner{}
	runner := &mockRunner{
		runFunc: func(ctx context.Context, testComposePath string, runID string, timeoutMs int) (*domain.TestExecutionResult, error) {
			return &domain.TestExecutionResult{Success: false, Summary: "Compose failed"}, errors.New("compose up failed")
		},
	}

	var buf bytes.Buffer
	engine := NewEngine(discover, pruner, runner, &buf)

	err := engine.Run(context.Background(), "/workspace/docker-compose.test.yml", 30000)
	if err == nil {
		t.Fatal("expected run error, got nil")
	}
	if !strings.Contains(err.Error(), "the following test scripts failed") {
		t.Errorf("expected failure summary error, got: %v", err)
	}
}
