package models

import (
	"context"

	"github.com/Cyclone1070/iav/internal/orchestrator/models"
)

// Provider defines the interface for LLM backends.
type Provider interface {
	// === Core Generation ===

	// Generate sends a request to the model and returns the response.
	// It may return a partial response AND an error (e.g., ErrContextLengthExceeded).
	// Callers should check if Response is non-nil even if Error is non-nil.
	Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error)

	// GenerateStream enables streaming responses for long-running generations.
	// Returns a stream that yields chunks incrementally.
	GenerateStream(ctx context.Context, req *GenerateRequest) (ResponseStream, error)

	// === Token Management ===

	// CountTokens returns the number of tokens in the provided messages.
	// This allows the orchestrator to proactively manage the context window.
	CountTokens(ctx context.Context, messages []models.Message) (int, error)

	// GetContextWindow returns the maximum context size for the current model.
	GetContextWindow() int

	// === Model Management ===

	// SetModel changes the active model at runtime.
	// Returns an error if the model is invalid or unavailable.
	SetModel(model string) error

	// GetModel returns the currently active model name.
	GetModel() string

	// ListModels returns a list of available model names.
	// Providers should implement this to allow users to discover available models.
	ListModels(ctx context.Context) ([]string, error)

	// === Capabilities ===

	// GetCapabilities returns what features the provider/model supports.
	GetCapabilities() Capabilities

	// === Native Tool Calling ===

	// DefineTools registers tool definitions with the provider for native tool calling.
	// This should be called before Generate if you want to use tools.
	DefineTools(ctx context.Context, tools []ToolDefinition) error
}
