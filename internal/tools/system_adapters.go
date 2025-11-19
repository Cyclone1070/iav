package tools

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
)

const (
	// DefaultMaxFileSize is the default maximum file size (5 MB)
	DefaultMaxFileSize = 5 * 1024 * 1024
	// BinaryDetectionSampleSize is how many bytes to sample for binary detection
	BinaryDetectionSampleSize = 4096
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

func (r *OSFileSystem) Stat(path string) (FileInfo, error) {
	return os.Stat(path)
}

func (r *OSFileSystem) Lstat(path string) (FileInfo, error) {
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
		return nil, ErrTooLarge
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
		return nil, ErrInvalidOffset
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

func (r *OSFileSystem) CreateTemp(dir, pattern string) (string, FileHandle, error) {
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

func (r *OSFileSystem) IsDir(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	return info.IsDir(), nil
}

func (r *OSFileSystem) Readlink(path string) (string, error) {
	return os.Readlink(path)
}

func (r *OSFileSystem) UserHomeDir() (string, error) {
	return os.UserHomeDir()
}

// SystemBinaryDetector implements BinaryDetector using local heuristics
type SystemBinaryDetector struct{}

func (r *SystemBinaryDetector) IsBinary(path string) (bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer file.Close()

	buf := make([]byte, BinaryDetectionSampleSize)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		return false, err
	}

	for i := range n {
		if buf[i] == 0 {
			return true, nil
		}
	}

	return false, nil
}

func (r *SystemBinaryDetector) IsBinaryContent(content []byte) bool {
	sampleSize := min(len(content), BinaryDetectionSampleSize)

	for i := range sampleSize {
		if content[i] == 0 {
			return true
		}
	}

	return false
}

// SHA256Checksum implements ChecksumComputer using SHA-256
type SHA256Checksum struct{}

func (r *SHA256Checksum) ComputeChecksum(data []byte) string {
	return computeChecksum(data)
}

// computeChecksum computes SHA-256 checksum of data
func computeChecksum(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}
