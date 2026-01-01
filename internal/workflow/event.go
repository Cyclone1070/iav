package workflow

import "github.com/Cyclone1070/iav/internal/tool"

// Event is the interface for all workflow events.
// UI handles events via type switch.
type Event interface {
	isEvent()
}

// TextEvent is emitted when the LLM produces text output.
type TextEvent struct {
	Text string
}

func (TextEvent) isEvent() {}

// ThinkingEvent is emitted when the LLM is processing.
type ThinkingEvent struct{}

func (ThinkingEvent) isEvent() {}

// DoneEvent is emitted when the workflow loop completes.
type DoneEvent struct{}

func (DoneEvent) isEvent() {}

// ToolStartEvent is emitted when a tool execution begins.
type ToolStartEvent struct {
	ToolName       string
	RequestDisplay string // e.g., "Reading src/index.ts"
}

func (ToolStartEvent) isEvent() {}

// ToolStreamEvent is emitted for streaming tool output (shell commands).
type ToolStreamEvent struct {
	ToolName string
	Chunk    string
}

func (ToolStreamEvent) isEvent() {}

// ToolEndEvent is emitted when a non-streaming tool completes.
type ToolEndEvent struct {
	ToolName string
	Display  tool.ToolDisplay
}

func (ToolEndEvent) isEvent() {}

// ShellEndEvent is emitted when a shell command completes.
type ShellEndEvent struct {
	ToolName string
	ExitCode int
}

func (ShellEndEvent) isEvent() {}
