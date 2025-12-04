package gemini

import (
	"context"
	"strings"

	"google.golang.org/genai"
)

// ModelInfo contains metadata about a Gemini model from the SDK
type ModelInfo struct {
	Name             string
	InputTokenLimit  int
	OutputTokenLimit int
}

// GeminiClient defines the interface for interacting with the Gemini API.
// This abstraction allows for easier testing and potential future implementations.
type GeminiClient interface {
	// GenerateContent sends a request to the Gemini API and returns the response
	GenerateContent(ctx context.Context, model string, contents []*genai.Content, config *genai.GenerateContentConfig) (*genai.GenerateContentResponse, error)

	// CountTokens counts the number of tokens in the given contents
	CountTokens(ctx context.Context, model string, contents []*genai.Content) (*genai.CountTokensResponse, error)

	// ListModels returns a list of available model information
	ListModels(ctx context.Context) ([]ModelInfo, error)
}

// RealGeminiClient wraps the official SDK client to satisfy GeminiClient.
type RealGeminiClient struct {
	client *genai.Client
}

// NewRealGeminiClient creates a new RealGeminiClient from an SDK client.
func NewRealGeminiClient(client *genai.Client) *RealGeminiClient {
	return &RealGeminiClient{client: client}
}

// GenerateContent calls the SDK's GenerateContent method.
func (c *RealGeminiClient) GenerateContent(ctx context.Context, model string, contents []*genai.Content, config *genai.GenerateContentConfig) (*genai.GenerateContentResponse, error) {
	return c.client.Models.GenerateContent(ctx, model, contents, config)
}

// CountTokens calls the SDK's CountTokens method.
func (c *RealGeminiClient) CountTokens(ctx context.Context, model string, contents []*genai.Content) (*genai.CountTokensResponse, error) {
	return c.client.Models.CountTokens(ctx, model, contents, nil)
}

// ListModels returns a list of available model information, filtered to only include gemini-* models
// (excluding embedding, image, audio, live, and robotic models)
func (c *RealGeminiClient) ListModels(ctx context.Context) ([]ModelInfo, error) {
	var models []ModelInfo
	for model, err := range c.client.Models.All(ctx) {
		if err != nil {
			return nil, err
		}
		// Filter to only include models starting with "models/gemini-" and exclude embedding, image, audio, live, and robotic models
		if strings.HasPrefix(model.Name, "models/gemini-") &&
			!strings.Contains(model.Name, "embedding") &&
			!strings.Contains(model.Name, "image") &&
			!strings.Contains(model.Name, "audio") &&
			!strings.Contains(model.Name, "live") &&
			!strings.Contains(model.Name, "robotic") {
			models = append(models, ModelInfo{
				Name:             model.Name,
				InputTokenLimit:  int(model.InputTokenLimit),
				OutputTokenLimit: int(model.OutputTokenLimit),
			})
		}
	}
	return models, nil
}
