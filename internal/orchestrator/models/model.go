package models

// Message represents a single message in the conversation history
type Message struct {
	Role    string // "user", "assistant", "system", "model", "function"
	Content string

	// For model messages with tool calls
	ToolCalls []ToolCall

	// For function messages with tool results
	ToolResults []ToolResult
}

// ToolCall represents a structured tool invocation from the model.
type ToolCall struct {
	ID   string
	Name string
	Args map[string]interface{}
}

// ToolResult represents the result of a tool execution.
type ToolResult struct {
	ID      string // Matches ToolCall.ID
	Name    string // Tool name
	Content string // Result content
	Error   string // Error message if tool failed
}

// Policy defines the rules for the agent.
// It covers both Shell commands and Tool usage.
type Policy struct {
	Shell ShellPolicy
	Tools ToolPolicy
}

// ShellPolicy defines rules for shell commands.
type ShellPolicy struct {
	Allow        []string        // Exact command matches (e.g. "ls -la")
	Deny         []string        // Exact command matches
	SessionAllow map[string]bool // Runtime approvals
}

// ToolPolicy defines rules for tool usage.
type ToolPolicy struct {
	Allow        []string        // Allowed tool names (e.g. "read_file")
	Deny         []string        // Denied tool names
	SessionAllow map[string]bool // Runtime approvals
}
