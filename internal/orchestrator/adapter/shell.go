package adapter

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Cyclone1070/deployforme/internal/tools"
	toolModels "github.com/Cyclone1070/deployforme/internal/tools/models"
)

// Shell adapts ShellTool to the Tool interface
type Shell struct {
	tool *tools.ShellTool
	wCtx *toolModels.WorkspaceContext
}

// NewShell creates a new Shell adapter
func NewShell(t *tools.ShellTool, w *toolModels.WorkspaceContext) *Shell {
	return &Shell{tool: t, wCtx: w}
}

// Name returns the tool name
func (s *Shell) Name() string {
	return "run_shell"
}

// Description returns the tool description
func (s *Shell) Description() string {
	return "Executes shell commands in the workspace"
}

// Schema returns the JSON schema for the tool arguments
func (s *Shell) Schema() string {
	return `{
		"type": "object",
		"properties": {
			"command": {
				"type": "array",
				"items": {"type": "string"},
				"description": "The command and arguments to execute"
			},
			"working_dir": {
				"type": "string",
				"description": "Working directory (relative to workspace root)"
			},
			"timeout_seconds": {
				"type": "integer",
				"description": "Timeout in seconds (default: 3600)"
			},
			"env": {
				"type": "object",
				"description": "Environment variables"
			},
			"env_files": {
				"type": "array",
				"items": {"type": "string"},
				"description": "Paths to .env files to load"
			}
		},
		"required": ["command"]
	}`
}

// Execute runs the shell command
func (s *Shell) Execute(ctx context.Context, args string) (string, error) {
	var req toolModels.ShellRequest
	if err := json.Unmarshal([]byte(args), &req); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	resp, err := s.tool.Run(ctx, s.wCtx, req)
	if err != nil {
		return "", err
	}

	bytes, err := json.Marshal(resp)
	if err != nil {
		return "", fmt.Errorf("failed to marshal response: %w", err)
	}
	return string(bytes), nil
}
