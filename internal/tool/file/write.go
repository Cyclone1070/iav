package file

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Cyclone1070/iav/internal/config"
	toolserrors "github.com/Cyclone1070/iav/internal/tool/errutil"
	"github.com/Cyclone1070/iav/internal/tool/pathutil"
)

// WriteFileTool handles file writing operations.
type WriteFileTool struct {
	fileOps         fileOps
	pathResolver    pathResolver
	binaryDetector  binaryDetector
	checksumManager checksumManager
	config          *config.Config
	workspaceRoot   string
}

// NewWriteFileTool creates a new WriteFileTool with injected dependencies.
func NewWriteFileTool(
	fileOps fileOps,
	pathResolver pathResolver,
	binaryDetector binaryDetector,
	checksumManager checksumManager,
	cfg *config.Config,
	workspaceRoot string,
) *WriteFileTool {
	return &WriteFileTool{
		fileOps:         fileOps,
		pathResolver:    pathResolver,
		binaryDetector:  binaryDetector,
		checksumManager: checksumManager,
		config:          cfg,
		workspaceRoot:   workspaceRoot,
	}
}

// Run creates a new file in the workspace with the specified content and permissions.
// It validates the path is within workspace boundaries, checks for binary content,
// enforces size limits, and writes atomically using a temp file + rename pattern.
// Returns an error if the file already exists, is binary, too large, or outside the workspace.
//
// Note: ctx is accepted for API consistency but not used - file I/O is synchronous.
func (t *WriteFileTool) Run(ctx context.Context, req WriteFileRequest) (*WriteFileResponse, error) {
	// Resolve path
	abs, rel, err := pathutil.Resolve(t.workspaceRoot, t.pathResolver, req.Path)
	if err != nil {
		return nil, err
	}

	// Check if file already exists
	_, err = t.fileOps.Stat(abs)
	if err == nil {
		return nil, toolserrors.ErrFileExists
	}
	if !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to check if file exists: %w", err)
	}

	parentDir := filepath.Dir(abs)
	if err := t.fileOps.EnsureDirs(parentDir); err != nil {
		return nil, fmt.Errorf("failed to create parent directories: %w", err)
	}

	contentBytes := []byte(req.Content)

	if t.binaryDetector.IsBinaryContent(contentBytes) {
		return nil, toolserrors.ErrBinaryFile
	}

	maxFileSize := t.config.Tools.MaxFileSize

	if int64(len(contentBytes)) > maxFileSize {
		return nil, toolserrors.ErrTooLarge
	}

	filePerm := os.FileMode(0644)
	if req.Perm != nil {
		filePerm = *req.Perm & 0777
	}

	// Write the file atomically
	if err := t.fileOps.WriteFileAtomic(abs, contentBytes, filePerm); err != nil {
		return nil, fmt.Errorf("failed to write file: %w", err)
	}

	// Compute checksum and update cache
	checksum := t.checksumManager.Compute(contentBytes)
	t.checksumManager.Update(abs, checksum)

	return &WriteFileResponse{
		AbsolutePath: abs,
		RelativePath: rel,
		BytesWritten: len(contentBytes),
		FileMode:     uint32(filePerm),
	}, nil
}
