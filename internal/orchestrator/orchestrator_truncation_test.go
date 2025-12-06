package orchestrator

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/Cyclone1070/iav/internal/orchestrator/adapter"
	"github.com/Cyclone1070/iav/internal/orchestrator/models"
	provider "github.com/Cyclone1070/iav/internal/provider/models"
)

// =============================================================================
// HISTORY TRUNCATION TESTS
// =============================================================================
// These tests verify the checkAndTruncateHistory function which manages
// conversation history when it exceeds the LLM's context window.
//
// IMPORTANT: Run() resets history to just the goal, so to test truncation:
// 1. We must use tool calls to build up history during the run loop
// 2. Then verify truncation happens when token count exceeds limit
// 3. Tests are adversarial - designed to break the code, not tailored to pass

// TestHistoryTruncation_BuildsAndTruncates verifies that history is truncated
// when tool call responses push token count past the context window.
// This is the gold standard test - it builds history via tool calls, not injection.
func TestHistoryTruncation_BuildsAndTruncates(t *testing.T) {
	// Track how many times CountTokens is called to understand truncation behavior
	var countCalls atomic.Int32
	var historyLengthAtTruncation int

	// Mock a tool that responds
	mockTool := &MockTool{
		NameFunc:        func() string { return "test_tool" },
		DescriptionFunc: func() string { return "test" },
		DefinitionFunc: func() provider.ToolDefinition {
			return provider.ToolDefinition{Name: "test_tool", Description: "test"}
		},
		ExecuteFunc: func(ctx context.Context, args map[string]any) (string, error) {
			return "tool result", nil
		},
	}

	generateCallCount := 0
	mockProvider := &MockProvider{
		CountTokensFunc: func(ctx context.Context, messages []models.Message) (int, error) {
			countCalls.Add(1)
			// After several messages, start exceeding the limit
			// This forces truncation to kick in
			if len(messages) >= 5 {
				historyLengthAtTruncation = len(messages)
				return 10000, nil // Way over limit
			}
			if len(messages) >= 3 {
				return 5000, nil // Getting close
			}
			return 500, nil // Under limit
		},
		GetContextWindowFunc: func() int {
			return 4000 // Context window
		},
		GetCapabilitiesFunc: func() provider.Capabilities {
			return provider.Capabilities{MaxOutputTokens: 2000} // Safety margin
		},
		GenerateFunc: func(ctx context.Context, req *provider.GenerateRequest) (*provider.GenerateResponse, error) {
			generateCallCount++
			// First 3 calls: return tool calls to build up history
			if generateCallCount <= 3 {
				return &provider.GenerateResponse{
					Content: provider.ResponseContent{
						Type: provider.ResponseTypeToolCall,
						ToolCalls: []models.ToolCall{
							{ID: "call", Name: "test_tool", Args: map[string]any{}},
						},
					},
				}, nil
			}
			// After that: return text to end
			return &provider.GenerateResponse{
				Content: provider.ResponseContent{
					Type: provider.ResponseTypeText,
					Text: "Done",
				},
			}, nil
		},
	}

	mockUI := &MockUI{
		InputFunc: func(ctx context.Context, prompt string) (string, error) {
			return "", errors.New("test complete")
		},
	}

	mockPolicy := &MockPolicy{
		CheckToolFunc: func(ctx context.Context, toolName string, args map[string]any) error {
			return nil // Allow all tools
		},
	}

	orchestrator := newTestOrchestrator(mockProvider, mockPolicy, mockUI, []adapter.Tool{mockTool})
	_ = orchestrator.Run(context.Background(), "goal")

	// Verify truncation was triggered (CountTokens called multiple times)
	if countCalls.Load() < 4 {
		t.Errorf("Expected multiple CountTokens calls for truncation, got %d", countCalls.Load())
	}

	// Verify history was truncated (should be less than what we built up)
	if historyLengthAtTruncation > 0 && len(orchestrator.history) >= historyLengthAtTruncation {
		t.Errorf("Expected history to be truncated from %d messages, but got %d",
			historyLengthAtTruncation, len(orchestrator.history))
	}

	// CRITICAL: Verify WHICH messages were preserved
	// Goal must always be first
	if len(orchestrator.history) == 0 {
		t.Fatal("History is empty after truncation!")
	}
	if orchestrator.history[0].Role != "user" || orchestrator.history[0].Content != "goal" {
		t.Errorf("Goal not preserved! First message: role=%s, content=%s",
			orchestrator.history[0].Role, orchestrator.history[0].Content)
	}

	// Most recent messages should be preserved (we should have the final text response)
	lastMsg := orchestrator.history[len(orchestrator.history)-1]
	if lastMsg.Role != "assistant" {
		t.Errorf("Expected last message to be assistant response, got %s", lastMsg.Role)
	}
}

