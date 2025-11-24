package services

import (
	"io"
	"os"

	"github.com/Cyclone1070/deployforme/internal/tools/models"
)

// OSFileSystem implements FileSystem using the local OS primitives.
// It enforces file size limits based on the MaxFileSize field.
type OSFileSystem struct {
	MaxFileSize int64
}

// NewOSFileSystem creates a new OSFileSystem with the specified max file size.
func NewOSFileSystem(maxFileSize int64) *OSFileSystem {
	return &OSFileSystem{
		MaxFileSize: maxFileSize,
	}
}

func (r *OSFileSystem) Stat(path string) (models.FileInfo, error) {
	return os.Stat(path)
}

func (r *OSFileSystem) Lstat(path string) (models.FileInfo, error) {
	return os.Lstat(path)
}

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

	// Check size limit
	if fileSize > r.MaxFileSize {
		return nil, models.ErrTooLarge
	}

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

func (r *OSFileSystem) CreateTemp(dir, pattern string) (string, models.FileHandle, error) {
	tmpFile, err := os.CreateTemp(dir, pattern)
	if err != nil {
		return "", nil, err
	}
	return tmpFile.Name(), tmpFile, nil
}

func (r *OSFileSystem) Rename(oldpath, newpath string) error {
	return os.Rename(oldpath, newpath)
}

func (r *OSFileSystem) Chmod(name string, mode os.FileMode) error {
	return os.Chmod(name, mode)
}

func (r *OSFileSystem) Remove(name string) error {
	return os.Remove(name)
}

func (r *OSFileSystem) EnsureDirs(path string) error {
	return os.MkdirAll(path, 0o755)
}

func (r *OSFileSystem) Readlink(path string) (string, error) {
	return os.Readlink(path)
}

func (r *OSFileSystem) UserHomeDir() (string, error) {
	return os.UserHomeDir()
}

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
