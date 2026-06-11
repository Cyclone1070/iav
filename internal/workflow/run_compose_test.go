package workflow

import (
	"context"
	"testing"
)

func TestRunCompose_Hermetic(t *testing.T) {
	m := &mockFS{
		files: map[string][]byte{
			"/workspace/docker-compose.test.yml": []byte("version: '3'"),
			"/workspace/docker-compose.yml":      []byte("version: '3'"),
		},
		currentDir: "/workspace",
	}

	client := &mockDockerClient{}
	var composeDownCalled bool
	client.composeDownFunc = func(ctx context.Context, composeFiles []string, runID string) error {
		composeDownCalled = true
		if len(composeFiles) != 2 || composeFiles[0] != "/workspace/docker-compose.yml" || composeFiles[1] != "/workspace/docker-compose.test.yml" {
			t.Errorf("unexpected compose files passed to ComposeDown: %v", composeFiles)
		}
		return nil
	}

	stages := []Stage{&mockStage{name: "test-stage"}}
	factory := func() Pipeline {
		return &mockPipeline{}
	}

	runner := NewRunCompose(m, client, stages, factory)
	res, err := runner.Run(context.Background(), "/workspace/docker-compose.test.yml", "run-id", 30000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !res.Success {
		t.Errorf("expected run success, got %+v", res)
	}

	if !composeDownCalled {
		t.Errorf("expected composeDown to be called")
	}
}
