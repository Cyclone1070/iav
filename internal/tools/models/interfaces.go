package models

import (
	"os"
)

// FileInfo represents file metadata
type FileInfo interface {
	Name() string
	Size() int64
	Mode() os.FileMode
	IsDir() bool
}

// FileSystem provides filesystem operations.
// ReadFileRange must respect size limits provided by the context.
// EnsureDirs should create parent directories recursively, but only within the workspace boundary.
type FileSystem interface {
	// Stat returns file info for a path (follows symlinks)
	Stat(path string) (FileInfo, error)

	// Lstat returns file info for a path without following symlinks
	Lstat(path string) (FileInfo, error)

	// ReadFileRange reads a range of bytes from a file.
	// If offset and limit are both 0, reads the entire file.
	// Must enforce size limits based on context configuration.
	ReadFileRange(path string, offset, limit int64) ([]byte, error)

	// EnsureDirs creates parent directories if they don't exist.
	// Must only create directories within the workspace boundary.
	EnsureDirs(path string) error

	// IsDir checks if a path is a directory
	IsDir(path string) (bool, error)

	// Readlink reads the target of a symlink
	Readlink(path string) (string, error)

	// UserHomeDir returns the current user's home directory
	UserHomeDir() (string, error)

	// CreateTemp creates a temporary file in the specified directory with the given pattern.
	// Returns the path to the temp file and a file handle.
	CreateTemp(dir, pattern string) (string, FileHandle, error)

	// Rename atomically renames oldpath to newpath.
	Rename(oldpath, newpath string) error

	// Chmod changes the mode of the named file.
	Chmod(name string, mode os.FileMode) error

	// Remove removes the named file or directory.
	Remove(name string) error

	// ListDir lists the contents of a directory.
	// Returns a slice of FileInfo for each entry in the directory.
	ListDir(path string) ([]FileInfo, error)
}

// BinaryDetector checks if content is binary
type BinaryDetector interface {
	// IsBinary checks if a file contains binary data
	IsBinary(path string) (bool, error)
	// IsBinaryContent checks if content bytes contain binary data
	IsBinaryContent(content []byte) bool
}

// ChecksumManager manages checksum computation and caching.
// Implementations must be thread-safe.
type ChecksumManager interface {
	// Compute computes SHA-256 checksum of data
	Compute(data []byte) string
	// Get retrieves checksum for a file path
	Get(path string) (checksum string, ok bool)
	// Update stores or updates checksum for a file path
	Update(path string, checksum string)
	// Clear removes all cached checksums
	Clear()
}

// FileHandle represents a file handle for writing operations.
// This contains low level methods for writing, syncing, and closing files.
// Both *os.File and mock implementations satisfy this interface.
type FileHandle interface {
	Write([]byte) (int, error)
	Sync() error
	Close() error
}
