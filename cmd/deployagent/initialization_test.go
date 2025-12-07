//go:build integration

package main

import (
	"context"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/Cyclone1070/iav/internal/config"
	providermodels "github.com/Cyclone1070/iav/internal/provider/models"
	"github.com/Cyclone1070/iav/internal/testing/testhelpers"
	"github.com/Cyclone1070/iav/internal/tools/models"
	"github.com/Cyclone1070/iav/internal/tools/services"
	"github.com/stretchr/testify/assert"
)

func TestMain_InitTools(t *testing.T) {
	t.Parallel()

	// Create workspace context
	workspaceRoot := t.TempDir()
	fileSystem := services.NewOSFileSystem(config.DefaultConfig().Tools.MaxFileSize)
	binaryDetector := &services.SystemBinaryDetector{}
	checksumMgr := services.NewChecksumManager()
	gitignoreSvc, _ := services.NewGitignoreService(workspaceRoot, fileSystem)

	ctx := &models.WorkspaceContext{
		FS:               fileSystem,
		BinaryDetector:   binaryDetector,
		ChecksumManager:  checksumMgr,
		WorkspaceRoot:    workspaceRoot,
		GitignoreService: gitignoreSvc,
		CommandExecutor:  &services.OSCommandExecutor{},
		DockerConfig: models.DockerConfig{
			CheckCommand: []string{"docker", "info"},
			StartCommand: []string{"docker", "desktop", "start"},
		},
	}

	// Initialize tools using helper
	toolList := createTools(ctx)

	// All expected tools present
	expectedTools := []string{
		"read_file",
		"write_file",
		"edit_file",
		"list_directory",
		"run_shell",
		"search_content",
		"find_file",
		"read_todos",
		"write_todos",
	}

	for _, expected := range expectedTools {
		found := false
		for _, tool := range toolList {
			if tool.Name() == expected {
				found = true
				break
			}
		}
		assert.True(t, found, "Tool %s should be in toolList", expected)
	}

	// Correct count
	assert.Equal(t, len(expectedTools), len(toolList), "Should have exactly 9 tools")

	// All tools have valid definitions
	for _, tool := range toolList {
		def := tool.Definition()
		assert.NotEmpty(t, def.Name)
		assert.NotEmpty(t, def.Description)
	}
}

func TestMain_GoroutineCleanup(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping goroutine cleanup test in short mode")
	}

	initialGoroutines := runtime.NumGoroutine()

	// Create cancellable context for shutdown
	appCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Workspace context creation removed as it is now internal to runInteractive

	// Create dependencies
	mockUI := &testhelpers.MockUI{
		StartBlocker: make(chan struct{}),
	}
	mockProvider := testhelpers.NewMockProvider()
	providerFactory := func(ctx context.Context) (providermodels.Provider, error) {
		return mockProvider, nil
	}

	deps := Dependencies{
		UI:              mockUI,
		ProviderFactory: providerFactory,
		Tools:           nil, // Created in goroutine
	}

	// Run interactive mode in background
	done := make(chan bool)
	go func() {
		runInteractive(appCtx, deps) // Use cancellable context
		done <- true
	}()

	// Wait for startup
	time.Sleep(100 * time.Millisecond)

	// Verify goroutines started (proxy for "app is running")
	midCount := runtime.NumGoroutine()
	assert.Greater(t, midCount, initialGoroutines, "Background goroutines should have started")

	// Trigger shutdown
	cancel()
	close(mockUI.StartBlocker)

	// Verify graceful shutdown
	select {
	case <-done:
		// Success - runInteractive exited
	case <-time.After(2 * time.Second):
		t.Fatal("runInteractive did not stop after context cancellation")
	}

	// Allow time for cleanup
	time.Sleep(100 * time.Millisecond)
	runtime.GC()

	// Verify goroutines cleaned up (no leaks)
	finalCount := runtime.NumGoroutine()
	assert.LessOrEqual(t, finalCount, initialGoroutines,
		"Goroutines should clean up after shutdown (no leaks allowed)")
}

func TestMain_UIStartsInstantly(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping UI startup timing test in short mode")
	}

	// Workspace context creation removed as it is now internal to runInteractive

	// Track event order
	events := []string{}
	mu := sync.Mutex{}

	// Track when UI ready
	readyChan := make(chan struct{}, 1)
	mockUI := &testhelpers.MockUI{}
	mockUI.OnReadyCalled = func() {
		mu.Lock()
		events = append(events, "UI_READY")
		mu.Unlock()
		close(readyChan)
	}

	// Track when provider starts init
	providerStartChan := make(chan struct{}, 1)
	providerFactory := func(ctx context.Context) (providermodels.Provider, error) {
		mu.Lock()
		events = append(events, "PROVIDER_START")
		mu.Unlock()
		close(providerStartChan)

		time.Sleep(100 * time.Millisecond) // Simulate slow init
		return testhelpers.NewMockProvider(), nil
	}

	deps := Dependencies{
		UI:              mockUI,
		ProviderFactory: providerFactory,
		Tools:           nil, // Created in runInteractive
	}

	// Run test
	start := time.Now()
	go func() {
		runInteractive(context.Background(), deps)
	}()

	// Assert UI ready within reasonable time
	select {
	case <-readyChan:
		elapsed := time.Since(start)
		assert.Less(t, elapsed, 1*time.Second,
			"UI should be ready within 1s (proves responsiveness)")
	case <-time.After(2 * time.Second):
		t.Fatal("UI never signaled ready")
	}

	// Assert provider starts eventually
	select {
	case <-providerStartChan:
		// Success
	case <-time.After(3 * time.Second):
		t.Fatal("Provider never started initializing")
	}

	// Assert correct order (sequencing, not timing)
	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, []string{"UI_READY", "PROVIDER_START"}, events,
		"UI must signal ready before provider starts")
}
