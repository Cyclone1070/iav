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
	"github.com/Cyclone1070/iav/internal/tools"
	"github.com/Cyclone1070/iav/internal/tools/models"
	"github.com/Cyclone1070/iav/internal/tools/services"
	"github.com/stretchr/testify/assert"
)

func TestToolAdapter_ReadFile(t *testing.T) {
	t.Parallel()

	// Create temporary workspace with test file
	workspaceRoot := t.TempDir()
	testFile := filepath.Join(workspaceRoot, "test.txt")
	err := os.WriteFile(testFile, []byte("Hello World"), 0644)
	assert.NoError(t, err)

	// Create workspace context
	fileSystem := services.NewOSFileSystem(config.DefaultConfig().Tools.MaxFileSize)
	ctx := &models.WorkspaceContext{
		FS:              fileSystem,
		BinaryDetector:  &services.SystemBinaryDetector{},
		ChecksumManager: services.NewChecksumManager(),
		WorkspaceRoot:   workspaceRoot,
		CommandExecutor: &services.OSCommandExecutor{},
	}

	// Create adapter
	adapter := orchadapter.NewReadFile(ctx)

	// Execute with args
	args := map[string]any{
		"path": "test.txt",
	}

	result, err := adapter.Execute(context.Background(), args)
	assert.NoError(t, err)

	// Result is valid JSON
	var response models.ReadFileResponse
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

	// Create workspace context
	workspaceRoot := t.TempDir()
	fileSystem := services.NewOSFileSystem(config.DefaultConfig().Tools.MaxFileSize)
	gitignoreSvc, _ := services.NewGitignoreService(workspaceRoot, fileSystem)

	ctx := &models.WorkspaceContext{
		FS:               fileSystem,
		BinaryDetector:   &services.SystemBinaryDetector{},
		ChecksumManager:  services.NewChecksumManager(),
		WorkspaceRoot:    workspaceRoot,
		GitignoreService: gitignoreSvc,
		CommandExecutor:  &services.OSCommandExecutor{},
	}

	// Create all adapters
	adapters := map[string]orchadapter.Tool{
		"read_file":      orchadapter.NewReadFile(ctx),
		"write_file":     orchadapter.NewWriteFile(ctx),
		"edit_file":      orchadapter.NewEditFile(ctx),
		"list_directory": orchadapter.NewListDirectory(ctx),
		"run_shell":      orchadapter.NewShell(&tools.ShellTool{CommandExecutor: ctx.CommandExecutor}, ctx),
		"search_content": orchadapter.NewSearchContent(ctx),
		"find_file":      orchadapter.NewFindFile(ctx),
		"read_todos":     orchadapter.NewReadTodos(ctx),
		"write_todos":    orchadapter.NewWriteTodos(ctx),
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

	// Create workspace context
	workspaceRoot := t.TempDir()
	ctx := &models.WorkspaceContext{
		FS:              services.NewOSFileSystem(config.DefaultConfig().Tools.MaxFileSize),
		BinaryDetector:  &services.SystemBinaryDetector{},
		ChecksumManager: services.NewChecksumManager(),
		WorkspaceRoot:   workspaceRoot,
		CommandExecutor: &services.OSCommandExecutor{},
	}

	// Create adapter
	adapter := orchadapter.NewReadFile(ctx)

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
