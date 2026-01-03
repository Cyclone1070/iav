package file

import (
	"bytes"
	"context"
	"fmt"
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
func (m *mockFileInfoForRead) Mode() os.FileMode  { return 0o644 }
func (m *mockFileInfoForRead) ModTime() time.Time { return time.Time{} }
func (m *mockFileInfoForRead) IsDir() bool        { return m.isDir }
func (m *mockFileInfoForRead) Sys() any           { return nil }

type mockFileSystemForRead struct {
	files  map[string][]byte
	dirs   map[string]bool
	config *config.Config
}

func newMockFileSystemForRead(cfg *config.Config) *mockFileSystemForRead {
	return &mockFileSystemForRead{
		files:  make(map[string][]byte),
		dirs:   make(map[string]bool),
		config: cfg,
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

func (m *mockFileSystemForRead) ReadFile(path string) ([]byte, error) {
	// Check if it's a directory
	if m.dirs[path] {
		return nil, fmt.Errorf("read %s: is a directory", path)
	}

	content, ok := m.files[path]
	if !ok {
		return nil, os.ErrNotExist
	}

	if m.config != nil && m.config.Tools.MaxFileSize > 0 && int64(len(content)) > m.config.Tools.MaxFileSize {
		return nil, fmt.Errorf("file %s exceeds max size (%d bytes)", path, m.config.Tools.MaxFileSize)
	}

	// Binary detection mock: if it contains null byte
	for _, b := range content {
		if b == 0 {
			return nil, fmt.Errorf("binary file: %s", path)
		}
	}

	return content, nil
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
	return fmt.Sprintf("mock-checksum-%x", content)
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

// executeRead is a test helper that calls Execute and type-asserts the result.
func executeRead(t *testing.T, rtool *ReadFileTool, req *ReadFileRequest) *ReadFileResponse {
	t.Helper()
	result, err := rtool.Execute(context.Background(), req)
	if err != nil {
		t.Fatalf("Execute returned infra error: %v", err)
	}
	resp, ok := result.(*ReadFileResponse)
	if !ok {
		t.Fatalf("Execute returned wrong type: %T", result)
	}
	return resp
}

func TestReadFile(t *testing.T) {
	workspaceRoot := "/workspace"
	maxFileSize := int64(1024 * 1024) // 1MB

	t.Run("full read caches checksum", func(t *testing.T) {
		cfg := config.DefaultConfig()
		cfg.Tools.MaxFileSize = maxFileSize
		fs := newMockFileSystemForRead(cfg)
		checksumManager := newMockChecksumManagerForRead()
		contentStr := "test content"
		content := []byte(contentStr)
		fs.createFile("/workspace/test.txt", content)

		readTool := NewReadFileTool(fs, checksumManager, path.NewResolver(workspaceRoot), cfg)

		readReq := &ReadFileRequest{Path: "test.txt", Offset: 0, Limit: 100}
		resp := executeRead(t, readTool, readReq)

		if resp.Error != "" {
			t.Fatalf("Execute failed: %s", resp.Error)
		}

		if resp.Content != contentStr {
			t.Errorf("expected content %q, got %q", contentStr, resp.Content)
		}

		// Verify cache was updated
		checksum, ok := checksumManager.Get("/workspace/test.txt")
		if !ok {
			t.Error("expected cache to be updated after full read")
		}
		if checksum == "" {
			t.Error("expected non-empty checksum in cache")
		}
	})

	t.Run("partial read using offset and limit", func(t *testing.T) {
		cfg := config.DefaultConfig()
		cfg.Tools.MaxFileSize = maxFileSize
		fs := newMockFileSystemForRead(cfg)
		checksumManager := newMockChecksumManagerForRead()
		content := []byte("line1\nline2\nline3\nline4")
		fs.createFile("/workspace/test.txt", content)

		readTool := NewReadFileTool(fs, checksumManager, path.NewResolver(workspaceRoot), cfg)

		// Read lines 2 and 3 (Offset=1, Limit=2)
		readReq := &ReadFileRequest{Path: "test.txt", Offset: 1, Limit: 2}
		resp := executeRead(t, readTool, readReq)

		if resp.Error != "" {
			t.Fatalf("Execute failed: %s", resp.Error)
		}

		expected := "line2\nline3"
		if resp.Content != expected {
			t.Errorf("expected content %q, got %q", expected, resp.Content)
		}

		// Verify metadata
		if resp.StartLine != 2 {
			t.Errorf("expected StartLine 2, got %d", resp.StartLine)
		}
		if resp.EndLine != 3 {
			t.Errorf("expected EndLine 3, got %d", resp.EndLine)
		}
		if resp.TotalLines != 4 {
			t.Errorf("expected TotalLines 4, got %d", resp.TotalLines)
		}

		// Verify LLMContent formatting
		llm := resp.LLMContent()
		assertContains(t, llm, "00002| line2")
		assertContains(t, llm, "00003| line3")
		assertContains(t, llm, "(File has more lines. Use offset=3 to read more)")
	})

	t.Run("binary detection rejection", func(t *testing.T) {
		cfg := config.DefaultConfig()
		cfg.Tools.MaxFileSize = maxFileSize
		fs := newMockFileSystemForRead(cfg)
		checksumManager := newMockChecksumManagerForRead()

		// Create file with null bytes (actual binary content)
		content := []byte{0x00, 0x01, 0x02, 't', 'e', 's', 't'}
		fs.createFile("/workspace/binary.bin", content)

		readTool := NewReadFileTool(fs, checksumManager, path.NewResolver(workspaceRoot), cfg)

		readReq := &ReadFileRequest{Path: "binary.bin"}
		resp := executeRead(t, readTool, readReq)
		if resp.Error == "" {
			t.Errorf("expected error for binary file, got success")
		}
	})

	t.Run("file size check still occurs", func(t *testing.T) {
		cfg := config.DefaultConfig()
		cfg.Tools.MaxFileSize = 10 // small limit
		fs := newMockFileSystemForRead(cfg)
		checksumManager := newMockChecksumManagerForRead()

		largeContent := []byte("this is more than 10 bytes")
		fs.createFile("/workspace/large.txt", largeContent)

		readTool := NewReadFileTool(fs, checksumManager, path.NewResolver(workspaceRoot), cfg)

		readReq := &ReadFileRequest{Path: "large.txt"}
		resp := executeRead(t, readTool, readReq)
		if resp.Error == "" {
			t.Fatalf("expected error for file exceeding MaxFileSize, got success")
		}
	})

	t.Run("offset beyond EOF", func(t *testing.T) {
		cfg := config.DefaultConfig()
		fs := newMockFileSystemForRead(cfg)
		checksumManager := newMockChecksumManagerForRead()
		content := []byte("line1")
		fs.createFile("/workspace/test.txt", content)

		readTool := NewReadFileTool(fs, checksumManager, path.NewResolver(workspaceRoot), cfg)
		offset := 100

		readReq := &ReadFileRequest{Path: "test.txt", Offset: offset, Limit: 10}
		resp := executeRead(t, readTool, readReq)

		if resp.Error != "" {
			t.Fatalf("Execute failed: %s", resp.Error)
		}

		if resp.Content != "" {
			t.Errorf("expected empty content, got %q", resp.Content)
		}
		if resp.TotalLines != 1 {
			t.Errorf("expected TotalLines 1, got %d", resp.TotalLines)
		}

		// Verify LLMContent handles offset beyond EOF correctly
		llm := resp.LLMContent()
		assertContains(t, llm, "(End of file - total 1 lines)")
	})

	t.Run("directory rejection", func(t *testing.T) {
		cfg := config.DefaultConfig()
		fs := newMockFileSystemForRead(cfg)
		checksumManager := newMockChecksumManagerForRead()
		fs.createDir("/workspace/subdir")

		readTool := NewReadFileTool(fs, checksumManager, path.NewResolver(workspaceRoot), cfg)

		readReq := &ReadFileRequest{Path: "subdir"}
		resp := executeRead(t, readTool, readReq)
		if resp.Error == "" {
			t.Error("expected error when reading directory")
		}
	})

	t.Run("file not found", func(t *testing.T) {
		cfg := config.DefaultConfig()
		fs := newMockFileSystemForRead(cfg)
		checksumManager := newMockChecksumManagerForRead()

		readTool := NewReadFileTool(fs, checksumManager, path.NewResolver(workspaceRoot), cfg)

		readReq := &ReadFileRequest{Path: "nonexistent.txt"}
		resp := executeRead(t, readTool, readReq)
		if resp.Error == "" {
			t.Errorf("expected error for nonexistent file, got success")
		}
	})
}

func assertContains(t *testing.T, s, substr string) {
	t.Helper()
	if !bytes.Contains([]byte(s), []byte(substr)) {
		t.Errorf("expected %q to contain %q", s, substr)
	}
}
