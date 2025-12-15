//go:build integration

package main

import (
	"context"
	"runtime"
	"sync"
	"testing"
	"time"

	providermodel "github.com/Cyclone1070/iav/internal/provider/model"
	"github.com/Cyclone1070/iav/internal/testing/mock"
	"github.com/Cyclone1070/iav/internal/tool/model"
	"github.com/Cyclone1070/iav/internal/tool/service"
	"github.com/stretchr/testify/assert"
)

func TestMain_InitTools(t *testing.T) {
	t.Parallel()

	// Create workspace context
	workspaceRoot := t.TempDir()
	fileSystem := service.NewOSFileSystem()
	binaryDetector := &service.SystemBinaryDetector{}
	checksumMgr := service.NewChecksumManager()
	gitignoreSvc, _ := service.NewGitignoreService(workspaceRoot, fileSystem)

	ctx := &model.WorkspaceContext{
		FS:               fileSystem,
		BinaryDetector:   binaryDetector,
		ChecksumManager:  checksumMgr,
		WorkspaceRoot:    workspaceRoot,
		GitignoreService: gitignoreSvc,
		CommandExecutor:  &service.OSCommandExecutor{},
		DockerConfig: model.DockerConfig{
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

	// Create readiness signal
	readyChan := make(chan struct{})

	// Create dependencies
	mockUI := mock.NewMockUI()
	mockUI.StartBlocker = make(chan struct{})
	mockUI.OnReadyCalled = func() {
		close(readyChan)
	}
	mockProvider := mock.NewMockProvider()
	providerFactory := func(ctx context.Context) (providermodel.Provider, error) {
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

	// Wait for startup signal (instead of sleep)
	select {
	case <-readyChan:
		// Ready to proceed
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for runInteractive to start")
	}

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
	mockUI := mock.NewMockUI()
	mockUI.OnReadyCalled = func() {
		mu.Lock()
		events = append(events, "UI_READY")
		mu.Unlock()
		close(readyChan)
	}

	// Track when provider starts init
	providerStartChan := make(chan struct{}, 1)
	providerFactory := func(ctx context.Context) (providermodel.Provider, error) {
		mu.Lock()
		events = append(events, "PROVIDER_START")
		mu.Unlock()
		close(providerStartChan)

		time.Sleep(100 * time.Millisecond) // Simulate slow init
		return mock.NewMockProvider(), nil
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
