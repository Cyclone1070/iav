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
	"github.com/Cyclone1070/iav/internal/tool/fsutil"
	"github.com/Cyclone1070/iav/internal/tool/gitutil"
	"github.com/Cyclone1070/iav/internal/tool/pathutil"
	"github.com/Cyclone1070/iav/internal/tool/search"
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
	canonicalRoot, err := pathutil.CanonicaliseRoot(workspaceRoot)
	assert.NoError(t, err)

	// Create dependencies
	cfg := config.DefaultConfig()
	osFS := fs.NewOSFileSystem()
	binaryDetector := fs.NewSystemBinaryDetector(cfg.Tools.BinaryDetectionSampleSize)
	checksumManager := fs.NewChecksumManager()

	// Create tool
	readFileTool := file.NewReadFileTool(osFS, binaryDetector, checksumManager, cfg, canonicalRoot)

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
	assert.Contains(t, def.InputSchema.Properties, "path")
}

func TestToolAdapter_AllTools(t *testing.T) {
	t.Parallel()

	// Create workspace
	workspaceRoot := t.TempDir()

	// Canonicalize workspace root
	canonicalRoot, err := pathutil.CanonicaliseRoot(workspaceRoot)
	assert.NoError(t, err)

	// Create dependencies
	cfg := config.DefaultConfig()
	osFS := fs.NewOSFileSystem()
	binaryDetector := fs.NewSystemBinaryDetector(cfg.Tools.BinaryDetectionSampleSize)
	checksumManager := fs.NewChecksumManager()
	commandExecutor := shell.NewOSCommandExecutor()
	todoStore := todo.NewInMemoryTodoStore()

	// Initialize gitignore service
	gitignoreService, err := gitignore.NewService(canonicalRoot, osFS)
	if err != nil {
		gitignoreService = &gitignore.NoOpService{}
	}

	// Docker configuration
	dockerConfig := shell.DockerConfig{
		CheckCommand: []string{"docker", "info"},
		StartCommand: []string{"open", "-a", "Docker"},
	}

	// Create all tools
	readFileTool := file.NewReadFileTool(osFS, binaryDetector, checksumManager, cfg, canonicalRoot)
	writeFileTool := file.NewWriteFileTool(osFS, binaryDetector, checksumManager, cfg, canonicalRoot)
	editFileTool := file.NewEditFileTool(osFS, binaryDetector, checksumManager, cfg, canonicalRoot)
	listDirectoryTool := directory.NewListDirectoryTool(osFS, gitignoreService, cfg, canonicalRoot)
	findFileTool := directory.NewFindFileTool(osFS, commandExecutor, cfg, canonicalRoot)
	searchContentTool := search.NewSearchContentTool(osFS, commandExecutor, cfg, canonicalRoot)
	shellTool := shell.NewShellTool(osFS, commandExecutor, cfg, dockerConfig, canonicalRoot)
	readTodosTool := todo.NewReadTodosTool(todoStore)
	writeTodosTool := todo.NewWriteTodosTool(todoStore)

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
		assert.NotNil(t, def.InputSchema)

		// Schema is non-empty (except for read_todos which has no args)
		if expected != "read_todos" {
			assert.NotEmpty(t, def.InputSchema.Properties, "Tool %s schema properties should not be empty", expected)
		}
	}
}

func TestToolAdapter_ErrorHandling(t *testing.T) {
	t.Parallel()

	// Create workspace
	workspaceRoot := t.TempDir()

	// Canonicalize workspace root
	canonicalRoot, err := pathutil.CanonicaliseRoot(workspaceRoot)
	assert.NoError(t, err)

	// Create dependencies
	cfg := config.DefaultConfig()
	osFS := fs.NewOSFileSystem()
	binaryDetector := fs.NewSystemBinaryDetector(cfg.Tools.BinaryDetectionSampleSize)
	checksumManager := fs.NewChecksumManager()

	// Create tool
	readFileTool := file.NewReadFileTool(osFS, binaryDetector, checksumManager, cfg, canonicalRoot)

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
