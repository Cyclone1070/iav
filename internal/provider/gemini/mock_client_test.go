package gemini

import (
	"context"
	"errors"

	"google.golang.org/genai"
)

// MockGeminiClient is a mock implementation of GeminiClient for testing.
type MockGeminiClient struct {
	GenerateContentFunc func(ctx context.Context, model string, contents []*genai.Content, config *genai.GenerateContentConfig) (*genai.GenerateContentResponse, error)
	CountTokensFunc     func(ctx context.Context, model string, contents []*genai.Content, config *genai.CountTokensConfig) (*genai.CountTokensResponse, error)
}

// GenerateContent calls the mock function if set, otherwise returns an error.
func (m *MockGeminiClient) GenerateContent(ctx context.Context, model string, contents []*genai.Content, config *genai.GenerateContentConfig) (*genai.GenerateContentResponse, error) {
	if m.GenerateContentFunc != nil {
		return m.GenerateContentFunc(ctx, model, contents, config)
	}
	return nil, errors.New("GenerateContentFunc not set")
}

// CountTokens calls the mock function if set, otherwise returns an error.
func (m *MockGeminiClient) CountTokens(ctx context.Context, model string, contents []*genai.Content, config *genai.CountTokensConfig) (*genai.CountTokensResponse, error) {
	if m.CountTokensFunc != nil {
		return m.CountTokensFunc(ctx, model, contents, config)
	}
	return nil, errors.New("CountTokensFunc not set")
}