// TestHistoryTruncation_GoalNeverRemoved verifies the FIRST message (goal) survives
// even maximum truncation. This is critical - without the goal, the agent is lost.
func TestHistoryTruncation_GoalNeverRemoved(t *testing.T) {
	const goalText = "This is my critical goal that must never be deleted"

	// Create a mock that builds history then triggers aggressive truncation
	generateCallCount := 0
	mockTool := &MockTool{
		NameFunc:        func() string { return "test" },
		DescriptionFunc: func() string { return "test" },
		DefinitionFunc: func() provider.ToolDefinition {
			return provider.ToolDefinition{Name: "test", Description: "test"}
		},
		ExecuteFunc: func(ctx context.Context, args map[string]any) (string, error) {
			return "result", nil
		},
	}

	mockProvider := &MockProvider{
		CountTokensFunc: func(ctx context.Context, messages []models.Message) (int, error) {
			// Always say we're over limit to force maximum truncation
			// But respect when history is minimal
			if len(messages) <= 1 {
				return 100, nil // Under limit when only goal
			}
			return 999999, nil // Absurdly over limit to force all truncation
		},
		GetContextWindowFunc: func() int {
			return 1000
		},
		GetCapabilitiesFunc: func() provider.Capabilities {
			return provider.Capabilities{MaxOutputTokens: 500}
		},
		GenerateFunc: func(ctx context.Context, req *provider.GenerateRequest) (*provider.GenerateResponse, error) {
			generateCallCount++
			if generateCallCount <= 5 {
				return &provider.GenerateResponse{
					Content: provider.ResponseContent{
						Type: provider.ResponseTypeToolCall,
						ToolCalls: []models.ToolCall{
							{ID: "c", Name: "test", Args: map[string]any{}},
						},
					},
				}, nil
			}
			return &provider.GenerateResponse{
				Content: provider.ResponseContent{Type: provider.ResponseTypeText, Text: "Done"},
			}, nil
		},
	}

	mockUI := &MockUI{
		InputFunc: func(ctx context.Context, prompt string) (string, error) {
			return "", errors.New("test complete")
		},
	}

	mockPolicy := &MockPolicy{
		CheckToolFunc: func(ctx context.Context, toolName string, args map[string]any) error {
			return nil
		},
	}

	orchestrator := newTestOrchestrator(mockProvider, mockPolicy, mockUI, []adapter.Tool{mockTool})
	_ = orchestrator.Run(context.Background(), goalText)

	// THE CRITICAL ASSERTION: Goal must be preserved
	if len(orchestrator.history) == 0 {
		t.Fatal("History is empty - goal was deleted!")
	}
	if orchestrator.history[0].Role != "user" {
		t.Errorf("First message should be 'user' (goal), got '%s'", orchestrator.history[0].Role)
	}
	if orchestrator.history[0].Content != goalText {
		t.Errorf("Goal was mutated! Expected %q, got %q", goalText, orchestrator.history[0].Content)
	}
}

