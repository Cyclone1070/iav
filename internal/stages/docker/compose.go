// Package docker encapsulates pipeline stage execution for docker-compose setups,
// including starting and stopping compose projects.
package docker

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/Cyclone1070/iav/internal/domain"
)

type composeClient interface {
	ComposeUp(ctx context.Context, composeFiles []string, runID string) error
	ComposeDown(ctx context.Context, composeFiles []string, runID string) error
}

type fileSystem interface {
	EvalSymlinks(path string) (string, error)
}

type ComposeUpStage struct {
	client    composeClient
	sys       fileSystem
}

func NewComposeUpStage(client composeClient, sys fileSystem) *ComposeUpStage {
	return &ComposeUpStage{
		client:    client,
		sys:       sys,
	}
}

func (s *ComposeUpStage) Name() string {
	return "Compose Up"
}

func (s *ComposeUpStage) DependsOn() []string {
	return []string{}
}

func (s *ComposeUpStage) Run(ctx context.Context, ec *domain.ExecutionContext) (*domain.StageResult, error) {
	composeAbs, err := resolveComposePaths(s.sys, ec.WorkspaceRoot, ec.ComposeFiles)
	if err != nil {
		return &domain.StageResult{
			StageName: s.Name(),
			Success:   false,
			Stderr:    err.Error(),
		}, err
	}

	err = s.client.ComposeUp(ctx, composeAbs, ec.RunID)
	if err != nil {
		return &domain.StageResult{
			StageName: s.Name(),
			Success:   false,
			Stderr:    err.Error(),
		}, err
	}

	return &domain.StageResult{
		StageName: s.Name(),
		Success:   true,
		Stdout:    "Docker Compose up successful for " + strings.Join(ec.ComposeFiles, ", "),
	}, nil
}

type ComposeDownStage struct {
	client    composeClient
	sys       fileSystem
}

func NewComposeDownStage(client composeClient, sys fileSystem) *ComposeDownStage {
	return &ComposeDownStage{
		client:    client,
		sys:       sys,
	}
}

func (s *ComposeDownStage) Name() string {
	return "Compose Down"
}

func (s *ComposeDownStage) DependsOn() []string {
	return []string{"Compose Up"}
}

func (s *ComposeDownStage) Run(ctx context.Context, ec *domain.ExecutionContext) (*domain.StageResult, error) {
	composeAbs, err := resolveComposePaths(s.sys, ec.WorkspaceRoot, ec.ComposeFiles)
	if err != nil {
		return &domain.StageResult{
			StageName: s.Name(),
			Success:   false,
			Stderr:    err.Error(),
		}, err
	}

	err = s.client.ComposeDown(ctx, composeAbs, ec.RunID)
	if err != nil {
		return &domain.StageResult{
			StageName: s.Name(),
			Success:   false,
			Stderr:    err.Error(),
		}, err
	}

	return &domain.StageResult{
		StageName: s.Name(),
		Success:   true,
		Stdout:    "Docker Compose down successful for " + strings.Join(ec.ComposeFiles, ", "),
	}, nil
}

func resolveComposePaths(sys fileSystem, workspaceRoot string, composeFiles []string) ([]string, error) {
	var composeAbs []string
	for _, f := range composeFiles {
		abs, err := resolveComposePath(sys, workspaceRoot, f)
		if err != nil {
			return nil, err
		}
		composeAbs = append(composeAbs, abs)
	}
	return composeAbs, nil
}

func resolveComposePath(sys fileSystem, workspaceRoot, composeFile string) (string, error) {
	workspaceAbs, err := filepath.Abs(workspaceRoot)
	if err != nil {
		return "", err
	}

	workspaceAbs, err = sys.EvalSymlinks(workspaceAbs)
	if err != nil {
		return "", err
	}

	composePath := composeFile
	if !filepath.IsAbs(composePath) {
		composePath = filepath.Join(workspaceAbs, composePath)
	}

	composePath, err = sys.EvalSymlinks(composePath)
	if err != nil {
		return "", err
	}

	return composePath, nil
}
