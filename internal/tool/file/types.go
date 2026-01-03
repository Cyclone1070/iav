package file

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Cyclone1070/iav/internal/config"
	"github.com/Cyclone1070/iav/internal/tool"
)

// -- Read File --

type ReadFileRequest struct {
	Path   string `json:"path"`
	Offset int    `json:"offset,omitempty"` // 0-based start line
	Limit  int    `json:"limit,omitempty"`  // Max lines to return
}

func (r *ReadFileRequest) Display() string {
	return filepath.Base(r.Path)
}

func (r *ReadFileRequest) Validate(cfg *config.Config) error {
	if r.Path == "" {
		return fmt.Errorf("path is required")
	}
	if r.Offset < 0 {
		r.Offset = 0
	}
	if r.Limit <= 0 {
		r.Limit = cfg.Tools.DefaultReadFileLimit
	}
	// No MaxReadFileLimit check as per refined plan
	return nil
}

type ReadFileResponse struct {
	Content    string // RAW content (no line numbers, no tags)
	StartLine  int    // calculated: Offset + 1 (1-indexed)
	EndLine    int    // calculated: StartLine + actual_lines - 1
	TotalLines int
	Error      string // Set if the tool failed (e.g. file not found)
}

// LLMContent returns the formatted XML block with pagination hints
func (r *ReadFileResponse) LLMContent() string {
	if r.Error != "" {
		return fmt.Sprintf("Error: %s", r.Error)
	}

	if r.Content == "" {
		return fmt.Sprintf("<file>\n\n(End of file - total %d lines)\n</file>", r.TotalLines)
	}

	var sb strings.Builder
	sb.WriteString("<file>\n")

	lines := strings.Split(r.Content, "\n")
	for i, line := range lines {
		// Avoid extra empty line at end if content ends with \n
		if line == "" && i == len(lines)-1 {
			break
		}
		sb.WriteString(fmt.Sprintf("%05d| %s\n", r.StartLine+i, line))
	}

	if r.EndLine < r.TotalLines {
		sb.WriteString(fmt.Sprintf("\n(File has more lines. Use offset=%d to read more)", r.EndLine))
	} else {
		sb.WriteString(fmt.Sprintf("\n(End of file - total %d lines)", r.TotalLines))
	}

	sb.WriteString("\n</file>")
	return sb.String()
}

// Display returns the UI representation
func (r *ReadFileResponse) Display() tool.ToolDisplay {
	if r.Error != "" {
		return tool.StringDisplay("Bad request")
	}
	return nil
}

func (r ReadFileResponse) Success() bool {
	return r.Error == ""
}

// -- Write File --

type WriteFileRequest struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

func (r *WriteFileRequest) Validate(cfg *config.Config) error {
	if r.Path == "" {
		return fmt.Errorf("path is required")
	}
	if r.Content == "" {
		return fmt.Errorf("content is required")
	}
	if int64(len(r.Content)) > cfg.Tools.MaxFileSize {
		return fmt.Errorf("content too large: %d bytes exceeds limit %d", len(r.Content), cfg.Tools.MaxFileSize)
	}
	return nil
}

type WriteFileResponse struct {
	AbsolutePath string
	RelativePath string
	BytesWritten int
}

func (r WriteFileResponse) Success() bool {
	return true
}

// -- Edit File --

type EditOperation struct {
	Before               string `json:"before"`
	After                string `json:"after"`
	ExpectedReplacements int    `json:"expected_replacements,omitempty"`
}

type EditFileRequest struct {
	Path       string          `json:"path"`
	Operations []EditOperation `json:"operations"`
}

func (r *EditFileRequest) Display() string {
	return filepath.Base(r.Path)
}

func (r *EditFileRequest) Validate() error {
	if r.Path == "" {
		return fmt.Errorf("path is required")
	}
	if len(r.Operations) == 0 {
		return fmt.Errorf("operations are required")
	}
	for i := range r.Operations {
		if r.Operations[i].ExpectedReplacements <= 0 {
			r.Operations[i].ExpectedReplacements = 1
		}
	}
	return nil
}

type EditFileResponse struct {
	Path  string // File path for success message
	Error string // Set if the tool failed

	// For DiffDisplay
	Diff         string // Unified diff content
	AddedLines   int
	RemovedLines int
}

// LLMContent returns success message or error
func (r *EditFileResponse) LLMContent() string {
	if r.Error != "" {
		return fmt.Sprintf("Error: %s", r.Error)
	}
	return fmt.Sprintf("Successfully modified file: %s", r.Path)
}

// Display returns DiffDisplay for UI rendering
func (r *EditFileResponse) Display() tool.ToolDisplay {
	if r.Error != "" {
		return tool.StringDisplay("Bad request")
	}
	return tool.DiffDisplay{
		Diff:         r.Diff,
		AddedLines:   r.AddedLines,
		RemovedLines: r.RemovedLines,
	}
}

func (r EditFileResponse) Success() bool {
	return r.Error == ""
}
