package file

import (
	"context"
	"fmt"
	"os"

	"github.com/Cyclone1070/iav/internal/config"
	"github.com/Cyclone1070/iav/internal/tool/pathutil"
)

// fileReader defines the minimal filesystem operations needed for reading files.
type fileReader interface {
	Stat(path string) (os.FileInfo, error)
	Lstat(path string) (os.FileInfo, error)
	Readlink(path string) (string, error)
	UserHomeDir() (string, error)
	ReadFileRange(path string, offset, limit int64) ([]byte, error)
}

// checksumComputer defines the interface for checksum computation and updates.
type checksumComputer interface {
	Compute(data []byte) string
	Update(path string, checksum string)
}

// ReadFileTool handles file reading operations.
type ReadFileTool struct {
	fileOps         fileReader
	binaryDetector  binaryDetector
	checksumManager checksumComputer
	config          *config.Config
	workspaceRoot   string
}

// NewReadFileTool creates a new ReadFileTool with injected dependencies.
func NewReadFileTool(
	fileOps fileReader,
	binaryDetector binaryDetector,
	checksumManager checksumComputer,
	cfg *config.Config,
	workspaceRoot string,
) *ReadFileTool {
	return &ReadFileTool{
		fileOps:         fileOps,
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
func (t *ReadFileTool) Run(ctx context.Context, req *ReadFileRequest) (*ReadFileResponse, error) {
	if err := req.Validate(t.config); err != nil {
		return nil, err
	}

	abs, rel, err := pathutil.Resolve(t.workspaceRoot, t.fileOps, req.Path)
	if err != nil {
		return nil, err
	}

	// Get file info (single stat syscall)
	info, err := t.fileOps.Stat(abs)
	if err != nil {
		return nil, &StatError{Path: abs, Cause: err}
	}

	// Check if it's a directory using info we already have
	if info.IsDir() {
		return nil, fmt.Errorf("%w: %s", ErrIsDirectory, abs)
	}

	// Enforce size limit
	maxFileSize := t.config.Tools.MaxFileSize
	if info.Size() > maxFileSize {
		return nil, fmt.Errorf("%w: %s (size %d, limit %d)", ErrFileTooLarge, abs, info.Size(), maxFileSize)
	}

	var offset int64
	if req.Offset != nil {
		offset = *req.Offset
	}

	limit := maxFileSize
	if req.Limit != nil && *req.Limit > 0 {
		limit = *req.Limit
	}

	// Read the file range (single open+read syscall)
	contentBytes, err := t.fileOps.ReadFileRange(abs, offset, limit)
	if err != nil {
		return nil, &ReadError{Path: abs, Cause: err}
	}

	// Check for binary using content we already read
	if t.binaryDetector.IsBinaryContent(contentBytes) {
		return nil, fmt.Errorf("%w: %s", ErrBinaryFile, abs)
	}

	// Convert to string
	content := string(contentBytes)

	// Only cache checksum if we read the entire file
	isFullRead := offset == 0 && int64(len(contentBytes)) == info.Size()

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
