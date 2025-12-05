package gemini

import (
	"context" // Added for fmt.Errorf
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/Cyclone1070/deployforme/internal/orchestrator/models"
	provider "github.com/Cyclone1070/deployforme/internal/provider/models"
)

// GeminiProvider implements the Provider interface for Google Gemini.
type GeminiProvider struct {
	client     GeminiClient
	model      string // Renamed from modelName
	mu         sync.RWMutex
	tools      []provider.ToolDefinition
	modelCache []ModelInfo // Cached list of available models with metadata
}

// modelVersion represents a model name with its parsed version for sorting
type modelVersion struct {
	name    string
	version float64 // Parsed version number (e.g., 2.0, 1.5)
}

// extractVersion extracts the version number from a model name.
// Model names are in format "models/gemini-X.Y-variant" or "models/gemini-X-variant" (e.g., "models/gemini-2.0-flash" or "models/gemini-3-pro").
// Returns the version as a float64 (e.g., 2.0 or 3.0) and true if successful, or 0.0 and false if parsing fails.
func extractVersion(modelName string) (float64, bool) {
	// Pattern to match "models/gemini-X" or "models/gemini-X.Y" where X or X.Y is the version
	// This handles formats like "models/gemini-2.0-flash", "models/gemini-1.5-pro", or "models/gemini-3-pro"
	re := regexp.MustCompile(`models/gemini-(\d+(?:\.\d+)?)`)
	matches := re.FindStringSubmatch(modelName)
	if len(matches) < 2 {
		return 0.0, false
	}

	version, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return 0.0, false
	}

	return version, true
}

// getModelInfo returns the ModelInfo for the given model name from the cache
func (p *GeminiProvider) getModelInfo(name string) *ModelInfo {
	p.mu.RLock()
	defer p.mu.RUnlock()

	for i := range p.modelCache {
		if p.modelCache[i].Name == name {
			return &p.modelCache[i]
		}
	}
	return nil
}

// stripModelPrefix removes the "models/" prefix from a model name for display purposes.
// If the model doesn't have the prefix, it returns the name unchanged.
func stripModelPrefix(modelName string) string {
	if strings.HasPrefix(modelName, "models/") {
		return strings.TrimPrefix(modelName, "models/")
	}
	return modelName
}

// addModelPrefix adds the "models/" prefix to a model name if it's not already present.
// This is used when accepting user input to convert it to SDK format.
func addModelPrefix(modelName string) string {
	if strings.HasPrefix(modelName, "models/") {
		return modelName
	}
	return "models/" + modelName
}

// hasLatestSuffix returns true if the model name contains "-latest"
func hasLatestSuffix(modelName string) bool {
	return strings.Contains(modelName, "-latest")
}

// getModelTypePriority returns a priority value for model types.
// Higher values are preferred. pro=2, flash=1, others=0
// Handles models with additional suffixes like "-pro-latest", "-flash-exp", etc.
func getModelTypePriority(modelName string) int {
	// Match "-pro" or "-pro-" followed by optional suffix (e.g., "-pro-latest", "-pro-exp")
	if strings.Contains(modelName, "-pro") {
		return 2
	}
	// Match "-flash" or "-flash-" followed by optional suffix (e.g., "-flash-latest", "-flash-exp")
	if strings.Contains(modelName, "-flash") {
		return 1
	}
	return 0
}

// sortModelsByVersion sorts models with the following priority (highest to lowest):
// 1. Models with "-latest" suffix (regardless of version)
// 2. Version number (descending: 3.0 > 2.5 > 2.0 > 1.5)
// 3. Model type (pro > flash > others)
// 4. Original API order (stable sort)
// sortModelsByVersion sorts models with the following priority (highest to lowest):
// 1. Models with "-latest" suffix (regardless of version)
// 2. Version number (descending: 3.0 > 2.5 > 2.0 > 1.5)
// 3. Model type (pro > flash > others)
// 4. Original API order (stable sort)
func sortModelsByVersion(models []ModelInfo) []ModelInfo {
	// Copy to avoid mutating input
	result := make([]ModelInfo, len(models))
	copy(result, models)

	// Sort directly - O(n log n)
	slices.SortStableFunc(result, func(a, b ModelInfo) int {
		// First priority: models with "-latest" suffix (highest priority)
		aLatest := hasLatestSuffix(a.Name)
		bLatest := hasLatestSuffix(b.Name)
		if aLatest && !bLatest {
			return -1
		}
		if !aLatest && bLatest {
			return 1
		}

		// Second priority: Compare versions (descending)
		aVer, _ := extractVersion(a.Name)
		bVer, _ := extractVersion(b.Name)
		if aVer > bVer {
			return -1
		}
		if aVer < bVer {
			return 1
		}

		// Third priority: If versions are equal, prefer pro over flash
		aPri := getModelTypePriority(a.Name)
		bPri := getModelTypePriority(b.Name)
		if aPri > bPri {
			return -1
		}
		if aPri < bPri {
			return 1
		}

		return 0 // SortStable preserves original order for equal elements
	})

	return result
}

