package tools

import (
	"fmt"
	"os"
	"strings"

	"github.com/Cyclone1070/deployforme/internal/tools/models"
	"github.com/Cyclone1070/deployforme/internal/tools/services"
)

// EditFile applies edit operations to an existing file in the workspace.
// It detects concurrent modifications by comparing file checksums and validates
// operations before applying them. The file is written atomically.
//
// Note: There is a narrow race condition window between checksum validation and write.
// For guaranteed conflict-free edits, external file locking would be required.
func EditFile(ctx *models.WorkspaceContext, req models.EditFileRequest) (*models.EditFileResponse, error) {
	// Resolve path
	abs, rel, err := services.Resolve(ctx, req.Path)
	if err != nil {
		return nil, err
	}

	// Check if file exists
	info, err := ctx.FS.Stat(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, models.ErrFileMissing
		}
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	// Read full file (single open+read syscall)
	contentBytes, err := ctx.FS.ReadFileRange(abs, 0, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Check for binary using content we already read
	if ctx.BinaryDetector.IsBinaryContent(contentBytes) {
		return nil, models.ErrBinaryFile
	}

	content := string(contentBytes)

	// Compute current checksum
	currentChecksum := ctx.ChecksumManager.Compute(contentBytes)

	// Check for conflicts with cached version
	priorChecksum, ok := ctx.ChecksumManager.Get(abs)
	if ok && priorChecksum != currentChecksum {
		return nil, models.ErrEditConflict
	}

	// Preserve original permissions
	originalPerm := info.Mode()

	// Apply operations sequentially
	operationsApplied := 0
	for i, op := range req.Operations {
		if op.Before == "" {
			return nil, fmt.Errorf("operation %d: Before must be non-empty, include nearest meaningful context for append-style edits", i+1)
		}

		if op.ExpectedReplacements < 1 {
			return nil, fmt.Errorf("operation %d: ExpectedReplacements must be >= 1", i+1)
		}

		count := strings.Count(content, op.Before)

		if count == 0 {
			return nil, models.ErrSnippetNotFound
		}

		if count != op.ExpectedReplacements {
			return nil, models.ErrSnippetAmbiguous
		}

		content = strings.Replace(content, op.Before, op.After, op.ExpectedReplacements)
		operationsApplied++
	}

	newContentBytes := []byte(content)

	// Check size limit
	if int64(len(newContentBytes)) > ctx.MaxFileSize {
		return nil, models.ErrTooLarge
	}

	// Only revalidate if we had a cached checksum to check against
	// This optimizes the common case where files are edited without being read first
	if ok {
		revalidationBytes, err := ctx.FS.ReadFileRange(abs, 0, 0)
		if err != nil {
			return nil, fmt.Errorf("failed to revalidate file before write: %w", err)
		}
		revalidationChecksum := ctx.ChecksumManager.Compute(revalidationBytes)
		if revalidationChecksum != currentChecksum {
			return nil, models.ErrEditConflict
		}
	}

	// Write the modified content atomically
	if err := writeFileAtomic(ctx, abs, newContentBytes, originalPerm); err != nil {
		return nil, fmt.Errorf("failed to write edited file: %w", err)
	}

	// Compute new checksum and update cache
	newChecksum := ctx.ChecksumManager.Compute(newContentBytes)
	ctx.ChecksumManager.Update(abs, newChecksum)

	return &models.EditFileResponse{
		AbsolutePath:      abs,
		RelativePath:      rel,
		OperationsApplied: operationsApplied,
		FileSize:          int64(len(newContentBytes)),
	}, nil
}
