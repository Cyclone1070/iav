// Package pipeline orchestrates execution steps using a Directed Acyclic Graph
// (DAG) model, collecting metrics and handling failures.
package pipeline

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	flow "github.com/Azure/go-workflow"
	"github.com/Cyclone1070/iav/internal/domain"
)

type Pipeline struct {
	stages []Stage
}

func NewPipeline() *Pipeline {
	return &Pipeline{
		stages: make([]Stage, 0),
	}
}

func (p *Pipeline) Add(s Stage) {
	p.stages = append(p.stages, s)
}

type stepWrapper struct {
	stage Stage
	ec    *domain.ExecutionContext
	res   *domain.StageResult
	err   error
	mu    sync.Mutex
}

func (w *stepWrapper) Do(ctx context.Context) error {
	start := time.Now()
	res, err := w.stage.Run(ctx, w.ec)
	duration := time.Since(start).Milliseconds()

	w.mu.Lock()
	defer w.mu.Unlock()

	if res != nil {
		res.DurationMs = duration
		w.res = res
	} else {
		success := err == nil
		var stderr string
		if err != nil {
			stderr = err.Error()
		}
		w.res = &domain.StageResult{
			StageName:  w.stage.Name(),
			Success:    success,
			DurationMs: duration,
			Stderr:     stderr,
		}
	}
	w.err = err
	return err
}

func (p *Pipeline) Execute(ctx context.Context, ec *domain.ExecutionContext) (*domain.TestExecutionResult, error) {
	w, _, err := p.buildWorkflow(ec)
	if err != nil {
		return nil, err
	}

	runErr := w.Do(ctx)

	stageResults, success, failedErr := collectWorkflowResults(w, runErr)

	exitCode := 0
	var summary string
	if !success {
		exitCode = 1
		summary = "Pipeline failed"
		if failedErr == nil {
			failedErr = errors.New("pipeline execution failed")
		}
	} else {
		summary = "Pipeline succeeded"
	}

	result := &domain.TestExecutionResult{
		Success:  success,
		ExitCode: exitCode,
		Stages:   stageResults,
		Summary:  summary,
	}

	return result, failedErr
}

func (p *Pipeline) buildWorkflow(ec *domain.ExecutionContext) (*flow.Workflow, map[string]*stepWrapper, error) {
	w := new(flow.Workflow)
	w.DontPanic = true

	// Build wrappers map
	wrappers := make(map[string]*stepWrapper)
	for _, s := range p.stages {
		wrappers[s.Name()] = &stepWrapper{
			stage: s,
			ec:    ec,
		}
	}

	// Add steps and build dependency graph
	for name, wrapper := range wrappers {
		builder := flow.Step(wrapper)

		var upstreams []flow.Steper
		for _, depName := range wrapper.stage.DependsOn() {
			if dep, ok := wrappers[depName]; ok {
				upstreams = append(upstreams, dep)
			} else {
				return nil, nil, fmt.Errorf("stage %q depends on unknown stage %q", name, depName)
			}
		}
		if len(upstreams) > 0 {
			builder = builder.DependsOn(upstreams...)
		}
		w.Add(builder)
	}

	return w, wrappers, nil
}

func collectWorkflowResults(w *flow.Workflow, runErr error) ([]domain.StageResult, bool, error) {
	var stageResults []domain.StageResult
	success := true
	var failedErr error
	if runErr != nil {
		failedErr = runErr
	}

	// w.Steps() returns steps in topological order
	steps := w.Steps()
	for _, step := range steps {
		if _, ok := step.(*stepWrapper); ok {
			res := processStepResult(step, &failedErr, &success)
			stageResults = append(stageResults, res)
		}
	}

	return stageResults, success, failedErr
}

func processStepResult(step flow.Steper, failedErr *error, success *bool) domain.StageResult {
	wrapper := step.(*stepWrapper)
	wrapper.mu.Lock()
	res := wrapper.res
	stepErr := wrapper.err
	wrapper.mu.Unlock()

	if res != nil {
		if !res.Success {
			*success = false
			if *failedErr == nil && stepErr != nil {
				*failedErr = stepErr
			}
		}
		return *res
	}

	// Step was skipped or not executed
	*success = false
	return domain.StageResult{
		StageName: wrapper.stage.Name(),
		Success:   false,
		Stderr:    "Skipped or not executed",
	}
}
