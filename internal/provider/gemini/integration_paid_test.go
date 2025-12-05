//go:build integration && llm_api

package gemini

import (
	"context"
	"os"
	"testing"

	orchmodels "github.com/Cyclone1070/iav/internal/orchestrator/models"
	"github.com/Cyclone1070/iav/internal/provider/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/genai"
)

func TestGeminiProvider_RealAPI_TextGeneration(t *testing.T) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		t.Skip("GEMINI_API_KEY not set, skipping paid API test")
	}

	// Create real Gemini client
	genaiClient, err := genai.NewClient(context.Background(), &genai.ClientConfig{APIKey: apiKey})
	assert.NoError(t, err)
	// defer genaiClient.Close() // Client doesn't have Close method

	geminiClient := NewRealGeminiClient(genaiClient)
	provider, err := NewGeminiProvider(geminiClient, "gemini-2.0-flash-exp")
	assert.NoError(t, err)

	// Generate text (short prompt to minimize cost)
	req := &models.GenerateRequest{
		Prompt: "Say hello",
		History: []orchmodels.Message{
			{Role: "user", Content: "Say hello"},
		},
	}

	resp, err := provider.Generate(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Should get a text response
	assert.Equal(t, models.ResponseTypeText, resp.Content.Type)
	assert.NotEmpty(t, resp.Content.Text)
}

func TestGeminiProvider_RealAPI_TokenCounting(t *testing.T) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		t.Skip("GEMINI_API_KEY not set, skipping paid API test")
	}

	// Create real Gemini client
	genaiClient, err := genai.NewClient(context.Background(), &genai.ClientConfig{APIKey: apiKey})
	assert.NoError(t, err)
	// defer genaiClient.Close() // Client doesn't have Close method

	geminiClient := NewRealGeminiClient(genaiClient)
	provider, err := NewGeminiProvider(geminiClient, "gemini-2.0-flash-exp")
	assert.NoError(t, err)

	// Count tokens for a short message
	history := []orchmodels.Message{
		{Role: "user", Content: "Hello world"},
	}

	tokens, err := provider.CountTokens(context.Background(), history)
	assert.NoError(t, err)
	assert.Greater(t, tokens, 0)
	assert.Less(t, tokens, 100) // Should be a small number
}

func TestGeminiProvider_MockValidation(t *testing.T) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		t.Skip("GEMINI_API_KEY not set, skipping mock validation test")
	}

	// This test validates that our MockProvider behaves similarly to the real one
	// We'll make a simple call to verify structure

	genaiClient, err := genai.NewClient(context.Background(), &genai.ClientConfig{APIKey: apiKey})
	assert.NoError(t, err)
	// defer genaiClient.Close() // Client doesn't have Close method

	geminiClient := NewRealGeminiClient(genaiClient)
	provider, err := NewGeminiProvider(geminiClient, "gemini-2.0-flash-exp")
	assert.NoError(t, err)

	// Verify capabilities match what we assume in MockProvider
	caps := provider.GetCapabilities()
	assert.NotNil(t, caps)
	assert.True(t, caps.SupportsToolCalling, "Real provider should support tool calling")

	// Verify context window is reasonable
	contextWindow := provider.GetContextWindow()
	assert.Greater(t, contextWindow, 10000, "Context window should be substantial")
}
