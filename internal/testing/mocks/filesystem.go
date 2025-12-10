package mocks

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/Cyclone1070/iav/internal/tools/models"
)

// MockFileInfo implements models.FileInfo
type MockFileInfo struct {
	NameVal  string
	SizeVal  int64
	ModeVal  os.FileMode
	IsDirVal bool
}

func (f *MockFileInfo) Name() string      { return f.NameVal }
func (f *MockFileInfo) Size() int64       { return f.SizeVal }
func (f *MockFileInfo) Mode() os.FileMode { return f.ModeVal }
func (f *MockFileInfo) IsDir() bool       { return f.IsDirVal }

// MockFileHandle represents a file handle for temp files
type MockFileHandle struct {
	Fs      *MockFileSystem
	Path    string
	Content []byte
	Closed  bool
}

// Write implements models.FileHandle.Write
func (h *MockFileHandle) Write(data []byte) (int, error) {
	h.Fs.Mu.Lock()
	defer h.Fs.Mu.Unlock()

	if err, ok := h.Fs.OpErrors["Write"]; ok {
		return 0, err
	}

	if h.Closed {
		return 0, fmt.Errorf("file is closed")
	}

	h.Content = append(h.Content, data...)
	return len(data), nil
}

// Sync implements models.FileHandle.Sync
func (h *MockFileHandle) Sync() error {
	h.Fs.Mu.Lock()
	defer h.Fs.Mu.Unlock()

	if err, ok := h.Fs.OpErrors["Sync"]; ok {
		return err
	}

	if h.Closed {
		return fmt.Errorf("file is closed")
	}

	// In mock, sync is a no-op
	return nil
}

// Close implements models.FileHandle.Close
func (h *MockFileHandle) Close() error {
	h.Fs.Mu.Lock()
	defer h.Fs.Mu.Unlock()

	if err, ok := h.Fs.OpErrors["Close"]; ok {
		return err
	}

	if h.Closed {
		return fmt.Errorf("file already closed")
	}

	h.Closed = true
	return nil
}

// MockFileSystem implements models.FileSystem with in-memory storage.
// This is the comprehensive mock for tool tests.
type MockFileSystem struct {
	Mu          sync.RWMutex
	Files       map[string][]byte          // path -> content
	FileInfos   map[string]*MockFileInfo   // path -> metadata
	Symlinks    map[string]string          // symlink path -> target path
	Dirs        map[string]bool            // path -> is directory
	Errors      map[string]error           // path -> error to return
	OpErrors    map[string]error           // operation -> error to return
	TempFiles   map[string]*MockFileHandle // temp path -> handle
	MaxFileSize int64
	// Counter for unique temp files
	TempCounter int
	HomeDir     string
	HomeDirErr  error
}

// NewMockFileSystem creates a new mock filesystem
func NewMockFileSystem(maxFileSize int64) *MockFileSystem {
	return &MockFileSystem{
		Files:       make(map[string][]byte),
		FileInfos:   make(map[string]*MockFileInfo),
		Symlinks:    make(map[string]string),
		Dirs:        make(map[string]bool),
		Errors:      make(map[string]error),
		OpErrors:    make(map[string]error),
		TempFiles:   make(map[string]*MockFileHandle),
		MaxFileSize: maxFileSize,
		TempCounter: 0,
		HomeDir:     "/home/user",
	}
}

// SetError sets an error to return for a specific path
func (f *MockFileSystem) SetError(path string, err error) {
	f.Mu.Lock()
	defer f.Mu.Unlock()
	f.Errors[path] = err
}

// SetOperationError sets an error to return for a specific operation.
func (f *MockFileSystem) SetOperationError(operation string, err error) {
	f.Mu.Lock()
	defer f.Mu.Unlock()
	f.OpErrors[operation] = err
}

// CreateFile creates a file with content
func (f *MockFileSystem) CreateFile(path string, content []byte, perm os.FileMode) {
	f.Mu.Lock()
	defer f.Mu.Unlock()
	f.Files[path] = content
	f.FileInfos[path] = &MockFileInfo{
		NameVal:  filepath.Base(path),
		SizeVal:  int64(len(content)),
		ModeVal:  perm,
		IsDirVal: false,
	}
	f.Dirs[path] = false
}

// CreateDir creates a directory
func (f *MockFileSystem) CreateDir(path string) {
	f.Mu.Lock()
	defer f.Mu.Unlock()
	f.Dirs[path] = true
	f.FileInfos[path] = &MockFileInfo{
		NameVal:  filepath.Base(path),
		SizeVal:  0,
		ModeVal:  os.ModeDir | 0o755,
		IsDirVal: true,
	}
}

// CreateSymlink creates a symlink
func (f *MockFileSystem) CreateSymlink(symlinkPath, targetPath string) {
	f.Mu.Lock()
	defer f.Mu.Unlock()
	f.Symlinks[symlinkPath] = targetPath
	f.FileInfos[symlinkPath] = &MockFileInfo{
		NameVal:  filepath.Base(symlinkPath),
		SizeVal:  0,
		ModeVal:  os.ModeSymlink | 0o777,
		IsDirVal: false,
	}
}

