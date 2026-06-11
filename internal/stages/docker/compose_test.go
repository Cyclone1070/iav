package docker

import (
	"context"
	"errors"
	"testing"

	"github.com/Cyclone1070/iav/internal/domain"
)

func TestComposeUpStage_Success(t *testing.T) {
	mockFS := &mockFileSystem{files: make(map[string][]byte)}
	mockFS.files["/workspace/docker-compose.yml"] = []byte("version: '3'")

	mockClient := &mockDockerClient{}
	mockClient.composeUpFunc = func(ctx context.Context, composeFiles []string, runID string) error {
		if runID != "run-compose-123" {
			t.Errorf("unexpected run ID: %s", runID)
		}
		if len(composeFiles) != 1 || composeFiles[0] != "/workspace/docker-compose.yml" {
			t.Errorf("unexpected compose files: %v", composeFiles)
		}
		return nil
	}

	stage := NewComposeUpStage(mockClient, mockFS)
	ec := &domain.ExecutionContext{
		WorkspaceRoot: "/workspace",
		ComposeFiles:  []string{"docker-compose.yml"},
		RunID:         "run-compose-123",
	}

	res, err := stage.Run(context.Background(), ec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !res.Success {
		t.Errorf("expected compose up to succeed")
	}
	if res.StageName != "Compose Up" {
		t.Errorf("unexpected stage name: %s", res.StageName)
	}
}

func TestComposeUpStage_Failure(t *testing.T) {
	mockFS := &mockFileSystem{files: make(map[string][]byte)}
	mockFS.files["/workspace/docker-compose.yml"] = []byte("version: '3'")

	mockClient := &mockDockerClient{}
	mockClient.composeUpFunc = func(ctx context.Context, composeFiles []string, runID string) error {
		return errors.New("compose up failed")
	}

	stage := NewComposeUpStage(mockClient, mockFS)
	ec := &domain.ExecutionContext{
		WorkspaceRoot: "/workspace",
		ComposeFiles:  []string{"docker-compose.yml"},
		RunID:         "run-compose-123",
	}

	res, err := stage.Run(context.Background(), ec)
	if err == nil {
		t.Fatalf("expected error from compose up failure")
	}

	if res.Success {
		t.Errorf("expected stage success=false")
	}
}

func TestComposeDownStage_Success(t *testing.T) {
	mockFS := &mockFileSystem{files: make(map[string][]byte)}
	mockFS.files["/workspace/docker-compose.yml"] = []byte("version: '3'")

	mockClient := &mockDockerClient{}
	mockClient.composeDownFunc = func(ctx context.Context, composeFiles []string, runID string) error {
		if runID != "run-compose-123" {
			t.Errorf("unexpected run ID: %s", runID)
		}
		if len(composeFiles) != 1 || composeFiles[0] != "/workspace/docker-compose.yml" {
			t.Errorf("unexpected compose files: %v", composeFiles)
		}
		return nil
	}

	stage := NewComposeDownStage(mockClient, mockFS)
	ec := &domain.ExecutionContext{
		WorkspaceRoot: "/workspace",
		ComposeFiles:  []string{"docker-compose.yml"},
		RunID:         "run-compose-123",
	}

	res, err := stage.Run(context.Background(), ec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !res.Success {
		t.Errorf("expected compose down to succeed")
	}
	if res.StageName != "Compose Down" {
		t.Errorf("unexpected stage name: %s", res.StageName)
	}
}
