package file

import (
	"context"
	"fmt"

	"github.com/Cyclone1070/iav/internal/config"
	toolserrors "github.com/Cyclone1070/iav/internal/tool/errutil"
	"github.com/Cyclone1070/iav/internal/tool/pathutil"
)

// ReadFileTool handles file reading operations.
type ReadFileTool struct {
	fileOps         fileOps
	pathResolver    pathResolver
	binaryDetector  binaryDetector
	checksumManager checksumManager
	config          *config.Config
	workspaceRoot   string
}

// NewReadFileTool creates a new ReadFileTool with injected dependencies.
func NewReadFileTool(
	fileOps fileOps,
	pathResolver pathResolver,
	binaryDetector binaryDetector,
	checksumManager checksumManager,
	cfg *config.Config,
	workspaceRoot string,
) *ReadFileTool {
	return &ReadFileTool{
		fileOps:         fileOps,
		pathResolver:    pathResolver,
		binaryDetector:  binaryDetector,
		checksumManager: checksumManager,
		config:          cfg,
		workspaceRoot:   workspaceRoot,
	}
}

// Run reads a file from the workspace with optional offset and limit for partial reads.
// It validates the path is within workspace boundaries, checks for binary content,
// enforces size limits, and caches checksums for full file reads.
// Returns an error if the file is binary, too large, or outside the workspace.
//
// Note: ctx is accepted for API consistency but not used - file I/O is synchronous.
func (t *ReadFileTool) Run(ctx context.Context, req ReadFileRequest) (*ReadFileResponse, error) {
	// Resolve path
	abs, rel, err := pathutil.Resolve(t.workspaceRoot, t.pathResolver, req.Path)
	if err != nil {
		return nil, err
	}

	// Get file info (single stat syscall)
	info, err := t.fileOps.Stat(abs)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	// Check if it's a directory using info we already have
	if info.IsDir() {
		return nil, fmt.Errorf("path is a directory, not a file")
	}

	// Enforce size limit
	maxFileSize := t.config.Tools.MaxFileSize
	if info.Size() > maxFileSize {
		return nil, toolserrors.ErrTooLarge
	}

	// Derive offset and limit
	var actualOffset, actualLimit int64
	if req.Offset != nil {
		actualOffset = *req.Offset
	}
	if req.Limit != nil {
		actualLimit = *req.Limit
	}

	// Read the file range (single open+read syscall)
	contentBytes, err := t.fileOps.ReadFileRange(abs, actualOffset, actualLimit)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Check for binary using content we already read
	if t.binaryDetector.IsBinaryContent(contentBytes) {
		return nil, toolserrors.ErrBinaryFile
	}

	// Convert to string
	content := string(contentBytes)

	// Only cache checksum if we read the entire file
	isFullRead := actualOffset == 0 && int64(len(contentBytes)) == info.Size()

	if isFullRead {
		checksum := t.checksumManager.Compute(contentBytes)
		t.checksumManager.Update(abs, checksum)
	}

	return &ReadFileResponse{
		AbsolutePath: abs,
		RelativePath: rel,
		Size:         info.Size(),
		Content:      content,
	}, nil
}