func (f *MockFileSystem) Stat(path string) (models.FileInfo, error) {
	f.Mu.RLock()
	defer f.Mu.RUnlock()

	if err, ok := f.Errors[path]; ok {
		return nil, err
	}

	if info, ok := f.FileInfos[path]; ok {
		return info, nil
	}

	return nil, &os.PathError{Op: "stat", Path: path, Err: os.ErrNotExist}
}

func (f *MockFileSystem) Lstat(path string) (models.FileInfo, error) {
	f.Mu.RLock()
	defer f.Mu.RUnlock()

	if err, ok := f.Errors[path]; ok {
		return nil, err
	}

	// Check if it's a symlink first - don't follow it
	if _, isSymlink := f.Symlinks[path]; isSymlink {
		if info, ok := f.FileInfos[path]; ok {
			return info, nil
		}
	}

	// For non-symlinks, return regular file info
	if info, ok := f.FileInfos[path]; ok {
		return info, nil
	}

	return nil, &os.PathError{Op: "lstat", Path: path, Err: os.ErrNotExist}
}

func (f *MockFileSystem) ReadFileRange(path string, offset, limit int64) ([]byte, error) {
	f.Mu.RLock()
	defer f.Mu.RUnlock()

	if err, ok := f.Errors[path]; ok {
		return nil, err
	}

	content, ok := f.Files[path]
	if !ok {
		return nil, &os.PathError{Op: "open", Path: path, Err: os.ErrNotExist}
	}

	fileSize := int64(len(content))

	if fileSize > f.MaxFileSize {
		return nil, models.ErrTooLarge
	}

	if offset == 0 && limit == 0 {
		return content, nil
	}

	if offset < 0 {
		return nil, models.ErrInvalidOffset
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

// ReadFile specifically for config.Loader interface compatibility
func (f *MockFileSystem) ReadFile(path string) ([]byte, error) {
	f.Mu.RLock()
	defer f.Mu.RUnlock()

	if err, ok := f.OpErrors["ReadFile"]; ok {
		return nil, err
	}

	if err, ok := f.Errors[path]; ok {
		return nil, err
	}

	content, ok := f.Files[path]
	if !ok {
		return nil, os.ErrNotExist
	}

	return content, nil
}

func (f *MockFileSystem) EnsureDirs(path string) error {
	f.Mu.Lock()
	defer f.Mu.Unlock()

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
		if !f.Dirs[current] {
			f.Dirs[current] = true
			f.FileInfos[current] = &MockFileInfo{
				NameVal:  part,
				SizeVal:  0,
				ModeVal:  os.ModeDir | 0o755,
				IsDirVal: true,
			}
		}
	}
	return nil
}

func (f *MockFileSystem) Readlink(path string) (string, error) {
	f.Mu.RLock()
	defer f.Mu.RUnlock()

	if err, ok := f.Errors[path]; ok {
		return "", err
	}

	target, ok := f.Symlinks[path]
	if !ok {
		return "", fmt.Errorf("not a symlink: %s", path)
	}

	return target, nil
}

func (f *MockFileSystem) UserHomeDir() (string, error) {
	return f.HomeDir, f.HomeDirErr
}

func (f *MockFileSystem) CreateTemp(dir, pattern string) (string, models.FileHandle, error) {
	f.Mu.Lock()
	defer f.Mu.Unlock()

	if err, ok := f.OpErrors["CreateTemp"]; ok {
		return "", nil, err
	}

	// Generate a temp file path
	f.TempCounter++
	tempPath := filepath.Join(dir, fmt.Sprintf(".tmp-%d", f.TempCounter))
	handle := &MockFileHandle{
		Fs:      f,
		Path:    tempPath,
		Content: []byte{},
		Closed:  false,
	}
	f.TempFiles[tempPath] = handle

	// Also add it to the regular files map and fileInfos for Stat/Lstat to work
	f.Files[tempPath] = []byte{}
	f.FileInfos[tempPath] = &MockFileInfo{
		NameVal:  filepath.Base(tempPath),
		SizeVal:  0,
		ModeVal:  0o600, // Default temp file permissions
		IsDirVal: false,
	}

	return tempPath, handle, nil
}

func (f *MockFileSystem) Rename(oldpath, newpath string) error {
	f.Mu.Lock()
	defer f.Mu.Unlock()

	if err, ok := f.OpErrors["Rename"]; ok {
		return err
	}

	// Check if oldpath is a temp file
	if handle, ok := f.TempFiles[oldpath]; ok {
		// Move temp file content to new path
		f.Files[newpath] = handle.Content
		f.FileInfos[newpath] = &MockFileInfo{
			NameVal:  filepath.Base(newpath),
			SizeVal:  int64(len(handle.Content)),
			ModeVal:  0o644,
			IsDirVal: false,
		}
		f.Dirs[newpath] = false
		delete(f.TempFiles, oldpath)
		return nil
	}

	// Regular rename
	if content, ok := f.Files[oldpath]; ok {
		f.Files[newpath] = content
		if info, ok := f.FileInfos[oldpath]; ok {
			f.FileInfos[newpath] = &MockFileInfo{
				NameVal:  filepath.Base(newpath),
				SizeVal:  info.SizeVal,
				ModeVal:  info.ModeVal,
				IsDirVal: info.IsDirVal,
			}
		}
		delete(f.Files, oldpath)
		delete(f.FileInfos, oldpath)
		return nil
	}

	return &os.PathError{Op: "rename", Path: oldpath, Err: os.ErrNotExist}
}

func (f *MockFileSystem) Chmod(name string, mode os.FileMode) error {
	f.Mu.Lock()
	defer f.Mu.Unlock()

	if err, ok := f.OpErrors["Chmod"]; ok {
		return err
	}

	if info, ok := f.FileInfos[name]; ok {
		info.ModeVal = mode
		return nil
	}

	return &os.PathError{Op: "chmod", Path: name, Err: os.ErrNotExist}
}

func (f *MockFileSystem) Remove(name string) error {
	f.Mu.Lock()
	defer f.Mu.Unlock()

	if err, ok := f.OpErrors["Remove"]; ok {
		return err
	}

	// Check if it's a temp file
	if _, ok := f.TempFiles[name]; ok {
		delete(f.TempFiles, name)
		return nil
	}

	// Regular file removal
	if _, ok := f.Files[name]; ok {
		delete(f.Files, name)
		delete(f.FileInfos, name)
		delete(f.Dirs, name)
		return nil
	}

	return &os.PathError{Op: "remove", Path: name, Err: os.ErrNotExist}
}

func (f *MockFileSystem) ListDir(path string) ([]models.FileInfo, error) {
	f.Mu.RLock()
	defer f.Mu.RUnlock()

	// Check for operation-level errors
	if err, ok := f.OpErrors["ListDir"]; ok {
		return nil, err
	}

	// Check for path-specific errors
	if err, ok := f.Errors[path]; ok {
		return nil, err
	}

	// Verify path exists and is a directory
	info, ok := f.FileInfos[path]
	if !ok {
		return nil, &os.PathError{Op: "readdir", Path: path, Err: os.ErrNotExist}
	}

	if !info.IsDir() {
		return nil, fmt.Errorf("path is not a directory: %s", path)
	}

	// Normalise the directory path
	dirPath := filepath.Clean(path)
	if dirPath == "." {
		dirPath = ""
	}

	// Collect direct children
	var entries []models.FileInfo
	for entryPath, entryInfo := range f.FileInfos {
		// Skip the directory itself
		if entryPath == path {
			continue
		}

		// Get the parent directory of this entry
		parent := filepath.Dir(entryPath)
		parent = filepath.Clean(parent)

		// Only include direct children
		if parent == dirPath {
			entries = append(entries, entryInfo)
		}
	}

	return entries, nil
}

// GetTempFiles returns all temp file paths (for testing cleanup verification)
func (f *MockFileSystem) GetTempFiles() []string {
	f.Mu.RLock()
	defer f.Mu.RUnlock()

	paths := make([]string, 0, len(f.TempFiles))
	for path := range f.TempFiles {
		paths = append(paths, path)
	}
	return paths
}

// MockCommandExecutor implements models.CommandExecutor for testing
type MockCommandExecutor struct {
	RunFunc   func(ctx context.Context, cmd []string) ([]byte, error)
	StartFunc func(ctx context.Context, command []string, opts models.ProcessOptions) (models.Process, io.Reader, io.Reader, error)
}

func (m *MockCommandExecutor) Run(ctx context.Context, cmd []string) ([]byte, error) {
	if m.RunFunc != nil {
		return m.RunFunc(ctx, cmd)
	}
	return nil, nil
}

func (m *MockCommandExecutor) Start(ctx context.Context, command []string, opts models.ProcessOptions) (models.Process, io.Reader, io.Reader, error) {
	if m.StartFunc != nil {
		return m.StartFunc(ctx, command, opts)
	}
	return nil, nil, nil, nil
}

// MockExitError simulates an exit error with a specific exit code
type MockExitError struct {
	Code int
}

func (e *MockExitError) Error() string {
	return fmt.Sprintf("exit status %d", e.Code)
}

func (e *MockExitError) ExitCode() int {
	return e.Code
}

// MockProcess implements models.Process for testing
type MockProcess struct {
	WaitFunc   func() error
	KillFunc   func() error
	SignalFunc func(sig os.Signal) error
}

func (p *MockProcess) Wait() error {
	if p.WaitFunc != nil {
		return p.WaitFunc()
	}
	return nil
}

func (p *MockProcess) Kill() error {
	if p.KillFunc != nil {
		return p.KillFunc()
	}
	return nil
}

func (p *MockProcess) Signal(sig os.Signal) error {
	if p.SignalFunc != nil {
		return p.SignalFunc(sig)
	}
	return nil
}
