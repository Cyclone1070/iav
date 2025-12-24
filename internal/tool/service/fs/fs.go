package fs

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// OSFileSystem implements filesystem operations using the local OS filesystem primitives.
type OSFileSystem struct{}

// NewOSFileSystem creates a new OSFileSystem.
func NewOSFileSystem() *OSFileSystem {
	return &OSFileSystem{}
}

// Stat returns file info for a path (follows symlinks).
func (fs *OSFileSystem) Stat(path string) (os.FileInfo, error) {
	return os.Stat(path)
}

// Lstat returns file info for a path without following symlinks.
func (fs *OSFileSystem) Lstat(path string) (os.FileInfo, error) {
	return os.Lstat(path)
}

// ReadFileRange reads a range of bytes from a file.
// If offset and limit are both 0, reads the entire file.
func (fs *OSFileSystem) ReadFileRange(path string, offset, limit int64) ([]byte, error) {
	if offset < 0 {
		return nil, fmt.Errorf("%w: %d", ErrInvalidOffset, offset)
	}

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
func (fs *OSFileSystem) WriteFileAtomic(path string, content []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)

	tmpFile, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return &TempFileError{Dir: dir, Cause: err}
	}

	tmpPath := tmpFile.Name()
	needsCleanup := true

	defer func() {
		if tmpFile != nil {
			_ = tmpFile.Close()
		}
		if needsCleanup {
			_ = os.Remove(tmpPath)
		}
	}()

	if _, err := tmpFile.Write(content); err != nil {
		return &TempWriteError{Path: tmpPath, Cause: err}
	}

	if err := tmpFile.Sync(); err != nil {
		return &TempSyncError{Path: tmpPath, Cause: err}
	}

	// Close file before rename (required on some systems)
	if err := tmpFile.Close(); err != nil {
		tmpFile = nil
		return &TempCloseError{Path: tmpPath, Cause: err}
	}
	tmpFile = nil

	// Atomic rename is the critical operation that ensures consistency
	if err := os.Rename(tmpPath, path); err != nil {
		return &RenameError{Old: tmpPath, New: path, Cause: err}
	}
	needsCleanup = false

	if err := os.Chmod(path, perm); err != nil {
		return &ChmodError{Path: path, Mode: perm, Cause: err}
	}

	return nil
}

// EnsureDirs creates parent directories recursively if they don't exist.
func (fs *OSFileSystem) EnsureDirs(path string) error {
	return os.MkdirAll(path, 0o755)
}

// Readlink reads the target of a symlink.
func (fs *OSFileSystem) Readlink(path string) (string, error) {
	return os.Readlink(path)
}

// UserHomeDir returns the current user's home directory.
func (fs *OSFileSystem) UserHomeDir() (string, error) {
	return os.UserHomeDir()
}

// ListDir lists the contents of a directory.
// Returns a slice of FileInfo for each entry in the directory.
func (fs *OSFileSystem) ListDir(path string) ([]os.FileInfo, error) {
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