// NewGeminiProviderWithLatest creates a new Gemini provider with the latest available gemini-* model.
// It fetches the list of available models, filters to gemini-* models, sorts by version (highest first),
// and uses the first model as the default.
// NewGeminiProviderWithLatest creates a new Gemini provider with the latest available gemini-* model.
// It fetches the list of available models, filters to gemini-* models, sorts by version (highest first),
// and uses the first model as the default.
func NewGeminiProviderWithLatest(ctx context.Context, client GeminiClient) (*GeminiProvider, error) {
	models, err := client.ListModels(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list models: %w", err)
	}

	if len(models) == 0 {
		return nil, fmt.Errorf("no gemini-* models available")
	}

	// Sort models by version (highest first), preserving API order for equal versions
	sortedModels := sortModelsByVersion(models)
	latestModel := sortedModels[0].Name

	p := &GeminiProvider{
		client:     client,
		model:      latestModel,
		modelCache: sortedModels,
	}

	return p, nil
}

// NewGeminiProvider creates a new Gemini provider with the given client and model.
// It validates that the provided model exists in the filtered list of available gemini-* models.
// The model parameter can be provided with or without the "models/" prefix.
func NewGeminiProvider(ctx context.Context, client GeminiClient, model string) (*GeminiProvider, error) {
	// Add "models/" prefix if not present (user input may not have it)
	modelWithPrefix := addModelPrefix(model)

	// Fetch and validate model list
	models, err := client.ListModels(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list models: %w", err)
	}

	// Validate that the provided model exists in the filtered list
	found := false
	for _, m := range models {
		if m.Name == modelWithPrefix {
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("invalid model: %s not found in available models", model)
	}

	p := &GeminiProvider{
		client:     client,
		model:      modelWithPrefix,
		modelCache: models,
	}

	return p, nil
}

// ListModels returns a list of available model names (without "models/" prefix for display)
func (p *GeminiProvider) ListModels(ctx context.Context) ([]string, error) {
	// Return cached list if available
	if len(p.modelCache) > 0 {
		names := make([]string, len(p.modelCache))
		for i, m := range p.modelCache {
			names[i] = stripModelPrefix(m.Name)
		}
		return names, nil
	}

	// Otherwise fetch from client
	models, err := p.client.ListModels(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list models: %w", err)
	}

	// Update cache
	p.modelCache = models

	// Extract names for return (strip prefix for display)
	names := make([]string, len(models))
	for i, m := range models {
		names[i] = stripModelPrefix(m.Name)
	}
	return names, nil
}

// Generate sends a request to the Gemini API and returns the response.
func (p *GeminiProvider) Generate(ctx context.Context, req *provider.GenerateRequest) (*provider.GenerateResponse, error) {
	p.mu.RLock()
	model := p.model
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

// CountTokens counts the number of tokens in the given messages.
func (p *GeminiProvider) CountTokens(ctx context.Context, messages []models.Message) (int, error) {
	p.mu.RLock()
	model := p.model
	p.mu.RUnlock()

	// Convert messages to Gemini contents
	contents := messagesToGeminiContents(messages)

	// Call Gemini API
	resp, err := p.client.CountTokens(ctx, model, contents)
	if err != nil {
		return 0, mapGeminiError(err)
	}

	return int(resp.TotalTokens), nil
}

// GetContextWindow returns the maximum context size for the current model.
func (p *GeminiProvider) GetContextWindow() int {
	p.mu.RLock()
	model := p.model
	p.mu.RUnlock()

	info := p.getModelInfo(model)
	if info != nil {
		return info.InputTokenLimit
	}
	return 1_000_000 // fallback
}

// SetModel sets the model to use for generation
func (p *GeminiProvider) SetModel(model string) error {
	// Add "models/" prefix if not present (user input may not have it)
	modelWithPrefix := addModelPrefix(model)

	// Validate model if cache is available
	if len(p.modelCache) > 0 {
		found := false
		for _, m := range p.modelCache {
			if m.Name == modelWithPrefix {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("invalid model name: %s", model)
		}
	}
	p.mu.Lock()
	p.model = modelWithPrefix
	p.mu.Unlock()
	return nil
}

// GetModel returns the current model name (without "models/" prefix for display)
func (p *GeminiProvider) GetModel() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return stripModelPrefix(p.model)
}

// GetCapabilities returns what features the provider/model supports.
func (p *GeminiProvider) GetCapabilities() provider.Capabilities {
	p.mu.RLock()
	model := p.model
	p.mu.RUnlock()

	info := p.getModelInfo(model)
	if info == nil {
		// Fallback if model not found
		return provider.Capabilities{
			SupportsStreaming:   true,
			SupportsToolCalling: true,
			SupportsJSONMode:    true,
			MaxContextTokens:    1_000_000,
			MaxOutputTokens:     8192,
		}
	}

	return provider.Capabilities{
		SupportsStreaming:   true, // All gemini models support this via SDK
		SupportsToolCalling: true, // All gemini models support this via SDK
		SupportsJSONMode:    true,
		MaxContextTokens:    info.InputTokenLimit,
		MaxOutputTokens:     info.OutputTokenLimit,
	}
}

// DefineTools registers tool definitions with the provider for native tool calling.
func (p *GeminiProvider) DefineTools(ctx context.Context, tools []provider.ToolDefinition) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.tools = tools
	return nil
}
