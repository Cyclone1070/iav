//go:build integration

package main

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	orchmodel "github.com/Cyclone1070/iav/internal/orchestrator/model"
	providermodel "github.com/Cyclone1070/iav/internal/provider/model"
	"github.com/Cyclone1070/iav/internal/testing/mock"
	uimodel "github.com/Cyclone1070/iav/internal/ui/model"
	"github.com/stretchr/testify/assert"
)

func TestInteractiveMode_FullFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Control when MockUI exits
	startBlocker := make(chan struct{})

	// Create synchronization for readiness
	readyChan := make(chan struct{})

	// Create Mock UI
	var inputCount int
	mockUI := mock.NewMockUI()
	mockUI.InputFunc = func(ctx context.Context, prompt string) (string, error) {
		inputCount++
		if inputCount > 1 {
			return "", fmt.Errorf("stop test")
		}
		return "List files", nil
	}
	mockUI.StartBlocker = startBlocker
	mockUI.OnReadyCalled = func() {
		close(readyChan)
	}

	// Track what orchestrator sends to provider
	var allProviderCalls []providermodel.GenerateRequest
	var mu sync.Mutex

	// Create mock provider
	mockProvider := mock.NewMockProvider().
		WithToolCallResponse([]orchmodel.ToolCall{
			{
				ID:   "call_1",
				Name: "list_directory",
				Args: map[string]any{
					"path":      ".",
					"max_depth": -1,
					"offset":    0,
					"limit":     100,
				},
			},
		}).
		WithTextResponse("Found files in current directory")

	// Capture provider inputs
	mockProvider.OnGenerateCalled = func(req *providermodel.GenerateRequest) {
		mu.Lock()
		defer mu.Unlock()
		allProviderCalls = append(allProviderCalls, *req)
	}

	providerFactory := func(ctx context.Context) (providermodel.Provider, error) {
		return mockProvider, nil
	}

	// Create dependencies
	deps := Dependencies{
		UI:              mockUI,
		ProviderFactory: providerFactory,
		Tools:           nil, // Created in goroutine
	}

	// Run interactive mode in background
	go func() {
		runInteractive(context.Background(), deps)
	}()

	// Wait for initialization using channel
	select {
	case <-readyChan:
		// System ready
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for UI to become ready")
	}

	// Verify provider called multiple times (tool call + final response)
	// Retry loop to allow async processing to finish
	var callCount int
	timeout := time.After(2 * time.Second)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

waitForCalls:
	for {
		select {
		case <-timeout:
			break waitForCalls
		case <-ticker.C:
			mu.Lock()
			callCount = len(allProviderCalls)
			mu.Unlock()
			if callCount >= 2 {
				break waitForCalls
			}
		}
	}

	// Let UI exit
	close(startBlocker)

	assert.GreaterOrEqual(t, callCount, 2,
		"Provider should be called at least twice (initial + after tool execution)")

	// Verify orchestrator sent tool results back to provider
	mu.Lock()
	if len(allProviderCalls) > 0 {
		lastHistory := allProviderCalls[len(allProviderCalls)-1].History
		foundToolResult := false
		for _, msg := range lastHistory {
			if msg.Role == "function" && len(msg.ToolResults) > 0 {
				foundToolResult = true
				// Verify tool result structure
				assert.Equal(t, "list_directory", msg.ToolResults[0].Name)
				assert.Equal(t, "call_1", msg.ToolResults[0].ID)
				break
			}
		}
		assert.True(t, foundToolResult,
			"Orchestrator should send tool results to provider in history")
	} else {
		t.Error("No provider calls captured")
	}
	mu.Unlock()

	// Verify UI received final message
	foundResponse := false
	timeout = time.After(2 * time.Second)
	ticker = time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

loop:
	for {
		select {
		case <-timeout:
			break loop
		case <-ticker.C:
			// Check messages
			for _, msg := range mockUI.GetMessages() {
				if msg == "Found files in current directory" {
					foundResponse = true
					break loop
				}
			}
		}
	}
	assert.True(t, foundResponse, "Should have received final response. Messages: %v", mockUI.GetMessages())
}

