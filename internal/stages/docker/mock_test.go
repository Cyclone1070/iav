package docker

import (
	"context"
	"io"

	"github.com/Cyclone1070/iav/internal/domain"
)

type mockFileSystem struct {
	files map[string][]byte
}

func (m *mockFileSystem) EvalSymlinks(path string) (string, error) {
	return path, nil
}

type mockDockerClient struct {
	runContainerInfoFunc        func(ctx context.Context, opts domain.RunOptions) (string, string, int, error)
	buildImageFunc              func(ctx context.Context, opts domain.BuildOptions, out io.Writer) error
	createAndStartContainerFunc func(ctx context.Context, opts domain.RunOptions) (string, error)
	inspectContainerFunc        func(ctx context.Context, id string) (*domain.ContainerState, error)
	composeUpFunc               func(ctx context.Context, composeFiles []string, runID string) error
	composeDownFunc             func(ctx context.Context, composeFiles []string, runID string) error
}

func (m *mockDockerClient) RunContainerInfo(ctx context.Context, opts domain.RunOptions) (string, string, int, error) {
	if m.runContainerInfoFunc != nil {
		return m.runContainerInfoFunc(ctx, opts)
	}
	return "", "", 0, nil
}

func (m *mockDockerClient) BuildImage(ctx context.Context, opts domain.BuildOptions, out io.Writer) error {
	if m.buildImageFunc != nil {
		return m.buildImageFunc(ctx, opts, out)
	}
	return nil
}

func (m *mockDockerClient) CreateAndStartContainer(ctx context.Context, opts domain.RunOptions) (string, error) {
	if m.createAndStartContainerFunc != nil {
		return m.createAndStartContainerFunc(ctx, opts)
	}
	return "", nil
}

func (m *mockDockerClient) InspectContainer(ctx context.Context, id string) (*domain.ContainerState, error) {
	if m.inspectContainerFunc != nil {
		return m.inspectContainerFunc(ctx, id)
	}
	return &domain.ContainerState{Running: true}, nil
}

func (m *mockDockerClient) ComposeUp(ctx context.Context, composeFiles []string, runID string) error {
	if m.composeUpFunc != nil {
		return m.composeUpFunc(ctx, composeFiles, runID)
	}
	return nil
}

func (m *mockDockerClient) ComposeDown(ctx context.Context, composeFiles []string, runID string) error {
	if m.composeDownFunc != nil {
		return m.composeDownFunc(ctx, composeFiles, runID)
	}
	return nil
}