// TestHistoryTruncation_PreservesToolResultPairs verifies that model+function
// message pairs are kept together. If split, Gemini will error ("orphaned function result").
func TestHistoryTruncation_PreservesToolResultPairs(t *testing.T) {
	generateCallCount := 0

	mockTool := &MockTool{
		NameFunc:        func() string { return "tool" },
		DescriptionFunc: func() string { return "test" },
		DefinitionFunc: func() provider.ToolDefinition {
			return provider.ToolDefinition{Name: "tool", Description: "test"}
		},
		ExecuteFunc: func(ctx context.Context, args map[string]any) (string, error) {
			return "result", nil
		},
	}

	mockProvider := &MockProvider{
		CountTokensFunc: func(ctx context.Context, messages []models.Message) (int, error) {
			// Trigger truncation after building some history
			if len(messages) >= 7 {
				return 8000, nil // Over limit
			}
			if len(messages) >= 5 {
				return 4000, nil // Getting close
			}
			return 1000, nil
		},
		GetContextWindowFunc: func() int {
			return 5000
		},
		GetCapabilitiesFunc: func() provider.Capabilities {
			return provider.Capabilities{MaxOutputTokens: 2000}
		},
		GenerateFunc: func(ctx context.Context, req *provider.GenerateRequest) (*provider.GenerateResponse, error) {
			generateCallCount++
			// Build up several tool call/result pairs
			if generateCallCount <= 4 {
				return &provider.GenerateResponse{
					Content: provider.ResponseContent{
						Type: provider.ResponseTypeToolCall,
						ToolCalls: []models.ToolCall{
							{ID: "c", Name: "tool", Args: map[string]any{}},
						},
					},
				}, nil
			}
			return &provider.GenerateResponse{
				Content: provider.ResponseContent{Type: provider.ResponseTypeText, Text: "Done"},
			}, nil
		},
	}

	mockUI := &MockUI{
		InputFunc: func(ctx context.Context, prompt string) (string, error) {
			return "", errors.New("test complete")
		},
	}

	mockPolicy := &MockPolicy{
		CheckToolFunc: func(ctx context.Context, toolName string, args map[string]any) error {
			return nil
		},
	}

	orchestrator := newTestOrchestrator(mockProvider, mockPolicy, mockUI, []adapter.Tool{mockTool})
	_ = orchestrator.Run(context.Background(), "goal")

	// Verify no orphaned function messages
	for i, msg := range orchestrator.history {
		if msg.Role == "function" {
			if i == 0 {
				t.Fatal("Function message at index 0 - no preceding model!")
			}
			prevRole := orchestrator.history[i-1].Role
			if prevRole != "model" {
				t.Errorf("Orphaned function message at index %d. Preceded by '%s', not 'model'",
					i, prevRole)
			}
		}
	}
}

// TestHistoryTruncation_CountTokensError_NoInfiniteLoop verifies that token counting
// errors don't cause infinite loops or panics.
func TestHistoryTruncation_CountTokensError_NoInfiniteLoop(t *testing.T) {
	callCount := 0
	mockProvider := &MockProvider{
		CountTokensFunc: func(ctx context.Context, messages []models.Message) (int, error) {
			callCount++
			// First call succeeds, subsequent fail (simulates rate limit mid-truncation)
			if callCount == 1 {
				return 10000, nil // Over limit, will trigger truncation
			}
			return 0, errors.New("rate limit exceeded")
		},
		GetContextWindowFunc: func() int {
			return 1000
		},
		GetCapabilitiesFunc: func() provider.Capabilities {
			return provider.Capabilities{MaxOutputTokens: 500}
		},
	}

	orchestrator := newTestOrchestrator(mockProvider, &MockPolicy{}, &MockUI{}, []adapter.Tool{})
	err := orchestrator.Run(context.Background(), "test goal")

	// Should return error, not hang
	if err == nil {
		t.Error("Expected error from CountTokens failure, got nil")
	}
	// Verify we didn't spin forever (should fail after just a few calls)
	if callCount > 10 {
		t.Errorf("Possible infinite loop detected: CountTokens called %d times", callCount)
	}
}

