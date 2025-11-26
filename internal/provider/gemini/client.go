package gemini

import (
	"context"

	"google.golang.org/genai"
)

// GeminiClient defines the interface for interacting with the Gemini API.
// This matches the signature of genai.Client.Models methods.
type GeminiClient interface {
	GenerateContent(ctx context.Context, model string, contents []*genai.Content, config *genai.GenerateContentConfig) (*genai.GenerateContentResponse, error)
	CountTokens(ctx context.Context, model string, contents []*genai.Content, config *genai.CountTokensConfig) (*genai.CountTokensResponse, error)
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
func (c *RealGeminiClient) CountTokens(ctx context.Context, model string, contents []*genai.Content, config *genai.CountTokensConfig) (*genai.CountTokensResponse, error) {
	return c.client.Models.CountTokens(ctx, model, contents, config)
}
