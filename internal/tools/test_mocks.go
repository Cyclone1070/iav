package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// mockFileInfo implements FileInfo
type mockFileInfo struct {
	name  string
	size  int64
	mode  os.FileMode
	isDir bool
}

func (f *mockFileInfo) Name() string      { return f.name }
func (f *mockFileInfo) Size() int64       { return f.size }
func (f *mockFileInfo) Mode() os.FileMode { return f.mode }
func (f *mockFileInfo) IsDir() bool       { return f.isDir }

// mockFileHandle represents a file handle for temp files
type mockFileHandle struct {
	fs      *MockFileSystem
	path    string
	content []byte
	closed  bool
}

// Write implements FileHandle.Write
func (h *mockFileHandle) Write(data []byte) (int, error) {
	h.fs.mu.Lock()
	defer h.fs.mu.Unlock()

	if err, ok := h.fs.opErrors["Write"]; ok {
		return 0, err
	}

	if h.closed {
		return 0, fmt.Errorf("file is closed")
	}

	h.content = append(h.content, data...)
	return len(data), nil
}

// Sync implements FileHandle.Sync
func (h *mockFileHandle) Sync() error {
	h.fs.mu.Lock()
	defer h.fs.mu.Unlock()

	if err, ok := h.fs.opErrors["Sync"]; ok {
		return err
	}

	if h.closed {
		return fmt.Errorf("file is closed")
	}

	// In mock, sync is a no-op
	return nil
}

// Close implements FileHandle.Close
func (h *mockFileHandle) Close() error {
	h.fs.mu.Lock()
	defer h.fs.mu.Unlock()

	if err, ok := h.fs.opErrors["Close"]; ok {
		return err
	}

	if h.closed {
		return fmt.Errorf("file already closed")
	}

	h.closed = true
	return nil
}

// MockFileSystem implements FileSystem with in-memory storage
type MockFileSystem struct {
	mu          sync.RWMutex
	files       map[string][]byte          // path -> content
	fileInfos   map[string]*mockFileInfo   // path -> metadata
	symlinks    map[string]string          // symlink path -> target path
	dirs        map[string]bool            // path -> is directory
	errors      map[string]error           // path -> error to return
	opErrors    map[string]error           // operation -> error to return (e.g., "CreateTemp", "Write", "Sync", "Close", "Rename", "Chmod", "Remove")
	tempFiles   map[string]*mockFileHandle // temp path -> handle
	maxFileSize int64
}

// NewMockFileSystem creates a new mock filesystem
func NewMockFileSystem(maxFileSize int64) *MockFileSystem {
	return &MockFileSystem{
		files:       make(map[string][]byte),
		fileInfos:   make(map[string]*mockFileInfo),
		symlinks:    make(map[string]string),
		dirs:        make(map[string]bool),
		errors:      make(map[string]error),
		opErrors:    make(map[string]error),
		tempFiles:   make(map[string]*mockFileHandle),
		maxFileSize: maxFileSize,
	}
}

// SetError sets an error to return for a specific path
func (f *MockFileSystem) SetError(path string, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.errors[path] = err
}

// SetOperationError sets an error to return for a specific operation.
// Operations: "CreateTemp", "Write", "Sync", "Close", "Rename", "Chmod", "Remove"
func (f *MockFileSystem) SetOperationError(operation string, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.opErrors[operation] = err
}

// CreateFile creates a file with content
func (f *MockFileSystem) CreateFile(path string, content []byte, perm os.FileMode) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.files[path] = content
	f.fileInfos[path] = &mockFileInfo{
		name:  filepath.Base(path),
		size:  int64(len(content)),
		mode:  perm,
		isDir: false,
	}
	f.dirs[path] = false
}

// CreateDir creates a directory
func (f *MockFileSystem) CreateDir(path string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.dirs[path] = true
	f.fileInfos[path] = &mockFileInfo{
		name:  filepath.Base(path),
		size:  0,
		mode:  os.ModeDir | 0o755,
		isDir: true,
	}
}

// CreateSymlink creates a symlink
func (f *MockFileSystem) CreateSymlink(symlinkPath, targetPath string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.symlinks[symlinkPath] = targetPath
	f.fileInfos[symlinkPath] = &mockFileInfo{
		name:  filepath.Base(symlinkPath),
		size:  0,
		mode:  os.ModeSymlink | 0o777,
		isDir: false,
	}
}

func (f *MockFileSystem) Stat(path string) (FileInfo, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	if err, ok := f.errors[path]; ok {
		return nil, err
	}

	if info, ok := f.fileInfos[path]; ok {
		return info, nil
	}

	return nil, os.ErrNotExist
}

func (f *MockFileSystem) Lstat(path string) (FileInfo, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	if err, ok := f.errors[path]; ok {
		return nil, err
	}

	// Check if it's a symlink first - don't follow it
	if _, isSymlink := f.symlinks[path]; isSymlink {
		if info, ok := f.fileInfos[path]; ok {
			return info, nil
		}
	}

	// For non-symlinks, return regular file info
	if info, ok := f.fileInfos[path]; ok {
		return info, nil
	}

	return nil, os.ErrNotExist
}

func (f *MockFileSystem) ReadFileRange(path string, offset, limit int64) ([]byte, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	if err, ok := f.errors[path]; ok {
		return nil, err
	}

	content, ok := f.files[path]
	if !ok {
		return nil, os.ErrNotExist
	}

	fileSize := int64(len(content))

	if fileSize > f.maxFileSize {
		return nil, ErrTooLarge
	}

	if offset == 0 && limit == 0 {
		return content, nil
	}

	if offset < 0 {
		return nil, ErrInvalidOffset
	}

	if offset >= fileSize {
		return []byte{}, nil
	}

	remaining := fileSize - offset
	var readSize int64

	if limit == 0 {
		readSize = remaining
	} else {
		readSize = min(remaining, limit)
	}

	end := min(offset+readSize, fileSize)

	return content[offset:end], nil
}

