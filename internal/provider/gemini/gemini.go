package gemini

import (
	"context"
	"sync"

	"github.com/Cyclone1070/deployforme/internal/orchestrator/models"
	provider "github.com/Cyclone1070/deployforme/internal/provider/models"
)

// GeminiProvider implements the Provider interface for Google Gemini.
type GeminiProvider struct {
	client    GeminiClient
	modelName string
	mu        sync.RWMutex
	tools     []provider.ToolDefinition
}

// New creates a new GeminiProvider with the specified client and model.
func New(client GeminiClient, modelName string) *GeminiProvider {
	return &GeminiProvider{
		client:    client,
		modelName: modelName,
	}
}

// Generate sends a request to the Gemini API and returns the response.
func (p *GeminiProvider) Generate(ctx context.Context, req *provider.GenerateRequest) (*provider.GenerateResponse, error) {
	p.mu.RLock()
	model := p.modelName
	tools := p.tools
	p.mu.RUnlock()

	// Convert internal types to Gemini types
	contents := toGeminiContents(req.Prompt, req.History)
	config := toGeminiConfig(req.Config)

	// Add tools if defined
	if len(tools) > 0 {
		config.Tools = toGeminiTools(tools)
	}

	// Call Gemini API
	resp, err := p.client.GenerateContent(ctx, model, contents, config)
	if err != nil {
		return nil, mapGeminiError(err)
	}

	// Convert response
	return fromGeminiResponse(resp, model)
}

// GenerateStream is not yet implemented.
func (p *GeminiProvider) GenerateStream(ctx context.Context, req *provider.GenerateRequest) (provider.ResponseStream, error) {
	return nil, provider.ErrStreamingNotSupported
}

// CountTokens returns the number of tokens in the provided messages.
func (p *GeminiProvider) CountTokens(ctx context.Context, messages []models.Message) (int, error) {
	p.mu.RLock()
	model := p.modelName
	p.mu.RUnlock()

	// Convert messages to Gemini contents
	contents := messagesToGeminiContents(messages)

	// Call Gemini API
	resp, err := p.client.CountTokens(ctx, model, contents, nil)
	if err != nil {
		return 0, mapGeminiError(err)
	}

	return int(resp.TotalTokens), nil
}

// GetContextWindow returns the maximum context size for the current model.
func (p *GeminiProvider) GetContextWindow() int {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Model-specific limits
	switch p.modelName {
	case "gemini-1.5-pro", "gemini-1.5-pro-latest":
		return 2_000_000
	case "gemini-1.5-flash", "gemini-1.5-flash-latest":
		return 1_000_000
	case "gemini-2.0-flash-exp":
		return 1_000_000
	default:
		return 1_000_000 // Default to 1M
	}
}

// SetModel changes the active model at runtime.
func (p *GeminiProvider) SetModel(model string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// TODO: Validate model name
	p.modelName = model
	return nil
}

// GetModel returns the currently active model name.
func (p *GeminiProvider) GetModel() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.modelName
}

// GetCapabilities returns what features the provider/model supports.
func (p *GeminiProvider) GetCapabilities() provider.Capabilities {
	return provider.Capabilities{
		SupportsStreaming:   false, // Not yet implemented
		SupportsToolCalling: true,
		SupportsJSONMode:    true,
		MaxContextTokens:    p.GetContextWindow(),
		MaxOutputTokens:     8192, // Gemini default
	}
}

// DefineTools registers tool definitions with the provider for native tool calling.
func (p *GeminiProvider) DefineTools(ctx context.Context, tools []provider.ToolDefinition) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.tools = tools
	return nil
}
