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
	}

	p := New(mockClient, "gemini-1.5-flash")

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
										Args: map[string]interface{}{
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
	}

	p := New(mockClient, "gemini-1.5-flash")

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
func TestCountTokens_HappyPath(t *testing.T) {
	mockClient := &MockGeminiClient{
		CountTokensFunc: func(ctx context.Context, model string, contents []*genai.Content, config *genai.CountTokensConfig) (*genai.CountTokensResponse, error) {
			return &genai.CountTokensResponse{
				TotalTokens: 150,
			}, nil
		},
	}

	p := New(mockClient, "gemini-1.5-flash")

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
func TestSetModel_GetModel(t *testing.T) {
	mockClient := &MockGeminiClient{}
	p := New(mockClient, "gemini-1.5-flash")

	if p.GetModel() != "gemini-1.5-flash" {
		t.Errorf("expected 'gemini-1.5-flash', got %q", p.GetModel())
	}

	err := p.SetModel("gemini-1.5-pro")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if p.GetModel() != "gemini-1.5-pro" {
		t.Errorf("expected 'gemini-1.5-pro', got %q", p.GetModel())
	}
}

// TestGenerate_UnhappyPath_RateLimit tests rate limit error handling.
func TestGenerate_UnhappyPath_RateLimit(t *testing.T) {
	mockClient := &MockGeminiClient{
		GenerateContentFunc: func(ctx context.Context, model string, contents []*genai.Content, config *genai.GenerateContentConfig) (*genai.GenerateContentResponse, error) {
			return nil, &genai.APIError{
				Code:    429,
				Message: "Rate limit exceeded",
			}
		},
	}

	p := New(mockClient, "gemini-1.5-flash")

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
func TestGenerate_UnhappyPath_AuthFailure(t *testing.T) {
	mockClient := &MockGeminiClient{
		GenerateContentFunc: func(ctx context.Context, model string, contents []*genai.Content, config *genai.GenerateContentConfig) (*genai.GenerateContentResponse, error) {
			return nil, &genai.APIError{
				Code:    401,
				Message: "Unauthorized",
			}
		},
	}

	p := New(mockClient, "gemini-1.5-flash")

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
func TestGenerate_EdgeCase_EmptyResponse(t *testing.T) {
	mockClient := &MockGeminiClient{
		GenerateContentFunc: func(ctx context.Context, model string, contents []*genai.Content, config *genai.GenerateContentConfig) (*genai.GenerateContentResponse, error) {
			return &genai.GenerateContentResponse{
				Candidates: []*genai.Candidate{},
			}, nil
		},
	}

	p := New(mockClient, "gemini-1.5-flash")

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
	}

	p := New(mockClient, "gemini-1.5-flash")

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
func TestGenerate_EdgeCase_NilConfig(t *testing.T) {
	mockClient := &MockGeminiClient{
		GenerateContentFunc: func(ctx context.Context, model string, contents []*genai.Content, config *genai.GenerateContentConfig) (*genai.GenerateContentResponse, error) {
			// Verify safety settings are still applied
			if len(config.SafetySettings) == 0 {
				t.Error("expected safety settings to be set")
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
	}

	p := New(mockClient, "gemini-1.5-flash")

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
	}

	p := New(mockClient, "gemini-1.5-flash")

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
func TestGetCapabilities(t *testing.T) {
	mockClient := &MockGeminiClient{}
	p := New(mockClient, "gemini-1.5-pro")

	caps := p.GetCapabilities()

	if !caps.SupportsToolCalling {
		t.Error("expected tool calling support")
	}

	if caps.SupportsStreaming {
		t.Error("expected streaming to not be supported yet")
	}

	if caps.MaxContextTokens != 2_000_000 {
		t.Errorf("expected 2M context tokens for gemini-1.5-pro, got %d", caps.MaxContextTokens)
	}
}

// TestDefineTools tests tool definition registration.
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
	}

	p := New(mockClient, "gemini-1.5-flash")

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
