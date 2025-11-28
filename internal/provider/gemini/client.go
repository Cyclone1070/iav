package gemini

import (
	"context"

	"google.golang.org/genai"
)

// GeminiClient defines the interface for interacting with the Gemini API.
// This abstraction allows for easier testing and potential future implementations.
type GeminiClient interface {
	// GenerateContent sends a request to the Gemini API and returns the response
	GenerateContent(ctx context.Context, model string, contents []*genai.Content, config *genai.GenerateContentConfig) (*genai.GenerateContentResponse, error)

	// CountTokens counts the number of tokens in the given contents
	CountTokens(ctx context.Context, model string, contents []*genai.Content) (*genai.CountTokensResponse, error)

	// ListModels returns a list of available model names
	ListModels(ctx context.Context) ([]string, error)
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

// ListModels returns a list of available model names
func (c *RealGeminiClient) ListModels(ctx context.Context) ([]string, error) {
	iter, err := c.client.Models.List(ctx, nil)
	if err != nil {
		return nil, err
	}

	var models []string
	for {
		page, err := iter.Next(ctx)
		if err != nil {
			break // End of list or error
		}
		// Each page contains multiple models in the Items field
		for _, model := range page.Items {
			models = append(models, model.Name)
		}
	}
	return models, nil
}
