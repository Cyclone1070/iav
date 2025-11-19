package tools

import (
	"fmt"
	"os"
	"path/filepath"
)

// WriteFile creates a new file using injected dependencies
func WriteFile(ctx *WorkspaceContext, path string, content string, perm *os.FileMode) (*WriteFileResponse, error) {
	// Resolve path
	abs, rel, err := Resolve(ctx, path)
	if err != nil {
		return nil, err
	}

	// Check if file already exists
	_, err = ctx.FS.Stat(abs)
	if err == nil {
		return nil, ErrFileExists
	}
	if !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to check if file exists: %w", err)
	}

	// Ensure parent directories exist
	if err := EnsureParentDirs(ctx, path); err != nil {
		return nil, err
	}

	contentBytes := []byte(content)

	// Check for binary content
	if ctx.BinaryDetector.IsBinaryContent(contentBytes) {
		return nil, ErrBinaryFile
	}

	// Enforce size limit
	if int64(len(contentBytes)) > ctx.MaxFileSize {
		return nil, ErrTooLarge
	}

	// Default permissions
	filePerm := os.FileMode(0644)
	// Determine custom permissions if provided
	if perm != nil {
		// Only allow standard permission bits (owner/group/other rwx)
		if *perm&^os.FileMode(0777) != 0 {
			return nil, fmt.Errorf("invalid file permissions: only standard permission bits (0-0777) allowed, got %o", *perm)
		}
		filePerm = *perm & 0777
	}

	// Write the file atomically
	if err := writeFileAtomic(ctx, abs, contentBytes, filePerm); err != nil {
		return nil, fmt.Errorf("failed to write file: %w", err)
	}

	// Compute checksum and update cache
	checksum := ctx.ChecksumManager.Compute(contentBytes)
	ctx.ChecksumManager.Update(abs, checksum)

	return &WriteFileResponse{
		AbsolutePath: abs,
		RelativePath: rel,
		BytesWritten: len(contentBytes),
		FileMode:     uint32(filePerm),
	}, nil
}

// writeFileAtomic writes content to a file atomically using temp file + rename pattern.
// This ensures that if the process crashes mid-write, the original file remains intact.
func writeFileAtomic(ctx *WorkspaceContext, path string, content []byte, perm os.FileMode) error {
	// Get directory for temp file
	dir := filepath.Dir(path)

	// Create temporary file in same directory
	tmpPath, tmpFile, err := ctx.FS.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}

	// Track whether we need to clean up the temp file
	// (set to false after successful rename)
	needsCleanup := true

	// Ensure cleanup on error
	defer func() {
		// Close file handle if still open
		if tmpFile != nil {
			// Best-effort close. File may already be closed or in bad state.
			// Any close errors here don't affect the write operation's success.
			_ = tmpFile.Close()
		}
		// Always try to remove temp file if rename didn't succeed
		if needsCleanup {
			// Best-effort cleanup. Temp file is in workspace directory with .tmp- prefix.
			// If removal fails, OS will eventually clean it up. This won't cause
			// security issues since temp files are within workspace boundaries.
			_ = ctx.FS.Remove(tmpPath)
		}
	}()

	// Write content to temp file
	if _, err := tmpFile.Write(content); err != nil {
		return fmt.Errorf("failed to write to temp file: %w", err)
	}

	// Sync to ensure data is written to disk
	if err := tmpFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync temp file: %w", err)
	}

	// Close file before rename (required on some systems)
	if err := tmpFile.Close(); err != nil {
		// Set to nil to prevent double-close in defer
		tmpFile = nil
		// Still return error - cleanup will be attempted in defer
		return fmt.Errorf("failed to close temp file: %w", err)
	}
	tmpFile = nil // Prevent cleanup in defer

	// Atomic rename - this is the critical operation that makes it atomic
	if err := ctx.FS.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}
	needsCleanup = false // Rename succeeded, no need to remove temp file

	// Set permissions on the final file
	if err := ctx.FS.Chmod(path, perm); err != nil {
		return fmt.Errorf("failed to set file permissions: %w", err)
	}

	return nil
}
