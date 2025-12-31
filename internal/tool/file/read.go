package file

import (
	"context"
	"fmt"
	"os"

	"strings"

	"github.com/Cyclone1070/iav/internal/tool/helper/content"
	"github.com/Cyclone1070/iav/internal/tool/service/fs"
)

// fileReader defines the minimal filesystem operations needed for reading files.
type fileReader interface {
	Stat(path string) (os.FileInfo, error)
	ReadFileLines(path string, startLine, endLine int) (*fs.ReadFileLinesResult, error)
}

// checksumComputer defines the interface for checksum computation and updates.
type checksumComputer interface {
	Compute(data []byte) string
	Update(path string, checksum string)
}

// ReadFileTool handles file reading operations.
type ReadFileTool struct {
	fileOps         fileReader
	checksumManager checksumComputer
	pathResolver    pathResolver
}

// NewReadFileTool creates a new ReadFileTool with injected dependencies.
func NewReadFileTool(
	fileOps fileReader,
	checksumManager checksumComputer,
	pathResolver pathResolver,
) *ReadFileTool {
	if fileOps == nil {
		panic("fileOps is required")
	}
	if checksumManager == nil {
		panic("checksumManager is required")
	}
	if pathResolver == nil {
		panic("pathResolver is required")
	}
	return &ReadFileTool{
		fileOps:         fileOps,
		checksumManager: checksumManager,
		pathResolver:    pathResolver,
	}
}

// Run reads a file from the workspace with line-based pagination.
// It validates the path is within workspace boundaries, checks for binary content,
// enforces size limits, and caches checksums for full file reads.
// Returns an error if the file is binary, too large, or outside the workspace.
//
// Note: ctx is accepted for API consistency but not used - file I/O is synchronous.
func (t *ReadFileTool) Run(ctx context.Context, req *ReadFileRequest) (*ReadFileResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	abs, err := t.pathResolver.Abs(req.Path)
	if err != nil {
		return nil, err
	}
	rel, err := t.pathResolver.Rel(abs)
	if err != nil {
		return nil, err
	}

	// Get file info for total size
	info, err := t.fileOps.Stat(abs)
	if err != nil {
		return nil, fmt.Errorf("failed to stat %s: %w", abs, err)
	}

	// Check if it's a directory
	if info.IsDir() {
		return nil, fmt.Errorf("path is a directory: %s", abs)
	}

	// Read the lines (binary check and size check are done inside ReadFileLines)
	result, err := t.fileOps.ReadFileLines(abs, req.StartLine, req.EndLine)
	if err != nil {
		return nil, err // propagates "binary file", "file exceeds max size", and "failed to stat" errors
	}

	// Only cache checksum if we read the entire file
	if req.StartLine == 1 && req.EndLine == 0 {
		checksum := t.checksumManager.Compute([]byte(result.Content))
		t.checksumManager.Update(abs, checksum)
	}

	return &ReadFileResponse{
		AbsolutePath: abs,
		RelativePath: rel,
		Size:         info.Size(),
		Content:      formatFileContent(result.Content, result.StartLine),
		StartLine:    result.StartLine,
		EndLine:      result.EndLine,
		TotalLines:   result.TotalLines,
	}, nil
}

// formatFileContent wraps file content in <file> tags and adds line number prefixes.
func formatFileContent(text string, startLine int) string {
	if text == "" {
		return "<file>\n(Empty file)\n</file>"
	}

	var sb strings.Builder
	sb.WriteString("<file>\n")

	lines := content.SplitLines(text)

	for i, line := range lines {
		lineNum := startLine + i
		sb.WriteString(fmt.Sprintf("%05d| %s\n", lineNum, line))
	}

	sb.WriteString("</file>")
	return sb.String()
}
