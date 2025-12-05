//go:build integration

package main

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	orchmodels "github.com/Cyclone1070/iav/internal/orchestrator/models"
	providermodels "github.com/Cyclone1070/iav/internal/provider/models"
	"github.com/Cyclone1070/iav/internal/testing/testhelpers"
	"github.com/Cyclone1070/iav/internal/ui"
	"github.com/stretchr/testify/assert"
)

func TestInteractiveMode_FullFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Workspace context creation removed as it is now internal to runInteractive

	// Control when MockUI exits
	startBlocker := make(chan struct{})

	// Create Mock UI
	var inputCount int
	mockUI := &testhelpers.MockUI{
		InputFunc: func(ctx context.Context, prompt string) (string, error) {
			inputCount++
			if inputCount > 1 {
				return "", fmt.Errorf("stop test")
			}
			return "List files", nil
		},
		StartBlocker: startBlocker,
	}

	// Track what orchestrator sends to provider
	var allProviderCalls []providermodels.GenerateRequest
	var mu sync.Mutex

	// Create mock provider
	mockProvider := testhelpers.NewMockProvider().
		WithToolCallResponse([]orchmodels.ToolCall{
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
	mockProvider.OnGenerateCalled = func(req *providermodels.GenerateRequest) {
		mu.Lock()
		defer mu.Unlock()
		allProviderCalls = append(allProviderCalls, *req)
	}

	providerFactory := func(ctx context.Context) (providermodels.Provider, error) {
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

	// Give orchestrator time to initialize and run
	time.Sleep(300 * time.Millisecond)

	// Let UI exit
	close(startBlocker)

	// Small delay for cleanup
	time.Sleep(50 * time.Millisecond)

	// Verify provider called multiple times (tool call + final response)
	mu.Lock()
	callCount := len(allProviderCalls)
	mu.Unlock()
	assert.GreaterOrEqual(t, callCount, 2,
		"Provider should be called at least twice (initial + after tool execution)")

	// Verify orchestrator sent tool results back to provider
	mu.Lock()
	lastHistory := allProviderCalls[len(allProviderCalls)-1].History
	mu.Unlock()

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

	// Verify UI received final message
	foundResponse := false
	timeout := time.After(2 * time.Second)
	ticker := time.NewTicker(10 * time.Millisecond)
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
	mockProvider := testhelpers.NewMockProvider()
	mockProvider.ListModelsFunc = func(ctx context.Context) ([]string, error) {
		mu.Lock()
		listModelsCalled = true
		mu.Unlock()
		return expectedModels, nil
	}

	providerFactory := func(ctx context.Context) (providermodels.Provider, error) {
		return mockProvider, nil
	}

	// Track what models UI received
	var receivedModels []string
	modelListChan := make(chan []string, 1)

	startBlocker := make(chan struct{})
	commandChan := make(chan ui.UICommand, 1)

	mockUI := &testhelpers.MockUI{
		StartBlocker: startBlocker,
		CommandsChan: commandChan,
		OnModelListWritten: func(models []string) {
			modelListChan <- models
		},
		// We need to provide InputFunc to avoid infinite loop if ReadInput is called
		InputFunc: func(ctx context.Context, prompt string) (string, error) {
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(10 * time.Second):
				return "timeout", nil
			}
		},
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

	// Wait for initialization
	time.Sleep(300 * time.Millisecond)

	// Send list_models command
	commandChan <- ui.UICommand{Type: "list_models"}

	// Wait for response
	select {
	case receivedModels = <-modelListChan:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for model list response")
	}

	// Stop test
	close(startBlocker)
	time.Sleep(50 * time.Millisecond)

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

	mockProvider := testhelpers.NewMockProvider()
	mockProvider.SetModelFunc = func(model string) error {
		mu.Lock()
		setModelCalled = true
		setModelArg = model
		mu.Unlock()
		return nil
	}

	providerFactory := func(ctx context.Context) (providermodels.Provider, error) {
		return mockProvider, nil
	}

	startBlocker := make(chan struct{})
	commandChan := make(chan ui.UICommand, 1)

	mockUI := &testhelpers.MockUI{
		StartBlocker: startBlocker,
		CommandsChan: commandChan,
		InputFunc: func(ctx context.Context, prompt string) (string, error) {
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(10 * time.Second):
				return "timeout", nil
			}
		},
	}

	deps := Dependencies{
		UI:              mockUI,
		ProviderFactory: providerFactory,
		Tools:           nil,
	}

	go func() {
		runInteractive(context.Background(), deps)
	}()

	// Wait for initialization
	time.Sleep(300 * time.Millisecond)

	// Send switch_model command
	targetModel := "gemini-1.5-flash"
	commandChan <- ui.UICommand{
		Type: "switch_model",
		Args: map[string]string{"model": targetModel},
	}

	// Give it time to process
	time.Sleep(100 * time.Millisecond)

	// Stop test
	close(startBlocker)
	time.Sleep(50 * time.Millisecond)

	// Verify provider.SetModel was called
	mu.Lock()
	called := setModelCalled
	modelArg := setModelArg
	mu.Unlock()

	assert.True(t, called, "provider.SetModel() should have been called")
	assert.Equal(t, targetModel, modelArg,
		"provider.SetModel() should be called with correct model name")
}
