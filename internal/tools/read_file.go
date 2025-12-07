package tools

import (
	"context"
	"fmt"

	"github.com/Cyclone1070/iav/internal/config"
	"github.com/Cyclone1070/iav/internal/tools/models"
	"github.com/Cyclone1070/iav/internal/tools/services"
)

// ReadFile reads a file from the workspace with optional offset and limit for partial reads.
// It validates the path is within workspace boundaries, checks for binary content,
// enforces size limits, and caches checksums for full file reads.
// Returns an error if the file is binary, too large, or outside the workspace.
func ReadFile(ctx context.Context, wCtx *models.WorkspaceContext, req models.ReadFileRequest) (*models.ReadFileResponse, error) {
	// Resolve path
	abs, rel, err := services.Resolve(wCtx, req.Path)
	if err != nil {
		return nil, err
	}

	// Get file info (single stat syscall)
	info, err := wCtx.FS.Stat(abs)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	// Check if it's a directory using info we already have
	if info.IsDir() {
		return nil, fmt.Errorf("path is a directory, not a file")
	}

	// Enforce size limit
	maxFileSize := config.DefaultConfig().Tools.MaxFileSize
	if wCtx.Config != nil {
		maxFileSize = wCtx.Config.Tools.MaxFileSize
	}
	if info.Size() > maxFileSize {
		return nil, models.ErrTooLarge
	}

	// Derive offset and limit
	var actualOffset, actualLimit int64
	if req.Offset != nil {
		actualOffset = *req.Offset
		if actualOffset < 0 {
			return nil, models.ErrInvalidOffset
		}
	}
	if req.Limit != nil {
		actualLimit = *req.Limit
		if actualLimit < 0 {
			return nil, models.ErrInvalidLimit
		}
	}

	// Read the file range (single open+read syscall)
	contentBytes, err := wCtx.FS.ReadFileRange(abs, actualOffset, actualLimit)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Check for binary using content we already read
	if wCtx.BinaryDetector.IsBinaryContent(contentBytes) {
		return nil, models.ErrBinaryFile
	}

	// Convert to string
	content := string(contentBytes)

	// Only cache checksum if we read the entire file
	isFullRead := actualOffset == 0 && int64(len(contentBytes)) == info.Size()

	if isFullRead {
		checksum := wCtx.ChecksumManager.Compute(contentBytes)
		wCtx.ChecksumManager.Update(abs, checksum)
	}

	return &models.ReadFileResponse{
		AbsolutePath: abs,
		RelativePath: rel,
		Size:         info.Size(),
		Content:      content,
	}, nil
}
