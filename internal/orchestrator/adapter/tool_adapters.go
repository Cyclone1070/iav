package adapter

import (
	"context"

	provider "github.com/Cyclone1070/deployforme/internal/provider/models"
	"github.com/Cyclone1070/deployforme/internal/tools"
	toolModels "github.com/Cyclone1070/deployforme/internal/tools/models"
)

// This file consolidates all tool adapters using the BaseAdapter pattern.
// Each adapter is now a simple constructor function instead of a full type definition.

// NewReadFile creates a read_file adapter
func NewReadFile(wCtx *toolModels.WorkspaceContext) Tool {
	return NewBaseAdapter(
		"read_file",
		"Reads a file from the workspace",
		&provider.Schema{
			Type: "object",
			Properties: map[string]provider.Schema{
				"path": {
					Type:        "string",
					Description: "Path to the file (relative to workspace root)",
				},
				"offset": {
					Type:        "integer",
					Description: "Byte offset to start reading from",
				},
				"limit": {
					Type:        "integer",
					Description: "Maximum number of bytes to read",
				},
			},
			Required: []string{"path"},
		},
		wCtx,
		tools.ReadFile,
	)
}

// NewWriteFile creates a write_file adapter
func NewWriteFile(wCtx *toolModels.WorkspaceContext) Tool {
	return NewBaseAdapter(
		"write_file",
		"Creates a new file in the workspace",
		&provider.Schema{
			Type: "object",
			Properties: map[string]provider.Schema{
				"path": {
					Type:        "string",
					Description: "Path to the file (relative to workspace root)",
				},
				"content": {
					Type:        "string",
					Description: "File content",
				},
				"perm": {
					Type:        "integer",
					Description: "File permissions (octal, e.g., 0644)",
				},
			},
			Required: []string{"path", "content"},
		},
		wCtx,
		tools.WriteFile,
	)
}

// NewEditFile creates an edit_file adapter
func NewEditFile(wCtx *toolModels.WorkspaceContext) Tool {
	return NewBaseAdapter(
		"edit_file",
		"Applies edit operations to an existing file",
		&provider.Schema{
			Type: "object",
			Properties: map[string]provider.Schema{
				"path": {
					Type:        "string",
					Description: "Path to the file (relative to workspace root)",
				},
				"operations": {
					Type:        "array",
					Description: "List of edit operations to apply",
					Items: &provider.Schema{
						Type: "object",
						Properties: map[string]provider.Schema{
							"before": {
								Type:        "string",
								Description: "Exact text to find and replace",
							},
							"after": {
								Type:        "string",
								Description: "Replacement text",
							},
							"expected_replacements": {
								Type:        "integer",
								Description: "Expected number of replacements (defaults to 1 if omitted)",
							},
						},
						Required: []string{"before", "after"},
					},
				},
			},
			Required: []string{"path", "operations"},
		},
		wCtx,
		tools.EditFile,
	)
}

// NewFindFile creates a find_file adapter
func NewFindFile(wCtx *toolModels.WorkspaceContext) Tool {
	return NewBaseAdapter(
		"find_file",
		"Finds files in the workspace matching a pattern",
		&provider.Schema{
			Type: "object",
			Properties: map[string]provider.Schema{
				"pattern": {
					Type:        "string",
					Description: "Glob pattern to match files",
				},
				"max_depth": {
					Type:        "integer",
					Description: "Maximum directory depth to search",
				},
				"include_ignored": {
					Type:        "boolean",
					Description: "Include gitignored files",
				},
			},
			Required: []string{"pattern"},
		},
		wCtx,
		tools.FindFile,
	)
}

