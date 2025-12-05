package orchestrator

import (
	"context"
	"fmt"

	"github.com/Cyclone1070/iav/internal/orchestrator/adapter"
	"github.com/Cyclone1070/iav/internal/orchestrator/models"
	provider "github.com/Cyclone1070/iav/internal/provider/models"
	"github.com/Cyclone1070/iav/internal/ui"
)

// Orchestrator manages the agent loop, tool execution, and conversation history
type Orchestrator struct {
	provider provider.Provider
	policy   models.PolicyService
	ui       ui.UserInterface
	tools    map[string]adapter.Tool
	history  []models.Message
}

// New creates a new Orchestrator instance
func New(p provider.Provider, pol models.PolicyService, userInterface ui.UserInterface, tools []adapter.Tool) *Orchestrator {
	toolMap := make(map[string]adapter.Tool)
	for _, t := range tools {
		toolMap[t.Name()] = t
	}

	return &Orchestrator{
		provider: p,
		policy:   pol,
		ui:       userInterface,
		tools:    toolMap,
		history:  make([]models.Message, 0),
	}
}

// Run executes the main agent loop
func (o *Orchestrator) Run(ctx context.Context, goal string) error {
	const maxTurns = 50

	// Initialize conversation with the goal
	o.history = []models.Message{
		{
			Role:    "user",
			Content: goal,
		},
	}

	// Pre-loop context check
	if err := o.checkAndTruncateHistory(ctx); err != nil {
		return fmt.Errorf("initial context check failed: %w", err)
	}

	for range maxTurns {
		// Check context
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Context management: check token count and truncate if needed
		if err := o.checkAndTruncateHistory(ctx); err != nil {
			return fmt.Errorf("context management failed: %w", err)
		}

		// Update UI status
		o.ui.WriteStatus("thinking", "Generating response...")

		// Generate response from LLM
		req := &provider.GenerateRequest{
			Prompt:  "",
			History: o.history,
		}
		response, err := o.provider.Generate(ctx, req)
		if err != nil {
			return fmt.Errorf("provider error: %w", err)
		}

		// Handle response based on type
		switch response.Content.Type {
		case provider.ResponseTypeToolCall:
			if err := o.handleToolCallResponse(ctx, response); err != nil {
				return err
			}
		case provider.ResponseTypeText:
			if err := o.handleTextResponse(ctx, response); err != nil {
				return err
			}
		case provider.ResponseTypeRefusal:
			o.handleRefusalResponse(response)
		default:
			o.history = append(o.history, models.Message{
				Role:    "system",
				Content: fmt.Sprintf("Error: unknown response type %v", response.Content.Type),
			})
		}
	}

	return fmt.Errorf("max turns (%d) reached", maxTurns)
}

// executeToolCall executes a single tool call and returns the result
func (o *Orchestrator) executeToolCall(ctx context.Context, toolCall models.ToolCall) models.ToolResult {
	// Check if tool exists
	tool, exists := o.tools[toolCall.Name]
	if !exists {
		return models.ToolResult{
			ID:      toolCall.ID,
			Name:    toolCall.Name,
			Content: "",
			Error:   fmt.Sprintf("unknown tool '%s'", toolCall.Name),
		}
	}

	// Policy check
	if err := o.policy.CheckTool(ctx, toolCall.Name, toolCall.Args); err != nil {
		return models.ToolResult{
			ID:      toolCall.ID,
			Name:    toolCall.Name,
			Content: "",
			Error:   fmt.Sprintf("policy denied: %v", err),
		}
	}

	// Execute tool with args map directly
	o.ui.WriteStatus("executing", fmt.Sprintf("Running %s...", toolCall.Name))
	result, err := tool.Execute(ctx, toolCall.Args)
	if err != nil {
		return models.ToolResult{
			ID:      toolCall.ID,
			Name:    toolCall.Name,
			Content: "",
			Error:   err.Error(),
		}
	}

	return models.ToolResult{
		ID:      toolCall.ID,
		Name:    toolCall.Name,
		Content: result,
		Error:   "",
	}
}

