package file

import (
	"context"
	"os"

	"github.com/Cyclone1070/iav/internal/config"
)

// fileReader defines the minimal filesystem operations needed for reading files.
type fileReader interface {
	Stat(path string) (os.FileInfo, error)
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
	// Runtime Validation
	abs := req.AbsPath()
	rel := req.RelPath()

	// Get file info (single stat syscall)
	info, err := t.fileOps.Stat(abs)
	if err != nil {
		return nil, &StatError{Path: abs, Cause: err}
	}

	// Check if it's a directory using info we already have
	if info.IsDir() {
		return nil, &IsDirectoryError{Path: abs}
	}

	// Enforce size limit
	maxFileSize := t.config.Tools.MaxFileSize
	if info.Size() > maxFileSize {
		return nil, &TooLargeError{Path: abs, Size: info.Size(), Limit: maxFileSize}
	}

	// Get offset and limit from validated request
	actualOffset := req.Offset()
	actualLimit := req.Limit()

	// Read the file range (single open+read syscall)
	contentBytes, err := t.fileOps.ReadFileRange(abs, actualOffset, actualLimit)
	if err != nil {
		return nil, &ReadError{Path: abs, Cause: err}
	}

	// Check for binary using content we already read
	if t.binaryDetector.IsBinaryContent(contentBytes) {
		return nil, &BinaryFileError{Path: abs}
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
