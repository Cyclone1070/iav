package file

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/Cyclone1070/iav/internal/config"
)

// Local mocks for read tests

type mockFileInfoForRead struct {
	name  string
	size  int64
	isDir bool
}

func (m *mockFileInfoForRead) Name() string       { return m.name }
func (m *mockFileInfoForRead) Size() int64        { return m.size }
func (m *mockFileInfoForRead) Mode() os.FileMode  { return 0644 }
func (m *mockFileInfoForRead) ModTime() time.Time { return time.Time{} }
func (m *mockFileInfoForRead) IsDir() bool        { return m.isDir }
func (m *mockFileInfoForRead) Sys() interface{}   { return nil }

type mockFileSystemForRead struct {
	files map[string][]byte
	dirs  map[string]bool
}

func newMockFileSystemForRead() *mockFileSystemForRead {
	return &mockFileSystemForRead{
		files: make(map[string][]byte),
		dirs:  make(map[string]bool),
	}
}

func (m *mockFileSystemForRead) createFile(path string, content []byte) {
	m.files[path] = content
}

func (m *mockFileSystemForRead) createDir(path string) {
	m.dirs[path] = true
}

func (m *mockFileSystemForRead) Stat(path string) (os.FileInfo, error) {
	if m.dirs[path] {
		return &mockFileInfoForRead{name: path, isDir: true}, nil
	}
	if content, ok := m.files[path]; ok {
		return &mockFileInfoForRead{name: path, size: int64(len(content)), isDir: false}, nil
	}
	return nil, os.ErrNotExist
}

func (m *mockFileSystemForRead) Lstat(path string) (os.FileInfo, error) {
	return m.Stat(path)
}

func (m *mockFileSystemForRead) Readlink(path string) (string, error) {
	return "", os.ErrInvalid
}

func (m *mockFileSystemForRead) UserHomeDir() (string, error) {
	return "/home/user", nil
}

func (m *mockFileSystemForRead) ReadFileRange(path string, offset, limit int64) ([]byte, error) {
	content, ok := m.files[path]
	if !ok {
		return nil, os.ErrNotExist
	}

	if offset >= int64(len(content)) {
		return []byte{}, nil
	}

	end := int64(len(content))
	if limit > 0 && offset+limit < end {
		end = offset + limit
	}

	return content[offset:end], nil
}

func (m *mockFileSystemForRead) EnsureDirs(path string) error {
	return nil
}

func (m *mockFileSystemForRead) WriteFileAtomic(path string, content []byte, perm os.FileMode) error {
	// Not used in read tests, but required by interface
	return nil
}

func (m *mockFileSystemForRead) Remove(name string) error {
	return nil
}

func (m *mockFileSystemForRead) Rename(oldpath, newpath string) error {
	return nil
}

func (m *mockFileSystemForRead) Chmod(name string, mode os.FileMode) error {
	return nil
}

type mockBinaryDetectorForRead struct {
	isBinaryFunc func([]byte) bool
}

func newMockBinaryDetectorForRead() *mockBinaryDetectorForRead {
	return &mockBinaryDetectorForRead{
		isBinaryFunc: func([]byte) bool { return false },
	}
}

func (m *mockBinaryDetectorForRead) IsBinaryContent(content []byte) bool {
	if m.isBinaryFunc != nil {
		return m.isBinaryFunc(content)
	}
	return false
}

type mockChecksumManagerForRead struct {
	checksums map[string]string
}

func newMockChecksumManagerForRead() *mockChecksumManagerForRead {
	return &mockChecksumManagerForRead{
		checksums: make(map[string]string),
	}
}

func (m *mockChecksumManagerForRead) Compute(content []byte) string {
	return "mock-checksum"
}

func (m *mockChecksumManagerForRead) Get(path string) (string, bool) {
	checksum, ok := m.checksums[path]
	return checksum, ok
}

func (m *mockChecksumManagerForRead) Update(path, checksum string) {
	m.checksums[path] = checksum
}

func (m *mockChecksumManagerForRead) Clear() {
	m.checksums = make(map[string]string)
}

// Test functions

