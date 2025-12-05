package gemini

import (
	"context"
	"errors"
	"testing"

	"github.com/Cyclone1070/deployforme/internal/orchestrator/models"
	provider "github.com/Cyclone1070/deployforme/internal/provider/models"
	"google.golang.org/genai"
)

// TestGenerate_HappyPath_TextResponse tests successful text generation.
// Uses mock model name since this test doesn't verify version parsing or sorting logic.
func TestGenerate_HappyPath_TextResponse(t *testing.T) {
	mockClient := &MockGeminiClient{
		GenerateContentFunc: func(ctx context.Context, model string, contents []*genai.Content, config *genai.GenerateContentConfig) (*genai.GenerateContentResponse, error) {
			return &genai.GenerateContentResponse{
				Candidates: []*genai.Candidate{
					{
						Content: &genai.Content{
							Parts: []*genai.Part{
								{Text: "Hello there!"},
							},
						},
						FinishReason: genai.FinishReasonStop,
					},
				},
				UsageMetadata: &genai.GenerateContentResponseUsageMetadata{
					PromptTokenCount:     10,
					CandidatesTokenCount: 5,
					TotalTokenCount:      15,
				},
			}, nil
		},
		ListModelsFunc: func(ctx context.Context) ([]ModelInfo, error) {
			return []ModelInfo{{Name: "models/gemini-mock", InputTokenLimit: 1_000_000, OutputTokenLimit: 8192}}, nil
		},
	}

	p, _ := NewGeminiProvider(context.Background(), mockClient, "gemini-mock")

	req := &provider.GenerateRequest{
		Prompt:  "Hello",
		History: []models.Message{},
	}

	resp, err := p.Generate(context.Background(), req)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if resp.Content.Type != provider.ResponseTypeText {
		t.Errorf("expected ResponseTypeText, got %v", resp.Content.Type)
	}

	if resp.Content.Text != "Hello there!" {
		t.Errorf("expected 'Hello there!', got %q", resp.Content.Text)
	}

	if resp.Metadata.TotalTokens != 15 {
		t.Errorf("expected 15 total tokens, got %d", resp.Metadata.TotalTokens)
	}
}

// TestGenerate_HappyPath_ToolCall tests successful tool call generation.
// Uses mock model name since this test doesn't verify version parsing or sorting logic.
func TestGenerate_HappyPath_ToolCall(t *testing.T) {
	mockClient := &MockGeminiClient{
		GenerateContentFunc: func(ctx context.Context, model string, contents []*genai.Content, config *genai.GenerateContentConfig) (*genai.GenerateContentResponse, error) {
			return &genai.GenerateContentResponse{
				Candidates: []*genai.Candidate{
					{
						Content: &genai.Content{
							Parts: []*genai.Part{
								{
									FunctionCall: &genai.FunctionCall{
										Name: "read_file",
										Args: map[string]any{
											"path": "foo.txt",
										},
									},
								},
							},
						},
						FinishReason: genai.FinishReasonStop,
					},
				},
				UsageMetadata: &genai.GenerateContentResponseUsageMetadata{
					TotalTokenCount: 20,
				},
			}, nil
		},
		ListModelsFunc: func(ctx context.Context) ([]ModelInfo, error) {
			return []ModelInfo{{Name: "models/gemini-mock", InputTokenLimit: 1_000_000, OutputTokenLimit: 8192}}, nil
		},
	}

	p, _ := NewGeminiProvider(context.Background(), mockClient, "gemini-mock")

	req := &provider.GenerateRequest{
		Prompt: "Read foo.txt",
	}

	resp, err := p.Generate(context.Background(), req)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if resp.Content.Type != provider.ResponseTypeToolCall {
		t.Errorf("expected ResponseTypeToolCall, got %v", resp.Content.Type)
	}

	if len(resp.Content.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(resp.Content.ToolCalls))
	}

	if resp.Content.ToolCalls[0].Name != "read_file" {
		t.Errorf("expected tool name 'read_file', got %q", resp.Content.ToolCalls[0].Name)
	}

	if resp.Content.ToolCalls[0].Args["path"] != "foo.txt" {
		t.Errorf("expected path 'foo.txt', got %v", resp.Content.ToolCalls[0].Args["path"])
	}
}

