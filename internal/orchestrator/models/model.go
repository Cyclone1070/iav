package models

// Message represents a single message in the conversation history
type Message struct {
	Role    string // "user", "assistant", "system"
	Content string
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
