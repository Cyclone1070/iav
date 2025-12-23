package file

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Cyclone1070/iav/internal/config"
	"github.com/Cyclone1070/iav/internal/tool/pathutil"
)

// Local mocks for write tests

// mockFileInfoForWrite implements os.FileInfo for testing
type mockFileInfoForWrite struct {
	name  string
	size  int64
	mode  os.FileMode
	isDir bool
}

func (m *mockFileInfoForWrite) Name() string       { return m.name }
func (m *mockFileInfoForWrite) Size() int64        { return m.size }
func (m *mockFileInfoForWrite) Mode() os.FileMode  { return m.mode }
func (m *mockFileInfoForWrite) ModTime() time.Time { return time.Time{} }
func (m *mockFileInfoForWrite) IsDir() bool        { return m.isDir }
func (m *mockFileInfoForWrite) Sys() any           { return nil }

type fileEntry struct {
	content []byte
	mode    os.FileMode
}

type symlinkEntry struct {
	target string
}

// mockFileSystemForWrite provides comprehensive filesystem mocking for write tests
type mockFileSystemForWrite struct {
	files           map[string]fileEntry
	dirs            map[string]bool
	symlinks        map[string]symlinkEntry
	tempFiles       []string
	operationErrors map[string]error
}

func newMockFileSystemForWrite() *mockFileSystemForWrite {
	return &mockFileSystemForWrite{
		files:           make(map[string]fileEntry),
		dirs:            make(map[string]bool),
		symlinks:        make(map[string]symlinkEntry),
		tempFiles:       []string{},
		operationErrors: make(map[string]error),
	}
}

func (m *mockFileSystemForWrite) createFile(path string, content []byte, mode os.FileMode) {
	m.files[path] = fileEntry{content: content, mode: mode}
}

func (m *mockFileSystemForWrite) createDir(path string) {
	m.dirs[path] = true
}

func (m *mockFileSystemForWrite) createSymlink(path, target string) {
	m.symlinks[path] = symlinkEntry{target: target}
}

func (m *mockFileSystemForWrite) setOperationError(operation string, err error) {
	m.operationErrors[operation] = err
}

func (m *mockFileSystemForWrite) getTempFiles() []string {
	return m.tempFiles
}

func (m *mockFileSystemForWrite) Stat(path string) (os.FileInfo, error) {
	// Check symlinks first
	if link, ok := m.symlinks[path]; ok {
		// Follow symlink
		return m.Stat(link.target)
	}

	if m.dirs[path] {
		return &mockFileInfoForWrite{name: filepath.Base(path), isDir: true, mode: 0755}, nil
	}
	if entry, ok := m.files[path]; ok {
		return &mockFileInfoForWrite{name: filepath.Base(path), size: int64(len(entry.content)), mode: entry.mode}, nil
	}
	return nil, os.ErrNotExist
}

func (m *mockFileSystemForWrite) Lstat(path string) (os.FileInfo, error) {
	// Lstat doesn't follow symlinks
	if _, ok := m.symlinks[path]; ok {
		return &mockFileInfoForWrite{name: filepath.Base(path), mode: os.ModeSymlink | 0777}, nil
	}

	if m.dirs[path] {
		return &mockFileInfoForWrite{name: filepath.Base(path), isDir: true, mode: 0755}, nil
	}
	if entry, ok := m.files[path]; ok {
		return &mockFileInfoForWrite{name: filepath.Base(path), size: int64(len(entry.content)), mode: entry.mode}, nil
	}
	return nil, os.ErrNotExist
}

func (m *mockFileSystemForWrite) Readlink(path string) (string, error) {
	if link, ok := m.symlinks[path]; ok {
		return link.target, nil
	}
	return "", os.ErrInvalid
}

func (m *mockFileSystemForWrite) UserHomeDir() (string, error) {
	return "/home/user", nil
}

func (m *mockFileSystemForWrite) ReadFileRange(path string, offset, limit int64) ([]byte, error) {
	// Fall back to mock data
	entry, ok := m.files[path]
	if !ok {
		return nil, os.ErrNotExist
	}

	content := entry.content
	if offset >= int64(len(content)) {
		return []byte{}, nil
	}

	end := int64(len(content))
	if limit > 0 && offset+limit < end {
		end = offset + limit
	}

	return content[offset:end], nil
}

func (m *mockFileSystemForWrite) EnsureDirs(path string) error {
	// Create all parent directories
	dir := filepath.Dir(path)
	parts := strings.Split(dir, "/")
	current := ""
	for _, part := range parts {
		if part == "" {
			continue
		}
		current = current + "/" + part
		m.dirs[current] = true
	}
	return nil
}

func (m *mockFileSystemForWrite) WriteFileAtomic(path string, content []byte, perm os.FileMode) error {
	// Check for injected error at the WriteFileAtomic level
	if m.operationErrors["WriteFileAtomic"] != nil {
		return m.operationErrors["WriteFileAtomic"]
	}

	// Simulate atomic write
	m.files[path] = fileEntry{
		content: content,
		mode:    perm,
	}
	return nil
}

