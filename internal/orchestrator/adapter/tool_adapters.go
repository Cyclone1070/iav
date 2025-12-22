package adapter

import (
	"context"
	"encoding/json"
	"fmt"

	provider "github.com/Cyclone1070/iav/internal/provider/model"
	"github.com/Cyclone1070/iav/internal/tool/directory"
	"github.com/Cyclone1070/iav/internal/tool/file"
	"github.com/Cyclone1070/iav/internal/tool/search"
	"github.com/Cyclone1070/iav/internal/tool/shell"
	"github.com/Cyclone1070/iav/internal/tool/todo"
)

// ReadFileAdapter adapts file.ReadFileTool to the Tool interface
type ReadFileAdapter struct {
	tool *file.ReadFileTool
}

// NewReadFileAdapter creates a new ReadFileAdapter
func NewReadFileAdapter(tool *file.ReadFileTool) *ReadFileAdapter {
	return &ReadFileAdapter{tool: tool}
}

func (a *ReadFileAdapter) Name() string {
	return "read_file"
}

func (a *ReadFileAdapter) Description() string {
	return "Reads a file from the workspace"
}

func (a *ReadFileAdapter) Definition() provider.ToolDefinition {
	return provider.ToolDefinition{
		Name:        a.Name(),
		Description: a.Description(),
		Parameters: &provider.Schema{
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
	}
}

func (a *ReadFileAdapter) Execute(ctx context.Context, args map[string]any) (string, error) {
	var req file.ReadFileRequest
	argsJSON, err := json.Marshal(args)
	if err != nil {
		return "", fmt.Errorf("failed to marshal args: %w", err)
	}
	if err := json.Unmarshal(argsJSON, &req); err != nil {
		return "", fmt.Errorf("failed to unmarshal request: %w", err)
	}

	resp, err := a.tool.Run(ctx, &req)
	if err != nil {
		return "", err
	}

	respJSON, err := json.Marshal(resp)
	if err != nil {
		return "", fmt.Errorf("failed to marshal response: %w", err)
	}
	return string(respJSON), nil
}

// WriteFileAdapter adapts file.WriteFileTool to the Tool interface
type WriteFileAdapter struct {
	tool *file.WriteFileTool
}

// NewWriteFileAdapter creates a new WriteFileAdapter
func NewWriteFileAdapter(tool *file.WriteFileTool) *WriteFileAdapter {
	return &WriteFileAdapter{tool: tool}
}

func (a *WriteFileAdapter) Name() string {
	return "write_file"
}

func (a *WriteFileAdapter) Description() string {
	return "Creates a new file in the workspace"
}

func (a *WriteFileAdapter) Definition() provider.ToolDefinition {
	return provider.ToolDefinition{
		Name:        a.Name(),
		Description: a.Description(),
		Parameters: &provider.Schema{
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
	}
}

func (a *WriteFileAdapter) Execute(ctx context.Context, args map[string]any) (string, error) {
	var req file.WriteFileRequest
	argsJSON, err := json.Marshal(args)
	if err != nil {
		return "", fmt.Errorf("failed to marshal args: %w", err)
	}
	if err := json.Unmarshal(argsJSON, &req); err != nil {
		return "", fmt.Errorf("failed to unmarshal request: %w", err)
	}

	resp, err := a.tool.Run(ctx, &req)
	if err != nil {
		return "", err
	}

	respJSON, err := json.Marshal(resp)
	if err != nil {
		return "", fmt.Errorf("failed to marshal response: %w", err)
	}
	return string(respJSON), nil
}

// EditFileAdapter adapts file.EditFileTool to the Tool interface
type EditFileAdapter struct {
	tool *file.EditFileTool
}

// NewEditFileAdapter creates a new EditFileAdapter
func NewEditFileAdapter(tool *file.EditFileTool) *EditFileAdapter {
	return &EditFileAdapter{tool: tool}
}

func (a *EditFileAdapter) Name() string {
	return "edit_file"
}

func (a *EditFileAdapter) Description() string {
	return "Applies edit operations to an existing file"
}

func (a *EditFileAdapter) Definition() provider.ToolDefinition {
	return provider.ToolDefinition{
		Name:        a.Name(),
		Description: a.Description(),
		Parameters: &provider.Schema{
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
	}
}

func (a *EditFileAdapter) Execute(ctx context.Context, args map[string]any) (string, error) {
	var req file.EditFileRequest
	argsJSON, err := json.Marshal(args)
	if err != nil {
		return "", fmt.Errorf("failed to marshal args: %w", err)
	}
	if err := json.Unmarshal(argsJSON, &req); err != nil {
		return "", fmt.Errorf("failed to unmarshal request: %w", err)
	}

	resp, err := a.tool.Run(ctx, &req)
	if err != nil {
		return "", err
	}

	respJSON, err := json.Marshal(resp)
	if err != nil {
		return "", fmt.Errorf("failed to marshal response: %w", err)
	}
	return string(respJSON), nil
}

// ListDirectoryAdapter adapts directory.ListDirectoryTool to the Tool interface
type ListDirectoryAdapter struct {
	tool *directory.ListDirectoryTool
}

// NewListDirectoryAdapter creates a new ListDirectoryAdapter
func NewListDirectoryAdapter(tool *directory.ListDirectoryTool) *ListDirectoryAdapter {
	return &ListDirectoryAdapter{tool: tool}
}

func (a *ListDirectoryAdapter) Name() string {
	return "list_directory"
}

func (a *ListDirectoryAdapter) Description() string {
	return "Lists the contents of a directory"
}

func (a *ListDirectoryAdapter) Definition() provider.ToolDefinition {
	return provider.ToolDefinition{
		Name:        a.Name(),
		Description: a.Description(),
		Parameters: &provider.Schema{
			Type: "object",
			Properties: map[string]provider.Schema{
				"path": {
					Type:        "string",
					Description: "Path to the directory (relative to workspace root, defaults to workspace root)",
				},
				"max_depth": {
					Type:        "integer",
					Description: "Maximum depth to recurse (0 = non-recursive, -1 = unlimited)",
				},
				"include_ignored": {
					Type:        "boolean",
					Description: "Include files ignored by .gitignore",
				},
				"offset": {
					Type:        "integer",
					Description: "Pagination offset",
				},
				"limit": {
					Type:        "integer",
					Description: "Pagination limit",
				},
			},
			Required: []string{},
		},
	}
}

func (a *ListDirectoryAdapter) Execute(ctx context.Context, args map[string]any) (string, error) {
	var req directory.ListDirectoryRequest
	argsJSON, err := json.Marshal(args)
	if err != nil {
		return "", fmt.Errorf("failed to marshal args: %w", err)
	}
	if err := json.Unmarshal(argsJSON, &req); err != nil {
		return "", fmt.Errorf("failed to unmarshal request: %w", err)
	}

	resp, err := a.tool.Run(ctx, &req)
	if err != nil {
		return "", err
	}

	respJSON, err := json.Marshal(resp)
	if err != nil {
		return "", fmt.Errorf("failed to marshal response: %w", err)
	}
	return string(respJSON), nil
}

// FindFileAdapter adapts directory.FindFileTool to the Tool interface
type FindFileAdapter struct {
	tool *directory.FindFileTool
}

// NewFindFileAdapter creates a new FindFileAdapter
func NewFindFileAdapter(tool *directory.FindFileTool) *FindFileAdapter {
	return &FindFileAdapter{tool: tool}
}

func (a *FindFileAdapter) Name() string {
	return "find_file"
}

func (a *FindFileAdapter) Description() string {
	return "Searches for files matching a glob pattern"
}

func (a *FindFileAdapter) Definition() provider.ToolDefinition {
	return provider.ToolDefinition{
		Name:        a.Name(),
		Description: a.Description(),
		Parameters: &provider.Schema{
			Type: "object",
			Properties: map[string]provider.Schema{
				"pattern": {
					Type:        "string",
					Description: "Glob pattern to match files",
				},
				"search_path": {
					Type:        "string",
					Description: "Path to search within (relative to workspace root)",
				},
				"max_depth": {
					Type:        "integer",
					Description: "Maximum depth to search",
				},
				"include_ignored": {
					Type:        "boolean",
					Description: "Include files ignored by .gitignore",
				},
				"offset": {
					Type:        "integer",
					Description: "Pagination offset",
				},
				"limit": {
					Type:        "integer",
					Description: "Pagination limit",
				},
			},
			Required: []string{"pattern"},
		},
	}
}

func (a *FindFileAdapter) Execute(ctx context.Context, args map[string]any) (string, error) {
	var req directory.FindFileRequest
	argsJSON, err := json.Marshal(args)
	if err != nil {
		return "", fmt.Errorf("failed to marshal args: %w", err)
	}
	if err := json.Unmarshal(argsJSON, &req); err != nil {
		return "", fmt.Errorf("failed to unmarshal request: %w", err)
	}

	resp, err := a.tool.Run(ctx, &req)
	if err != nil {
		return "", err
	}

	respJSON, err := json.Marshal(resp)
	if err != nil {
		return "", fmt.Errorf("failed to marshal response: %w", err)
	}
	return string(respJSON), nil
}

// SearchContentAdapter adapts search.SearchContentTool to the Tool interface
type SearchContentAdapter struct {
	tool *search.SearchContentTool
}

// NewSearchContentAdapter creates a new SearchContentAdapter
func NewSearchContentAdapter(tool *search.SearchContentTool) *SearchContentAdapter {
	return &SearchContentAdapter{tool: tool}
}

func (a *SearchContentAdapter) Name() string {
	return "search_content"
}

func (a *SearchContentAdapter) Description() string {
	return "Searches for content matching a regex pattern using ripgrep"
}

func (a *SearchContentAdapter) Definition() provider.ToolDefinition {
	return provider.ToolDefinition{
		Name:        a.Name(),
		Description: a.Description(),
		Parameters: &provider.Schema{
			Type: "object",
			Properties: map[string]provider.Schema{
				"query": {
					Type:        "string",
					Description: "Search query (regex pattern)",
				},
				"search_path": {
					Type:        "string",
					Description: "Path to search within (relative to workspace root)",
				},
				"case_sensitive": {
					Type:        "boolean",
					Description: "Whether the search should be case sensitive",
				},
				"include_ignored": {
					Type:        "boolean",
					Description: "Include files ignored by .gitignore",
				},
				"offset": {
					Type:        "integer",
					Description: "Pagination offset",
				},
				"limit": {
					Type:        "integer",
					Description: "Pagination limit",
				},
			},
			Required: []string{"query"},
		},
	}
}

func (a *SearchContentAdapter) Execute(ctx context.Context, args map[string]any) (string, error) {
	var req search.SearchContentRequest
	argsJSON, err := json.Marshal(args)
	if err != nil {
		return "", fmt.Errorf("failed to marshal args: %w", err)
	}
	if err := json.Unmarshal(argsJSON, &req); err != nil {
		return "", fmt.Errorf("failed to unmarshal request: %w", err)
	}

	resp, err := a.tool.Run(ctx, &req)
	if err != nil {
		return "", err
	}

	respJSON, err := json.Marshal(resp)
	if err != nil {
		return "", fmt.Errorf("failed to marshal response: %w", err)
	}
	return string(respJSON), nil
}

// ShellAdapter adapts shell.ShellTool to the Tool interface
type ShellAdapter struct {
	tool *shell.ShellTool
}

// NewShellAdapter creates a new ShellAdapter
func NewShellAdapter(tool *shell.ShellTool) *ShellAdapter {
	return &ShellAdapter{tool: tool}
}

func (a *ShellAdapter) Name() string {
	return "run_shell"
}

func (a *ShellAdapter) Description() string {
	return "Executes a shell command on the local machine"
}

func (a *ShellAdapter) Definition() provider.ToolDefinition {
	return provider.ToolDefinition{
		Name:        a.Name(),
		Description: a.Description(),
		Parameters: &provider.Schema{
			Type: "object",
			Properties: map[string]provider.Schema{
				"command": {
					Type:        "array",
					Description: "Command and arguments to execute",
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
					Description: "Timeout in seconds",
				},
				"env": {
					Type:        "object",
					Description: "Environment variables",
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
	}
}

func (a *ShellAdapter) Execute(ctx context.Context, args map[string]any) (string, error) {
	var req shell.ShellRequest
	argsJSON, err := json.Marshal(args)
	if err != nil {
		return "", fmt.Errorf("failed to marshal args: %w", err)
	}
	if err := json.Unmarshal(argsJSON, &req); err != nil {
		return "", fmt.Errorf("failed to unmarshal request: %w", err)
	}

	resp, err := a.tool.Run(ctx, &req)
	if err != nil {
		return "", err
	}

	respJSON, err := json.Marshal(resp)
	if err != nil {
		return "", fmt.Errorf("failed to marshal response: %w", err)
	}
	return string(respJSON), nil
}

// ReadTodosAdapter adapts todo.ReadTodosTool to the Tool interface
type ReadTodosAdapter struct {
	tool *todo.ReadTodosTool
}

// NewReadTodosAdapter creates a new ReadTodosAdapter
func NewReadTodosAdapter(tool *todo.ReadTodosTool) *ReadTodosAdapter {
	return &ReadTodosAdapter{tool: tool}
}

func (a *ReadTodosAdapter) Name() string {
	return "read_todos"
}

func (a *ReadTodosAdapter) Description() string {
	return "Reads all todos from the in-memory store"
}

func (a *ReadTodosAdapter) Definition() provider.ToolDefinition {
	return provider.ToolDefinition{
		Name:        a.Name(),
		Description: a.Description(),
		Parameters: &provider.Schema{
			Type:       "object",
			Properties: map[string]provider.Schema{},
			Required:   []string{},
		},
	}
}

func (a *ReadTodosAdapter) Execute(ctx context.Context, args map[string]any) (string, error) {
	var req todo.ReadTodosRequest
	argsJSON, err := json.Marshal(args)
	if err != nil {
		return "", fmt.Errorf("failed to marshal args: %w", err)
	}
	if err := json.Unmarshal(argsJSON, &req); err != nil {
		return "", fmt.Errorf("failed to unmarshal request: %w", err)
	}

	resp, err := a.tool.Run(ctx, &req)
	if err != nil {
		return "", err
	}

	respJSON, err := json.Marshal(resp)
	if err != nil {
		return "", fmt.Errorf("failed to marshal response: %w", err)
	}
	return string(respJSON), nil
}

// WriteTodosAdapter adapts todo.WriteTodosTool to the Tool interface
type WriteTodosAdapter struct {
	tool *todo.WriteTodosTool
}

// NewWriteTodosAdapter creates a new WriteTodosAdapter
func NewWriteTodosAdapter(tool *todo.WriteTodosTool) *WriteTodosAdapter {
	return &WriteTodosAdapter{tool: tool}
}

func (a *WriteTodosAdapter) Name() string {
	return "write_todos"
}

func (a *WriteTodosAdapter) Description() string {
	return "Replaces all todos in the in-memory store"
}

func (a *WriteTodosAdapter) Definition() provider.ToolDefinition {
	return provider.ToolDefinition{
		Name:        a.Name(),
		Description: a.Description(),
		Parameters: &provider.Schema{
			Type: "object",
			Properties: map[string]provider.Schema{
				"todos": {
					Type:        "array",
					Description: "List of todos",
					Items: &provider.Schema{
						Type: "object",
						Properties: map[string]provider.Schema{
							"description": {
								Type:        "string",
								Description: "Todo description",
							},
							"status": {
								Type:        "string",
								Description: "Todo status (pending, in_progress, completed, cancelled)",
							},
						},
						Required: []string{"description", "status"},
					},
				},
			},
			Required: []string{"todos"},
		},
	}
}

func (a *WriteTodosAdapter) Execute(ctx context.Context, args map[string]any) (string, error) {
	var req todo.WriteTodosRequest
	argsJSON, err := json.Marshal(args)
	if err != nil {
		return "", fmt.Errorf("failed to marshal args: %w", err)
	}
	if err := json.Unmarshal(argsJSON, &req); err != nil {
		return "", fmt.Errorf("failed to unmarshal request: %w", err)
	}

	resp, err := a.tool.Run(ctx, &req)
	if err != nil {
		return "", err
	}

	respJSON, err := json.Marshal(resp)
	if err != nil {
		return "", fmt.Errorf("failed to marshal response: %w", err)
	}
	return string(respJSON), nil
}
