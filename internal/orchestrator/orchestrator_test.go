package orchestrator

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/Cyclone1070/iav/internal/config"
	"github.com/Cyclone1070/iav/internal/orchestrator/adapter"
	"github.com/Cyclone1070/iav/internal/orchestrator/models"
	provider "github.com/Cyclone1070/iav/internal/provider/models"
	"github.com/Cyclone1070/iav/internal/ui"
	uimodels "github.com/Cyclone1070/iav/internal/ui/models"
)

// MockProvider implements provider.Provider for testing
type MockProvider struct {
	GenerateFunc         func(ctx context.Context, req *provider.GenerateRequest) (*provider.GenerateResponse, error)
	CountTokensFunc      func(ctx context.Context, messages []models.Message) (int, error)
	GetContextWindowFunc func() int
	SetModelFunc         func(model string) error
	GetModelFunc         func() string
	GetCapabilitiesFunc  func() provider.Capabilities
	DefineToolsFunc      func(ctx context.Context, tools []provider.ToolDefinition) error
	GenerateStreamFunc   func(ctx context.Context, req *provider.GenerateRequest) (provider.ResponseStream, error)
	ListModelsFunc       func(ctx context.Context) ([]string, error)
}

func (m *MockProvider) Generate(ctx context.Context, req *provider.GenerateRequest) (*provider.GenerateResponse, error) {
	if m.GenerateFunc != nil {
		return m.GenerateFunc(ctx, req)
	}
	return nil, errors.New("not implemented")
}

func (m *MockProvider) GenerateStream(ctx context.Context, req *provider.GenerateRequest) (provider.ResponseStream, error) {
	if m.GenerateStreamFunc != nil {
		return m.GenerateStreamFunc(ctx, req)
	}
	return nil, errors.New("not implemented")
}

func (m *MockProvider) CountTokens(ctx context.Context, messages []models.Message) (int, error) {
	if m.CountTokensFunc != nil {
		return m.CountTokensFunc(ctx, messages)
	}
	return 0, nil
}

func (m *MockProvider) GetContextWindow() int {
	if m.GetContextWindowFunc != nil {
		return m.GetContextWindowFunc()
	}
	return 1000000
}

func (m *MockProvider) SetModel(model string) error {
	if m.SetModelFunc != nil {
		return m.SetModelFunc(model)
	}
	return nil
}

func (m *MockProvider) GetModel() string {
	if m.GetModelFunc != nil {
		return m.GetModelFunc()
	}
	return "test-model"
}

func (m *MockProvider) GetCapabilities() provider.Capabilities {
	if m.GetCapabilitiesFunc != nil {
		return m.GetCapabilitiesFunc()
	}
	return provider.Capabilities{}
}

func (m *MockProvider) DefineTools(ctx context.Context, tools []provider.ToolDefinition) error {
	if m.DefineToolsFunc != nil {
		return m.DefineToolsFunc(ctx, tools)
	}
	return nil
}

func (m *MockProvider) ListModels(ctx context.Context) ([]string, error) {
	if m.ListModelsFunc != nil {
		return m.ListModelsFunc(ctx)
	}
	return []string{"test-model"}, nil
}

// MockTool implements adapter.Tool for testing
type MockTool struct {
	NameFunc        func() string
	DescriptionFunc func() string
	DefinitionFunc  func() provider.ToolDefinition
	ExecuteFunc     func(ctx context.Context, args map[string]any) (string, error)
}

func (m *MockTool) Name() string {
	if m.NameFunc != nil {
		return m.NameFunc()
	}
	return "mock_tool"
}

func (m *MockTool) Description() string {
	if m.DescriptionFunc != nil {
		return m.DescriptionFunc()
	}
	return "Mock tool for testing"
}

func (m *MockTool) Definition() provider.ToolDefinition {
	if m.DefinitionFunc != nil {
		return m.DefinitionFunc()
	}
	return provider.ToolDefinition{
		Name:        m.Name(),
		Description: m.Description(),
	}
}

func (m *MockTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	if m.ExecuteFunc != nil {
		return m.ExecuteFunc(ctx, args)
	}
	return "mock result", nil
}

// MockPolicy implements models.PolicyService for testing
type MockPolicy struct {
	CheckToolFunc  func(ctx context.Context, toolName string, args map[string]any) error
	CheckShellFunc func(ctx context.Context, command []string) error
}

func (m *MockPolicy) CheckTool(ctx context.Context, toolName string, args map[string]any) error {
	if m.CheckToolFunc != nil {
		return m.CheckToolFunc(ctx, toolName, args)
	}
	return nil
}

