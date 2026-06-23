package docker

import (
	"context"
	"os"
	"path/filepath"
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

func TestHadolintStageHostSuccess(t *testing.T) {
	mockClient := &mockDockerClient{}
	stage := NewLintStage(mockClient)
	stage.lookPath = func(name string) (string, error) {
		if name != "hadolint" {
			t.Errorf("unexpected lookPath call for: %s", name)
		}
		return "/usr/local/bin/hadolint", nil
	}

	tmpDir := t.TempDir()
	dockerfilePath := filepath.Join(tmpDir, "Dockerfile")
	if err := os.WriteFile(dockerfilePath, []byte("FROM alpine"), 0600); err != nil {
		t.Fatalf("failed to write test Dockerfile: %v", err)
	}

	stage.hostRunner = func(ctx context.Context, name string, arg string) (string, string, int, error) {
		if name != "/usr/local/bin/hadolint" {
			t.Errorf("unexpected name: %s", name)
		}
		if arg != dockerfilePath {
			t.Errorf("unexpected arg: %s", arg)
		}
		return "HOST_OK", "", 0, nil
	}

	ec := &domain.ExecutionContext{
		WorkspaceRoot: tmpDir,
	}

	res, err := stage.Run(context.Background(), ec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !res.Success {
		t.Errorf("expected success")
	}
	if res.Stdout != "HOST_OK" {
		t.Errorf("unexpected stdout: %s", res.Stdout)
	}
}

func TestHadolintStageHostFailure(t *testing.T) {
	mockClient := &mockDockerClient{}
	stage := NewLintStage(mockClient)
	stage.lookPath = func(name string) (string, error) {
		return "/usr/local/bin/hadolint", nil
	}

	tmpDir := t.TempDir()
	dockerfilePath := filepath.Join(tmpDir, "Dockerfile")
	if err := os.WriteFile(dockerfilePath, []byte("FROM alpine"), 0600); err != nil {
		t.Fatalf("failed to write test Dockerfile: %v", err)
	}

	stage.hostRunner = func(ctx context.Context, name string, arg string) (string, string, int, error) {
		return "", "host error: DL3006", 1, nil
	}

	ec := &domain.ExecutionContext{
		WorkspaceRoot: tmpDir,
	}

	res, err := stage.Run(context.Background(), ec)
	if err == nil {
		t.Errorf("expected error, got nil")
	}

	if res.Success {
		t.Errorf("expected stage success=false")
	}
	if res.Stderr != "host error: DL3006" {
		t.Errorf("unexpected stderr: %s", res.Stderr)
	}
}
