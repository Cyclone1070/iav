package loop

import (
	"context"

	"github.com/Cyclone1070/iav/internal/provider"
	"github.com/Cyclone1070/iav/internal/tool"
	"github.com/Cyclone1070/iav/internal/workflow"
)

// llmProvider communicates with an LLM.
type llmProvider interface {
	// Generate sends messages to the LLM and returns its response.
	Generate(ctx context.Context, messages []provider.Message, tools []tool.Declaration) (*provider.Message, error)
}

// toolManager manages tool storage and execution.
type toolManager interface {
	// Declarations returns all tool schemas for the LLM.
	Declarations() []tool.Declaration

	// Execute runs a tool call and returns the result as a provider.Message.
	// It emits ToolStartEvent, ToolEndEvent, ToolStreamEvent, and ShellEndEvent to the events channel.
	Execute(ctx context.Context, tc provider.ToolCall, events chan<- workflow.Event) (provider.Message, error)
}