func (m *mockFileSystemForWrite) Remove(name string) error {
	// Remove from temp files list
	for i, tf := range m.tempFiles {
		if tf == name {
			m.tempFiles = append(m.tempFiles[:i], m.tempFiles[i+1:]...)
			break
		}
	}

	delete(m.files, name)
	delete(m.dirs, name)
	delete(m.symlinks, name)
	return nil
}

func (m *mockFileSystemForWrite) Rename(oldpath, newpath string) error {
	if m.operationErrors["Rename"] != nil {
		return m.operationErrors["Rename"]
	}

	// Store the content in our mock filesystem at the new path
	if entry, ok := m.files[oldpath]; ok {
		m.files[newpath] = entry
		delete(m.files, oldpath)
	}

	// Remove from temp files
	for i, tf := range m.tempFiles {
		if tf == oldpath {
			m.tempFiles = append(m.tempFiles[:i], m.tempFiles[i+1:]...)
			break
		}
	}

	return nil
}

func (m *mockFileSystemForWrite) Chmod(name string, mode os.FileMode) error {
	if m.operationErrors["Chmod"] != nil {
		return m.operationErrors["Chmod"]
	}

	if entry, ok := m.files[name]; ok {
		entry.mode = mode
		m.files[name] = entry
	}

	return nil
}

type mockBinaryDetectorForWrite struct {
	isBinaryFunc func([]byte) bool
}

func newMockBinaryDetectorForWrite() *mockBinaryDetectorForWrite {
	return &mockBinaryDetectorForWrite{
		isBinaryFunc: func([]byte) bool { return false },
	}
}

func (m *mockBinaryDetectorForWrite) IsBinaryContent(content []byte) bool {
	if m.isBinaryFunc != nil {
		return m.isBinaryFunc(content)
	}
	return false
}

type mockChecksumManagerForWrite struct {
	checksums map[string]string
}

func newMockChecksumManagerForWrite() *mockChecksumManagerForWrite {
	return &mockChecksumManagerForWrite{
		checksums: make(map[string]string),
	}
}

func (m *mockChecksumManagerForWrite) Compute(content []byte) string {
	return fmt.Sprintf("checksum-%d", len(content))
}

func (m *mockChecksumManagerForWrite) Get(path string) (string, bool) {
	checksum, ok := m.checksums[path]
	return checksum, ok
}

func (m *mockChecksumManagerForWrite) Update(path, checksum string) {
	m.checksums[path] = checksum
}

func (m *mockChecksumManagerForWrite) Clear() {
	m.checksums = make(map[string]string)
}

// Test functions

