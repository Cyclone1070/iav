package toolmanager

import (
	"context"

	"github.com/Cyclone1070/iav/internal/tool"
)

// toolResult is returned by tools after execution.
type toolResult interface {
	// LLMContent returns the string content sent to the LLM.
	LLMContent() string

	// Display returns the display type for UI rendering.
	Display() tool.ToolDisplay
}

// toolImpl defines the interface for individual tools.
// Request structs should implement fmt.Stringer for display.
type toolImpl interface {
	// Name returns the tool's identifier.
	Name() string

	// Declaration returns the tool's schema for the LLM.
	Declaration() tool.Declaration

	// Input returns a pointer to the input struct (e.g., &ReadFileRequest{}).
	// The request struct should implement fmt.Stringer for the request display.
	Input() any

	// Execute runs the tool with typed input and returns a toolResult.
	Execute(ctx context.Context, input any) (toolResult, error)
}
