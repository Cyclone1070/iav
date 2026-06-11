package docker

import (
	"context"
	"testing"

	"github.com/Cyclone1070/iav/internal/domain"
)

func TestHadolintStageSuccess(t *testing.T) {
	mockClient := &mockDockerClient{}
	mockClient.runContainerInfoFunc = func(ctx context.Context, opts domain.RunOptions) (stdout string, stderr string, exitCode int, err error) {
		// verify parameters
		if opts.Image != "hadolint/hadolint:latest-alpine" {
			t.Errorf("unexpected image: %s", opts.Image)
		}
		if len(opts.Cmd) != 2 || opts.Cmd[0] != "hadolint" || opts.Cmd[1] != "/workspace/Dockerfile" {
			t.Errorf("unexpected cmd: %v", opts.Cmd)
		}
		if len(opts.Binds) != 1 || opts.Binds[0] != "/tmp:/workspace:ro" {
			t.Errorf("unexpected binds: %v", opts.Binds)
		}
		return "OK", "", 0, nil
	}

	stage := NewLintStage(mockClient)
	ec := &domain.ExecutionContext{
		WorkspaceRoot: "/tmp",
	}

	res, err := stage.Run(context.Background(), ec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !res.Success {
		t.Errorf("expected success")
	}
	if res.StageName != "Hadolint" {
		t.Errorf("unexpected name: %s", res.StageName)
	}
}

func TestHadolintStageFailure(t *testing.T) {
	mockClient := &mockDockerClient{}
	mockClient.runContainerInfoFunc = func(ctx context.Context, opts domain.RunOptions) (stdout string, stderr string, exitCode int, err error) {
		return "", "error: DL3006 Always tag the version of an image", 1, nil
	}

	stage := NewLintStage(mockClient)
	ec := &domain.ExecutionContext{
		WorkspaceRoot: "/tmp",
	}

	res, err := stage.Run(context.Background(), ec)
	if err == nil {
		t.Errorf("expected error, got nil")
	}

	if res.Success {
		t.Errorf("expected stage success=false")
	}
	if res.Stderr != "error: DL3006 Always tag the version of an image" {
		t.Errorf("unexpected stderr recorded: %s", res.Stderr)
	}
}