func TestWriteFile(t *testing.T) {
	workspaceRoot := "/workspace"
	maxFileSize := int64(1024 * 1024) // 1MB

	t.Run("create new file succeeds and updates cache", func(t *testing.T) {
		fs := newMockFileSystemForWrite()
		checksumManager := newMockChecksumManagerForWrite()
		cfg := config.DefaultConfig()
		cfg.Tools.MaxFileSize = maxFileSize

		writeTool := NewWriteFileTool(fs, newMockBinaryDetectorForWrite(), checksumManager, cfg, pathutil.NewResolver(workspaceRoot, fs))
		content := "test content"

		req := &WriteFileRequest{Path: "new.txt", Content: content}
		resp, err := writeTool.Run(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if resp.BytesWritten != len(content) {
			t.Errorf("expected %d bytes written, got %d", len(content), resp.BytesWritten)
		}

		// Verify file was created
		fileContent, err := fs.ReadFileRange("/workspace/new.txt", 0, 0)
		if err != nil {
			t.Fatalf("failed to read created file: %v", err)
		}
		if string(fileContent) != content {
			t.Errorf("expected content %q, got %q", content, string(fileContent))
		}

		// Verify cache was updated
		checksum, ok := checksumManager.Get(resp.AbsolutePath)
		if !ok {
			t.Error("expected cache to be updated after write")
		}
		if checksum == "" {
			t.Error("expected non-empty checksum in cache")
		}
	})

	t.Run("existing file rejection", func(t *testing.T) {
		fs := newMockFileSystemForWrite()
		checksumManager := newMockChecksumManagerForWrite()
		fs.createFile("/workspace/existing.txt", []byte("existing"), 0644)

		writeTool := NewWriteFileTool(fs, newMockBinaryDetectorForWrite(), checksumManager, config.DefaultConfig(), pathutil.NewResolver(workspaceRoot, fs))

		req := &WriteFileRequest{Path: "existing.txt", Content: "new content"}
		_, err := writeTool.Run(context.Background(), req)
		if err == nil || !errors.Is(err, ErrFileExists) {
			t.Errorf("expected ErrFileExists, got %v", err)
		}
	})

	t.Run("symlink escape prevention", func(t *testing.T) {
		fs := newMockFileSystemForWrite()
		// Create symlink pointing outside workspace
		fs.createSymlink("/workspace/escape", "/outside/target.txt")

		writeTool := NewWriteFileTool(fs, newMockBinaryDetectorForWrite(), newMockChecksumManagerForWrite(), config.DefaultConfig(), pathutil.NewResolver(workspaceRoot, fs))
		req := &WriteFileRequest{Path: "escape", Content: "content"}
		_, err := writeTool.Run(context.Background(), req)

		if err == nil {
			t.Error("expected error for symlink escape, got nil")
		}
		if !errors.Is(err, pathutil.ErrOutsideWorkspace) {
			t.Errorf("expected ErrOutsideWorkspace, got %v", err)
		}
	})

	t.Run("large content rejection", func(t *testing.T) {
		fs := newMockFileSystemForWrite()
		checksumManager := newMockChecksumManagerForWrite()
		cfg := config.DefaultConfig()
		cfg.Tools.MaxFileSize = maxFileSize

		// Create content larger than limit
		largeContent := make([]byte, maxFileSize+1)
		for i := range largeContent {
			largeContent[i] = 'A'
		}

		writeTool := NewWriteFileTool(fs, newMockBinaryDetectorForWrite(), checksumManager, cfg, pathutil.NewResolver(workspaceRoot, fs))

		req := &WriteFileRequest{Path: "large.txt", Content: string(largeContent)}
		_, err := writeTool.Run(context.Background(), req)
		if err == nil || !errors.Is(err, ErrFileTooLarge) {
			t.Errorf("expected ErrFileTooLarge, got %v", err)
		}
	})

	t.Run("binary content rejection", func(t *testing.T) {
		fs := newMockFileSystemForWrite()
		checksumManager := newMockChecksumManagerForWrite()
		detector := newMockBinaryDetectorForWrite()
		detector.isBinaryFunc = func(content []byte) bool {
			return true
		}

		writeTool := NewWriteFileTool(fs, detector, checksumManager, config.DefaultConfig(), pathutil.NewResolver(workspaceRoot, fs))
		// Content with NUL byte
		binaryContent := []byte{0x48, 0x65, 0x6C, 0x00, 0x6C, 0x6F}

		req := &WriteFileRequest{Path: "binary.bin", Content: string(binaryContent)}
		_, err := writeTool.Run(context.Background(), req)
		if err == nil || !errors.Is(err, ErrBinaryFile) {
			t.Errorf("expected ErrBinaryFile, got %v", err)
		}
	})

	t.Run("custom permissions", func(t *testing.T) {
		fs := newMockFileSystemForWrite()
		checksumManager := newMockChecksumManagerForWrite()

		cfg := config.DefaultConfig()
		writeTool := NewWriteFileTool(fs, newMockBinaryDetectorForWrite(), checksumManager, cfg, pathutil.NewResolver(workspaceRoot, fs))

		perm := os.FileMode(0755)

		req := &WriteFileRequest{Path: "executable.txt", Content: "content", Perm: &perm}
		resp, err := writeTool.Run(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		info, err := fs.Stat("/workspace/executable.txt")
		if err != nil {
			t.Fatalf("failed to stat file: %v", err)
		}

		if info.Mode().Perm() != os.FileMode(perm) {
			t.Errorf("expected permissions %o, got %o", perm, info.Mode().Perm())
		}

		if resp.FileMode != uint32(perm) {
			t.Errorf("expected FileMode %o, got %o", perm, resp.FileMode)
		}
	})

	t.Run("nested directory creation", func(t *testing.T) {
		fs := newMockFileSystemForWrite()
		checksumManager := newMockChecksumManagerForWrite()

		cfg := config.DefaultConfig()
		writeTool := NewWriteFileTool(fs, newMockBinaryDetectorForWrite(), checksumManager, cfg, pathutil.NewResolver(workspaceRoot, fs))

		req := &WriteFileRequest{Path: "nested/deep/file.txt", Content: "content"}
		_, err := writeTool.Run(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify file was created
		fileContent, err := fs.ReadFileRange("/workspace/nested/deep/file.txt", 0, 0)
		if err != nil {
			t.Fatalf("failed to read created file: %v", err)
		}
		if string(fileContent) != "content" {
			t.Errorf("expected content %q, got %q", "content", string(fileContent))
		}
	})

	t.Run("symlink directory escape prevention", func(t *testing.T) {
		fs := newMockFileSystemForWrite()
		// Create symlink directory pointing outside workspace
		fs.createSymlink("/workspace/link", "/outside")

		cfg := config.DefaultConfig()
		writeTool := NewWriteFileTool(fs, newMockBinaryDetectorForWrite(), newMockChecksumManagerForWrite(), cfg, pathutil.NewResolver(workspaceRoot, fs))
		req := &WriteFileRequest{Path: "link/escape.txt", Content: "content"}
		_, err := writeTool.Run(context.Background(), req)

		if err == nil {
			t.Error("expected error for symlink directory escape, got nil")
		}
		if !errors.Is(err, pathutil.ErrOutsideWorkspace) {
			t.Errorf("expected ErrOutsideWorkspace, got %v", err)
		}
	})
}