// NewListDirectory creates a list_directory adapter
func NewListDirectory(wCtx *toolModels.WorkspaceContext) Tool {
	return NewBaseAdapter(
		"list_directory",
		"Lists contents of a directory",
		&provider.Schema{
			Type: "object",
			Properties: map[string]provider.Schema{
				"path": {
					Type:        "string",
					Description: "Directory path (relative to workspace root)",
				},
				"max_depth": {
					Type:        "integer",
					Description: "Maximum depth for recursive listing (0 = current dir only, -1 = unlimited)",
				},
				"include_ignored": {
					Type:        "boolean",
					Description: "Include gitignored files",
				},
				"offset": {
					Type:        "integer",
					Description: "Pagination offset",
				},
				"limit": {
					Type:        "integer",
					Description: "Maximum number of entries to return",
				},
			},
			Required: []string{"path"},
		},
		wCtx,
		tools.ListDirectory,
	)
}

// NewSearchContent creates a search_content adapter
func NewSearchContent(wCtx *toolModels.WorkspaceContext) Tool {
	return NewBaseAdapter(
		"search_content",
		"Searches for content within files",
		&provider.Schema{
			Type: "object",
			Properties: map[string]provider.Schema{
				"query": {
					Type:        "string",
					Description: "Search query",
				},
				"path": {
					Type:        "string",
					Description: "Path to search in",
				},
				"case_sensitive": {
					Type:        "boolean",
					Description: "Case sensitive search",
				},
				"include_ignored": {
					Type:        "boolean",
					Description: "Include gitignored files",
				},
			},
			Required: []string{"query"},
		},
		wCtx,
		tools.SearchContent,
	)
}

// NewShell creates a run_shell adapter
// Note: Shell is special because it needs both ShellTool and WorkspaceContext
func NewShell(tool *tools.ShellTool, wCtx *toolModels.WorkspaceContext) Tool {
	return NewBaseAdapter(
		"run_shell",
		"Executes shell commands in the workspace",
		&provider.Schema{
			Type: "object",
			Properties: map[string]provider.Schema{
				"command": {
					Type:        "array",
					Description: "The command and arguments to execute",
					Items: &provider.Schema{
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
					Description: "Environment variables as key-value pairs",
				},
				"env_files": {
					Type:        "array",
					Description: "Paths to .env files to load",
					Items: &provider.Schema{
						Type: "string",
					},
				},
			},
			Required: []string{"command"},
		},
		wCtx,
		// Use a closure to adapt ShellTool.Run to the expected signature
		func(ctx context.Context, wCtx *toolModels.WorkspaceContext, req toolModels.ShellRequest) (toolModels.ShellResponse, error) {
			// ShellTool.Run returns *ShellResponse, so we need to dereference it
			resp, err := tool.Run(ctx, wCtx, req)
			if err != nil {
				// Return zero value on error
				return toolModels.ShellResponse{}, err
			}
			return *resp, nil
		},
	)
}

// NewReadTodos creates a read_todos adapter
func NewReadTodos(wCtx *toolModels.WorkspaceContext) Tool {
	return NewBaseAdapter(
		"read_todos",
		"Reads all TODO items",
		&provider.Schema{
			Type:       "object",
			Properties: map[string]provider.Schema{},
			Required:   []string{},
		},
		wCtx,
		tools.ReadTodos,
	)
}

// NewWriteTodos creates a write_todos adapter
func NewWriteTodos(wCtx *toolModels.WorkspaceContext) Tool {
	return NewBaseAdapter(
		"write_todos",
		"Writes TODO items",
		&provider.Schema{
			Type: "object",
			Properties: map[string]provider.Schema{
				"todos": {
					Type:        "array",
					Description: "List of TODO items",
					Items: &provider.Schema{
						Type: "object",
						Properties: map[string]provider.Schema{
							"description": {
								Type:        "string",
								Description: "Description of the todo item",
							},
							"status": {
								Type:        "string",
								Description: "Status of the todo",
								Enum:        []string{"pending", "in_progress", "completed", "cancelled"},
							},
						},
						Required: []string{"description", "status"},
					},
				},
			},
			Required: []string{"todos"},
		},
		wCtx,
		tools.WriteTodos,
	)
}
