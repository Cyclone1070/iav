package gemini

import (
	"fmt"
	"time"

	"github.com/Cyclone1070/deployforme/internal/orchestrator/models"
	provider "github.com/Cyclone1070/deployforme/internal/provider/models"
	"google.golang.org/genai"
)

// toGeminiContents converts a prompt and history to Gemini Content format.
func toGeminiContents(prompt string, history []models.Message) []*genai.Content {
	contents := make([]*genai.Content, 0, len(history)+1)

	// Add history
	for _, msg := range history {
		content := messageToGeminiContent(msg)
		if content != nil {
			contents = append(contents, content)
		}
	}

	// Add current prompt
	if prompt != "" {
		contents = append(contents, &genai.Content{
			Role: "user",
			Parts: []*genai.Part{
				genai.NewPartFromText(prompt),
			},
		})
	}

	return contents
}

// messageToGeminiContent converts a single message to Gemini Content format.
func messageToGeminiContent(msg models.Message) *genai.Content {
	// Determine role
	role := "user"
	if msg.Role == "assistant" || msg.Role == "model" {
		role = "model"
	}

	parts := make([]*genai.Part, 0)

	// Add text content if present
	if msg.Content != "" {
		parts = append(parts, genai.NewPartFromText(msg.Content))
	}

	// Add tool calls if present (model messages)
	if len(msg.ToolCalls) > 0 {
		for _, toolCall := range msg.ToolCalls {
			parts = append(parts, &genai.Part{
				FunctionCall: &genai.FunctionCall{
					Name: toolCall.Name,
					Args: toolCall.Args,
				},
			})
		}
	}

	// Add tool results if present (function messages)
	if len(msg.ToolResults) > 0 {
		for _, result := range msg.ToolResults {
			// Build response content
			responseContent := result.Content
			if result.Error != "" {
				responseContent = fmt.Sprintf("Error: %s", result.Error)
			}

			parts = append(parts, &genai.Part{
				FunctionResponse: &genai.FunctionResponse{
					Name: result.Name,
					Response: map[string]interface{}{
						"content": responseContent,
					},
				},
			})
		}
	}

	// Skip empty messages
	if len(parts) == 0 {
		return nil
	}

	return &genai.Content{
		Role:  role,
		Parts: parts,
	}
}

// messagesToGeminiContents converts messages to Gemini Content format.
func messagesToGeminiContents(messages []models.Message) []*genai.Content {
	contents := make([]*genai.Content, 0, len(messages))

	for _, msg := range messages {
		content := messageToGeminiContent(msg)
		if content != nil {
			contents = append(contents, content)
		}
	}

	return contents
}

// toGeminiConfig converts internal GenerateConfig to Gemini config.
func toGeminiConfig(config *provider.GenerateConfig) *genai.GenerateContentConfig {
	geminiConfig := &genai.GenerateContentConfig{
		SafetySettings: defaultSafetySettings(),
	}

	if config == nil {
		return geminiConfig
	}

	if config.Temperature != nil {
		geminiConfig.Temperature = config.Temperature
	}
	if config.TopP != nil {
		geminiConfig.TopP = config.TopP
	}
	if config.TopK != nil {
		topK := float32(*config.TopK)
		geminiConfig.TopK = &topK
	}
	if len(config.StopSequences) > 0 {
		geminiConfig.StopSequences = config.StopSequences
	}

	return geminiConfig
}

// defaultSafetySettings returns safety settings with BLOCK_NONE for all categories.
func defaultSafetySettings() []*genai.SafetySetting {
	return []*genai.SafetySetting{
		{
			Category:  genai.HarmCategoryHateSpeech,
			Threshold: genai.HarmBlockThresholdOff,
		},
		{
			Category:  genai.HarmCategoryDangerousContent,
			Threshold: genai.HarmBlockThresholdOff,
		},
		{
			Category:  genai.HarmCategoryHarassment,
			Threshold: genai.HarmBlockThresholdOff,
		},
		{
			Category:  genai.HarmCategorySexuallyExplicit,
			Threshold: genai.HarmBlockThresholdOff,
		},
	}
}

// toGeminiTools converts internal ToolDefinition to Gemini tools.
func toGeminiTools(tools []provider.ToolDefinition) []*genai.Tool {
	if len(tools) == 0 {
		return nil
	}

	functionDeclarations := make([]*genai.FunctionDeclaration, 0, len(tools))

	for _, tool := range tools {
		fd := &genai.FunctionDeclaration{
			Name:        tool.Name,
			Description: tool.Description,
		}

		if tool.Parameters != nil {
			fd.Parameters = toGeminiSchema(tool.Parameters)
		}

		functionDeclarations = append(functionDeclarations, fd)
	}

	return []*genai.Tool{
		{FunctionDeclarations: functionDeclarations},
	}
}

// toGeminiSchema converts ParameterSchema to Gemini Schema.
func toGeminiSchema(params *provider.ParameterSchema) *genai.Schema {
	schema := &genai.Schema{
		Type: genai.TypeObject,
	}

	if params.Properties != nil {
		schema.Properties = make(map[string]*genai.Schema)
		for name, prop := range params.Properties {
			schema.Properties[name] = &genai.Schema{
				Type:        toGeminiType(prop.Type),
				Description: prop.Description,
			}

			if len(prop.Enum) > 0 {
				schema.Properties[name].Enum = prop.Enum
			}

			if prop.Items != nil {
				schema.Properties[name].Items = &genai.Schema{
					Type:        toGeminiType(prop.Items.Type),
					Description: prop.Items.Description,
				}
			}
		}
	}

	if len(params.Required) > 0 {
		schema.Required = params.Required
	}

	return schema
}