func TestInteractiveMode_ListModelsFromProvider(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Expected models from provider
	expectedModels := []string{
		"gemini-2.0-flash-exp",
		"gemini-1.5-pro-latest",
		"gemini-1.5-flash",
	}

	// Track whether ListModels was called
	var listModelsCalled bool
	var mu sync.Mutex

	// Create mock provider with ListModels implementation
	mockProvider := mock.NewMockProvider()
	mockProvider.ListModelsFunc = func(ctx context.Context) ([]string, error) {
		mu.Lock()
		listModelsCalled = true
		mu.Unlock()
		return expectedModels, nil
	}

	providerFactory := func(ctx context.Context) (providermodel.Provider, error) {
		return mockProvider, nil
	}

	// Track what models UI received
	var receivedModels []string
	modelListChan := make(chan []string, 1)

	// Create synchronization for readiness
	readyChan := make(chan struct{})

	startBlocker := make(chan struct{})
	commandChan := make(chan uimodel.UICommand, 1)

	// Create Mock UI using constructor
	mockUI := mock.NewMockUI()
	mockUI.StartBlocker = startBlocker
	mockUI.CommandsChan = commandChan
	mockUI.OnModelListWritten = func(models []string) {
		modelListChan <- models
	}
	mockUI.OnReadyCalled = func() {
		close(readyChan)
	}
	// We need to provide InputFunc to avoid infinite loop if ReadInput is called
	mockUI.InputFunc = func(ctx context.Context, prompt string) (string, error) {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(10 * time.Second):
			return "timeout", nil
		}
	}

	deps := Dependencies{
		UI:              mockUI,
		ProviderFactory: providerFactory,
		Tools:           nil,
	}

	// Run interactive mode in background
	go func() {
		runInteractive(context.Background(), deps)
	}()

	// Wait for initialization using channel
	select {
	case <-readyChan:
		// System ready
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for UI to become ready")
	}

	// Send list_models command
	commandChan <- uimodel.UICommand{Type: "list_models"}

	// Wait for response
	select {
	case receivedModels = <-modelListChan:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for model list response")
	}

	// Stop test
	close(startBlocker)

	// Wait for cleanup
	// Ideally we'd wait for runInteractive to exit via channel, but MockUI doesn't expose it.
	// We rely on startBlocker closing causing loop exit.

	// Verify provider.ListModels was called
	mu.Lock()
	called := listModelsCalled
	mu.Unlock()

	assert.True(t, called, "provider.ListModels() should have been called")
	assert.Equal(t, expectedModels, receivedModels,
		"UI should receive models from provider, not hardcoded list")
}

func TestInteractiveMode_SwitchModelCallsProvider(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	var setModelCalled bool
	var setModelArg string
	var mu sync.Mutex

	mockProvider := mock.NewMockProvider()
	mockProvider.SetModelFunc = func(model string) error {
		mu.Lock()
		setModelCalled = true
		setModelArg = model
		mu.Unlock()
		return nil
	}

	providerFactory := func(ctx context.Context) (providermodel.Provider, error) {
		return mockProvider, nil
	}

	// Create synchronization for readiness
	readyChan := make(chan struct{})

	startBlocker := make(chan struct{})
	commandChan := make(chan uimodel.UICommand, 1)

	// Create Mock UI using constructor
	mockUI := mock.NewMockUI()
	mockUI.StartBlocker = startBlocker
	mockUI.CommandsChan = commandChan
	mockUI.OnReadyCalled = func() {
		close(readyChan)
	}
	mockUI.InputFunc = func(ctx context.Context, prompt string) (string, error) {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(10 * time.Second):
			return "timeout", nil
		}
	}

	deps := Dependencies{
		UI:              mockUI,
		ProviderFactory: providerFactory,
		Tools:           nil,
	}

	go func() {
		runInteractive(context.Background(), deps)
	}()

	// Wait for initialization using channel
	select {
	case <-readyChan:
		// System ready
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for UI to become ready")
	}

	// Send switch_model command
	targetModel := "gemini-1.5-flash"
	commandChan <- uimodel.UICommand{
		Type: "switch_model",
		Args: map[string]string{"model": targetModel},
	}

	// Wait for cleanup later...

	// Poll for changes instead of sleep
	timeout := time.After(2 * time.Second)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	var called bool

waitForCall:
	for {
		select {
		case <-timeout:
			break waitForCall
		case <-ticker.C:
			mu.Lock()
			called = setModelCalled
			mu.Unlock()
			if called {
				break waitForCall
			}
		}
	}

	// Stop test
	close(startBlocker)

	// Verify provider.SetModel was called
	mu.Lock()
	called = setModelCalled
	modelArg := setModelArg
	mu.Unlock()

	assert.True(t, called, "provider.SetModel() should have been called")
	assert.Equal(t, targetModel, modelArg,
		"provider.SetModel() should be called with correct model name")
}