// checkAndTruncateHistory checks token count and truncates history if needed,
// respecting message pair boundaries (model+function, user+assistant)
func (o *Orchestrator) checkAndTruncateHistory(ctx context.Context) error {
	tokens, err := o.provider.CountTokens(ctx, o.history)
	if err != nil {
		return fmt.Errorf("failed to count tokens: %w", err)
	}

	contextWindow := o.provider.GetContextWindow()
	// Reserve tokens for response - use model's max output capability
	maxOutput := o.provider.GetCapabilities().MaxOutputTokens
	if maxOutput == 0 {
		maxOutput = 8192 // Fallback for models without capability info
	}
	safetyMargin := maxOutput

	if tokens <= contextWindow-safetyMargin {
		return nil // No truncation needed
	}

	// Truncate history while respecting message pairs
	// Strategy: Keep first message (goal) + most recent messages
	// Remove messages in pairs to maintain conversation structure
	for tokens > contextWindow-safetyMargin && len(o.history) > 2 {
		// Find the next pair to remove (starting from index 1)
		// We need to identify if messages form pairs:
		// - "model" + "function" (tool call + result)
		// - "user" + "assistant" (user input + text response)

		if len(o.history) < 3 {
			break // Can't remove more without losing the goal
		}

		// Check if messages at index 1 and 2 form a pair
		msg1 := o.history[1]
		msg2 := o.history[2]

		isPair := false
		if msg1.Role == "model" && msg2.Role == "function" {
			isPair = true
		} else if msg1.Role == "user" && msg2.Role == "assistant" {
			isPair = true
		}

		if isPair && len(o.history) > 3 {
			// Remove the pair
			o.history = append(o.history[:1], o.history[3:]...)
		} else {
			// Remove single message (fallback)
			o.history = append(o.history[:1], o.history[2:]...)
		}

		// Recount tokens
		tokens, err = o.provider.CountTokens(ctx, o.history)
		if err != nil {
			// If counting fails, stop truncating to avoid infinite loop
			return fmt.Errorf("failed to recount tokens during truncation: %w", err)
		}
	}

	return nil
}

// handleToolCallResponse processes a tool call response from the provider
func (o *Orchestrator) handleToolCallResponse(ctx context.Context, response *provider.GenerateResponse) error {
	// Validate tool calls
	if len(response.Content.ToolCalls) == 0 {
		o.history = append(o.history, models.Message{
			Role:    "system",
			Content: "Error: empty tool call list",
		})
		return nil
	}

	// Add model message with tool calls to history
	o.history = append(o.history, models.Message{
		Role:      "model",
		ToolCalls: response.Content.ToolCalls,
	})

	// Execute ALL tool calls
	toolResults := make([]models.ToolResult, 0, len(response.Content.ToolCalls))
	for _, toolCall := range response.Content.ToolCalls {
		result := o.executeToolCall(ctx, toolCall)
		toolResults = append(toolResults, result)
	}

	// Add function message with all results to history
	o.history = append(o.history, models.Message{
		Role:        "function",
		ToolResults: toolResults,
	})

	return nil
}

// handleTextResponse processes a text response from the provider
func (o *Orchestrator) handleTextResponse(ctx context.Context, response *provider.GenerateResponse) error {
	// Display text to user
	o.ui.WriteMessage(response.Content.Text)
	o.history = append(o.history, models.Message{
		Role:    "assistant",
		Content: response.Content.Text,
	})

	// Wait for user input
	userInput, err := o.ui.ReadInput(ctx, "You: ")
	if err != nil {
		return fmt.Errorf("failed to read user input: %w", err)
	}

	o.history = append(o.history, models.Message{
		Role:    "user",
		Content: userInput,
	})

	return nil
}

// handleRefusalResponse processes a refusal response from the provider
func (o *Orchestrator) handleRefusalResponse(response *provider.GenerateResponse) {
	o.ui.WriteStatus("blocked", "Model refused to generate")
	o.history = append(o.history, models.Message{
		Role:    "system",
		Content: fmt.Sprintf("Model refused: %s", response.Content.RefusalReason),
	})
}