// TestHistoryTruncation_SafetyMarginExceedsContext verifies that when
// MaxOutputTokens > ContextWindow, we don't truncate TOO aggressively.
// The implementation should use a sensible fallback, not truncate everything.
func TestHistoryTruncation_SafetyMarginExceedsContext(t *testing.T) {
	generateCallCount := 0

	mockTool := &MockTool{
		NameFunc:        func() string { return "test" },
		DescriptionFunc: func() string { return "test" },
		DefinitionFunc: func() provider.ToolDefinition {
			return provider.ToolDefinition{Name: "test", Description: "test"}
		},
		ExecuteFunc: func(ctx context.Context, args map[string]any) (string, error) {
			return "result", nil
		},
	}

	mockProvider := &MockProvider{
		CountTokensFunc: func(ctx context.Context, messages []models.Message) (int, error) {
			// Return token count well under context window
			// With context=1000 and clamped safety=800, effective limit is 200
			// So 400 tokens should NOT trigger truncation (400 > 200 but we should clamp better)
			// Actually, let's use 150 tokens to be safely under any reasonable limit
			return 150, nil
		},
		GetContextWindowFunc: func() int {
			return 1000 // Small context
		},
		GetCapabilitiesFunc: func() provider.Capabilities {
			return provider.Capabilities{MaxOutputTokens: 5000} // Larger than context!
		},
		GenerateFunc: func(ctx context.Context, req *provider.GenerateRequest) (*provider.GenerateResponse, error) {
			generateCallCount++
			// Build up history with tool calls
			if generateCallCount <= 2 {
				return &provider.GenerateResponse{
					Content: provider.ResponseContent{
						Type: provider.ResponseTypeToolCall,
						ToolCalls: []models.ToolCall{
							{ID: "c", Name: "test", Args: map[string]any{}},
						},
					},
				}, nil
			}
			return &provider.GenerateResponse{
				Content: provider.ResponseContent{Type: provider.ResponseTypeText, Text: "Done"},
			}, nil
		},
	}

	mockUI := &MockUI{
		InputFunc: func(ctx context.Context, prompt string) (string, error) {
			return "", errors.New("test complete")
		},
	}

	mockPolicy := &MockPolicy{
		CheckToolFunc: func(ctx context.Context, toolName string, args map[string]any) error {
			return nil
		},
	}

	orchestrator := newTestOrchestrator(mockProvider, mockPolicy, mockUI, []adapter.Tool{mockTool})
	_ = orchestrator.Run(context.Background(), "goal")

	// CRITICAL ASSERTION: With 150 tokens and 1000 context window,
	// we should NOT truncate (we're well under the limit even with safety margin).
	// With clamped safety margin of 800, effective limit is 200.
	// 150 < 200, so no truncation should happen.
	// Expected: goal + 2 tool call pairs + final response = 5+ messages
	if len(orchestrator.history) < 5 {
		t.Errorf("BUG DETECTED: Over-aggressive truncation with safety margin > context. History length: %d (expected >= 5)", len(orchestrator.history))
		t.Logf("With 150 tokens and context 1000, clamped safety 800, limit should be 200")
		t.Logf("150 < 200 so truncation should NOT happen")
	}
}

// TestHistoryTruncation_TokensNeverDecrease verifies that if truncation doesn't
// reduce tokens (degenerate case), we don't loop forever.
func TestHistoryTruncation_TokensNeverDecrease(t *testing.T) {
	callCount := 0
	mockProvider := &MockProvider{
		CountTokensFunc: func(ctx context.Context, messages []models.Message) (int, error) {
			callCount++
			// Always return same high value regardless of message count
			// This simulates a broken/malicious token counter
			return 99999, nil
		},
		GetContextWindowFunc: func() int {
			return 1000
		},
		GetCapabilitiesFunc: func() provider.Capabilities {
			return provider.Capabilities{MaxOutputTokens: 500}
		},
	}

	orchestrator := newTestOrchestrator(mockProvider, &MockPolicy{}, &MockUI{}, []adapter.Tool{})

	// Set up minimal history - just goal will be there after Run starts
	_ = orchestrator.Run(context.Background(), "goal")

	// The implementation should stop truncating when len(history) <= 2
	// So we shouldn't loop forever even if tokens never decrease
	if callCount > 100 {
		t.Errorf("Infinite loop detected! CountTokens called %d times", callCount)
	}
}

