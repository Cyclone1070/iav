package services

import (
	"fmt"
	"strings"

	"github.com/Cyclone1070/iav/internal/ui/models"
)

// FormatToolDescription generates a user-friendly description from tool args
func FormatToolDescription(name string, args map[string]any) string {
	switch name {
	case "EditFile", "edit_file":
		if path, ok := args["path"].(string); ok {
			return fmt.Sprintf("EditFile %s", path)
		}
	case "ReadFile", "read_file":
		if path, ok := args["path"].(string); ok {
			return fmt.Sprintf("ReadFile %s", path)
		}
	case "WriteFile", "write_file":
		if path, ok := args["path"].(string); ok {
			return fmt.Sprintf("WriteFile %s", path)
		}
	case "Shell", "shell", "run_shell":
		if cmd, ok := args["command"].(string); ok {
			return fmt.Sprintf("Shell '%s'", cmd)
		}
		if cmdSlice, ok := args["command"].([]string); ok && len(cmdSlice) > 0 {
			return fmt.Sprintf("Shell '%s'", strings.Join(cmdSlice, " "))
		}
	case "FindFile", "find_file":
		if pattern, ok := args["pattern"].(string); ok {
			return fmt.Sprintf("FindFile '%s'", pattern)
		}
	case "SearchContent", "search_content":
		if query, ok := args["query"].(string); ok {
			return fmt.Sprintf("SearchContent '%s'", query)
		}
	case "ListDirectory", "list_directory":
		if path, ok := args["path"].(string); ok {
			return fmt.Sprintf("ListDirectory %s", path)
		}
	}
	return name
}

// RenderPreview renders a tool preview for permission requests
func RenderPreview(preview *models.ToolPreview) string {
	if preview == nil {
		return ""
	}

	switch preview.Type {
	case "edit_operations":
		return renderEditPreview(preview.Data)
	case "shell_command":
		return renderShellPreview(preview.Data)
	default:
		return ""
	}
}

func renderEditPreview(data map[string]any) string {
	var sb strings.Builder
	path, _ := data["path"].(string)
	sb.WriteString(fmt.Sprintf("File: %s\n\n", path))

	ops, ok := data["operations"].([]any)
	if !ok {
		return fmt.Sprintf("Edit operations for %s (details unavailable)", path)
	}

	for i, op := range ops {
		sb.WriteString(fmt.Sprintf("Operation %d:\n", i+1))

		if opMap, ok := op.(map[string]any); ok {
			if before, ok := opMap["before"].(string); ok {
				sb.WriteString("  - " + strings.ReplaceAll(before, "\n", "\n  - ") + "\n")
			}
			if after, ok := opMap["after"].(string); ok {
				sb.WriteString("  + " + strings.ReplaceAll(after, "\n", "\n  + ") + "\n")
			}
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

func renderShellPreview(data map[string]any) string {
	cmd, ok := data["command"].(string)
	if !ok {
		if cmdSlice, ok := data["command"].([]string); ok {
			cmd = strings.Join(cmdSlice, " ")
		}
	}
	return fmt.Sprintf("$ %s", cmd)
}
