//go:build integration

package orchestrator

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	orchadapter "github.com/Cyclone1070/iav/internal/orchestrator/adapter"
	orchmodel "github.com/Cyclone1070/iav/internal/orchestrator/model"
	pmodel "github.com/Cyclone1070/iav/internal/provider/model"
	"github.com/Cyclone1070/iav/internal/testing/mock"
	"github.com/Cyclone1070/iav/internal/tool/model"
	"github.com/Cyclone1070/iav/internal/tool/service"
	"github.com/Cyclone1070/iav/internal/ui"
	uiservices "github.com/Cyclone1070/iav/internal/ui/service"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/stretchr/testify/assert"
)

func TestOrchestratorProvider_ToolCallResponse(t *testing.T) {
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

	// Create UI
	channels := ui.NewUIChannels(nil)
	renderer := uiservices.NewGlamourRenderer()
	spinnerFactory := func() spinner.Model {
		return spinner.New(spinner.WithSpinner(spinner.Dot))
	}
	userInterface := ui.NewUI(channels, renderer, spinnerFactory)

	// Create cancellable context
	runCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Service UI input requests in background
	go func() {
		count := 0
		for {
			select {
			case req := <-channels.InputReq:
				count++
				if count > 1 {
					// Cancel context to stop the loop
					cancel()
					// Try to send exit, but don't block if context is done
					select {
					case channels.InputResp <- "exit":
					case <-runCtx.Done():
					}
				} else {
					// Send next input for continuation
					channels.InputResp <- "continue"
				}
				_ = req
			case <-runCtx.Done():
				return
			}
		}
	}()

	// Track UI messages
	messageDone := make(chan []string)
	go func() {
		var msgs []string
		for msg := range channels.MessageChan {
			msgs = append(msgs, msg)
		}
		messageDone <- msgs
	}()

	// Initialize tools
	toolList := []orchadapter.Tool{
		orchadapter.NewListDirectory(ctx),
	}

	// Track what orchestrator sends to provider
	var allHistories [][]orchmodel.Message
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
		WithTextResponse("Found 0 files")

	// Capture provider inputs
	mockProvider.OnGenerateCalled = func(req *pmodel.GenerateRequest) {
		mu.Lock()
		defer mu.Unlock()
		// Capture history
		historyCopy := make([]orchmodel.Message, len(req.History))
		copy(historyCopy, req.History)
		allHistories = append(allHistories, historyCopy)
	}

	// Create policy
	policy := &orchmodel.Policy{
		Shell: orchmodel.ShellPolicy{
			SessionAllow: make(map[string]bool),
		},
		Tools: orchmodel.ToolPolicy{
			Allow:        []string{"list_directory"},
			SessionAllow: make(map[string]bool),
		},
	}
	policyService := NewPolicyService(policy, userInterface)

	// Create orchestrator
	orch := New(nil, mockProvider, policyService, userInterface, toolList)

	// Run orchestrator
	err := orch.Run(runCtx, "List files")

	// Close channel to signal completion and get collected messages
	close(channels.MessageChan)
	messages := <-messageDone

	// Should complete with cancellation error or nil
	if err != nil && !errors.Is(err, context.Canceled) {
		t.Errorf("Run returned unexpected error: %v", err)
	}

	// Verify progression through provider calls
	mu.Lock()
	historyCount := len(allHistories)
	mu.Unlock()
	assert.GreaterOrEqual(t, historyCount, 2,
		"Provider should be called multiple times")

	// First call: user goal
	mu.Lock()
	firstHistory := allHistories[0]
	mu.Unlock()
	assert.Len(t, firstHistory, 1, "First call should have just user goal")
	assert.Equal(t, "user", firstHistory[0].Role)

	// Second call: should include model + function messages
	if historyCount >= 2 {
		mu.Lock()
		secondHistory := allHistories[1]
		mu.Unlock()
		assert.GreaterOrEqual(t, len(secondHistory), 3,
			"Second call should have goal + tool call + tool result")

		// Find model message with tool calls
		foundToolCall := false
		for i, msg := range secondHistory {
			if msg.Role == "model" && len(msg.ToolCalls) > 0 {
				foundToolCall = true
				assert.Equal(t, "list_directory", msg.ToolCalls[0].Name)

				// Next message should be function result
				if i+1 < len(secondHistory) {
					nextMsg := secondHistory[i+1]
					assert.Equal(t, "function", nextMsg.Role)
					assert.NotEmpty(t, nextMsg.ToolResults)
				}
				break
			}
		}
		assert.True(t, foundToolCall, "Should have tool call in second history")
	}

	// Check if expected message was delivered
	found := false
	for _, msg := range messages {
		if strings.Contains(msg, "Found") {
			found = true
			break
		}
	}
	assert.True(t, found, "Expected message containing 'Found' not received. Messages: %v", messages)
}

