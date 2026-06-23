package cmd

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/Cyclone1070/iav/internal/docker"
	"github.com/Cyclone1070/iav/internal/domain"
	iavfs "github.com/Cyclone1070/iav/internal/fs"
	"github.com/Cyclone1070/iav/internal/pipeline"
	dockerstages "github.com/Cyclone1070/iav/internal/stages/docker"
)

func TestDockerCompose_Integration(t *testing.T) {
	// 1. Check if Docker is available
	client, err := docker.NewDockerClient()
	if err != nil {
		t.Skipf("Skipping integration test: Docker daemon not available: %v", err)
	}
	defer func() { _ = client.Close() }()

	// 2. Create temp directory for workspace
	tmpDir := t.TempDir()

	// 3. Write a simple valid docker-compose.yml
	composeContent := `version: '3.8'
services:
  target:
    image: alpine:3.20
    command: ["sleep", "3600"]
    healthcheck:
      test: ["CMD-SHELL", "true"]
      interval: 1s
      timeout: 1s
      retries: 2
`
	composePath := filepath.Join(tmpDir, "docker-compose.yml")
	if err := os.WriteFile(composePath, []byte(composeContent), 0600); err != nil {
		t.Fatalf("failed to write docker-compose.yml: %v", err)
	}

	// 4. Write docker-compose.test.yml containing the 'sut' service
	composeTestContent := `version: '3.8'
services:
  sut:
    image: alpine:3.20
    command: ["ping", "-c", "1", "target"]
    depends_on:
      target:
        condition: service_healthy
`
	composeTestPath := filepath.Join(tmpDir, "docker-compose.test.yml")
	if err := os.WriteFile(composeTestPath, []byte(composeTestContent), 0600); err != nil {
		t.Fatalf("failed to write docker-compose.test.yml: %v", err)
	}

	// 5. Construct Compose pipeline
	p := pipeline.NewPipeline(nil)
	p.Add(dockerstages.NewComposeUpStage(client, iavfs.RealFS{}))
	p.Add(dockerstages.NewComposeDownStage(client, iavfs.RealFS{}))

	// Generate a unique run ID to avoid network/container collisions
	randBytes := make([]byte, 8)
	_, _ = rand.Read(randBytes)
	runID := "int-comp-" + hex.EncodeToString(randBytes)

	ec := &domain.ExecutionContext{
		WorkspaceRoot:        tmpDir,
		ComposeFiles:         []string{"docker-compose.yml", "docker-compose.test.yml"},
		RunID:                runID,
		EnvironmentVariables: make(map[string]string),
	}

	defer func() {
		_ = client.ComposeDown(context.Background(), []string{composePath, composeTestPath}, runID)
	}()

	res, err := p.Execute(context.Background(), ec)
	if err != nil {
		t.Fatalf("compose pipeline execution failed: %v, stages: %+v", err, res.Stages)
	}

	if !res.Success {
		t.Errorf("compose pipeline reported failure: %+v", res)
	}
}