func (m *MockPolicy) CheckShell(ctx context.Context, command []string) error {
	if m.CheckShellFunc != nil {
		return m.CheckShellFunc(ctx, command)
	}
	return nil
}

// MockUI implements ui.UserInterface for testing
type MockUI struct {
	Messages           []string
	Statuses           []string
	InputFunc          func(ctx context.Context, prompt string) (string, error)
	ReadPermissionFunc func(ctx context.Context, prompt string, preview *uimodels.ToolPreview) (ui.PermissionDecision, error)
}

func (m *MockUI) WriteMessage(message string) {
	m.Messages = append(m.Messages, message)
}

func (m *MockUI) WriteStatus(status, message string) {
	m.Statuses = append(m.Statuses, status+": "+message)
}

func (m *MockUI) ReadInput(ctx context.Context, prompt string) (string, error) {
	if m.InputFunc != nil {
		return m.InputFunc(ctx, prompt)
	}
	return "test input", nil
}

func (m *MockUI) ReadPermission(ctx context.Context, prompt string, preview *uimodels.ToolPreview) (ui.PermissionDecision, error) {
	if m.ReadPermissionFunc != nil {
		return m.ReadPermissionFunc(ctx, prompt, preview)
	}
	return ui.DecisionAllow, nil
}

func (m *MockUI) WriteModelList(models []string) {
	// No-op for tests
}

func (m *MockUI) Commands() <-chan ui.UICommand {
	// Return nil channel for tests
	return nil
}

func (m *MockUI) SetModel(model string) {
	// No-op for tests
}

func (m *MockUI) Ready() <-chan struct{} {
	// Return closed channel (always ready)
	ch := make(chan struct{})
	close(ch)
	return ch
}

func (m *MockUI) Start() error {
	return nil
}

// newTestOrchestrator creates an orchestrator with default config for testing
func newTestOrchestrator(p provider.Provider, pol models.PolicyService, ui ui.UserInterface, tools []adapter.Tool) *Orchestrator {
	return New(config.DefaultConfig(), p, pol, ui, tools)
}

// Test Case 1: Happy Path - Text Response
func TestRun_HappyPath_TextResponse(t *testing.T) {
	mockProvider := &MockProvider{
		GenerateFunc: func(ctx context.Context, req *provider.GenerateRequest) (*provider.GenerateResponse, error) {
			return &provider.GenerateResponse{
				Content: provider.ResponseContent{
					Type: provider.ResponseTypeText,
					Text: "Hello, how can I help?",
				},
			}, nil
		},
	}

	mockUI := &MockUI{
		InputFunc: func(ctx context.Context, prompt string) (string, error) {
			// Return error to exit loop after first response
			return "", errors.New("test complete")
		},
	}

	mockPolicy := &MockPolicy{}

	orchestrator := New(config.DefaultConfig(), mockProvider, mockPolicy, mockUI, []adapter.Tool{})

	err := orchestrator.Run(context.Background(), "test goal")

	// Should fail with "test complete" from ReadInput
	if err == nil || err.Error() != "failed to read user input: test complete" {
		t.Errorf("Expected 'failed to read user input: test complete', got: %v", err)
	}

	// Verify UI received the message
	if len(mockUI.Messages) != 1 || mockUI.Messages[0] != "Hello, how can I help?" {
		t.Errorf("Expected UI to receive 'Hello, how can I help?', got: %v", mockUI.Messages)
	}

	// Verify history
	if len(orchestrator.history) != 2 {
		t.Fatalf("Expected 2 messages in history, got: %d", len(orchestrator.history))
	}

	if orchestrator.history[1].Role != "assistant" || orchestrator.history[1].Content != "Hello, how can I help?" {
		t.Errorf("Expected assistant message in history, got: %+v", orchestrator.history[1])
	}
}

