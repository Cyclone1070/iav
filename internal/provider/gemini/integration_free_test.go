//go:build integration

package gemini

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/genai"
)

func TestGeminiProvider_FreeAPI_ListModels(t *testing.T) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		t.Skip("GEMINI_API_KEY not set, skipping free API test")
	}

	// Create real Gemini client
	genaiClient, err := genai.NewClient(context.Background(), &genai.ClientConfig{APIKey: apiKey})
	assert.NoError(t, err)
	// defer genaiClient.Close() // Client doesn't have Close method

	geminiClient := NewRealGeminiClient(genaiClient)

	// Test NewGeminiProviderWithLatest
	provider, err := NewGeminiProviderWithLatest(context.Background(), geminiClient)
	assert.NoError(t, err)

	// Verify it selected a model
	model := provider.GetModel()
	assert.NotEmpty(t, model)
	assert.True(t, strings.HasPrefix(model, "gemini-"), "Expected model to start with 'gemini-', got %q", model)
	assert.False(t, strings.HasPrefix(model, "models/"), "Expected model to NOT have 'models/' prefix, got %q", model)
	t.Logf("Selected latest model: %s", model)

	// List models (free API call)
	models, err := provider.ListModels(context.Background())
	assert.NoError(t, err)

	// Should return some models (all should be gemini-* models now)
	assert.NotEmpty(t, models)

	// All models should be gemini models (filtered, without models/ prefix)
	for _, m := range models {
		assert.True(t, strings.HasPrefix(m, "gemini-"), "Expected all models to start with 'gemini-', got %q", m)
		assert.False(t, strings.HasPrefix(m, "models/"), "Expected models to NOT have 'models/' prefix, got %q", m)
		t.Logf("Found gemini model: %s", m)
	}
}

// TestGeminiProvider_FreeAPI_NamingConventions verifies that Google's naming conventions haven't changed.
// This test will fail if Google changes their model naming patterns, alerting us to update our code.
func TestGeminiProvider_FreeAPI_NamingConventions(t *testing.T) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		t.Skip("GEMINI_API_KEY not set, skipping naming conventions test")
	}

	// Create real Gemini client
	genaiClient, err := genai.NewClient(context.Background(), &genai.ClientConfig{APIKey: apiKey})
	assert.NoError(t, err)

	geminiClient := NewRealGeminiClient(genaiClient)

	// Get all models directly from client (before filtering)
	var allModels []string
	for model, err := range genaiClient.Models.All(context.Background()) {
		if err != nil {
			t.Fatalf("Failed to list models: %v", err)
		}
		if strings.HasPrefix(model.Name, "models/gemini-") {
			allModels = append(allModels, model.Name)
		}
	}
	assert.NotEmpty(t, allModels, "Should have at least one gemini model")

	// Verify naming conventions
	hasProModel := false
	hasFlashModel := false
	hasLatestModel := false
	hasEmbeddingModel := false
	hasImageModel := false
	hasAudioModel := false
	hasLiveModel := false
	hasRoboticModel := false

	for _, m := range allModels {
		if strings.Contains(m, "-pro") {
			hasProModel = true
		}
		if strings.Contains(m, "-flash") {
			hasFlashModel = true
		}
		if strings.Contains(m, "-latest") {
			hasLatestModel = true
		}
		if strings.Contains(m, "embedding") {
			hasEmbeddingModel = true
		}
		if strings.Contains(m, "image") {
			hasImageModel = true
		}
		if strings.Contains(m, "audio") {
			hasAudioModel = true
		}
		if strings.Contains(m, "live") {
			hasLiveModel = true
		}
		if strings.Contains(m, "robotic") {
			hasRoboticModel = true
		}
	}

	// These assertions verify Google's naming conventions haven't changed
	assert.True(t, hasProModel, "Expected at least one model with '-pro' suffix. Got models: %v", allModels)
	assert.True(t, hasFlashModel, "Expected at least one model with '-flash' suffix. Got models: %v", allModels)
	assert.True(t, hasLatestModel, "Expected at least one model with '-latest' suffix. Got models: %v", allModels)

	// Verify embedding/image/audio/live/robotic models exist (they should be filtered out by our ListModels)
	if !hasEmbeddingModel {
		t.Log("Note: No embedding models found in API response (this is fine)")
	}
	if !hasImageModel {
		t.Log("Note: No image models found in API response (this is fine)")
	}
	if !hasAudioModel {
		t.Log("Note: No audio models found in API response (this is fine)")
	}
	if !hasLiveModel {
		t.Log("Note: No live models found in API response (this is fine)")
	}
	if !hasRoboticModel {
		t.Log("Note: No robotic models found in API response (this is fine)")
	}

	// Verify our filtering works correctly
	provider, err := NewGeminiProviderWithLatest(context.Background(), geminiClient)
	assert.NoError(t, err)

	filteredModels, err := provider.ListModels(context.Background())
	assert.NoError(t, err)

	// Verify no embedding/image/audio/live/robotic models in filtered list
	for _, m := range filteredModels {
		assert.False(t, strings.Contains(m, "embedding"),
			"Embedding models should be filtered out, but found: %s", m)
		assert.False(t, strings.Contains(m, "image"),
			"Image models should be filtered out, but found: %s", m)
		assert.False(t, strings.Contains(m, "audio"),
			"Audio models should be filtered out, but found: %s", m)
		assert.False(t, strings.Contains(m, "live"),
			"Live models should be filtered out, but found: %s", m)
		assert.False(t, strings.Contains(m, "robotic"),
			"Robotic models should be filtered out, but found: %s", m)
	}
}