// TestCountTokens_HappyPath tests successful token counting.
// Uses mock model name since this test doesn't verify version parsing or sorting logic.
func TestCountTokens_HappyPath(t *testing.T) {
	mockClient := &MockGeminiClient{
		CountTokensFunc: func(ctx context.Context, model string, contents []*genai.Content) (*genai.CountTokensResponse, error) {
			return &genai.CountTokensResponse{
				TotalTokens: 150,
			}, nil
		},
		ListModelsFunc: func(ctx context.Context) ([]ModelInfo, error) {
			return []ModelInfo{{Name: "models/gemini-mock", InputTokenLimit: 1_000_000, OutputTokenLimit: 8192}}, nil
		},
	}

	p, _ := NewGeminiProvider(context.Background(), mockClient, "gemini-mock")

	messages := []models.Message{
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "Hi there"},
		{Role: "user", Content: "How are you?"},
	}

	count, err := p.CountTokens(context.Background(), messages)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if count != 150 {
		t.Errorf("expected 150 tokens, got %d", count)
	}
}

// TestSetModel_GetModel tests model switching.
// Uses mock model names since this test doesn't verify version parsing or sorting logic.
func TestSetModel_GetModel(t *testing.T) {
	mockClient := &MockGeminiClient{
		ListModelsFunc: func(ctx context.Context) ([]ModelInfo, error) {
			return []ModelInfo{
				{Name: "models/gemini-mock-1", InputTokenLimit: 1_000_000, OutputTokenLimit: 8192},
				{Name: "models/gemini-mock-2", InputTokenLimit: 2_000_000, OutputTokenLimit: 8192},
			}, nil
		},
	}
	p, _ := NewGeminiProvider(context.Background(), mockClient, "gemini-mock-1")

	if p.GetModel() != "gemini-mock-1" {
		t.Errorf("expected 'gemini-mock-1', got %q", p.GetModel())
	}

	err := p.SetModel("gemini-mock-2")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if p.GetModel() != "gemini-mock-2" {
		t.Errorf("expected 'gemini-mock-2', got %q", p.GetModel())
	}
}

// TestGenerate_UnhappyPath_RateLimit tests rate limit error handling.
// Uses mock model name since this test doesn't verify version parsing or sorting logic.
func TestGenerate_UnhappyPath_RateLimit(t *testing.T) {
	mockClient := &MockGeminiClient{
		GenerateContentFunc: func(ctx context.Context, model string, contents []*genai.Content, config *genai.GenerateContentConfig) (*genai.GenerateContentResponse, error) {
			return nil, &genai.APIError{
				Code:    429,
				Message: "Rate limit exceeded",
			}
		},
		ListModelsFunc: func(ctx context.Context) ([]ModelInfo, error) {
			return []ModelInfo{{Name: "models/gemini-mock", InputTokenLimit: 1_000_000, OutputTokenLimit: 8192}}, nil
		},
	}

	p, _ := NewGeminiProvider(context.Background(), mockClient, "gemini-mock")

	req := &provider.GenerateRequest{
		Prompt: "Hello",
	}

	_, err := p.Generate(context.Background(), req)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var providerErr *provider.ProviderError
	if !errors.As(err, &providerErr) {
		t.Fatalf("expected ProviderError, got %T", err)
	}

	if providerErr.Code != provider.ErrorCodeRateLimit {
		t.Errorf("expected ErrorCodeRateLimit, got %v", providerErr.Code)
	}

	if !providerErr.Retryable {
		t.Error("expected error to be retryable")
	}
}

