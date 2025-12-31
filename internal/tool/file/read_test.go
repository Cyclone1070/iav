package file

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"testing"
	"time"

	"github.com/Cyclone1070/iav/internal/config"
	"github.com/Cyclone1070/iav/internal/tool/service/fs"
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

func (m *mockFileSystemForRead) ReadFileLines(path string, startLine, endLine int) (*fs.ReadFileLinesResult, error) {
	content, ok := m.files[path]
	if !ok {
		return nil, os.ErrNotExist
	}

	if m.config != nil && m.config.Tools.MaxFileSize > 0 && int64(len(content)) > m.config.Tools.MaxFileSize {
		return nil, fmt.Errorf("file too large: %d bytes exceeds limit %d", len(content), m.config.Tools.MaxFileSize)
	}

	// Binary detection mock: if it contains null byte
	for _, b := range content {
		if b == 0 {
			return nil, fmt.Errorf("binary file: %s", path)
		}
	}

	// Count lines using same logic as fs.go
	totalLines := 0
	if len(content) > 0 {
		for _, b := range content {
			if b == '\n' {
				totalLines++
			}
		}
		if content[len(content)-1] != '\n' {
			totalLines++
		}
	}

	if startLine <= 0 {
		startLine = 1
	}

	if startLine > totalLines {
		return &fs.ReadFileLinesResult{
			Content:    "",
			TotalLines: totalLines,
			StartLine:  startLine,
			EndLine:    0,
		}, nil
	}

	// Read content using same logic as fs.go (preserving newlines)
	actualEndLine := 0
	var buffer bytes.Buffer
	currentLine := 1

	reader := bufio.NewReader(bytes.NewReader(content))
	for {
		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return nil, err
		}

		if currentLine >= startLine {
			if endLine == 0 || currentLine <= endLine {
				buffer.WriteString(line)
				actualEndLine = currentLine
			}
		}

		if endLine > 0 && currentLine >= endLine {
			break
		}
		if err == io.EOF {
			break
		}
		currentLine++
	}

	return &fs.ReadFileLinesResult{
		Content:    buffer.String(),
		TotalLines: totalLines,
		StartLine:  startLine,
		EndLine:    actualEndLine,
	}, nil
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

