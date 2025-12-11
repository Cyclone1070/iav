package services

import (
	"io"
	"os"

	"github.com/Cyclone1070/iav/internal/tools/models"
)

// OSFileSystem implements FileSystem using the local OS filesystem primitives.
// It enforces file size limits based on the MaxFileSize field.
type OSFileSystem struct {
}

// NewOSFileSystem creates a new OSFileSystem.
func NewOSFileSystem() *OSFileSystem {
	return &OSFileSystem{}
}

// Stat returns file info for a path (follows symlinks).
func (r *OSFileSystem) Stat(path string) (models.FileInfo, error) {
	return os.Stat(path)
}

// Lstat returns file info for a path without following symlinks.
func (r *OSFileSystem) Lstat(path string) (models.FileInfo, error) {
	return os.Lstat(path)
}

// ReadFileRange reads a range of bytes from a file.
// If offset and limit are both 0, reads the entire file.
// Enforces the MaxFileSize limit.
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
		return nil, models.ErrInvalidOffset
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

// CreateTemp creates a temporary file in the specified directory with the given pattern.
// Returns the path to the temp file and a file handle.
func (r *OSFileSystem) CreateTemp(dir, pattern string) (string, models.FileHandle, error) {
	tmpFile, err := os.CreateTemp(dir, pattern)
	if err != nil {
		return "", nil, err
	}
	return tmpFile.Name(), tmpFile, nil
}

// Rename atomically renames oldpath to newpath.
func (r *OSFileSystem) Rename(oldpath, newpath string) error {
	return os.Rename(oldpath, newpath)
}

// Chmod changes the mode of the named file.
func (r *OSFileSystem) Chmod(name string, mode os.FileMode) error {
	return os.Chmod(name, mode)
}

// Remove removes the named file or directory.
func (r *OSFileSystem) Remove(name string) error {
	return os.Remove(name)
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
func (r *OSFileSystem) ListDir(path string) ([]models.FileInfo, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	infos := make([]models.FileInfo, 0, len(entries))
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			return nil, err
		}
		infos = append(infos, info)
	}

	return infos, nil
}
