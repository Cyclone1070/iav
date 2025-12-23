package file

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/Cyclone1070/iav/internal/config"
	"github.com/Cyclone1070/iav/internal/tool/service/path"
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
func (m *mockFileInfoForRead) Sys() any           { return nil }

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

		readTool := NewReadFileTool(fs, checksumManager, cfg, path.NewResolver(workspaceRoot))

		readReq := &ReadFileRequest{Path: "test.txt"}
		resp, err := readTool.Run(context.Background(), readReq)
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

		readTool := NewReadFileTool(fs, checksumManager, cfg, path.NewResolver(workspaceRoot))
		offset := int64(5)

		readReq := &ReadFileRequest{Path: "test.txt", Offset: &offset}
		resp, err := readTool.Run(context.Background(), readReq)
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

		// Create file with null bytes (actual binary content)
		content := []byte{0x00, 0x01, 0x02, 't', 'e', 's', 't'}
		fs.createFile("/workspace/binary.bin", content)

		cfg := config.DefaultConfig()
		cfg.Tools.MaxFileSize = maxFileSize

		readTool := NewReadFileTool(fs, checksumManager, cfg, path.NewResolver(workspaceRoot))

		readReq := &ReadFileRequest{Path: "binary.bin"}
		_, err := readTool.Run(context.Background(), readReq)
		if err == nil || !errors.Is(err, ErrBinaryFile) {
			t.Errorf("expected ErrBinaryFile, got %v", err)
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

		readTool := NewReadFileTool(fs, checksumManager, cfg, path.NewResolver(workspaceRoot))

		readReq := &ReadFileRequest{Path: "large.txt"}
		_, err := readTool.Run(context.Background(), readReq)
		if err == nil || !errors.Is(err, ErrFileTooLarge) {
			t.Errorf("expected ErrFileTooLarge, got %v", err)
		}
	})

	t.Run("offset beyond EOF", func(t *testing.T) {
		fs := newMockFileSystemForRead()
		checksumManager := newMockChecksumManagerForRead()
		content := []byte("test")
		fs.createFile("/workspace/test.txt", content)

		cfg := config.DefaultConfig()

		readTool := NewReadFileTool(fs, checksumManager, cfg, path.NewResolver(workspaceRoot))
		offset := int64(10000)

		readReq := &ReadFileRequest{Path: "test.txt", Offset: &offset}
		resp, err := readTool.Run(context.Background(), readReq)
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

		readTool := NewReadFileTool(fs, checksumManager, cfg, path.NewResolver(workspaceRoot))

		readReq := &ReadFileRequest{Path: "subdir"}
		_, err := readTool.Run(context.Background(), readReq)
		if err == nil || !errors.Is(err, ErrIsDirectory) {
			t.Error("expected ErrIsDirectory when reading directory")
		}
	})

	t.Run("file not found", func(t *testing.T) {
		fs := newMockFileSystemForRead()
		checksumManager := newMockChecksumManagerForRead()

		cfg := config.DefaultConfig()

		readTool := NewReadFileTool(fs, checksumManager, cfg, path.NewResolver(workspaceRoot))

		readReq := &ReadFileRequest{Path: "nonexistent.txt"}
		_, err := readTool.Run(context.Background(), readReq)
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

		readTool := NewReadFileTool(fs, checksumManager, cfg, path.NewResolver(workspaceRoot))
		limit := int64(4)

		readReq := &ReadFileRequest{Path: "test.txt", Limit: &limit}
		resp, err := readTool.Run(context.Background(), readReq)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expected := string(content[:4])
		if resp.Content != expected {
			t.Errorf("expected content %q, got %q", expected, resp.Content)
		}
	})
}