func TestReadFile(t *testing.T) {
	workspaceRoot := "/workspace"
	maxFileSize := int64(1024 * 1024) // 1MB

	t.Run("full read caches checksum", func(t *testing.T) {
		fs := newMockFileSystemForRead()
		checksumManager := newMockChecksumManagerForRead()
		content := []byte("test content")
		fs.createFile("/workspace/test.txt", content)

		cfg := config.DefaultConfig()
		cfg.Tools.MaxFileSize = maxFileSize

		tool := NewReadFileTool(fs, newMockBinaryDetectorForRead(), checksumManager, cfg, workspaceRoot)

		reqDTO := ReadFileDTO{Path: "test.txt"}
		req, err := NewReadFileRequest(reqDTO, cfg, workspaceRoot, fs)
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}

		resp, err := tool.Run(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if resp.Content != string(content) {
			t.Errorf("expected content %q, got %q", string(content), resp.Content)
		}

		// Verify cache was updated
		checksum, ok := checksumManager.Get(resp.AbsolutePath)
		if !ok {
			t.Error("expected cache to be updated after full read")
		}
		if checksum == "" {
			t.Error("expected non-empty checksum in cache")
		}
	})

	t.Run("partial read skips cache update", func(t *testing.T) {
		fs := newMockFileSystemForRead()
		checksumManager := newMockChecksumManagerForRead()
		content := []byte("test content")
		fs.createFile("/workspace/test.txt", content)

		cfg := config.DefaultConfig()
		cfg.Tools.MaxFileSize = maxFileSize

		tool := NewReadFileTool(fs, newMockBinaryDetectorForRead(), checksumManager, cfg, workspaceRoot)
		offset := int64(5)

		reqDTO := ReadFileDTO{Path: "test.txt", Offset: &offset}
		req, err := NewReadFileRequest(reqDTO, cfg, workspaceRoot, fs)
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}

		resp, err := tool.Run(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expected := string(content[5:])
		if resp.Content != expected {
			t.Errorf("expected content %q, got %q", expected, resp.Content)
		}

		// Verify cache was NOT updated
		_, ok := checksumManager.Get(resp.AbsolutePath)
		if ok {
			t.Error("expected cache to NOT be updated after partial read")
		}
	})

	t.Run("binary detection rejection", func(t *testing.T) {
		fs := newMockFileSystemForRead()
		checksumManager := newMockChecksumManagerForRead()
		detector := newMockBinaryDetectorForRead()
		detector.isBinaryFunc = func(content []byte) bool {
			return true
		}

		// Create file with null bytes (actual binary content)
		content := []byte{0x00, 0x01, 0x02, 't', 'e', 's', 't'}
		fs.createFile("/workspace/binary.bin", content)

		cfg := config.DefaultConfig()
		cfg.Tools.MaxFileSize = maxFileSize

		tool := NewReadFileTool(fs, detector, checksumManager, cfg, workspaceRoot)

		reqDTO := ReadFileDTO{Path: "binary.bin"}
		req, err := NewReadFileRequest(reqDTO, cfg, workspaceRoot, fs)
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}

		_, err = tool.Run(context.Background(), req)
		var binaryErr *BinaryFileError
		if err == nil || !errors.As(err, &binaryErr) {
			t.Errorf("expected BinaryFileError, got %v", err)
		}
	})

	t.Run("size limit enforcement", func(t *testing.T) {
		fs := newMockFileSystemForRead()
		checksumManager := newMockChecksumManagerForRead()
		// Create file larger than limit
		largeContent := make([]byte, maxFileSize+1)
		fs.createFile("/workspace/large.txt", largeContent)

		cfg := config.DefaultConfig()
		cfg.Tools.MaxFileSize = maxFileSize

		tool := NewReadFileTool(fs, newMockBinaryDetectorForRead(), checksumManager, cfg, workspaceRoot)

		reqDTO := ReadFileDTO{Path: "large.txt"}
		req, err := NewReadFileRequest(reqDTO, cfg, workspaceRoot, fs)
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}

		_, err = tool.Run(context.Background(), req)
		var tooLargeErr *TooLargeError
		if err == nil || !errors.As(err, &tooLargeErr) {
			t.Errorf("expected TooLargeError, got %v", err)
		}
	})

	t.Run("offset beyond EOF", func(t *testing.T) {
		fs := newMockFileSystemForRead()
		checksumManager := newMockChecksumManagerForRead()
		content := []byte("test")
		fs.createFile("/workspace/test.txt", content)

		cfg := config.DefaultConfig()

		tool := NewReadFileTool(fs, newMockBinaryDetectorForRead(), checksumManager, cfg, workspaceRoot)
		offset := int64(10000)

		reqDTO := ReadFileDTO{Path: "test.txt", Offset: &offset}
		req, err := NewReadFileRequest(reqDTO, cfg, workspaceRoot, fs)
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}

		resp, err := tool.Run(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Content != "" {
			t.Errorf("expected empty content for offset beyond EOF, got %q", resp.Content)
		}
	})

	t.Run("directory rejection", func(t *testing.T) {
		fs := newMockFileSystemForRead()
		checksumManager := newMockChecksumManagerForRead()
		fs.createDir("/workspace/subdir")

		cfg := config.DefaultConfig()

		tool := NewReadFileTool(fs, newMockBinaryDetectorForRead(), checksumManager, cfg, workspaceRoot)

		reqDTO := ReadFileDTO{Path: "subdir"}
		req, err := NewReadFileRequest(reqDTO, cfg, workspaceRoot, fs)
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}

		_, err = tool.Run(context.Background(), req)
		var isDirErr *IsDirectoryError
		if err == nil || !errors.As(err, &isDirErr) {
			t.Error("expected IsDirectoryError when reading directory")
		}
	})

	t.Run("file not found", func(t *testing.T) {
		fs := newMockFileSystemForRead()
		checksumManager := newMockChecksumManagerForRead()

		cfg := config.DefaultConfig()

		tool := NewReadFileTool(fs, newMockBinaryDetectorForRead(), checksumManager, cfg, workspaceRoot)

		// NewReadFileRequest does Path resolution using FS, so it might fail there if Lstat fails?
		// resolvePathWithFS calls pathutil.Resolve which calls Lstat.
		// If file doesn't exist, Lstat returns IsNotExist.
		// pathutil.CanonicaliseRoot checks root existence.
		// pathutil.Resolve checks symlinks.
		// If file passed to Resolve doesn't exist, Resolve returns abs path and NO error (unless component is not a dir).
		// So validation passes. Tool.Run calling Stat will fail.

		reqDTO := ReadFileDTO{Path: "nonexistent.txt"}
		req, err := NewReadFileRequest(reqDTO, cfg, workspaceRoot, fs)
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}

		_, err = tool.Run(context.Background(), req)
		// Run calls fs.Stat, which on mock returns os.ErrNotExist.
		// Run wraps it in &StatError{Cause: err}.
		// Wait, StatError wrapped cause is os.IsNotExist.
		// BUT wait, I replaced the behavior:
		/*
			60: 	info, err := t.fileOps.Stat(abs)
			61: 	if err != nil {
			62: 		return nil, &StatError{Path: abs, Cause: err}
			63: 	}
		*/
		// So it returns &StatError.
		// Does StatError implement specific behavior? `IOError`.
		// If I want to check for "not found", I should probably check if cause is NotExist?
		// Or should I have returned `FileMissingError` in Run?
		// Run:
		// if err != nil { return nil, &StatError... }
		// NOT checking IsNotExist.
		// In `edit.go` I did check IsNotExist and returned `FileMissingError`.
		// In `read.go` I did NOT.
		// Using strict interpretation: "failed to stat file: %w".
		// If I want "not found" behavior, `read.go` logic should have been updated to check NotExist.
		// BUT the original code was: `return nil, fmt.Errorf("failed to stat file: %w", err)`.
		// It did NOT explicitly return ErrFileMissing.
		// So expecting `StatError` is correct for now.

		var statErr *StatError
		if err == nil || !errors.As(err, &statErr) {
			t.Errorf("expected StatError for nonexistent file, got %v", err)
		}
	})

	t.Run("limit truncation", func(t *testing.T) {
		fs := newMockFileSystemForRead()
		checksumManager := newMockChecksumManagerForRead()
		content := []byte("test content")
		fs.createFile("/workspace/test.txt", content)

		cfg := config.DefaultConfig()

		tool := NewReadFileTool(fs, newMockBinaryDetectorForRead(), checksumManager, cfg, workspaceRoot)
		limit := int64(4)

		reqDTO := ReadFileDTO{Path: "test.txt", Limit: &limit}
		req, err := NewReadFileRequest(reqDTO, cfg, workspaceRoot, fs)
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}

		resp, err := tool.Run(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expected := string(content[:4])
		if resp.Content != expected {
			t.Errorf("expected content %q, got %q", expected, resp.Content)
		}
	})
}
