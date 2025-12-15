package fsutil

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	toolserrors "github.com/Cyclone1070/iav/internal/tool/errutil"
)

// writeSyncCloser defines the minimal interface for a writable file handle.
// This abstraction allows testing without depending on concrete *os.File.
type writeSyncCloser interface {
	io.Writer
	Sync() error
	Close() error
	Name() string
}

// OSFileSystem implements filesystem operations using the local OS filesystem primitives.
// It uses internal function fields to enable testability via functional injection.
type OSFileSystem struct {
	// Internal syscall wrappers for testability
	createTemp func(dir, pattern string) (writeSyncCloser, error)
	rename     func(oldpath, newpath string) error
	chmod      func(name string, mode os.FileMode) error
	remove     func(name string) error
}

// NewOSFileSystem creates a new OSFileSystem with real OS syscalls.
func NewOSFileSystem() *OSFileSystem {
	return &OSFileSystem{
		createTemp: func(dir, pattern string) (writeSyncCloser, error) {
			return os.CreateTemp(dir, pattern)
		},
		rename: os.Rename,
		chmod:  os.Chmod,
		remove: os.Remove,
	}
}

// Stat returns file info for a path (follows symlinks).
func (r *OSFileSystem) Stat(path string) (os.FileInfo, error) {
	return os.Stat(path)
}

// Lstat returns file info for a path without following symlinks.
func (r *OSFileSystem) Lstat(path string) (os.FileInfo, error) {
	return os.Lstat(path)
}

// ReadFileRange reads a range of bytes from a file.
// If offset and limit are both 0, reads the entire file.
func (r *OSFileSystem) ReadFileRange(path string, offset, limit int64) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return nil, err
	}

	fileSize := info.Size()

	// If both offset and limit are 0, read entire file
	if offset == 0 && limit == 0 {
		content, err := io.ReadAll(file)
		if err != nil {
			return nil, err
		}
		return content, nil
	}

	// Validate offset
	if offset < 0 {
		return nil, toolserrors.ErrInvalidOffset
	}

	if offset >= fileSize {
		return []byte{}, nil
	}

	// Seek to offset
	_, err = file.Seek(offset, io.SeekStart)
	if err != nil {
		return nil, err
	}

	// Calculate how much to read
	remaining := fileSize - offset
	var readSize int64

	if limit == 0 {
		readSize = remaining
	} else {
		readSize = min(remaining, limit)
	}

	content := make([]byte, readSize)
	n, err := file.Read(content)
	if err != nil && err != io.EOF {
		return nil, err
	}

	return content[:n], nil
}

// WriteFileAtomic writes content to a file atomically using temp file + rename pattern.
// This ensures that if the process crashes mid-write, the original file remains intact.
// The temp file is created in the same directory as the target to ensure atomic rename.
func (r *OSFileSystem) WriteFileAtomic(path string, content []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)

	tmpFile, err := r.createTemp(dir, ".tmp-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}

	tmpPath := tmpFile.Name()
	needsCleanup := true

	defer func() {
		if tmpFile != nil {
			_ = tmpFile.Close()
		}
		if needsCleanup {
			_ = r.remove(tmpPath)
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
	if err := r.rename(tmpPath, path); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}
	needsCleanup = false

	if err := r.chmod(path, perm); err != nil {
		return fmt.Errorf("failed to set file permissions: %w", err)
	}

	return nil
}

// EnsureDirs creates parent directories recursively if they don't exist.
func (r *OSFileSystem) EnsureDirs(path string) error {
	return os.MkdirAll(path, 0o755)
}

// Readlink reads the target of a symlink.
func (r *OSFileSystem) Readlink(path string) (string, error) {
	return os.Readlink(path)
}

// UserHomeDir returns the current user's home directory.
func (r *OSFileSystem) UserHomeDir() (string, error) {
	return os.UserHomeDir()
}

// ListDir lists the contents of a directory.
// Returns a slice of FileInfo for each entry in the directory.
func (r *OSFileSystem) ListDir(path string) ([]os.FileInfo, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	infos := make([]os.FileInfo, 0, len(entries))
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			return nil, err
		}
		infos = append(infos, info)
	}

	return infos, nil
}
