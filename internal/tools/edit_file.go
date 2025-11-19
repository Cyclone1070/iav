package tools

import (
	"fmt"
	"os"
	"strings"
)

// EditFile applies edit operations using injected dependencies.
// It detects concurrent modifications by revalidating the file checksum
// immediately before writing to prevent race conditions.
func EditFile(ctx *WorkspaceContext, path string, operations []Operation) (*EditFileResponse, error) {
	// Resolve path
	abs, rel, err := Resolve(ctx, path)
	if err != nil {
		return nil, err
	}

	// Check if file exists
	info, err := ctx.FS.Stat(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrFileMissing
		}
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	// Check for binary
	isBinary, err := ctx.BinaryDetector.IsBinary(abs)
	if err != nil {
		return nil, fmt.Errorf("failed to check if file is binary: %w", err)
	}
	if isBinary {
		return nil, ErrBinaryFile
	}

	// Read full file
	contentBytes, err := ctx.FS.ReadFileRange(abs, 0, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	content := string(contentBytes)

	// Compute current checksum
	currentChecksum := ctx.ChecksumManager.Compute(contentBytes)

	// Check for conflicts with cached version
	priorChecksum, ok := ctx.ChecksumManager.Get(abs)
	if ok && priorChecksum != currentChecksum {
		return nil, ErrEditConflict
	}

	// Preserve original permissions
	originalPerm := info.Mode()

	// Apply operations sequentially
	operationsApplied := 0
	for i, op := range operations {
		if op.Before == "" {
			return nil, fmt.Errorf("operation %d: Before must be non-empty, include nearest meaningful context for append-style edits", i+1)
		}

		if op.ExpectedReplacements < 1 {
			return nil, fmt.Errorf("operation %d: ExpectedReplacements must be >= 1", i+1)
		}

		count := strings.Count(content, op.Before)

		if count == 0 {
			return nil, ErrSnippetNotFound
		}

		if count != op.ExpectedReplacements {
			return nil, ErrSnippetAmbiguous
		}

		content = strings.Replace(content, op.Before, op.After, op.ExpectedReplacements)
		operationsApplied++
	}

	newContentBytes := []byte(content)

	// Check size limit
	if int64(len(newContentBytes)) > ctx.MaxFileSize {
		return nil, ErrTooLarge
	}

	// Revalidate file hasn't changed before writing (race condition prevention)
	// Re-read the file to ensure it hasn't been modified by another process
	revalidationBytes, err := ctx.FS.ReadFileRange(abs, 0, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to revalidate file before write: %w", err)
	}
	revalidationChecksum := ctx.ChecksumManager.Compute(revalidationBytes)
	if revalidationChecksum != currentChecksum {
		return nil, ErrEditConflict
	}

	// Write the modified content atomically
	if err := writeFileAtomic(ctx, abs, newContentBytes, originalPerm); err != nil {
		return nil, fmt.Errorf("failed to write edited file: %w", err)
	}

	// Get updated file info
	newInfo, err := ctx.FS.Stat(abs)
	if err != nil {
		return nil, fmt.Errorf("failed to stat edited file: %w", err)
	}

	// Compute new checksum and update cache
	newChecksum := ctx.ChecksumManager.Compute(newContentBytes)
	ctx.ChecksumManager.Update(abs, newChecksum)

	return &EditFileResponse{
		AbsolutePath:      abs,
		RelativePath:      rel,
		OperationsApplied: operationsApplied,
		FileSize:          newInfo.Size(),
	}, nil
}
