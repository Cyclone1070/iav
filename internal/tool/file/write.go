package file

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Cyclone1070/iav/internal/config"
	"github.com/Cyclone1070/iav/internal/tool/helper/content"
	"github.com/Cyclone1070/iav/internal/tool/service/path"
)

// fileWriter defines the minimal filesystem operations needed for writing files.
type fileWriter interface {
	Stat(path string) (os.FileInfo, error)
	WriteFileAtomic(path string, content []byte, perm os.FileMode) error
	EnsureDirs(path string) error
}

// checksumUpdater defines the interface for checksum computation and updates.
type checksumUpdater interface {
	Compute(data []byte) string
	Update(path string, checksum string)
}

// WriteFileTool handles file writing operations.
type WriteFileTool struct {
	fileOps         fileWriter
	checksumManager checksumUpdater
	config          *config.Config
	pathResolver    *path.Resolver
}

// NewWriteFileTool creates a new WriteFileTool with injected dependencies.
func NewWriteFileTool(
	fileOps fileWriter,
	checksumManager checksumUpdater,
	cfg *config.Config,
	pathResolver *path.Resolver,
) *WriteFileTool {
	return &WriteFileTool{
		fileOps:         fileOps,
		checksumManager: checksumManager,
		config:          cfg,
		pathResolver:    pathResolver,
	}
}

// Run creates a new file in the workspace with the specified content and permissions.
// It validates the path is within workspace boundaries, checks for binary content,
// enforces size limits, and writes atomically using a temp file + rename pattern.
// Returns an error if the file already exists, is binary, too large, or outside the workspace.
//
// Note: ctx is accepted for API consistency but not used - file I/O is synchronous.
func (t *WriteFileTool) Run(ctx context.Context, req *WriteFileRequest) (*WriteFileResponse, error) {
	if err := req.Validate(t.config); err != nil {
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

	// Check if file already exists
	_, err = t.fileOps.Stat(abs)
	if err == nil {
		return nil, fmt.Errorf("%w: %s", ErrFileExists, abs)
	}
	if !os.IsNotExist(err) {
		return nil, &StatError{Path: abs, Cause: err}
	}

	parentDir := filepath.Dir(abs)
	if err := t.fileOps.EnsureDirs(parentDir); err != nil {
		return nil, &EnsureDirsError{Path: parentDir, Cause: err}
	}

	contentBytes := []byte(req.Content)

	// Check for binary content
	if content.IsBinaryContent(contentBytes) {
		return nil, fmt.Errorf("%w: %s", ErrBinaryFile, abs)
	}

	// Runtime limit check (redundant but safe)
	maxFileSize := t.config.Tools.MaxFileSize
	if int64(len(contentBytes)) > maxFileSize {
		return nil, fmt.Errorf("%w: %s (size %d, limit %d)", ErrFileTooLarge, abs, len(contentBytes), maxFileSize)
	}

	perm := os.FileMode(0644)
	if req.Perm != nil {
		perm = *req.Perm
	}

	// Write the file atomically
	if err := t.fileOps.WriteFileAtomic(abs, contentBytes, perm); err != nil {
		return nil, &WriteError{Path: abs, Cause: err}
	}

	// Compute checksum and update cache
	checksum := t.checksumManager.Compute(contentBytes)
	t.checksumManager.Update(abs, checksum)

	return &WriteFileResponse{
		AbsolutePath: abs,
		RelativePath: rel,
		BytesWritten: len(contentBytes),
		FileMode:     uint32(perm),
	}, nil
}
