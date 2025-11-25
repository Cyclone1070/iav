package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Cyclone1070/deployforme/internal/orchestrator/adapter"
	"github.com/Cyclone1070/deployforme/internal/orchestrator/models"
	"github.com/Cyclone1070/deployforme/internal/provider"
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
	// Add initial user goal to history
	o.history = append(o.history, models.Message{
		Role:    "user",
		Content: goal,
	})

	maxTurns := 50 // Prevent infinite loops
	for turn := 0; turn < maxTurns; turn++ {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// 1. Update UI status
		o.ui.WriteStatus("thinking", "Generating response...")

		// 2. Generate response from LLM
		response, err := o.provider.Generate(ctx, "", o.history)
		if err != nil {
			return fmt.Errorf("provider error: %w", err)
		}

		// 3. Parse response
		parsed, err := o.parseResponse(response)
		if err != nil {
			// Add system message about parse error
			o.history = append(o.history, models.Message{
				Role:    "system",
				Content: fmt.Sprintf("Error parsing response: %v", err),
			})
			continue
		}

		// 4. Handle response type
		if parsed.IsText {
			// Text response: display to user and wait for input
			o.ui.WriteMessage(parsed.Text)
			o.history = append(o.history, models.Message{
				Role:    "assistant",
				Content: parsed.Text,
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
			continue
		}

		// Tool call response
		if parsed.ToolName == "" {
			o.history = append(o.history, models.Message{
				Role:    "system",
				Content: "Error: tool call missing tool name",
			})
			continue
		}

		// 5. Check if tool exists
		tool, exists := o.tools[parsed.ToolName]
		if !exists {
			o.history = append(o.history, models.Message{
				Role:    "system",
				Content: fmt.Sprintf("Error: unknown tool '%s'", parsed.ToolName),
			})
			continue
		}

		// 6. Policy check
		if err := o.policy.CheckTool(ctx, parsed.ToolName); err != nil {
			o.history = append(o.history, models.Message{
				Role:    "system",
				Content: fmt.Sprintf("Policy denied tool '%s': %v", parsed.ToolName, err),
			})
			continue
		}

		// 7. Execute tool
		o.ui.WriteStatus("executing", fmt.Sprintf("Running tool: %s", parsed.ToolName))
		result, err := o.executeTool(ctx, tool, parsed.ToolArgs)
		if err != nil {
			o.history = append(o.history, models.Message{
				Role:    "system",
				Content: fmt.Sprintf("Tool '%s' failed: %v", parsed.ToolName, err),
			})
			continue
		}

		// 8. Add tool result to history
		o.history = append(o.history, models.Message{
			Role:    "assistant",
			Content: fmt.Sprintf("Tool '%s' result: %s", parsed.ToolName, result),
		})
	}

	return fmt.Errorf("max turns (%d) reached", maxTurns)
}

// parsedResponse represents a parsed LLM response
type parsedResponse struct {
	IsText   bool
	Text     string
	ToolName string
	ToolArgs string
}

// parseResponse attempts to parse the LLM response as either text or a tool call
func (o *Orchestrator) parseResponse(response string) (*parsedResponse, error) {
	// Try to parse as JSON (tool call)
	var toolCall struct {
		Tool string `json:"tool"`
		Args string `json:"args"`
	}

	if err := json.Unmarshal([]byte(response), &toolCall); err == nil && toolCall.Tool != "" {
		return &parsedResponse{
			IsText:   false,
			ToolName: toolCall.Tool,
			ToolArgs: toolCall.Args,
		}, nil
	}

	// Otherwise, treat as text
	if response == "" {
		return nil, fmt.Errorf("empty response from LLM")
	}

	return &parsedResponse{
		IsText: true,
		Text:   response,
	}, nil
}

// executeTool safely executes a tool with panic recovery
func (o *Orchestrator) executeTool(ctx context.Context, tool adapter.Tool, args string) (result string, err error) {
	// Recover from panics
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("tool panicked: %v", r)
		}
	}()

	return tool.Execute(ctx, args)
}