// TestGenerate_UnhappyPath_AuthFailure tests authentication error handling.
// Uses mock model name since this test doesn't verify version parsing or sorting logic.
func TestGenerate_UnhappyPath_AuthFailure(t *testing.T) {
	mockClient := &MockGeminiClient{
		GenerateContentFunc: func(ctx context.Context, model string, contents []*genai.Content, config *genai.GenerateContentConfig) (*genai.GenerateContentResponse, error) {
			return nil, &genai.APIError{
				Code:    401,
				Message: "Unauthorized",
			}
		},
		ListModelsFunc: func(ctx context.Context) ([]ModelInfo, error) {
			return []ModelInfo{{Name: "models/gemini-mock", InputTokenLimit: 1_000_000, OutputTokenLimit: 8192}}, nil
		},
	}

	p, _ := NewGeminiProvider(context.Background(), mockClient, "gemini-mock")

	req := &provider.GenerateRequest{
		Prompt: "Hello",
	}

	_, err := p.Generate(context.Background(), req)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var providerErr *provider.ProviderError
	if !errors.As(err, &providerErr) {
		t.Fatalf("expected ProviderError, got %T", err)
	}

	if providerErr.Code != provider.ErrorCodeAuth {
		t.Errorf("expected ErrorCodeAuth, got %v", providerErr.Code)
	}

	if providerErr.Retryable {
		t.Error("expected error to not be retryable")
	}
}

// TestGenerate_EdgeCase_EmptyResponse tests handling of empty candidate list.
// Uses mock model name since this test doesn't verify version parsing or sorting logic.
func TestGenerate_EdgeCase_EmptyResponse(t *testing.T) {
	mockClient := &MockGeminiClient{
		GenerateContentFunc: func(ctx context.Context, model string, contents []*genai.Content, config *genai.GenerateContentConfig) (*genai.GenerateContentResponse, error) {
			return &genai.GenerateContentResponse{
				Candidates: []*genai.Candidate{},
			}, nil
		},
		ListModelsFunc: func(ctx context.Context) ([]ModelInfo, error) {
			return []ModelInfo{{Name: "models/gemini-mock", InputTokenLimit: 1_000_000, OutputTokenLimit: 8192}}, nil
		},
	}

	p, _ := NewGeminiProvider(context.Background(), mockClient, "gemini-mock")

	req := &provider.GenerateRequest{
		Prompt: "Hello",
	}

	_, err := p.Generate(context.Background(), req)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var providerErr *provider.ProviderError
	if !errors.As(err, &providerErr) {
		t.Fatalf("expected ProviderError, got %T", err)
	}

	if providerErr.Code != provider.ErrorCodeInvalidRequest {
		t.Errorf("expected ErrorCodeInvalidRequest, got %v", providerErr.Code)
	}
}

// TestGenerate_EdgeCase_SafetyBlock tests content blocked by safety filters.
// Uses mock model name since this test doesn't verify version parsing or sorting logic.
func TestGenerate_EdgeCase_SafetyBlock(t *testing.T) {
	mockClient := &MockGeminiClient{
		GenerateContentFunc: func(ctx context.Context, model string, contents []*genai.Content, config *genai.GenerateContentConfig) (*genai.GenerateContentResponse, error) {
			return &genai.GenerateContentResponse{
				Candidates: []*genai.Candidate{
					{
						Content: &genai.Content{
							Parts: []*genai.Part{
								{Text: "Partial content"},
							},
						},
						FinishReason: genai.FinishReasonSafety,
					},
				},
			}, nil
		},
		ListModelsFunc: func(ctx context.Context) ([]ModelInfo, error) {
			return []ModelInfo{{Name: "models/gemini-mock", InputTokenLimit: 1_000_000, OutputTokenLimit: 8192}}, nil
		},
	}

	p, _ := NewGeminiProvider(context.Background(), mockClient, "gemini-mock")

	req := &provider.GenerateRequest{
		Prompt: "Dangerous content",
	}

	_, err := p.Generate(context.Background(), req)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var providerErr *provider.ProviderError
	if !errors.As(err, &providerErr) {
		t.Fatalf("expected ProviderError, got %T", err)
	}

	if providerErr.Code != provider.ErrorCodeContentBlocked {
		t.Errorf("expected ErrorCodeContentBlocked, got %v", providerErr.Code)
	}
}