func TestOrchestratorProvider_ContextTruncation(t *testing.T) {
	t.Parallel()

	// Create small context window provider
	mockProvider := mock.NewMockProvider().
		WithContextWindow(200) // Very small window

	// Track history sent to provider
	var lastHistory []orchmodel.Message
	var mu sync.Mutex

	mockProvider.OnGenerateCalled = func(req *pmodel.GenerateRequest) {
		mu.Lock()
		defer mu.Unlock()
		lastHistory = make([]orchmodel.Message, len(req.History))
		copy(lastHistory, req.History)
	}

	// Create workspace context
	workspaceRoot := t.TempDir()
	fileSystem := service.NewOSFileSystem()
	if err := fileSystem.EnsureDirs(workspaceRoot); err != nil {
		t.Fatalf("Failed to create workspace: %v", err)
	}

	wCtx := &model.WorkspaceContext{
		WorkspaceRoot:   workspaceRoot,
		FS:              fileSystem,
		BinaryDetector:  &service.SystemBinaryDetector{SampleSize: 4096},
		ChecksumManager: service.NewChecksumManager(),
		CommandExecutor: &service.OSCommandExecutor{},
	}

	// Create UI
	channels := ui.NewUIChannels(nil)
	renderer := uiservices.NewGlamourRenderer()
	userInterface := ui.NewUI(channels, renderer, func() spinner.Model {
		return spinner.New(spinner.WithSpinner(spinner.Dot))
	})

	// Service UI
	go func() {
		for range channels.InputReq {
			channels.InputResp <- "done"
		}
	}()
	go func() {
		for range channels.MessageChan {
		}
	}()

	// Create policy
	policy := &orchmodel.Policy{
		Shell: orchmodel.ShellPolicy{SessionAllow: make(map[string]bool)},
		Tools: orchmodel.ToolPolicy{SessionAllow: make(map[string]bool)},
	}
	policyService := NewPolicyService(policy, userInterface)

	// Create orchestrator with tools
	toolList := []orchadapter.Tool{
		orchadapter.NewListDirectory(wCtx),
	}

	orch := New(nil, mockProvider, policyService, userInterface, toolList)

	// Set a clear goal message
	goalMsg := orchmodel.Message{
		Role:    "user",
		Content: "GOAL: This is the critical goal message that must be preserved.",
	}
	orch.history = append(orch.history, goalMsg)

	// Build large history to force truncation
	// Add enough messages to definitely exceed 200 tokens
	for i := 0; i < 20; i++ {
		orch.history = append(orch.history, orchmodel.Message{
			Role:    "user",
			Content: "This is filler message " + strings.Repeat("long content ", 5),
		})
		orch.history = append(orch.history, orchmodel.Message{
			Role:    "model",
			Content: "This is filler response " + strings.Repeat("long content ", 5),
		})
	}

	// Initial token count should be high
	initialTokens, err := mockProvider.CountTokens(context.Background(), orch.history)
	assert.NoError(t, err)
	assert.Greater(t, initialTokens, 200)

	// Trigger truncation via checkAndTruncateHistory
	err = orch.checkAndTruncateHistory(context.Background())
	assert.NoError(t, err)

	// Verify internal state (white-box)
	finalTokens, err := mockProvider.CountTokens(context.Background(), orch.history)
	assert.NoError(t, err)
	assert.LessOrEqual(t, finalTokens, 200)

	// Verify what provider sees (black-box via OnGenerateCalled)
	// We need to trigger a Generate call to see what the provider gets
	_, err = mockProvider.Generate(context.Background(), &pmodel.GenerateRequest{
		History: orch.history,
	})
	assert.NoError(t, err)

	mu.Lock()
	capturedHistory := lastHistory
	mu.Unlock()

	assert.NotEmpty(t, capturedHistory)
	// First message MUST be the goal
	assert.Equal(t, goalMsg.Content, capturedHistory[0].Content,
		"First message (goal) should be preserved after truncation")
	assert.Equal(t, goalMsg.Role, capturedHistory[0].Role)
}
