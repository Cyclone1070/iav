package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Cyclone1070/deployforme/internal/orchestrator/adapter"
	"github.com/Cyclone1070/deployforme/internal/orchestrator/models"
	provider "github.com/Cyclone1070/deployforme/internal/provider/models"
	"github.com/Cyclone1070/deployforme/internal/ui"
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

	for range maxTurns {
		// Check context
		if ctx.Err() != nil {
			return ctx.Err()
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
			// Native tool call from provider
			if len(response.Content.ToolCalls) == 0 {
				o.history = append(o.history, models.Message{
					Role:    "system",
					Content: "Error: empty tool call list",
				})
				continue
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

		case provider.ResponseTypeText:
			// Text response: display to user and wait for input
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

		case provider.ResponseTypeRefusal:
			// Model refused to generate (safety block, policy violation)
			o.ui.WriteStatus("blocked", "Model refused to generate")
			o.history = append(o.history, models.Message{
				Role:    "system",
				Content: fmt.Sprintf("Model refused: %s", response.Content.RefusalReason),
			})

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
	if err := o.policy.CheckTool(ctx, toolCall.Name); err != nil {
		return models.ToolResult{
			ID:      toolCall.ID,
			Name:    toolCall.Name,
			Content: "",
			Error:   fmt.Sprintf("policy denied: %v", err),
		}
	}

	// Convert args to JSON string
	argsJSON, err := json.Marshal(toolCall.Args)
	if err != nil {
		return models.ToolResult{
			ID:      toolCall.ID,
			Name:    toolCall.Name,
			Content: "",
			Error:   fmt.Sprintf("error marshaling args: %v", err),
		}
	}

	// Execute tool
	o.ui.WriteStatus("executing", fmt.Sprintf("Running %s...", toolCall.Name))
	result, err := tool.Execute(ctx, string(argsJSON))
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
