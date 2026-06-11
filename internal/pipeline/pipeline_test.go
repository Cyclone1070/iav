package pipeline

import (
	"context"
	"errors"
	"testing"

	"github.com/Cyclone1070/iav/internal/domain"
)

type dummyStage struct {
	name    string
	runFunc func(ctx context.Context, ec *domain.ExecutionContext) (*domain.StageResult, error)
	deps    []string
}

func (d *dummyStage) Name() string { return d.name }
func (d *dummyStage) Run(ctx context.Context, ec *domain.ExecutionContext) (*domain.StageResult, error) {
	return d.runFunc(ctx, ec)
}
func (d *dummyStage) DependsOn() []string { return d.deps }

func TestPipelineSuccess(t *testing.T) {
	ctx := context.Background()
	ec := &domain.ExecutionContext{
		WorkspaceRoot: "/tmp",
	}

	var executed []string

	s1 := &dummyStage{
		name: "stage1",
		runFunc: func(ctx context.Context, ec *domain.ExecutionContext) (*domain.StageResult, error) {
			executed = append(executed, "stage1")
			return &domain.StageResult{StageName: "stage1", Success: true}, nil
		},
	}

	s2 := &dummyStage{
		name: "stage2",
		deps: []string{"stage1"},
		runFunc: func(ctx context.Context, ec *domain.ExecutionContext) (*domain.StageResult, error) {
			executed = append(executed, "stage2")
			return &domain.StageResult{StageName: "stage2", Success: true}, nil
		},
	}

	p := NewPipeline()
	p.Add(s1)
	p.Add(s2)

	res, err := p.Execute(ctx, ec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !res.Success {
		t.Errorf("expected success")
	}

	if len(executed) != 2 || executed[0] != "stage1" || executed[1] != "stage2" {
		t.Errorf("unexpected execution order: %v", executed)
	}
}

func TestPipelineFailure(t *testing.T) {
	ctx := context.Background()
	ec := &domain.ExecutionContext{
		WorkspaceRoot: "/tmp",
	}

	s1 := &dummyStage{
		name: "stage1",
		runFunc: func(ctx context.Context, ec *domain.ExecutionContext) (*domain.StageResult, error) {
			return &domain.StageResult{StageName: "stage1", Success: false, Stderr: "error here"}, errors.New("failed")
		},
	}

	s2 := &dummyStage{
		name: "stage2",
		deps: []string{"stage1"},
		runFunc: func(ctx context.Context, ec *domain.ExecutionContext) (*domain.StageResult, error) {
			t.Errorf("stage2 should not have run")
			return &domain.StageResult{StageName: "stage2", Success: true}, nil
		},
	}

	p := NewPipeline()
	p.Add(s1)
	p.Add(s2)

	res, err := p.Execute(ctx, ec)
	if err == nil {
		t.Errorf("expected error, got nil")
	}

	if res.Success {
		t.Errorf("expected pipeline to report failure")
	}

	stagesMap := make(map[string]domain.StageResult)
	for _, s := range res.Stages {
		stagesMap[s.StageName] = s
	}

	if len(stagesMap) != 2 {
		t.Fatalf("expected 2 stages, got %d", len(stagesMap))
	}

	res1, ok1 := stagesMap["stage1"]
	res2, ok2 := stagesMap["stage2"]
	if !ok1 || !ok2 || res1.Success || res2.Success {
		t.Errorf("unexpected stages results: %+v", res.Stages)
	}
}
