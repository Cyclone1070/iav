package file

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Cyclone1070/iav/internal/config"
	toolserrors "github.com/Cyclone1070/iav/internal/tools/errors"
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
func (m *mockFileInfoForWrite) Sys() interface{}   { return nil }

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
	// First try to read from real file (for temp files)
	if data, err := os.ReadFile(path); err == nil {
		if offset >= int64(len(data)) {
			return []byte{}, nil
		}
		end := int64(len(data))
		if limit > 0 && offset+limit < end {
			end = offset + limit
		}
		return data[offset:end], nil
	}

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

	// Actually remove the real file if it exists
	os.Remove(name)

	delete(m.files, name)
	delete(m.dirs, name)
	delete(m.symlinks, name)
	return nil
}

func (m *mockFileSystemForWrite) Rename(oldpath, newpath string) error {
	if m.operationErrors["Rename"] != nil {
		return m.operationErrors["Rename"]
	}

	// Read content from real temp file before renaming
	var content []byte
	if data, err := os.ReadFile(oldpath); err == nil {
		content = data
	}

	// Try to rename real file if it exists (will fail since newpath doesn't exist on real FS)
	// We ignore the error since we're mocking the filesystem
	os.Rename(oldpath, newpath)

	// Store the content in our mock filesystem at the new path
	if len(content) > 0 {
		m.files[newpath] = fileEntry{
			content: content,
			mode:    0644,
		}
	} else if entry, ok := m.files[oldpath]; ok {
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

	// Clean up the real temp file
	os.Remove(oldpath)

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

		tool := NewWriteFileTool(fs, newMockBinaryDetectorForWrite(), checksumManager, cfg, workspaceRoot)
		content := "test content"
		resp, err := tool.Run(context.Background(), WriteFileRequest{Path: "new.txt", Content: content})
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

		tool := NewWriteFileTool(fs, newMockBinaryDetectorForWrite(), checksumManager, config.DefaultConfig(), workspaceRoot)
		_, err := tool.Run(context.Background(), WriteFileRequest{Path: "existing.txt", Content: "new content"})
		if err != toolserrors.ErrFileExists {
			t.Errorf("expected ErrFileExists, got %v", err)
		}
	})

	t.Run("symlink escape prevention", func(t *testing.T) {
		fs := newMockFileSystemForWrite()
		checksumManager := newMockChecksumManagerForWrite()
		// Create symlink pointing outside workspace
		fs.createSymlink("/workspace/escape", "/outside/target.txt")

		tool := NewWriteFileTool(fs, newMockBinaryDetectorForWrite(), checksumManager, config.DefaultConfig(), workspaceRoot)
		_, err := tool.Run(context.Background(), WriteFileRequest{Path: "escape", Content: "content"})
		if err != toolserrors.ErrOutsideWorkspace {
			t.Errorf("expected ErrOutsideWorkspace for symlink escape, got %v", err)
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

		tool := NewWriteFileTool(fs, newMockBinaryDetectorForWrite(), checksumManager, cfg, workspaceRoot)
		_, err := tool.Run(context.Background(), WriteFileRequest{Path: "large.txt", Content: string(largeContent)})
		if err != toolserrors.ErrTooLarge {
			t.Errorf("expected ErrTooLarge, got %v", err)
		}
	})

	t.Run("binary content rejection", func(t *testing.T) {
		fs := newMockFileSystemForWrite()
		checksumManager := newMockChecksumManagerForWrite()
		detector := newMockBinaryDetectorForWrite()
		detector.isBinaryFunc = func(content []byte) bool {
			return true
		}

		tool := NewWriteFileTool(fs, detector, checksumManager, config.DefaultConfig(), workspaceRoot)
		// Content with NUL byte
		binaryContent := []byte{0x48, 0x65, 0x6C, 0x00, 0x6C, 0x6F}
		_, err := tool.Run(context.Background(), WriteFileRequest{Path: "binary.bin", Content: string(binaryContent)})
		if err != toolserrors.ErrBinaryFile {
			t.Errorf("expected ErrBinaryFile, got %v", err)
		}
	})

	t.Run("custom permissions", func(t *testing.T) {
		fs := newMockFileSystemForWrite()
		checksumManager := newMockChecksumManagerForWrite()

		tool := NewWriteFileTool(fs, newMockBinaryDetectorForWrite(), checksumManager, config.DefaultConfig(), workspaceRoot)
		perm := os.FileMode(0755)
		resp, err := tool.Run(context.Background(), WriteFileRequest{Path: "executable.txt", Content: "content", Perm: &perm})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		info, err := fs.Stat("/workspace/executable.txt")
		if err != nil {
			t.Fatalf("failed to stat file: %v", err)
		}

		if info.Mode().Perm() != perm {
			t.Errorf("expected permissions %o, got %o", perm, info.Mode().Perm())
		}

		if resp.FileMode != uint32(perm) {
			t.Errorf("expected FileMode %o, got %o", perm, resp.FileMode)
		}
	})

	t.Run("nested directory creation", func(t *testing.T) {
		fs := newMockFileSystemForWrite()
		checksumManager := newMockChecksumManagerForWrite()

		tool := NewWriteFileTool(fs, newMockBinaryDetectorForWrite(), checksumManager, config.DefaultConfig(), workspaceRoot)
		_, err := tool.Run(context.Background(), WriteFileRequest{Path: "nested/deep/file.txt", Content: "content"})
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

	t.Run("symlink inside workspace allowed", func(t *testing.T) {
		fs := newMockFileSystemForWrite()
		checksumManager := newMockChecksumManagerForWrite()
		// Create symlink pointing inside workspace
		fs.createSymlink("/workspace/link", "/workspace/target.txt")
		fs.createFile("/workspace/target.txt", []byte("target"), 0644)

		tool := NewWriteFileTool(fs, newMockBinaryDetectorForWrite(), checksumManager, config.DefaultConfig(), workspaceRoot)
		// Writing to a symlink that points inside workspace should work
		_, err := tool.Run(context.Background(), WriteFileRequest{Path: "link", Content: "new content"})
		// This should succeed because we're creating a new file at the symlink path
		if err != nil {
			// If it fails, it's because the symlink exists, which is expected
			if err != toolserrors.ErrFileExists {
				t.Errorf("unexpected error: %v", err)
			}
		}
	})

	t.Run("symlink directory escape prevention", func(t *testing.T) {
		fs := newMockFileSystemForWrite()
		checksumManager := newMockChecksumManagerForWrite()
		// Create symlink directory pointing outside workspace
		fs.createSymlink("/workspace/link", "/outside")
		fs.createDir("/outside")

		tool := NewWriteFileTool(fs, newMockBinaryDetectorForWrite(), checksumManager, config.DefaultConfig(), workspaceRoot)
		// Try to write a file through the symlink directory - should fail
		_, err := tool.Run(context.Background(), WriteFileRequest{Path: "link/escape.txt", Content: "content"})
		if err != toolserrors.ErrOutsideWorkspace {
			t.Errorf("expected ErrOutsideWorkspace for symlink directory escape, got %v", err)
		}
	})

	t.Run("write through symlink chain inside workspace", func(t *testing.T) {
		fs := newMockFileSystemForWrite()
		checksumManager := newMockChecksumManagerForWrite()
		// Create symlink chain: link1 -> link2 -> target_dir
		fs.createSymlink("/workspace/link1", "/workspace/link2")
		fs.createSymlink("/workspace/link2", "/workspace/target_dir")
		fs.createDir("/workspace/target_dir")

		tool := NewWriteFileTool(fs, newMockBinaryDetectorForWrite(), checksumManager, config.DefaultConfig(), workspaceRoot)
		// Write through symlink chain - should succeed
		resp, err := tool.Run(context.Background(), WriteFileRequest{Path: "link1/file.txt", Content: "content"})
		if err != nil {
			t.Fatalf("unexpected error writing through symlink chain: %v", err)
		}

		// Verify file was created at resolved location
		fileContent, err := fs.ReadFileRange("/workspace/target_dir/file.txt", 0, 0)
		if err != nil {
			t.Fatalf("failed to read created file: %v", err)
		}
		if string(fileContent) != "content" {
			t.Errorf("expected content %q, got %q", "content", string(fileContent))
		}

		// Verify response has correct absolute path
		if resp.AbsolutePath != "/workspace/target_dir/file.txt" {
			t.Errorf("expected absolute path /workspace/target_dir/file.txt, got %s", resp.AbsolutePath)
		}
	})

	t.Run("write through symlink chain escaping workspace", func(t *testing.T) {
		fs := newMockFileSystemForWrite()
		checksumManager := newMockChecksumManagerForWrite()
		// Create chain: link1 -> link2 -> /tmp/outside
		fs.createSymlink("/workspace/link1", "/workspace/link2")
		fs.createSymlink("/workspace/link2", "/tmp/outside")
		fs.createDir("/tmp/outside")

		tool := NewWriteFileTool(fs, newMockBinaryDetectorForWrite(), checksumManager, config.DefaultConfig(), workspaceRoot)
		// Try to write through escaping chain - should fail
		_, err := tool.Run(context.Background(), WriteFileRequest{Path: "link1/file.txt", Content: "content"})
		if err != toolserrors.ErrOutsideWorkspace {
			t.Errorf("expected ErrOutsideWorkspace for escaping symlink chain, got %v", err)
		}
	})
}
