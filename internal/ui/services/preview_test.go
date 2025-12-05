package services

import (
	"testing"

	"github.com/Cyclone1070/iav/internal/ui/models"
	"github.com/stretchr/testify/assert"
)

func TestFormatToolDescription_EditFile(t *testing.T) {
	name := "EditFile"
	args := map[string]any{"path": "internal/main.go"}
	result := FormatToolDescription(name, args)
	assert.Equal(t, "EditFile internal/main.go", result)
}

func TestFormatToolDescription_Shell(t *testing.T) {
	name := "Shell"
	args := map[string]any{"command": "docker ps -a"}
	result := FormatToolDescription(name, args)
	assert.Equal(t, "Shell 'docker ps -a'", result)
}

func TestFormatToolDescription_Shell_Slice(t *testing.T) {
	name := "Shell"
	args := map[string]any{"command": []string{"ls", "-la"}}
	result := FormatToolDescription(name, args)
	assert.Equal(t, "Shell 'ls -la'", result)
}

func TestFormatToolDescription_UnknownTool(t *testing.T) {
	name := "MysteryTool"
	args := map[string]any{}
	result := FormatToolDescription(name, args)
	assert.Equal(t, "MysteryTool", result)
}

func TestFormatToolDescription_MissingArgs(t *testing.T) {
	name := "EditFile"
	args := map[string]any{} // No "path"
	result := FormatToolDescription(name, args)
	assert.Equal(t, "EditFile", result)
}

func TestFormatToolDescription_WrongArgType(t *testing.T) {
	name := "EditFile"
	args := map[string]any{"path": 12345} // int not string
	result := FormatToolDescription(name, args)
	assert.Equal(t, "EditFile", result)
}

func TestRenderPreview_EditOperations(t *testing.T) {
	preview := &models.ToolPreview{
		Type: "edit_operations",
		Data: map[string]any{"path": "main.go"},
	}
	result := RenderPreview(preview)
	assert.Contains(t, result, "main.go")
}

func TestRenderPreview_ShellCommand(t *testing.T) {
	preview := &models.ToolPreview{
		Type: "shell_command",
		Data: map[string]any{"command": "ls -la"},
	}
	result := RenderPreview(preview)
	assert.Equal(t, "$ ls -la", result)
}

func TestRenderPreview_NilPreview(t *testing.T) {
	var preview *models.ToolPreview = nil
	result := RenderPreview(preview)
	assert.Empty(t, result)
}
