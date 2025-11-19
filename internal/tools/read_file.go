package tools

import (
	"fmt"

	"github.com/Cyclone1070/deployforme/internal/tools/models"
	"github.com/Cyclone1070/deployforme/internal/tools/services"
)

// ReadFile reads a file using injected dependencies
func ReadFile(ctx *models.WorkspaceContext, path string, offset *int64, limit *int64) (*models.ReadFileResponse, error) {
	// Resolve path
	abs, rel, err := services.Resolve(ctx, path)
	if err != nil {
		return nil, err
	}

	// Get file info (single stat syscall)
	info, err := ctx.FS.Stat(abs)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	// Check if it's a directory using info we already have
	if info.IsDir() {
		return nil, fmt.Errorf("path is a directory, not a file")
	}

	// Enforce size limit
	if info.Size() > ctx.MaxFileSize {
		return nil, models.ErrTooLarge
	}

	// Derive offset and limit
	var actualOffset, actualLimit int64
	if offset != nil {
		actualOffset = *offset
		if actualOffset < 0 {
			return nil, models.ErrInvalidOffset
		}
	}
	if limit != nil {
		actualLimit = *limit
		if actualLimit < 0 {
			return nil, models.ErrInvalidLimit
		}
	}

	// Read the file range (single open+read syscall)
	contentBytes, err := ctx.FS.ReadFileRange(abs, actualOffset, actualLimit)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Check for binary using content we already read
	if ctx.BinaryDetector.IsBinaryContent(contentBytes) {
		return nil, models.ErrBinaryFile
	}

	// Convert to string
	content := string(contentBytes)

	// Only cache checksum if we read the entire file
	isFullRead := actualOffset == 0 && int64(len(contentBytes)) == info.Size()

	if isFullRead {
		checksum := ctx.ChecksumManager.Compute(contentBytes)
		ctx.ChecksumManager.Update(abs, checksum)
	}

	return &models.ReadFileResponse{
		AbsolutePath: abs,
		RelativePath: rel,
		Size:         info.Size(),
		Content:      content,
	}, nil
}
