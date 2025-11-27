package gemini

import (
	"testing"

	"github.com/Cyclone1070/deployforme/internal/orchestrator/models"
	provider "github.com/Cyclone1070/deployforme/internal/provider/models"
	"google.golang.org/genai"
)

func TestToGeminiContents(t *testing.T) {
	tests := []struct {
		name     string
		prompt   string
		history  []models.Message
		expected int // Expected number of contents
	}{
		{
			name:     "Empty history with prompt",
			prompt:   "Hello",
			history:  []models.Message{},
			expected: 1,
		},
		{
			name:   "User and assistant messages",
			prompt: "",
			history: []models.Message{
				{Role: "user", Content: "Hi"},
				{Role: "assistant", Content: "Hello"},
			},
			expected: 2,
		},
		{
			name:   "Model message with tool calls",
			prompt: "",
			history: []models.Message{
				{
					Role: "model",
					ToolCalls: []models.ToolCall{
						{ID: "1", Name: "test_tool", Args: map[string]any{"arg": "value"}},
					},
				},
			},
			expected: 1,
		},
		{
			name:   "Function message with tool results",
			prompt: "",
			history: []models.Message{
				{
					Role: "function",
					ToolResults: []models.ToolResult{
						{ID: "1", Name: "test_tool", Content: "result"},
					},
				},
			},
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			contents := toGeminiContents(tt.prompt, tt.history)
			if len(contents) != tt.expected {
				t.Errorf("Expected %d contents, got %d", tt.expected, len(contents))
			}
		})
	}
}

func TestFromGeminiResponse_TextResponse(t *testing.T) {
	geminiResp := &genai.GenerateContentResponse{
		Candidates: []*genai.Candidate{
			{
				Content: &genai.Content{
					Parts: []*genai.Part{
						{Text: "Hello, how can I help?"},
					},
				},
				FinishReason: genai.FinishReasonStop,
			},
		},
	}

	resp, err := fromGeminiResponse(geminiResp, "test-model")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if resp.Content.Type != provider.ResponseTypeText {
		t.Errorf("Expected ResponseTypeText, got %v", resp.Content.Type)
	}

	if resp.Content.Text != "Hello, how can I help?" {
		t.Errorf("Expected 'Hello, how can I help?', got %s", resp.Content.Text)
	}

	if resp.Metadata.ModelUsed != "test-model" {
		t.Errorf("Expected model 'test-model', got %s", resp.Metadata.ModelUsed)
	}
}

func TestFromGeminiResponse_ToolCallResponse(t *testing.T) {
	geminiResp := &genai.GenerateContentResponse{
		Candidates: []*genai.Candidate{
			{
				Content: &genai.Content{
					Parts: []*genai.Part{
						{
							FunctionCall: &genai.FunctionCall{
								Name: "test_tool",
								Args: map[string]any{"arg": "value"},
							},
						},
					},
				},
				FinishReason: genai.FinishReasonStop,
			},
		},
	}

	resp, err := fromGeminiResponse(geminiResp, "test-model")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if resp.Content.Type != provider.ResponseTypeToolCall {
		t.Errorf("Expected ResponseTypeToolCall, got %v", resp.Content.Type)
	}

	if len(resp.Content.ToolCalls) != 1 {
		t.Fatalf("Expected 1 tool call, got %d", len(resp.Content.ToolCalls))
	}

	if resp.Content.ToolCalls[0].Name != "test_tool" {
		t.Errorf("Expected tool name 'test_tool', got %s", resp.Content.ToolCalls[0].Name)
	}
}

func TestFromGeminiResponse_SafetyBlock(t *testing.T) {
	geminiResp := &genai.GenerateContentResponse{
		Candidates: []*genai.Candidate{
			{
				Content:      &genai.Content{},
				FinishReason: genai.FinishReasonSafety,
			},
		},
	}

	_, err := fromGeminiResponse(geminiResp, "test-model")
	if err == nil {
		t.Fatal("Expected error for safety block")
	}

	provErr, ok := err.(*provider.ProviderError)
	if !ok {
		t.Fatalf("Expected ProviderError, got %T", err)
	}

	if provErr.Code != provider.ErrorCodeContentBlocked {
		t.Errorf("Expected ErrorCodeContentBlocked, got %v", provErr.Code)
	}
}

