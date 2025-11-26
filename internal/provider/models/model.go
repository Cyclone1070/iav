package models

import (
	"github.com/Cyclone1070/deployforme/internal/orchestrator/models"
)

// GenerateRequest encapsulates all parameters for a generation request.
type GenerateRequest struct {
	// Prompt is the user's input for this turn
	Prompt string

	// History contains the conversation history
	History []models.Message

	// Config contains optional generation parameters
	Config *GenerateConfig

	// Tools contains tool definitions for native tool calling
	Tools []ToolDefinition
}

// GenerateConfig contains optional generation parameters.
// All fields are pointers to distinguish between "not set" and "zero value".
type GenerateConfig struct {
	// Generation parameters
	Temperature   *float32
	TopP          *float32
	TopK          *int
	StopSequences []string
}

// GenerateResponse contains the model's response and metadata.
type GenerateResponse struct {
	// Content contains the generated response
	Content ResponseContent

	// Metadata contains information about the generation
	Metadata ResponseMetadata
}

// ResponseContent is a union type representing different response types.
type ResponseContent struct {
	// Type indicates what the model produced
	Type ResponseType

	// For Type = ResponseTypeText
	Text string

	// For Type = ResponseTypeToolCall
	ToolCalls []models.ToolCall

	// For Type = ResponseTypeRefusal (safety block, policy violation)
	RefusalReason string
}

// ResponseType indicates the type of response from the model.
type ResponseType string

const (
	ResponseTypeText     ResponseType = "text"
	ResponseTypeToolCall ResponseType = "tool_call"
	ResponseTypeRefusal  ResponseType = "refusal"
)

// ResponseMetadata contains information about the generation.
type ResponseMetadata struct {
	// Token usage
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int

	// Cost estimation (per 1M tokens)
	EstimatedCost *float64

	// Model used
	ModelUsed string

	// Performance
	LatencyMs int64
}

// ResponseStream provides access to streaming response chunks.
type ResponseStream interface {
	// Next returns the next chunk, or io.EOF when done
	Next() (*StreamChunk, error)

	// Close releases resources
	Close() error
}

// StreamChunk represents a single chunk in a streaming response.
type StreamChunk struct {
	// Delta is the incremental text
	Delta string

	// ToolCallDelta is the incremental tool call (if applicable)
	ToolCallDelta *models.ToolCall

	// Done indicates this is the final chunk
	Done bool
}

// ToolDefinition defines a tool that the model can invoke.
type ToolDefinition struct {
	Name        string
	Description string
	Parameters  *ParameterSchema // Pointer to allow nil (no params)
}

// ParameterSchema maps directly to standard JSON Schema.
type ParameterSchema struct {
	Type       string                    `json:"type"`
	Properties map[string]PropertySchema `json:"properties"`
	Required   []string                  `json:"required,omitempty"`
}

// PropertySchema defines a single parameter property.
type PropertySchema struct {
	Type        string          `json:"type"`
	Description string          `json:"description,omitempty"`
	Enum        []string        `json:"enum,omitempty"`
	Items       *PropertySchema `json:"items,omitempty"`
}

// Capabilities describes what features a provider supports.
type Capabilities struct {
	// Feature support
	SupportsStreaming   bool
	SupportsToolCalling bool
	SupportsJSONMode    bool

	// Model limits
	MaxContextTokens int
	MaxOutputTokens  int
}
