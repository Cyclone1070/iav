package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Cyclone1070/iav/internal/config"
	"github.com/Cyclone1070/iav/internal/tools/models"
	"github.com/Cyclone1070/iav/internal/tools/services"
)

// WriteFile creates a new file in the workspace with the specified content and permissions.
// It validates the path is within workspace boundaries, checks for binary content,
// enforces size limits, and writes atomically using a temp file + rename pattern.
// Returns an error if the file already exists, is binary, too large, or outside the workspace.
func WriteFile(ctx context.Context, wCtx *models.WorkspaceContext, req models.WriteFileRequest) (*models.WriteFileResponse, error) {
	// Resolve path
	abs, rel, err := services.Resolve(wCtx, req.Path)
	if err != nil {
		return nil, err
	}

	// Check if file already exists
	_, err = wCtx.FS.Stat(abs)
	if err == nil {
		return nil, models.ErrFileExists
	}
	if !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to check if file exists: %w", err)
	}

	parentDir := filepath.Dir(abs)
	if err := wCtx.FS.EnsureDirs(parentDir); err != nil {
		return nil, fmt.Errorf("failed to create parent directories: %w", err)
	}

	contentBytes := []byte(req.Content)

	if wCtx.BinaryDetector.IsBinaryContent(contentBytes) {
		return nil, models.ErrBinaryFile
	}

	maxFileSize := config.DefaultConfig().Tools.MaxFileSize
	if wCtx.Config != nil {
		maxFileSize = wCtx.Config.Tools.MaxFileSize
	}

	if int64(len(contentBytes)) > maxFileSize {
		return nil, models.ErrTooLarge
	}

	filePerm := os.FileMode(0644)
	if req.Perm != nil {
		// Only allow standard permission bits (owner/group/other rwx)
		if *req.Perm&^os.FileMode(0777) != 0 {
			return nil, fmt.Errorf("invalid file permissions: only standard permission bits (0-0777) allowed, got %o", *req.Perm)
		}
		filePerm = *req.Perm & 0777
	}

	// Write the file atomically
	if err := writeFileAtomic(wCtx, abs, contentBytes, filePerm); err != nil {
		return nil, fmt.Errorf("failed to write file: %w", err)
	}

	// Compute checksum and update cache
	checksum := wCtx.ChecksumManager.Compute(contentBytes)
	wCtx.ChecksumManager.Update(abs, checksum)

	return &models.WriteFileResponse{
		AbsolutePath: abs,
		RelativePath: rel,
		BytesWritten: len(contentBytes),
		FileMode:     uint32(filePerm),
	}, nil
}

// writeFileAtomic writes content to a file atomically using temp file + rename pattern.
// This ensures that if the process crashes mid-write, the original file remains intact.
// The temp file is created in the same directory as the target to ensure atomic rename.
func writeFileAtomic(ctx *models.WorkspaceContext, path string, content []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)

	tmpPath, tmpFile, err := ctx.FS.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}

	needsCleanup := true

	defer func() {
		if tmpFile != nil {
			_ = tmpFile.Close()
		}
		if needsCleanup {
			_ = ctx.FS.Remove(tmpPath)
		}
	}()

	if _, err := tmpFile.Write(content); err != nil {
		return fmt.Errorf("failed to write to temp file: %w", err)
	}

	if err := tmpFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync temp file: %w", err)
	}

	// Close file before rename (required on some systems)
	if err := tmpFile.Close(); err != nil {
		tmpFile = nil
		return fmt.Errorf("failed to close temp file: %w", err)
	}
	tmpFile = nil

	// Atomic rename is the critical operation that ensures consistency
	if err := ctx.FS.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}
	needsCleanup = false

	if err := ctx.FS.Chmod(path, perm); err != nil {
		return fmt.Errorf("failed to set file permissions: %w", err)
	}

	return nil
}