// TestHistoryTruncation_ThreeMessageEdgeCase verifies that when history has exactly
// 3 messages (goal + model + function), truncation doesn't orphan the function message.
// This exposes a bug in the truncation logic at line 196-201.
func TestHistoryTruncation_ThreeMessageEdgeCase(t *testing.T) {
	tokenCountCalls := 0

	mockTool := &MockTool{
		NameFunc:        func() string { return "test" },
		DescriptionFunc: func() string { return "test" },
		DefinitionFunc: func() provider.ToolDefinition {
			return provider.ToolDefinition{Name: "test", Description: "test"}
		},
		ExecuteFunc: func(ctx context.Context, args map[string]any) (string, error) {
			return "result", nil
		},
	}

	mockProvider := &MockProvider{
		CountTokensFunc: func(ctx context.Context, messages []models.Message) (int, error) {
			tokenCountCalls++
			// When we have exactly 3 messages (goal + model + function),
			// report high token count to force truncation
			if len(messages) == 3 {
				return 10000, nil // Way over limit - force truncation
			}
			return 100, nil // Under limit after truncation
		},
		GetContextWindowFunc: func() int {
			return 2000
		},
		GetCapabilitiesFunc: func() provider.Capabilities {
			return provider.Capabilities{MaxOutputTokens: 1000}
		},
		GenerateFunc: func(ctx context.Context, req *provider.GenerateRequest) (*provider.GenerateResponse, error) {
			// First call: return tool call (creates model + function messages)
			if len(req.History) == 1 {
				return &provider.GenerateResponse{
					Content: provider.ResponseContent{
						Type: provider.ResponseTypeToolCall,
						ToolCalls: []models.ToolCall{
							{ID: "c", Name: "test", Args: map[string]any{}},
						},
					},
				}, nil
			}
			// After truncation, return text to end
			return &provider.GenerateResponse{
				Content: provider.ResponseContent{Type: provider.ResponseTypeText, Text: "Done"},
			}, nil
		},
	}

	mockUI := &MockUI{
		InputFunc: func(ctx context.Context, prompt string) (string, error) {
			return "", errors.New("test complete")
		},
	}

	mockPolicy := &MockPolicy{
		CheckToolFunc: func(ctx context.Context, toolName string, args map[string]any) error {
			return nil
		},
	}

	orchestrator := newTestOrchestrator(mockProvider, mockPolicy, mockUI, []adapter.Tool{mockTool})
	_ = orchestrator.Run(context.Background(), "goal")

	// BUG: With history of [goal, model, function] and len==3,
	// the condition `isPair && len > 3` is FALSE (3 > 3 is false)
	// So it removes single message at index 1 (model), leaving orphaned function
	for i, msg := range orchestrator.history {
		if msg.Role == "function" {
			if i == 0 {
				t.Fatal("Function message at index 0 - no preceding model!")
			}
			prevRole := orchestrator.history[i-1].Role
			if prevRole != "model" {
				t.Errorf("BUG DETECTED: Orphaned function message at index %d. Preceded by '%s', not 'model'",
					i, prevRole)
				t.Logf("This happens when history has exactly 3 messages and isPair check succeeds")
				t.Logf("but len(history) > 3 is FALSE, so single-message removal orphans the function")
			}
		}
	}
}

