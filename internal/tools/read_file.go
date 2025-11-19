package tools

import (
	"fmt"
)

// ReadFile reads a file using injected dependencies
func ReadFile(ctx *WorkspaceContext, path string, offset *int64, limit *int64) (*ReadFileResponse, error) {
	// Resolve path
	abs, rel, err := Resolve(ctx, path)
	if err != nil {
		return nil, err
	}

	// Check if it's a directory
	isDir, err := IsDirectory(ctx, path)
	if err != nil {
		return nil, err
	}
	if isDir {
		return nil, fmt.Errorf("path is a directory, not a file")
	}

	// Get file info
	info, err := ctx.FS.Stat(abs)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	// Enforce size limit
	if info.Size() > ctx.MaxFileSize {
		return nil, ErrTooLarge
	}

	// Check for binary
	isBinary, err := ctx.BinaryDetector.IsBinary(abs)
	if err != nil {
		return nil, fmt.Errorf("failed to check if file is binary: %w", err)
	}
	if isBinary {
		return nil, ErrBinaryFile
	}

	// Derive offset and limit
	var actualOffset, actualLimit int64
	if offset != nil {
		actualOffset = *offset
		if actualOffset < 0 {
			return nil, ErrInvalidOffset
		}
	}
	if limit != nil {
		actualLimit = *limit
		if actualLimit < 0 {
			return nil, ErrInvalidLimit
		}
	}

	// Read the file range
	contentBytes, err := ctx.FS.ReadFileRange(abs, actualOffset, actualLimit)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Convert to string
	content := string(contentBytes)

	// Only cache checksum if we read the entire file
	isFullRead := actualOffset == 0 && int64(len(contentBytes)) == info.Size()

	if isFullRead {
		checksum := ctx.ChecksumManager.Compute(contentBytes)
		ctx.ChecksumManager.Update(abs, checksum)
	}

	return &ReadFileResponse{
		AbsolutePath: abs,
		RelativePath: rel,
		Size:         info.Size(),
		Content:      content,
	}, nil
}