func TestFromGeminiResponse_MaxTokens(t *testing.T) {
	geminiResp := &genai.GenerateContentResponse{
		Candidates: []*genai.Candidate{
			{
				Content: &genai.Content{
					Parts: []*genai.Part{
						{Text: "Partial response"},
					},
				},
				FinishReason: genai.FinishReasonMaxTokens,
			},
		},
	}

	resp, err := fromGeminiResponse(geminiResp, "test-model")
	if err == nil {
		t.Fatal("Expected error for max tokens")
	}

	// Should still return partial response
	if resp == nil {
		t.Fatal("Expected partial response")
	}

	if resp.Content.Text != "Partial response" {
		t.Errorf("Expected partial response text, got %s", resp.Content.Text)
	}

	provErr, ok := err.(*provider.ProviderError)
	if !ok {
		t.Fatalf("Expected ProviderError, got %T", err)
	}

	if provErr.Code != provider.ErrorCodeContextLength {
		t.Errorf("Expected ErrorCodeContextLength, got %v", provErr.Code)
	}
}

func TestMapGeminiError_RateLimit(t *testing.T) {
	apiErr := &genai.APIError{
		Code:    429,
		Message: "Rate limit exceeded",
	}

	err := mapGeminiError(apiErr)
	provErr, ok := err.(*provider.ProviderError)
	if !ok {
		t.Fatalf("Expected ProviderError, got %T", err)
	}

	if provErr.Code != provider.ErrorCodeRateLimit {
		t.Errorf("Expected ErrorCodeRateLimit, got %v", provErr.Code)
	}

	if !provErr.Retryable {
		t.Error("Expected error to be retryable")
	}
}

func TestMapGeminiError_Auth(t *testing.T) {
	tests := []struct {
		code int
		name string
	}{
		{401, "Unauthorized"},
		{403, "Forbidden"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apiErr := &genai.APIError{
				Code:    tt.code,
				Message: "Auth failed",
			}

			err := mapGeminiError(apiErr)
			provErr, ok := err.(*provider.ProviderError)
			if !ok {
				t.Fatalf("Expected ProviderError, got %T", err)
			}

			if provErr.Code != provider.ErrorCodeAuth {
				t.Errorf("Expected ErrorCodeAuth, got %v", provErr.Code)
			}

			if provErr.Retryable {
				t.Error("Expected error to NOT be retryable")
			}
		})
	}
}

func TestMapGeminiError_InvalidRequest(t *testing.T) {
	apiErr := &genai.APIError{
		Code:    400,
		Message: "Invalid request",
	}

	err := mapGeminiError(apiErr)
	provErr, ok := err.(*provider.ProviderError)
	if !ok {
		t.Fatalf("Expected ProviderError, got %T", err)
	}

	if provErr.Code != provider.ErrorCodeInvalidRequest {
		t.Errorf("Expected ErrorCodeInvalidRequest, got %v", provErr.Code)
	}

	if provErr.Retryable {
		t.Error("Expected error to NOT be retryable")
	}
}

func TestMapGeminiError_Unavailable(t *testing.T) {
	codes := []int{500, 502, 503, 504}

	for _, code := range codes {
		t.Run(string(rune(code)), func(t *testing.T) {
			apiErr := &genai.APIError{
				Code:    code,
				Message: "Service unavailable",
			}

			err := mapGeminiError(apiErr)
			provErr, ok := err.(*provider.ProviderError)
			if !ok {
				t.Fatalf("Expected ProviderError, got %T", err)
			}

			if provErr.Code != provider.ErrorCodeUnavailable {
				t.Errorf("Expected ErrorCodeUnavailable, got %v", provErr.Code)
			}

			if !provErr.Retryable {
				t.Error("Expected error to be retryable")
			}
		})
	}
}

func TestToGeminiTools(t *testing.T) {
	tools := []provider.ToolDefinition{
		{
			Name:        "test_tool",
			Description: "A test tool",
			Parameters: &provider.ParameterSchema{
				Properties: map[string]provider.PropertySchema{
					"arg1": {Type: "string", Description: "First argument"},
				},
				Required: []string{"arg1"},
			},
		},
	}

	geminiTools := toGeminiTools(tools)
	if len(geminiTools) != 1 {
		t.Fatalf("Expected 1 tool, got %d", len(geminiTools))
	}

	if len(geminiTools[0].FunctionDeclarations) != 1 {
		t.Fatalf("Expected 1 function declaration, got %d", len(geminiTools[0].FunctionDeclarations))
	}

	fd := geminiTools[0].FunctionDeclarations[0]
	if fd.Name != "test_tool" {
		t.Errorf("Expected name 'test_tool', got %s", fd.Name)
	}

	if fd.Description != "A test tool" {
		t.Errorf("Expected description 'A test tool', got %s", fd.Description)
	}
}
