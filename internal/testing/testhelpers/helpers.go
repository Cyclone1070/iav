// Package testhelpers provides shared utilities for integration testing
package testhelpers

import (
	"context"
	"sync"
	"testing"

	orchmodels "github.com/Cyclone1070/deployforme/internal/orchestrator/models"
	"github.com/Cyclone1070/deployforme/internal/provider/models"
	"github.com/Cyclone1070/deployforme/internal/ui"
	uimodels "github.com/Cyclone1070/deployforme/internal/ui/models"
)

// MockProvider is a controllable mock for the Gemini provider
type MockProvider struct {
	responses      []models.GenerateResponse
	responseIndex  int
	countTokensVal int
	contextWindow  int
	modelName      string
	// OnGenerateCalled is a callback for observing Generate calls
	OnGenerateCalled func(*models.GenerateRequest)

	// Callbacks for new methods
	ListModelsFunc func(ctx context.Context) ([]string, error)
	SetModelFunc   func(model string) error
}

// NewMockProvider creates a new mock provider with default settings
func NewMockProvider() *MockProvider {
	return &MockProvider{
		responses:     make([]models.GenerateResponse, 0),
		responseIndex: 0,
		contextWindow: 100000,
		modelName:     "mock-model",
	}
}

// WithTextResponse adds a text response to the queue
func (m *MockProvider) WithTextResponse(text string) *MockProvider {
	m.responses = append(m.responses, models.GenerateResponse{
		Content: models.ResponseContent{
			Type: models.ResponseTypeText,
			Text: text,
		},
	})
	return m
}

// WithToolCallResponse adds a tool call response to the queue
func (m *MockProvider) WithToolCallResponse(toolCalls []orchmodels.ToolCall) *MockProvider {
	m.responses = append(m.responses, models.GenerateResponse{
		Content: models.ResponseContent{
			Type:      models.ResponseTypeToolCall,
			ToolCalls: toolCalls,
		},
	})
	return m
}

// WithContextWindow sets the context window size
func (m *MockProvider) WithContextWindow(size int) *MockProvider {
	m.contextWindow = size
	return m
}

// Generate implements the Provider interface
func (m *MockProvider) Generate(ctx context.Context, req *models.GenerateRequest) (*models.GenerateResponse, error) {
	// Invoke callback if set
	if m.OnGenerateCalled != nil {
		m.OnGenerateCalled(req)
	}

	if m.responseIndex >= len(m.responses) {
		// Return a default text response if we run out
		return &models.GenerateResponse{
			Content: models.ResponseContent{
				Type: models.ResponseTypeText,
				Text: "Done",
			},
		}, nil
	}

	resp := m.responses[m.responseIndex]
	m.responseIndex++
	return &resp, nil
}

// GenerateStream implements the Provider interface
func (m *MockProvider) GenerateStream(ctx context.Context, req *models.GenerateRequest) (models.ResponseStream, error) {
	// Return a nil stream - not used in tests yet
	return nil, nil
}

// CountTokens implements the Provider interface
func (m *MockProvider) CountTokens(ctx context.Context, history []orchmodels.Message) (int, error) {
	if m.countTokensVal > 0 {
		return m.countTokensVal, nil
	}
	// Simple estimation: 50 tokens per message
	return len(history) * 50, nil
}

// ListModels implements the Provider interface
func (m *MockProvider) ListModels(ctx context.Context) ([]string, error) {
	if m.ListModelsFunc != nil {
		return m.ListModelsFunc(ctx)
	}
	// Default behavior
	return []string{"mock-model-1", "mock-model-2"}, nil
}

// SetModel sets the model
func (m *MockProvider) SetModel(model string) error {
	if m.SetModelFunc != nil {
		return m.SetModelFunc(model)
	}
	m.modelName = model
	return nil
}

// GetModel implements the Provider interface
func (m *MockProvider) GetModel() string {
	return m.modelName
}

// GetContextWindow implements the Provider interface
func (m *MockProvider) GetContextWindow() int {
	return m.contextWindow
}

// GetCapabilities implements the Provider interface
func (m *MockProvider) GetCapabilities() models.Capabilities {
	return models.Capabilities{
		SupportsToolCalling: true,
		SupportsStreaming:   true,
	}
}

// DefineTools implements the Provider interface
func (m *MockProvider) DefineTools(ctx context.Context, tools []models.ToolDefinition) error {
	return nil
}

// ListModels implements the Provider interface
// (Removed duplicate)

// CreateTestWorkspace creates a temporary workspace for integration tests
func CreateTestWorkspace(t *testing.T) string {
	return t.TempDir()
}

// MockUI implements ui.UserInterface for testing
type MockUI struct {
	mu                 sync.Mutex
	Messages           []string
	Statuses           []string
	InputFunc          func(ctx context.Context, prompt string) (string, error)
	ReadPermissionFunc func(ctx context.Context, prompt string, preview *uimodels.ToolPreview) (ui.PermissionDecision, error)
	ReadyChan          chan struct{}
	// OnReadyCalled is a callback for observing Ready calls
	OnReadyCalled func()
	// StartBlocker controls when Start() returns (for tests)
	StartBlocker       chan struct{}
	OnModelListWritten func(models []string)
	CommandsChan       chan ui.UICommand
	ModelList          []string
}

func (m *MockUI) WriteMessage(message string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Messages = append(m.Messages, message)
}

func (m *MockUI) WriteStatus(status, message string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Statuses = append(m.Statuses, status+": "+message)
}

func (m *MockUI) GetMessages() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Return copy
	msgs := make([]string, len(m.Messages))
	copy(msgs, m.Messages)
	return msgs
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
	m.mu.Lock()
	m.ModelList = models
	m.mu.Unlock()

	if m.OnModelListWritten != nil {
		m.OnModelListWritten(models)
	}
}

func (m *MockUI) Commands() <-chan ui.UICommand {
	if m.CommandsChan != nil {
		return m.CommandsChan
	}
	ch := make(chan ui.UICommand)
	close(ch)
	return ch
}

func (m *MockUI) SetModel(model string) {
	// No-op for tests
}

func (m *MockUI) Ready() <-chan struct{} {
	// Invoke callback if set
	if m.OnReadyCalled != nil {
		m.OnReadyCalled()
	}

	if m.ReadyChan != nil {
		return m.ReadyChan
	}
	// Return closed channel (always ready)
	ch := make(chan struct{})
	close(ch)
	return ch
}

func (m *MockUI) Start() error {
	if m.StartBlocker != nil {
		<-m.StartBlocker // Block until test signals
	}
	return nil
}