func TestReadFile(t *testing.T) {
	workspaceRoot := "/workspace"
	maxFileSize := int64(1024 * 1024) // 1MB

	t.Run("full read caches checksum", func(t *testing.T) {
		cfg := config.DefaultConfig()
		cfg.Tools.MaxFileSize = maxFileSize
		fs := newMockFileSystemForRead(cfg)
		checksumManager := newMockChecksumManagerForRead()
		content := []byte("test content")
		fs.createFile("/workspace/test.txt", content)

		readTool := NewReadFileTool(fs, checksumManager, path.NewResolver(workspaceRoot))

		readReq := &ReadFileRequest{Path: "test.txt"}
		resp, err := readTool.Run(context.Background(), readReq)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expected := formatFileContent(string(content), 1)
		if resp.Content != expected {
			t.Errorf("expected content %q, got %q", expected, resp.Content)
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
		cfg := config.DefaultConfig()
		cfg.Tools.MaxFileSize = maxFileSize
		fs := newMockFileSystemForRead(cfg)
		checksumManager := newMockChecksumManagerForRead()
		content := []byte("line1\nline2\nline3\nline4")
		fs.createFile("/workspace/test.txt", content)

		readTool := NewReadFileTool(fs, checksumManager, path.NewResolver(workspaceRoot))
		startLine := 2

		readReq := &ReadFileRequest{Path: "test.txt", StartLine: startLine}
		resp, err := readTool.Run(context.Background(), readReq)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expected := formatFileContent("line2\nline3\nline4", startLine)
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
		cfg := config.DefaultConfig()
		cfg.Tools.MaxFileSize = maxFileSize
		fs := newMockFileSystemForRead(cfg)
		checksumManager := newMockChecksumManagerForRead()

		// Create file with null bytes (actual binary content)
		content := []byte{0x00, 0x01, 0x02, 't', 'e', 's', 't'}
		fs.createFile("/workspace/binary.bin", content)

		readTool := NewReadFileTool(fs, checksumManager, path.NewResolver(workspaceRoot))

		readReq := &ReadFileRequest{Path: "binary.bin"}
		_, err := readTool.Run(context.Background(), readReq)
		if err == nil {
			t.Errorf("expected error for binary file, got nil")
		}
	})

	t.Run("file size check still occurs via Stat", func(t *testing.T) {
		cfg := config.DefaultConfig()
		cfg.Tools.MaxFileSize = maxFileSize
		fs := newMockFileSystemForRead(cfg)
		checksumManager := newMockChecksumManagerForRead()
		// Create file larger than limit
		largeContent := make([]byte, maxFileSize+1)
		for i := range largeContent {
			largeContent[i] = ' '
		}
		fs.createFile("/workspace/large.txt", largeContent)

		readTool := NewReadFileTool(fs, checksumManager, path.NewResolver(workspaceRoot))

		readReq := &ReadFileRequest{Path: "large.txt"}
		_, err := readTool.Run(context.Background(), readReq)
		if err == nil {
			t.Fatalf("expected error for file exceeding MaxFileSize, got nil")
		}
	})

	t.Run("start line beyond EOF", func(t *testing.T) {
		cfg := config.DefaultConfig()
		fs := newMockFileSystemForRead(cfg)
		checksumManager := newMockChecksumManagerForRead()
		content := []byte("line1")
		fs.createFile("/workspace/test.txt", content)

		readTool := NewReadFileTool(fs, checksumManager, path.NewResolver(workspaceRoot))
		startLine := 100

		readReq := &ReadFileRequest{Path: "test.txt", StartLine: startLine}
		resp, err := readTool.Run(context.Background(), readReq)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected := formatFileContent("", startLine)
		if resp.Content != expected {
			t.Errorf("expected empty content with line number %d, got %q", startLine, resp.Content)
		}
	})

	t.Run("directory rejection", func(t *testing.T) {
		cfg := config.DefaultConfig()
		fs := newMockFileSystemForRead(cfg)
		checksumManager := newMockChecksumManagerForRead()
		fs.createDir("/workspace/subdir")

		readTool := NewReadFileTool(fs, checksumManager, path.NewResolver(workspaceRoot))

		readReq := &ReadFileRequest{Path: "subdir"}
		_, err := readTool.Run(context.Background(), readReq)
		if err == nil {
			t.Error("expected error when reading directory")
		}
	})

	t.Run("file not found", func(t *testing.T) {
		cfg := config.DefaultConfig()
		fs := newMockFileSystemForRead(cfg)
		checksumManager := newMockChecksumManagerForRead()

		readTool := NewReadFileTool(fs, checksumManager, path.NewResolver(workspaceRoot))

		readReq := &ReadFileRequest{Path: "nonexistent.txt"}
		_, err := readTool.Run(context.Background(), readReq)
		if err == nil {
			t.Errorf("expected error for nonexistent file, got nil")
		}
	})

	t.Run("end line truncation", func(t *testing.T) {
		cfg := config.DefaultConfig()
		fs := newMockFileSystemForRead(cfg)
		checksumManager := newMockChecksumManagerForRead()
		content := []byte("line1\nline2\nline3\nline4")
		fs.createFile("/workspace/test.txt", content)

		readTool := NewReadFileTool(fs, checksumManager, path.NewResolver(workspaceRoot))
		endLine := 2

		readReq := &ReadFileRequest{Path: "test.txt", EndLine: endLine}
		resp, err := readTool.Run(context.Background(), readReq)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expected := formatFileContent("line1\nline2", 1)
		if resp.Content != expected {
			t.Errorf("expected content %q, got %q", expected, resp.Content)
		}
	})

	t.Run("checksum accuracy on file without trailing newline", func(t *testing.T) {
		cfg := config.DefaultConfig()
		fs := newMockFileSystemForRead(cfg)
		checksumManager := newMockChecksumManagerForRead()
		content := []byte("no trailing newline")
		fs.createFile("/workspace/test.txt", content)

		readTool := NewReadFileTool(fs, checksumManager, path.NewResolver(workspaceRoot))

		readReq := &ReadFileRequest{Path: "test.txt", StartLine: 1, EndLine: 0} // Full read
		resp, err := readTool.Run(context.Background(), readReq)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify checksum in manager matches the one for our raw bytes
		cached, ok := checksumManager.Get(resp.AbsolutePath)
		if !ok {
			t.Fatal("expected checksum to be cached")
		}

		expectedChecksum := checksumManager.Compute(content)
		if cached != expectedChecksum {
			t.Errorf("checksum mismatch: expected %q, got %q", expectedChecksum, cached)
		}
	})
}
