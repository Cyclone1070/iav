//go:build integration

package adapter_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Cyclone1070/iav/internal/config"
	orchadapter "github.com/Cyclone1070/iav/internal/orchestrator/adapter"
	"github.com/Cyclone1070/iav/internal/tool/directory"
	"github.com/Cyclone1070/iav/internal/tool/file"
	"github.com/Cyclone1070/iav/internal/tool/search"
	"github.com/Cyclone1070/iav/internal/tool/service/executor"
	"github.com/Cyclone1070/iav/internal/tool/service/fs"
	"github.com/Cyclone1070/iav/internal/tool/service/git"
	"github.com/Cyclone1070/iav/internal/tool/service/hash"
	"github.com/Cyclone1070/iav/internal/tool/service/path"
	"github.com/Cyclone1070/iav/internal/tool/shell"
	"github.com/Cyclone1070/iav/internal/tool/todo"
	"github.com/stretchr/testify/assert"
)

func TestToolAdapter_ReadFile(t *testing.T) {
	t.Parallel()

	// Create temporary workspace with test file
	workspaceRoot := t.TempDir()
	testFile := filepath.Join(workspaceRoot, "test.txt")
	err := os.WriteFile(testFile, []byte("Hello World"), 0644)
	assert.NoError(t, err)

	// Canonicalize workspace root
	canonicalRoot, err := path.CanonicaliseRoot(workspaceRoot)
	assert.NoError(t, err)

	// Create dependencies
	cfg := config.DefaultConfig()
	osFS := fs.NewOSFileSystem()
	checksumManager := hash.NewChecksumManager()
	pathResolver := path.NewResolver(canonicalRoot)

	// Create tool
	readFileTool := file.NewReadFileTool(osFS, checksumManager, cfg, pathResolver)

	// Create adapter
	adapter := orchadapter.NewReadFileAdapter(readFileTool)

	// Execute with args
	args := map[string]any{
		"path": "test.txt",
	}

	result, err := adapter.Execute(context.Background(), args)
	assert.NoError(t, err)

	// Result is valid JSON
	var response file.ReadFileResponse
	assert.NoError(t, json.Unmarshal([]byte(result), &response))

	// Content matches
	assert.Equal(t, "Hello World", response.Content)
	assert.Equal(t, "test.txt", response.RelativePath)

	// Tool definition matches signature
	def := adapter.Definition()
	assert.Equal(t, "read_file", def.Name)
	assert.Contains(t, def.Parameters.Properties, "path")
}

func TestToolAdapter_AllTools(t *testing.T) {
	t.Parallel()

	// Create workspace
	workspaceRoot := t.TempDir()

	// Canonicalize workspace root
	canonicalRoot, err := path.CanonicaliseRoot(workspaceRoot)
	assert.NoError(t, err)

	// Create dependencies
	cfg := config.DefaultConfig()
	osFS := fs.NewOSFileSystem()
	checksumManager := hash.NewChecksumManager()
	commandExecutor := executor.NewOSCommandExecutor(cfg)
	todoStore := todo.NewInMemoryTodoStore()
	pathResolver := path.NewResolver(canonicalRoot)

	// Initialize gitignore service
	var gitignoreService interface {
		ShouldIgnore(relativePath string) bool
	}
	svc, err := git.NewService(canonicalRoot, osFS)
	if err != nil {
		gitignoreService = &git.NoOpService{}
	} else {
		gitignoreService = svc
	}

	// Docker configuration
	dockerConfig := shell.DockerConfig{
		CheckCommand: []string{"docker", "info"},
		StartCommand: []string{"open", "-a", "Docker"},
	}

	// Create all tools
	readFileTool := file.NewReadFileTool(osFS, checksumManager, cfg, pathResolver)
	writeFileTool := file.NewWriteFileTool(osFS, checksumManager, cfg, pathResolver)
	editFileTool := file.NewEditFileTool(osFS, checksumManager, cfg, pathResolver)
	listDirectoryTool := directory.NewListDirectoryTool(osFS, gitignoreService, cfg, pathResolver)
	findFileTool := directory.NewFindFileTool(osFS, commandExecutor, cfg, pathResolver)
	searchContentTool := search.NewSearchContentTool(osFS, commandExecutor, cfg, pathResolver)
	shellTool := shell.NewShellTool(osFS, commandExecutor, cfg, dockerConfig, pathResolver)
	readTodosTool := todo.NewReadTodosTool(todoStore, cfg)
	writeTodosTool := todo.NewWriteTodosTool(todoStore, cfg)

	// Create all adapters
	adapters := map[string]orchadapter.Tool{
		"read_file":      orchadapter.NewReadFileAdapter(readFileTool),
		"write_file":     orchadapter.NewWriteFileAdapter(writeFileTool),
		"edit_file":      orchadapter.NewEditFileAdapter(editFileTool),
		"list_directory": orchadapter.NewListDirectoryAdapter(listDirectoryTool),
		"find_file":      orchadapter.NewFindFileAdapter(findFileTool),
		"search_content": orchadapter.NewSearchContentAdapter(searchContentTool),
		"run_shell":      orchadapter.NewShellAdapter(shellTool),
		"read_todos":     orchadapter.NewReadTodosAdapter(readTodosTool),
		"write_todos":    orchadapter.NewWriteTodosAdapter(writeTodosTool),
	}

	// All expected tools present
	expectedTools := []string{
		"read_file", "write_file", "edit_file",
		"list_directory", "run_shell", "search_content",
		"find_file", "read_todos", "write_todos",
	}

	for _, expected := range expectedTools {
		adapter := adapters[expected]
		assert.NotNil(t, adapter, "Tool %s should exist", expected)

		// Definition is valid
		def := adapter.Definition()
		assert.Equal(t, expected, def.Name)
		assert.NotEmpty(t, def.Description)
		assert.NotNil(t, def.Parameters)

		// Schema is non-empty (except for read_todos which has no args)
		if expected != "read_todos" {
			assert.NotEmpty(t, def.Parameters.Properties, "Tool %s schema properties should not be empty", expected)
		}
	}
}

func TestToolAdapter_ErrorHandling(t *testing.T) {
	t.Parallel()

	// Create workspace
	workspaceRoot := t.TempDir()

	// Canonicalize workspace root
	canonicalRoot, err := path.CanonicaliseRoot(workspaceRoot)
	assert.NoError(t, err)

	// Create dependencies
	cfg := config.DefaultConfig()
	osFS := fs.NewOSFileSystem()
	checksumManager := hash.NewChecksumManager()
	pathResolver := path.NewResolver(canonicalRoot)

	// Create tool
	readFileTool := file.NewReadFileTool(osFS, checksumManager, cfg, pathResolver)

	// Create adapter
	adapter := orchadapter.NewReadFileAdapter(readFileTool)

	// Invalid arguments (wrong type)
	invalidArgs := map[string]any{
		"path": 12345, // Should be string
	}

	// Adapter should handle validation
	result, err := adapter.Execute(context.Background(), invalidArgs)

	// Error is returned (not panic)
	assert.Error(t, err)
	assert.Empty(t, result)
}