func (f *MockFileSystem) EnsureDirs(path string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	cleaned := filepath.Clean(path)
	parts := strings.Split(cleaned, string(filepath.Separator))

	// Handle absolute paths
	var current string
	startIdx := 0
	if filepath.IsAbs(cleaned) {
		if len(parts) > 0 && parts[0] == "" {
			startIdx = 1
			current = "/"
		}
	}

	for i := startIdx; i < len(parts); i++ {
		part := parts[i]
		if part == "" {
			continue
		}
		switch current {
		case "":
			current = part
		case "/":
			current = "/" + part
		default:
			current = filepath.Join(current, part)
		}
		if !f.dirs[current] {
			f.dirs[current] = true
			f.fileInfos[current] = &mockFileInfo{
				name:  part,
				size:  0,
				mode:  os.ModeDir | 0o755,
				isDir: true,
			}
		}
	}
	return nil
}

func (f *MockFileSystem) IsDir(path string) (bool, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	if err, ok := f.errors[path]; ok {
		return false, err
	}

	if info, ok := f.fileInfos[path]; ok {
		return info.IsDir(), nil
	}

	return false, os.ErrNotExist
}

func (f *MockFileSystem) Readlink(path string) (string, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	if err, ok := f.errors[path]; ok {
		return "", err
	}

	target, ok := f.symlinks[path]
	if !ok {
		return "", fmt.Errorf("not a symlink: %s", path)
	}

	return target, nil
}

func (f *MockFileSystem) UserHomeDir() (string, error) {
	return "/home/user", nil
}

func (f *MockFileSystem) CreateTemp(dir, pattern string) (string, FileHandle, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if err, ok := f.opErrors["CreateTemp"]; ok {
		return "", nil, err
	}

	// Generate a temp file path
	tempPath := filepath.Join(dir, ".tmp-12345")
	handle := &mockFileHandle{
		fs:      f,
		path:    tempPath,
		content: []byte{},
		closed:  false,
	}
	f.tempFiles[tempPath] = handle

	return tempPath, handle, nil
}

func (f *MockFileSystem) Rename(oldpath, newpath string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if err, ok := f.opErrors["Rename"]; ok {
		return err
	}

	// Check if oldpath is a temp file
	if handle, ok := f.tempFiles[oldpath]; ok {
		// Move temp file content to new path
		f.files[newpath] = handle.content
		f.fileInfos[newpath] = &mockFileInfo{
			name:  filepath.Base(newpath),
			size:  int64(len(handle.content)),
			mode:  0o644,
			isDir: false,
		}
		f.dirs[newpath] = false
		delete(f.tempFiles, oldpath)
		return nil
	}

	// Regular rename
	if content, ok := f.files[oldpath]; ok {
		f.files[newpath] = content
		if info, ok := f.fileInfos[oldpath]; ok {
			f.fileInfos[newpath] = &mockFileInfo{
				name:  filepath.Base(newpath),
				size:  info.size,
				mode:  info.mode,
				isDir: info.isDir,
			}
		}
		delete(f.files, oldpath)
		delete(f.fileInfos, oldpath)
		return nil
	}

	return os.ErrNotExist
}

func (f *MockFileSystem) Chmod(name string, mode os.FileMode) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if err, ok := f.opErrors["Chmod"]; ok {
		return err
	}

	if info, ok := f.fileInfos[name]; ok {
		info.mode = mode
		return nil
	}

	return os.ErrNotExist
}

func (f *MockFileSystem) Remove(name string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if err, ok := f.opErrors["Remove"]; ok {
		return err
	}

	// Check if it's a temp file
	if _, ok := f.tempFiles[name]; ok {
		delete(f.tempFiles, name)
		return nil
	}

	// Regular file removal
	if _, ok := f.files[name]; ok {
		delete(f.files, name)
		delete(f.fileInfos, name)
		delete(f.dirs, name)
		return nil
	}

	return os.ErrNotExist
}

// GetTempFiles returns all temp file paths (for testing cleanup verification)
func (f *MockFileSystem) GetTempFiles() []string {
	f.mu.RLock()
	defer f.mu.RUnlock()

	paths := make([]string, 0, len(f.tempFiles))
	for path := range f.tempFiles {
		paths = append(paths, path)
	}
	return paths
}

// MockBinaryDetector implements BinaryDetector with configurable behaviour
type MockBinaryDetector struct {
	binaryPaths   map[string]bool
	binaryContent map[string]bool // content hash -> is binary
}

// NewMockBinaryDetector creates a new mock binary detector
func NewMockBinaryDetector() *MockBinaryDetector {
	return &MockBinaryDetector{
		binaryPaths:   make(map[string]bool),
		binaryContent: make(map[string]bool),
	}
}

// SetBinaryPath marks a path as binary
func (f *MockBinaryDetector) SetBinaryPath(path string, isBinary bool) {
	f.binaryPaths[path] = isBinary
}

func (f *MockBinaryDetector) IsBinary(path string) (bool, error) {
	if isBinary, ok := f.binaryPaths[path]; ok {
		return isBinary, nil
	}
	// Default: check for NUL bytes
	return false, nil
}

func (f *MockBinaryDetector) IsBinaryContent(content []byte) bool {
	sampleSize := min(len(content), BinaryDetectionSampleSize)

	for i := range sampleSize {
		if content[i] == 0 {
			return true
		}
	}

	return false
}