// toGeminiType converts string type to Gemini Type.
func toGeminiType(typeStr string) genai.Type {
	switch typeStr {
	case "string":
		return genai.TypeString
	case "number":
		return genai.TypeNumber
	case "integer":
		return genai.TypeInteger
	case "boolean":
		return genai.TypeBoolean
	case "array":
		return genai.TypeArray
	case "object":
		return genai.TypeObject
	default:
		return genai.TypeString
	}
}

// fromGeminiResponse converts Gemini response to internal format.
func fromGeminiResponse(resp *genai.GenerateContentResponse, modelUsed string) (*provider.GenerateResponse, error) {
	if len(resp.Candidates) == 0 {
		return nil, &provider.ProviderError{
			Code:    provider.ErrorCodeInvalidRequest,
			Message: "no candidates in response",
		}
	}

	candidate := resp.Candidates[0]

	// Check finish reason
	if candidate.FinishReason == genai.FinishReasonSafety {
		return nil, &provider.ProviderError{
			Code:      provider.ErrorCodeContentBlocked,
			Message:   "content blocked by safety filters",
			Retryable: false,
		}
	}

	if candidate.FinishReason == genai.FinishReasonMaxTokens {
		// Return partial response with error
		response := buildResponse(candidate, resp.UsageMetadata, modelUsed)
		return response, &provider.ProviderError{
			Code:      provider.ErrorCodeContextLength,
			Message:   "response truncated due to max tokens",
			Retryable: false,
		}
	}

	// Check for function calls
	if len(candidate.Content.Parts) > 0 {
		for _, part := range candidate.Content.Parts {
			if part.FunctionCall != nil {
				return buildToolCallResponse(candidate, resp.UsageMetadata, modelUsed), nil
			}
		}
	}

	// Normal text response
	return buildResponse(candidate, resp.UsageMetadata, modelUsed), nil
}

// buildResponse builds a text response from a candidate.
func buildResponse(candidate *genai.Candidate, usage *genai.GenerateContentResponseUsageMetadata, modelUsed string) *provider.GenerateResponse {
	var text string
	for _, part := range candidate.Content.Parts {
		if part.Text != "" {
			text += part.Text
		}
	}

	return &provider.GenerateResponse{
		Content: provider.ResponseContent{
			Type: provider.ResponseTypeText,
			Text: text,
		},
		Metadata: buildMetadata(usage, modelUsed),
	}
}

// buildToolCallResponse builds a tool call response from a candidate.
func buildToolCallResponse(candidate *genai.Candidate, usage *genai.GenerateContentResponseUsageMetadata, modelUsed string) *provider.GenerateResponse {
	toolCalls := make([]models.ToolCall, 0)

	for _, part := range candidate.Content.Parts {
		if part.FunctionCall != nil {
			toolCalls = append(toolCalls, models.ToolCall{
				ID:   "", // Gemini doesn't provide IDs
				Name: part.FunctionCall.Name,
				Args: part.FunctionCall.Args,
			})
		}
	}

	return &provider.GenerateResponse{
		Content: provider.ResponseContent{
			Type:      provider.ResponseTypeToolCall,
			ToolCalls: toolCalls,
		},
		Metadata: buildMetadata(usage, modelUsed),
	}
}

// buildMetadata builds response metadata from usage data.
func buildMetadata(usage *genai.GenerateContentResponseUsageMetadata, modelUsed string) provider.ResponseMetadata {
	metadata := provider.ResponseMetadata{
		ModelUsed: modelUsed,
	}

	if usage != nil {
		metadata.PromptTokens = int(usage.PromptTokenCount)
		metadata.CompletionTokens = int(usage.CandidatesTokenCount)
		metadata.TotalTokens = int(usage.TotalTokenCount)
	}

	return metadata
}

// mapGeminiError maps Gemini API errors to provider errors.
func mapGeminiError(err error) error {
	if err == nil {
		return nil
	}

	// Check if it's an APIError
	if apiErr, ok := err.(*genai.APIError); ok {
		switch apiErr.Code {
		case 401, 403:
			return &provider.ProviderError{
				Code:       provider.ErrorCodeAuth,
				Message:    "authentication failed",
				Underlying: err,
				Retryable:  false,
			}
		case 429:
			return &provider.ProviderError{
				Code:       provider.ErrorCodeRateLimit,
				Message:    "rate limit exceeded",
				Underlying: err,
				Retryable:  true,
				RetryAfter: parseRetryAfter(apiErr),
			}
		case 400:
			return &provider.ProviderError{
				Code:       provider.ErrorCodeInvalidRequest,
				Message:    fmt.Sprintf("invalid request: %s", apiErr.Message),
				Underlying: err,
				Retryable:  false,
			}
		case 500, 502, 503, 504:
			return &provider.ProviderError{
				Code:       provider.ErrorCodeUnavailable,
				Message:    "service unavailable",
				Underlying: err,
				Retryable:  true,
			}
		default:
			return &provider.ProviderError{
				Code:       provider.ErrorCodeNetwork,
				Message:    fmt.Sprintf("API error: %s", apiErr.Message),
				Underlying: err,
				Retryable:  true,
			}
		}
	}

	// Generic network error
	return &provider.ProviderError{
		Code:       provider.ErrorCodeNetwork,
		Message:    "network error",
		Underlying: err,
		Retryable:  true,
	}
}

// parseRetryAfter attempts to parse Retry-After from error details.
func parseRetryAfter(apiErr *genai.APIError) *time.Duration {
	// TODO: Parse Retry-After header if available in error details
	// For now, return nil
	return nil
}