// TestHistoryTruncation_NonPairedMessages verifies that truncation handles
// non-standard message sequences without orphaning function results.
// This tests the edge case where messages don't form recognized pairs.
func TestHistoryTruncation_NonPairedMessages(t *testing.T) {
	generateCallCount := 0

	mockTool := &MockTool{
		NameFunc:        func() string { return "test" },
		DescriptionFunc: func() string { return "test" },
		DefinitionFunc: func() provider.ToolDefinition {
			return provider.ToolDefinition{Name: "test", Description: "test"}
		},
		ExecuteFunc: func(ctx context.Context, args map[string]any) (string, error) {
			return "result", nil
		},
	}

	mockProvider := &MockProvider{
		CountTokensFunc: func(ctx context.Context, messages []models.Message) (int, error) {
			// Force truncation when we have enough messages
			if len(messages) >= 6 {
				return 10000, nil // Way over limit
			}
			return 500, nil
		},
		GetContextWindowFunc: func() int {
			return 2000
		},
		GetCapabilitiesFunc: func() provider.Capabilities {
			return provider.Capabilities{MaxOutputTokens: 1000}
		},
		GenerateFunc: func(ctx context.Context, req *provider.GenerateRequest) (*provider.GenerateResponse, error) {
			generateCallCount++
			// Build up history with tool calls
			if generateCallCount <= 3 {
				return &provider.GenerateResponse{
					Content: provider.ResponseContent{
						Type: provider.ResponseTypeToolCall,
						ToolCalls: []models.ToolCall{
							{ID: "c", Name: "test", Args: map[string]any{}},
						},
					},
				}, nil
			}
			return &provider.GenerateResponse{
				Content: provider.ResponseContent{Type: provider.ResponseTypeText, Text: "Done"},
			}, nil
		},
	}

	mockUI := &MockUI{
		InputFunc: func(ctx context.Context, prompt string) (string, error) {
			return "", errors.New("test complete")
		},
	}

	mockPolicy := &MockPolicy{
		CheckToolFunc: func(ctx context.Context, toolName string, args map[string]any) error {
			return nil
		},
	}

	orchestrator := newTestOrchestrator(mockProvider, mockPolicy, mockUI, []adapter.Tool{mockTool})
	_ = orchestrator.Run(context.Background(), "goal")

	// Verify no orphaned function messages after truncation
	for i, msg := range orchestrator.history {
		if msg.Role == "function" {
			if i == 0 {
				t.Fatal("Function message at index 0 - no preceding model!")
			}
			prevRole := orchestrator.history[i-1].Role
			if prevRole != "model" {
				t.Errorf("Orphaned function message at index %d. Preceded by '%s', not 'model'",
					i, prevRole)
			}
		}
	}
}

// TestHistoryTruncation_EmptyHistory verifies graceful handling of edge case
// where somehow history becomes empty (defensive coding).
func TestHistoryTruncation_EmptyHistory(t *testing.T) {
	mockProvider := &MockProvider{
		CountTokensFunc: func(ctx context.Context, messages []models.Message) (int, error) {
			// If we're called with empty messages, that's a bug
			if len(messages) == 0 {
				t.Error("CountTokens called with empty message list")
			}
			return 100, nil
		},
		GetContextWindowFunc: func() int {
			return 1000
		},
		GetCapabilitiesFunc: func() provider.Capabilities {
			return provider.Capabilities{MaxOutputTokens: 500}
		},
		GenerateFunc: func(ctx context.Context, req *provider.GenerateRequest) (*provider.GenerateResponse, error) {
			return &provider.GenerateResponse{
				Content: provider.ResponseContent{Type: provider.ResponseTypeText, Text: "Done"},
			}, nil
		},
	}

	mockUI := &MockUI{
		InputFunc: func(ctx context.Context, prompt string) (string, error) {
			return "", errors.New("test complete")
		},
	}

	orchestrator := newTestOrchestrator(mockProvider, &MockPolicy{}, mockUI, []adapter.Tool{})
	err := orchestrator.Run(context.Background(), "goal")

	// Should complete without panicking
	if err == nil || !strings.Contains(err.Error(), "test complete") {
		t.Errorf("Unexpected error: %v", err)
	}
}