// Test Case 2: Happy Path - Tool Call
func TestRun_HappyPath_ToolCall(t *testing.T) {
	toolExecuted := false

	mockTool := &MockTool{
		NameFunc: func() string {
			return "test_tool"
		},
		ExecuteFunc: func(ctx context.Context, args map[string]any) (string, error) {
			toolExecuted = true
			return "tool result", nil
		},
	}

	callCount := 0
	mockProvider := &MockProvider{
		GenerateFunc: func(ctx context.Context, req *provider.GenerateRequest) (*provider.GenerateResponse, error) {
			callCount++
			if callCount == 1 {
				// First call: return tool call
				return &provider.GenerateResponse{
					Content: provider.ResponseContent{
						Type: provider.ResponseTypeToolCall,
						ToolCalls: []models.ToolCall{
							{
								ID:   "call_1",
								Name: "test_tool",
								Args: map[string]any{"arg": "value"},
							},
						},
					},
				}, nil
			}
			// Second call: return text to exit
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

	mockPolicy := &MockPolicy{}

	orchestrator := New(config.DefaultConfig(), mockProvider, mockPolicy, mockUI, []adapter.Tool{mockTool})

	_ = orchestrator.Run(context.Background(), "test goal")

	if !toolExecuted {
		t.Error("Expected tool to be executed")
	}

	// Verify history has model message with tool calls and function message with results
	if len(orchestrator.history) < 3 {
		t.Fatalf("Expected at least 3 messages in history, got: %d", len(orchestrator.history))
	}

	// Check model message
	if orchestrator.history[1].Role != "model" || len(orchestrator.history[1].ToolCalls) != 1 {
		t.Errorf("Expected model message with tool calls, got: %+v", orchestrator.history[1])
	}

	// Check function message
	if orchestrator.history[2].Role != "function" || len(orchestrator.history[2].ToolResults) != 1 {
		t.Errorf("Expected function message with tool results, got: %+v", orchestrator.history[2])
	}

	if orchestrator.history[2].ToolResults[0].Content != "tool result" {
		t.Errorf("Expected tool result content 'tool result', got: %s", orchestrator.history[2].ToolResults[0].Content)
	}
}

// Test Case 3: Multiple Tool Calls
func TestRun_MultipleToolCalls(t *testing.T) {
	tool1Executed := false
	tool2Executed := false

	mockTool1 := &MockTool{
		NameFunc: func() string {
			return "tool1"
		},
		ExecuteFunc: func(ctx context.Context, args map[string]any) (string, error) {
			tool1Executed = true
			return "result1", nil
		},
	}

	mockTool2 := &MockTool{
		NameFunc: func() string {
			return "tool2"
		},
		ExecuteFunc: func(ctx context.Context, args map[string]any) (string, error) {
			tool2Executed = true
			return "result2", nil
		},
	}

	callCount := 0
	mockProvider := &MockProvider{
		GenerateFunc: func(ctx context.Context, req *provider.GenerateRequest) (*provider.GenerateResponse, error) {
			callCount++
			if callCount == 1 {
				return &provider.GenerateResponse{
					Content: provider.ResponseContent{
						Type: provider.ResponseTypeToolCall,
						ToolCalls: []models.ToolCall{
							{ID: "call_1", Name: "tool1", Args: map[string]any{}},
							{ID: "call_2", Name: "tool2", Args: map[string]any{}},
						},
					},
				}, nil
			}
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

	mockPolicy := &MockPolicy{}

	orchestrator := New(config.DefaultConfig(), mockProvider, mockPolicy, mockUI, []adapter.Tool{mockTool1, mockTool2})

	_ = orchestrator.Run(context.Background(), "test goal")

	if !tool1Executed || !tool2Executed {
		t.Error("Expected both tools to be executed")
	}

	// Verify function message has 2 results
	if len(orchestrator.history) < 3 {
		t.Fatalf("Expected at least 3 messages, got: %d", len(orchestrator.history))
	}

	if len(orchestrator.history[2].ToolResults) != 2 {
		t.Errorf("Expected 2 tool results, got: %d", len(orchestrator.history[2].ToolResults))
	}
}

// Test Case 4: Refusal
func TestRun_Refusal(t *testing.T) {
	callCount := 0
	mockProvider := &MockProvider{
		GenerateFunc: func(ctx context.Context, req *provider.GenerateRequest) (*provider.GenerateResponse, error) {
			callCount++
			if callCount == 1 {
				return &provider.GenerateResponse{
					Content: provider.ResponseContent{
						Type:          provider.ResponseTypeRefusal,
						RefusalReason: "Safety violation",
					},
				}, nil
			}
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

	mockPolicy := &MockPolicy{}

	orchestrator := newTestOrchestrator(mockProvider, mockPolicy, mockUI, []adapter.Tool{})

	_ = orchestrator.Run(context.Background(), "test goal")

	// Verify system message was added
	if len(orchestrator.history) < 2 {
		t.Fatalf("Expected at least 2 messages, got: %d", len(orchestrator.history))
	}

	if orchestrator.history[1].Role != "system" || orchestrator.history[1].Content != "Model refused: Safety violation" {
		t.Errorf("Expected system message about refusal, got: %+v", orchestrator.history[1])
	}

	// Verify UI status
	found := slices.Contains(mockUI.Statuses, "blocked: Model refused to generate")
	if !found {
		t.Error("Expected UI to show blocked status")
	}
}

// Test Case 5: Provider Error
func TestRun_ProviderError(t *testing.T) {
	mockProvider := &MockProvider{
		GenerateFunc: func(ctx context.Context, req *provider.GenerateRequest) (*provider.GenerateResponse, error) {
			return nil, errors.New("provider error")
		},
	}

	mockUI := &MockUI{}
	mockPolicy := &MockPolicy{}

	orchestrator := newTestOrchestrator(mockProvider, mockPolicy, mockUI, []adapter.Tool{})

	err := orchestrator.Run(context.Background(), "test goal")

	if err == nil || err.Error() != "provider error: provider error" {
		t.Errorf("Expected provider error, got: %v", err)
	}
}

// Test Case 6: Max Turns Reached
func TestRun_MaxTurnsReached(t *testing.T) {
	mockProvider := &MockProvider{
		GenerateFunc: func(ctx context.Context, req *provider.GenerateRequest) (*provider.GenerateResponse, error) {
			// Always return tool call to keep looping
			return &provider.GenerateResponse{
				Content: provider.ResponseContent{
					Type: provider.ResponseTypeToolCall,
					ToolCalls: []models.ToolCall{
						{ID: "call_1", Name: "test_tool", Args: map[string]any{}},
					},
				},
			}, nil
		},
	}

	mockTool := &MockTool{
		NameFunc: func() string {
			return "test_tool"
		},
		ExecuteFunc: func(ctx context.Context, args map[string]any) (string, error) {
			return "result", nil
		},
	}

	mockUI := &MockUI{}
	mockPolicy := &MockPolicy{}

	orchestrator := newTestOrchestrator(mockProvider, mockPolicy, mockUI, []adapter.Tool{mockTool})

	err := orchestrator.Run(context.Background(), "test goal")

	if err == nil || err.Error() != "max turns (50) reached" {
		t.Errorf("Expected max turns error, got: %v", err)
	}
}

// Test Case 7: Context Cancellation
func TestRun_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	mockProvider := &MockProvider{}
	mockUI := &MockUI{}
	mockPolicy := &MockPolicy{}

	orchestrator := newTestOrchestrator(mockProvider, mockPolicy, mockUI, []adapter.Tool{})

	err := orchestrator.Run(ctx, "test goal")

	if err != context.Canceled {
		t.Errorf("Expected context.Canceled, got: %v", err)
	}
}

// Test Case 8: Empty Tool List
func TestRun_EmptyToolList(t *testing.T) {
	callCount := 0
	mockProvider := &MockProvider{
		GenerateFunc: func(ctx context.Context, req *provider.GenerateRequest) (*provider.GenerateResponse, error) {
			callCount++
			if callCount == 1 {
				return &provider.GenerateResponse{
					Content: provider.ResponseContent{
						Type:      provider.ResponseTypeToolCall,
						ToolCalls: []models.ToolCall{}, // Empty!
					},
				}, nil
			}
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

	mockPolicy := &MockPolicy{}

	orchestrator := newTestOrchestrator(mockProvider, mockPolicy, mockUI, []adapter.Tool{})

	_ = orchestrator.Run(context.Background(), "test goal")

	// Verify system message was added
	if len(orchestrator.history) < 2 {
		t.Fatalf("Expected at least 2 messages, got: %d", len(orchestrator.history))
	}

	if orchestrator.history[1].Role != "system" || orchestrator.history[1].Content != "Error: empty tool call list" {
		t.Errorf("Expected system error message, got: %+v", orchestrator.history[1])
	}
}

// Test Case 9: Unknown Tool
func TestRun_UnknownTool(t *testing.T) {
	callCount := 0
	mockProvider := &MockProvider{
		GenerateFunc: func(ctx context.Context, req *provider.GenerateRequest) (*provider.GenerateResponse, error) {
			callCount++
			if callCount == 1 {
				return &provider.GenerateResponse{
					Content: provider.ResponseContent{
						Type: provider.ResponseTypeToolCall,
						ToolCalls: []models.ToolCall{
							{ID: "call_1", Name: "unknown_tool", Args: map[string]any{}},
						},
					},
				}, nil
			}
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

	mockPolicy := &MockPolicy{}

	orchestrator := newTestOrchestrator(mockProvider, mockPolicy, mockUI, []adapter.Tool{})

	_ = orchestrator.Run(context.Background(), "test goal")

	// Verify function message has error result
	if len(orchestrator.history) < 3 {
		t.Fatalf("Expected at least 3 messages, got: %d", len(orchestrator.history))
	}

	if orchestrator.history[2].ToolResults[0].Error != "unknown tool 'unknown_tool'" {
		t.Errorf("Expected unknown tool error, got: %s", orchestrator.history[2].ToolResults[0].Error)
	}
}

// Test Case 10: Policy Denial
func TestRun_PolicyDenial(t *testing.T) {
	callCount := 0
	mockProvider := &MockProvider{
		GenerateFunc: func(ctx context.Context, req *provider.GenerateRequest) (*provider.GenerateResponse, error) {
			callCount++
			if callCount == 1 {
				return &provider.GenerateResponse{
					Content: provider.ResponseContent{
						Type: provider.ResponseTypeToolCall,
						ToolCalls: []models.ToolCall{
							{ID: "call_1", Name: "test_tool", Args: map[string]any{}},
						},
					},
				}, nil
			}
			return &provider.GenerateResponse{
				Content: provider.ResponseContent{
					Type: provider.ResponseTypeText,
					Text: "Done",
				},
			}, nil
		},
	}

	mockTool := &MockTool{
		NameFunc: func() string {
			return "test_tool"
		},
	}

	mockPolicy := &MockPolicy{
		CheckToolFunc: func(ctx context.Context, toolName string, args map[string]any) error {
			return errors.New("policy denied")
		},
	}

	mockUI := &MockUI{
		InputFunc: func(ctx context.Context, prompt string) (string, error) {
			return "", errors.New("test complete")
		},
	}

	orchestrator := newTestOrchestrator(mockProvider, mockPolicy, mockUI, []adapter.Tool{mockTool})

	_ = orchestrator.Run(context.Background(), "test goal")

	// Verify function message has error result
	if len(orchestrator.history) < 3 {
		t.Fatalf("Expected at least 3 messages, got: %d", len(orchestrator.history))
	}

	if orchestrator.history[2].ToolResults[0].Error != "policy denied: policy denied" {
		t.Errorf("Expected policy denial error, got: %s", orchestrator.history[2].ToolResults[0].Error)
	}
}

// Test Case 11: Tool Execution Error
func TestRun_ToolExecutionError(t *testing.T) {
	callCount := 0
	mockProvider := &MockProvider{
		GenerateFunc: func(ctx context.Context, req *provider.GenerateRequest) (*provider.GenerateResponse, error) {
			callCount++
			if callCount == 1 {
				return &provider.GenerateResponse{
					Content: provider.ResponseContent{
						Type: provider.ResponseTypeToolCall,
						ToolCalls: []models.ToolCall{
							{ID: "call_1", Name: "test_tool", Args: map[string]any{}},
						},
					},
				}, nil
			}
			return &provider.GenerateResponse{
				Content: provider.ResponseContent{
					Type: provider.ResponseTypeText,
					Text: "Done",
				},
			}, nil
		},
	}

	mockTool := &MockTool{
		NameFunc: func() string {
			return "test_tool"
		},
		ExecuteFunc: func(ctx context.Context, args map[string]any) (string, error) {
			return "", errors.New("tool execution failed")
		},
	}

	mockPolicy := &MockPolicy{}

	mockUI := &MockUI{
		InputFunc: func(ctx context.Context, prompt string) (string, error) {
			return "", errors.New("test complete")
		},
	}

	orchestrator := newTestOrchestrator(mockProvider, mockPolicy, mockUI, []adapter.Tool{mockTool})

	_ = orchestrator.Run(context.Background(), "test goal")

	// Verify function message has error result
	if len(orchestrator.history) < 3 {
		t.Fatalf("Expected at least 3 messages, got: %d", len(orchestrator.history))
	}

	if orchestrator.history[2].ToolResults[0].Error != "tool execution failed" {
		t.Errorf("Expected tool execution error, got: %s", orchestrator.history[2].ToolResults[0].Error)
	}
}

// Test Case 12: User Input Error
func TestRun_UserInputError(t *testing.T) {
	mockProvider := &MockProvider{
		GenerateFunc: func(ctx context.Context, req *provider.GenerateRequest) (*provider.GenerateResponse, error) {
			return &provider.GenerateResponse{
				Content: provider.ResponseContent{
					Type: provider.ResponseTypeText,
					Text: "Hello",
				},
			}, nil
		},
	}

	mockUI := &MockUI{
		InputFunc: func(ctx context.Context, prompt string) (string, error) {
			return "", errors.New("input error")
		},
	}

	mockPolicy := &MockPolicy{}

	orchestrator := newTestOrchestrator(mockProvider, mockPolicy, mockUI, []adapter.Tool{})

	err := orchestrator.Run(context.Background(), "test goal")

	if err == nil || err.Error() != "failed to read user input: input error" {
		t.Errorf("Expected user input error, got: %v", err)
	}
}
