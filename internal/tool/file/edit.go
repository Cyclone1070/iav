package file

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/Cyclone1070/iav/internal/config"
	"github.com/Cyclone1070/iav/internal/tool/helper/content"
)

// fileEditor defines the minimal filesystem operations needed for editing files.
type fileEditor interface {
	Stat(path string) (os.FileInfo, error)
	ReadFileRange(path string, offset, limit int64) ([]byte, error)
	WriteFileAtomic(path string, content []byte, perm os.FileMode) error
}

// checksumManager defines the interface for full checksum management.
type checksumManager interface {
	Compute(data []byte) string
	Get(path string) (checksum string, ok bool)
	Update(path string, checksum string)
}

// EditFileTool handles file editing operations.
type EditFileTool struct {
	fileOps         fileEditor
	checksumManager checksumManager
	config          *config.Config
	pathResolver    pathResolver
}

// NewEditFileTool creates a new EditFileTool with injected dependencies.
func NewEditFileTool(
	fileOps fileEditor,
	checksumManager checksumManager,
	cfg *config.Config,
	pathResolver pathResolver,
) *EditFileTool {
	if fileOps == nil {
		panic("fileOps is required")
	}
	if checksumManager == nil {
		panic("checksumManager is required")
	}
	if cfg == nil {
		panic("cfg is required")
	}
	if pathResolver == nil {
		panic("pathResolver is required")
	}
	return &EditFileTool{
		fileOps:         fileOps,
		checksumManager: checksumManager,
		config:          cfg,
		pathResolver:    pathResolver,
	}
}

// Run applies edit operations to an existing file in the workspace.
// It detects concurrent modifications by comparing file checksums and validates
// operations before applying them. The file is written atomically.
//
// Note: There is a narrow race condition window between checksum validation and write.
// For guaranteed conflict-free edits, external file locking would be required.
//
// Note: ctx is accepted for API consistency but not used - file I/O is synchronous.
func (t *EditFileTool) Run(ctx context.Context, req *EditFileRequest) (*EditFileResponse, error) {
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

	// Check if file exists
	info, err := t.fileOps.Stat(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s", ErrFileMissing, abs)
		}
		return nil, &StatError{Path: abs, Cause: err}
	}

	// Read full file (single open+read syscall)
	contentBytes, err := t.fileOps.ReadFileRange(abs, 0, 0)
	if err != nil {
		return nil, &ReadError{Path: abs, Cause: err}
	}

	// Check for binary content
	if content.IsBinaryContent(contentBytes) {
		return nil, fmt.Errorf("%w: %s", ErrBinaryFile, abs)
	}

	content := string(contentBytes)

	// Compute current checksum
	currentChecksum := t.checksumManager.Compute(contentBytes)

	// Check for conflicts with cached version
	priorChecksum, ok := t.checksumManager.Get(abs)
	if ok && priorChecksum != currentChecksum {
		return nil, fmt.Errorf("%w: %s", ErrEditConflict, abs)
	}

	// Preserve original permissions
	originalPerm := info.Mode()

	// Apply operations sequentially
	operationsApplied := 0
	for _, op := range req.Operations {
		// Empty Before means append to end of file
		if op.Before == "" {
			// Append has exactly 1 logical "target" (end of file).
			// If count > 1 is specified, it's a mismatch since there's only 1 place to append.
			if op.ExpectedReplacements > 1 {
				return nil, fmt.Errorf("%w: append has 1 target, got %d", ErrReplacementCountMismatch, op.ExpectedReplacements)
			}
			content += op.After
			operationsApplied++
			continue
		}

		count := strings.Count(content, op.Before)
		if count == 0 {
			return nil, fmt.Errorf("%w: %s in %s", ErrSnippetNotFound, op.Before, abs)
		}

		expected := op.ExpectedReplacements

		if count != expected {
			return nil, fmt.Errorf("%w in %s: expected %d, found %d", ErrReplacementCountMismatch, abs, expected, count)
		}

		content = strings.Replace(content, op.Before, op.After, expected)
		operationsApplied++
	}

	newContentBytes := []byte(content)

	// Check size limit
	maxFileSize := t.config.Tools.MaxFileSize
	if int64(len(newContentBytes)) > maxFileSize {
		return nil, fmt.Errorf("%w: %s (size %d, limit %d)", ErrFileTooLarge, abs, len(newContentBytes), maxFileSize)
	}

	// Only revalidate if we had a cached checksum to check against
	// This optimizes the common case where files are edited without being read first
	if ok {
		revalidationBytes, err := t.fileOps.ReadFileRange(abs, 0, 0)
		if err != nil {
			return nil, &RevalidateError{Path: abs, Cause: err}
		}
		revalidationChecksum := t.checksumManager.Compute(revalidationBytes)
		if revalidationChecksum != currentChecksum {
			return nil, fmt.Errorf("%w during revalidation: %s", ErrEditConflict, abs)
		}
	}

	// Write the modified content atomically
	if err := t.fileOps.WriteFileAtomic(abs, newContentBytes, originalPerm); err != nil {
		return nil, &WriteError{Path: abs, Cause: err}
	}

	// Compute new checksum and update cache
	newChecksum := t.checksumManager.Compute(newContentBytes)
	t.checksumManager.Update(abs, newChecksum)

	return &EditFileResponse{
		AbsolutePath:      abs,
		RelativePath:      rel,
		OperationsApplied: operationsApplied,
		FileSize:          int64(len(newContentBytes)),
	}, nil
}