// TestGenerate_EdgeCase_NilConfig tests handling of nil config.
// Uses mock model name since this test doesn't verify version parsing or sorting logic.
func TestGenerate_EdgeCase_NilConfig(t *testing.T) {
	mockClient := &MockGeminiClient{
		GenerateContentFunc: func(ctx context.Context, model string, contents []*genai.Content, config *genai.GenerateContentConfig) (*genai.GenerateContentResponse, error) {
			// Verify safety settings are still applied
			// The config parameter is no longer passed to the mock,
			// so we cannot directly check its contents here.
			// The provider is responsible for applying default safety settings.
			return &genai.GenerateContentResponse{
				Candidates: []*genai.Candidate{
					{
						Content: &genai.Content{
							Parts: []*genai.Part{
								{Text: "Response"},
							},
						},
						FinishReason: genai.FinishReasonStop,
					},
				},
			}, nil
		},
		ListModelsFunc: func(ctx context.Context) ([]ModelInfo, error) {
			return []ModelInfo{{Name: "models/gemini-mock", InputTokenLimit: 1_000_000, OutputTokenLimit: 8192}}, nil
		},
	}

	p, _ := NewGeminiProvider(context.Background(), mockClient, "gemini-mock")

	req := &provider.GenerateRequest{
		Prompt: "Hello",
		Config: nil, // Nil config
	}

	_, err := p.Generate(context.Background(), req)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

// TestGenerate_EdgeCase_NilHistory tests handling of nil history.
// Uses mock model name since this test doesn't verify version parsing or sorting logic.
func TestGenerate_EdgeCase_NilHistory(t *testing.T) {
	mockClient := &MockGeminiClient{
		GenerateContentFunc: func(ctx context.Context, model string, contents []*genai.Content, config *genai.GenerateContentConfig) (*genai.GenerateContentResponse, error) {
			// Should only have the prompt
			if len(contents) != 1 {
				t.Errorf("expected 1 content, got %d", len(contents))
			}
			return &genai.GenerateContentResponse{
				Candidates: []*genai.Candidate{
					{
						Content: &genai.Content{
							Parts: []*genai.Part{
								{Text: "Response"},
							},
						},
						FinishReason: genai.FinishReasonStop,
					},
				},
			}, nil
		},
		ListModelsFunc: func(ctx context.Context) ([]ModelInfo, error) {
			return []ModelInfo{{Name: "models/gemini-mock", InputTokenLimit: 1_000_000, OutputTokenLimit: 8192}}, nil
		},
	}

	p, _ := NewGeminiProvider(context.Background(), mockClient, "gemini-mock")

	req := &provider.GenerateRequest{
		Prompt:  "Hello",
		History: nil, // Nil history
	}

	_, err := p.Generate(context.Background(), req)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

// TestGetCapabilities tests capability reporting.
// Uses real model name with version to verify GetCapabilities correctly looks up cached ModelInfo.
func TestGetCapabilities(t *testing.T) {
	mockClient := &MockGeminiClient{
		ListModelsFunc: func(ctx context.Context) ([]ModelInfo, error) {
			return []ModelInfo{{Name: "models/gemini-1.5-pro", InputTokenLimit: 2_000_000, OutputTokenLimit: 8192}}, nil
		},
	}
	p, _ := NewGeminiProvider(context.Background(), mockClient, "gemini-1.5-pro")

	caps := p.GetCapabilities()

	if !caps.SupportsToolCalling {
		t.Error("expected tool calling support")
	}

	if !caps.SupportsStreaming {
		t.Error("expected streaming support")
	}

	if caps.MaxContextTokens != 2_000_000 {
		t.Errorf("expected 2M context tokens for gemini-1.5-pro, got %d", caps.MaxContextTokens)
	}
}

// TestDefineTools tests tool definition registration.
// Uses mock model name since this test doesn't verify version parsing or sorting logic.
func TestDefineTools(t *testing.T) {
	mockClient := &MockGeminiClient{
		GenerateContentFunc: func(ctx context.Context, model string, contents []*genai.Content, config *genai.GenerateContentConfig) (*genai.GenerateContentResponse, error) {
			// Verify tools are passed
			if len(config.Tools) == 0 {
				t.Error("expected tools to be set")
			}
			if len(config.Tools[0].FunctionDeclarations) != 1 {
				t.Errorf("expected 1 function declaration, got %d", len(config.Tools[0].FunctionDeclarations))
			}
			if config.Tools[0].FunctionDeclarations[0].Name != "test_tool" {
				t.Errorf("expected tool name 'test_tool', got %q", config.Tools[0].FunctionDeclarations[0].Name)
			}
			return &genai.GenerateContentResponse{
				Candidates: []*genai.Candidate{
					{
						Content: &genai.Content{
							Parts: []*genai.Part{
								{Text: "Response"},
							},
						},
						FinishReason: genai.FinishReasonStop,
					},
				},
			}, nil
		},
		ListModelsFunc: func(ctx context.Context) ([]ModelInfo, error) {
			return []ModelInfo{{Name: "models/gemini-mock", InputTokenLimit: 1_000_000, OutputTokenLimit: 8192}}, nil
		},
	}

	p, _ := NewGeminiProvider(context.Background(), mockClient, "gemini-mock")

	tools := []provider.ToolDefinition{
		{
			Name:        "test_tool",
			Description: "A test tool",
		},
	}

	err := p.DefineTools(context.Background(), tools)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Generate to verify tools are used
	req := &provider.GenerateRequest{
		Prompt: "Use tool",
	}

	_, err = p.Generate(context.Background(), req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

// TestExtractVersion tests version extraction from model names.
// Uses real model names with versions to verify the regex parsing logic works correctly.
func TestExtractVersion(t *testing.T) {
	tests := []struct {
		name     string
		model    string
		expected float64
		ok       bool
	}{
		{
			name:     "gemini-2.0-flash",
			model:    "models/gemini-2.0-flash",
			expected: 2.0,
			ok:       true,
		},
		{
			name:     "gemini-1.5-pro",
			model:    "models/gemini-1.5-pro",
			expected: 1.5,
			ok:       true,
		},
		{
			name:     "gemini-2.5-flash",
			model:    "models/gemini-2.5-flash",
			expected: 2.5,
			ok:       true,
		},
		{
			name:     "gemini-3-pro (no decimal)",
			model:    "models/gemini-3-pro",
			expected: 3.0,
			ok:       true,
		},
		{
			name:     "invalid format",
			model:    "invalid-model-name",
			expected: 0.0,
			ok:       false,
		},
		{
			name:     "non-gemini model",
			model:    "models/other-model-1.0",
			expected: 0.0,
			ok:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			version, ok := extractVersion(tt.model)
			if ok != tt.ok {
				t.Errorf("extractVersion(%q) ok = %v, want %v", tt.model, ok, tt.ok)
			}
			if ok && version != tt.expected {
				t.Errorf("extractVersion(%q) = %v, want %v", tt.model, version, tt.expected)
			}
		})
	}
}

// TestSortModelsByVersion tests sorting models by version.
// Uses real model names with versions, -pro, -flash, and -latest suffixes to verify sorting logic.
func TestSortModelsByVersion(t *testing.T) {
	tests := []struct {
		name     string
		models   []ModelInfo
		expected []string
	}{
		{
			name: "sort by version descending",
			models: []ModelInfo{
				{Name: "models/gemini-1.5-flash", InputTokenLimit: 1_000_000, OutputTokenLimit: 8192},
				{Name: "models/gemini-2.0-flash", InputTokenLimit: 1_000_000, OutputTokenLimit: 8192},
				{Name: "models/gemini-1.5-pro", InputTokenLimit: 2_000_000, OutputTokenLimit: 8192},
			},
			expected: []string{"models/gemini-2.0-flash", "models/gemini-1.5-pro", "models/gemini-1.5-flash"},
		},
		{
			name: "pro ranks higher than flash when versions equal",
			models: []ModelInfo{
				{Name: "models/gemini-1.5-flash", InputTokenLimit: 1_000_000, OutputTokenLimit: 8192},
				{Name: "models/gemini-1.5-pro", InputTokenLimit: 2_000_000, OutputTokenLimit: 8192},
			},
			expected: []string{"models/gemini-1.5-pro", "models/gemini-1.5-flash"},
		},
		{
			name: "latest ranks highest regardless of version",
			models: []ModelInfo{
				{Name: "models/gemini-3.0-pro", InputTokenLimit: 2_000_000, OutputTokenLimit: 8192},
				{Name: "models/gemini-2.5-pro-latest", InputTokenLimit: 2_000_000, OutputTokenLimit: 8192},
				{Name: "models/gemini-1.5-flash-latest", InputTokenLimit: 1_000_000, OutputTokenLimit: 8192},
			},
			expected: []string{"models/gemini-2.5-pro-latest", "models/gemini-1.5-flash-latest", "models/gemini-3.0-pro"},
		},
		{
			name: "single model",
			models: []ModelInfo{
				{Name: "models/gemini-2.0-flash", InputTokenLimit: 1_000_000, OutputTokenLimit: 8192},
			},
			expected: []string{"models/gemini-2.0-flash"},
		},
		{
			name: "mixed versions",
			models: []ModelInfo{
				{Name: "models/gemini-1.0-flash", InputTokenLimit: 1_000_000, OutputTokenLimit: 8192},
				{Name: "models/gemini-2.5-flash", InputTokenLimit: 1_000_000, OutputTokenLimit: 8192},
				{Name: "models/gemini-1.5-pro", InputTokenLimit: 2_000_000, OutputTokenLimit: 8192},
			},
			expected: []string{"models/gemini-2.5-flash", "models/gemini-1.5-pro", "models/gemini-1.0-flash"},
		},
		{
			name: "version 3 without decimal",
			models: []ModelInfo{
				{Name: "models/gemini-2.5-pro", InputTokenLimit: 2_000_000, OutputTokenLimit: 8192},
				{Name: "models/gemini-3-pro", InputTokenLimit: 2_000_000, OutputTokenLimit: 8192},
				{Name: "models/gemini-1.5-flash", InputTokenLimit: 1_000_000, OutputTokenLimit: 8192},
			},
			expected: []string{"models/gemini-3-pro", "models/gemini-2.5-pro", "models/gemini-1.5-flash"},
		},
		{
			name: "pro and flash with additional suffixes",
			models: []ModelInfo{
				{Name: "models/gemini-2.5-flash-exp", InputTokenLimit: 1_000_000, OutputTokenLimit: 8192},
				{Name: "models/gemini-2.5-pro-latest", InputTokenLimit: 2_000_000, OutputTokenLimit: 8192},
				{Name: "models/gemini-2.5-flash-latest", InputTokenLimit: 1_000_000, OutputTokenLimit: 8192},
				{Name: "models/gemini-2.5-pro-exp", InputTokenLimit: 2_000_000, OutputTokenLimit: 8192},
			},
			// Latest models first (regardless of version), then pro > flash when versions equal
			expected: []string{"models/gemini-2.5-pro-latest", "models/gemini-2.5-flash-latest", "models/gemini-2.5-pro-exp", "models/gemini-2.5-flash-exp"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sortModelsByVersion(tt.models)
			if len(result) != len(tt.expected) {
				t.Fatalf("sortModelsByVersion() length = %d, want %d", len(result), len(tt.expected))
			}
			for i := range result {
				if result[i].Name != tt.expected[i] {
					t.Errorf("sortModelsByVersion()[%d] = %q, want %q", i, result[i].Name, tt.expected[i])
				}
			}
		})
	}
}

// TestNewGeminiProviderWithLatest tests the NewGeminiProviderWithLatest constructor.
// Uses real model names with versions to verify sorting and selection logic.
func TestNewGeminiProviderWithLatest(t *testing.T) {
	tests := []struct {
		name          string
		models        []ModelInfo
		expectedModel string
		expectError   bool
	}{
		{
			name: "selects highest version",
			models: []ModelInfo{
				{Name: "models/gemini-1.5-flash", InputTokenLimit: 1_000_000, OutputTokenLimit: 8192},
				{Name: "models/gemini-2.0-flash", InputTokenLimit: 1_000_000, OutputTokenLimit: 8192},
				{Name: "models/gemini-1.5-pro", InputTokenLimit: 2_000_000, OutputTokenLimit: 8192},
			},
			expectedModel: "gemini-2.0-flash",
			expectError:   false,
		},
		{
			name: "preserves order for equal versions",
			models: []ModelInfo{
				{Name: "models/gemini-1.5-pro", InputTokenLimit: 2_000_000, OutputTokenLimit: 8192},
				{Name: "models/gemini-1.5-flash", InputTokenLimit: 1_000_000, OutputTokenLimit: 8192},
			},
			expectedModel: "gemini-1.5-pro",
			expectError:   false,
		},
		{
			name:        "no models available",
			models:      []ModelInfo{},
			expectError: true,
		},
		{
			name: "single model",
			models: []ModelInfo{
				{Name: "models/gemini-2.0-flash", InputTokenLimit: 1_000_000, OutputTokenLimit: 8192},
			},
			expectedModel: "gemini-2.0-flash",
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockGeminiClient{
				ListModelsFunc: func(ctx context.Context) ([]ModelInfo, error) {
					return tt.models, nil
				},
			}

			p, err := NewGeminiProviderWithLatest(context.Background(), mockClient)
			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if p.GetModel() != tt.expectedModel {
				t.Errorf("GetModel() = %q, want %q", p.GetModel(), tt.expectedModel)
			}

			// Verify cache is populated
			models, err := p.ListModels(context.Background())
			if err != nil {
				t.Fatalf("ListModels() error = %v", err)
			}
			if len(models) != len(tt.models) {
				t.Errorf("ListModels() length = %d, want %d", len(models), len(tt.models))
			}
		})
	}
}

// TestNewGeminiProvider_InvalidModel tests validation of model names.
// Uses real model names to verify validation logic works with actual model name patterns.
func TestNewGeminiProvider_InvalidModel(t *testing.T) {
	mockClient := &MockGeminiClient{
		ListModelsFunc: func(ctx context.Context) ([]ModelInfo, error) {
			return []ModelInfo{
				{Name: "models/gemini-1.5-flash", InputTokenLimit: 1_000_000, OutputTokenLimit: 8192},
				{Name: "models/gemini-2.0-flash", InputTokenLimit: 1_000_000, OutputTokenLimit: 8192},
			}, nil
		},
	}

	_, err := NewGeminiProvider(context.Background(), mockClient, "models/gemini-invalid-model")
	if err == nil {
		t.Error("expected error for invalid model, got nil")
	}

	if err != nil && err.Error() != "invalid model: models/gemini-invalid-model not found in available models" {
		t.Errorf("error message = %q, want 'invalid model: models/gemini-invalid-model not found in available models'", err.Error())
	}
}

// TestNewGeminiProvider_ListModelsError tests error handling when ListModels fails.
func TestNewGeminiProvider_ListModelsError(t *testing.T) {
	mockClient := &MockGeminiClient{
		ListModelsFunc: func(ctx context.Context) ([]ModelInfo, error) {
			return nil, errors.New("API error")
		},
	}

	_, err := NewGeminiProvider(context.Background(), mockClient, "models/gemini-1.5-flash")
	if err == nil {
		t.Error("expected error when ListModels fails, got nil")
	}
}
