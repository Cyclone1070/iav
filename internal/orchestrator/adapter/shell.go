package adapter

import (
	"context"
	"encoding/json"
	"fmt"

	provider "github.com/Cyclone1070/deployforme/internal/provider/models"

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

// Definition returns the structured tool definition
func (s *Shell) Definition() provider.ToolDefinition {
	return provider.ToolDefinition{
		Name:        "run_shell",
		Description: "Executes shell commands in the workspace",
		Parameters: &provider.ParameterSchema{
			Type: "object",
			Properties: map[string]provider.PropertySchema{
				"command": {
					Type:        "array",
					Description: "The command and arguments to execute",
					Items: &provider.PropertySchema{
						Type: "string",
					},
				},
				"working_dir": {
					Type:        "string",
					Description: "Working directory (relative to workspace root)",
				},
				"timeout_seconds": {
					Type:        "integer",
					Description: "Timeout in seconds (default: 3600)",
				},
				"env": {
					Type:        "object",
					Description: "Environment variables",
				},
				"env_files": {
					Type:        "array",
					Description: "Paths to .env files to load",
					Items: &provider.PropertySchema{
						Type: "string",
					},
				},
			},
			Required: []string{"command"},
		},
	}
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
