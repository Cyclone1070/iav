package directory

// List directory tests - mocks shared from file package write_test.go pattern

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

// Local mocks for directory listing tests

type mockFileInfoForList struct {
	name  string
	size  int64
	mode  os.FileMode
	isDir bool
}

func (m *mockFileInfoForList) Name() string       { return m.name }
func (m *mockFileInfoForList) Size() int64        { return m.size }
func (m *mockFileInfoForList) Mode() os.FileMode  { return m.mode }
func (m *mockFileInfoForList) ModTime() time.Time { return time.Time{} }
func (m *mockFileInfoForList) IsDir() bool        { return m.isDir }
func (m *mockFileInfoForList) Sys() interface{}   { return nil }

type mockFileSystemForList struct {
	files    map[string][]byte
	dirs     map[string]bool
	symlinks map[string]string
	errors   map[string]error
}

func newMockFileSystemForList() *mockFileSystemForList {
	return &mockFileSystemForList{
		files:    make(map[string][]byte),
		dirs:     make(map[string]bool),
		symlinks: make(map[string]string),
		errors:   make(map[string]error),
	}
}

func (m *mockFileSystemForList) createFile(path string, content []byte, mode os.FileMode) {
	m.files[path] = content
	m.dirs[path] = false
}

func (m *mockFileSystemForList) createDir(path string) {
	m.dirs[path] = true
}

func (m *mockFileSystemForList) createSymlink(symlinkPath, targetPath string) {
	m.symlinks[symlinkPath] = targetPath
}

func (m *mockFileSystemForList) setError(path string, err error) {
	m.errors[path] = err
}

func (m *mockFileSystemForList) Stat(path string) (os.FileInfo, error) {
	if err, ok := m.errors[path]; ok {
		return nil, err
	}

	// Follow symlinks
	finalPath := path
	for {
		if target, ok := m.symlinks[finalPath]; ok {
			finalPath = target
		} else {
			break
		}
	}

	if isDir, ok := m.dirs[finalPath]; ok {
		mode := os.FileMode(0755)
		if !isDir {
			mode = 0644
		}
		if isDir {
			mode |= os.ModeDir
		}
		return &mockFileInfoForList{
			name:  filepath.Base(finalPath),
			size:  0,
			mode:  mode,
			isDir: isDir,
		}, nil
	}

	if content, ok := m.files[finalPath]; ok {
		return &mockFileInfoForList{
			name:  filepath.Base(finalPath),
			size:  int64(len(content)),
			mode:  0644,
			isDir: false,
		}, nil
	}

	return nil, os.ErrNotExist
}

func (m *mockFileSystemForList) Lstat(path string) (os.FileInfo, error) {
	if err, ok := m.errors[path]; ok {
		return nil, err
	}

	// Don't follow symlinks
	if _, ok := m.symlinks[path]; ok {
		return &mockFileInfoForList{
			name:  filepath.Base(path),
			size:  0,
			mode:  os.ModeSymlink | 0777,
			isDir: false,
		}, nil
	}

	return m.Stat(path)
}

func (m *mockFileSystemForList) Readlink(path string) (string, error) {
	if target, ok := m.symlinks[path]; ok {
		return target, nil
	}
	return "", fmt.Errorf("not a symlink")
}

func (m *mockFileSystemForList) UserHomeDir() (string, error) {
	return "/home/user", nil
}

func (m *mockFileSystemForList) ListDir(path string) ([]os.FileInfo, error) {
	if err, ok := m.errors[path]; ok {
		return nil, err
	}

	if isDir, ok := m.dirs[path]; !ok || !isDir {
		return nil, fmt.Errorf("not a directory")
	}

	var entries []os.FileInfo
	prefix := path
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	// Find all direct children
	seen := make(map[string]bool)
	for p := range m.files {
		if strings.HasPrefix(p, prefix) {
			rel := strings.TrimPrefix(p, prefix)
			parts := strings.Split(rel, "/")
			if len(parts) > 0 && parts[0] != "" && !seen[parts[0]] {
				seen[parts[0]] = true
				childPath := filepath.Join(path, parts[0])
				info, _ := m.Lstat(childPath)
				if info != nil {
					entries = append(entries, info)
				}
			}
		}
	}

	for p := range m.dirs {
		if p == path {
			continue
		}
		if strings.HasPrefix(p, prefix) {
			rel := strings.TrimPrefix(p, prefix)
			parts := strings.Split(rel, "/")
			if len(parts) > 0 && parts[0] != "" && !seen[parts[0]] {
				seen[parts[0]] = true
				childPath := filepath.Join(path, parts[0])
				info, _ := m.Lstat(childPath)
				if info != nil {
					entries = append(entries, info)
				}
			}
		}
	}

	for p := range m.symlinks {
		if strings.HasPrefix(p, prefix) {
			rel := strings.TrimPrefix(p, prefix)
			parts := strings.Split(rel, "/")
			if len(parts) > 0 && parts[0] != "" && !seen[parts[0]] {
				seen[parts[0]] = true
				childPath := filepath.Join(path, parts[0])
				info, _ := m.Lstat(childPath)
				if info != nil {
					entries = append(entries, info)
				}
			}
		}
	}

	return entries, nil
}

type mockGitignoreService struct {
	shouldIgnore func(string) bool
}

func newMockGitignoreService() *mockGitignoreService {
	return &mockGitignoreService{
		shouldIgnore: func(string) bool { return false },
	}
}

func (m *mockGitignoreService) ShouldIgnore(path string) bool {
	if m.shouldIgnore != nil {
		return m.shouldIgnore(path)
	}
	return false
}

// Test functions

func TestListDirectory(t *testing.T) {
	workspaceRoot := "/workspace"

	t.Run("list workspace root with mixed files and directories", func(t *testing.T) {
		fs := newMockFileSystemForList()
		fs.createDir("/workspace")
		fs.createFile("/workspace/file1.txt", []byte("content1"), 0644)
		fs.createFile("/workspace/file2.txt", []byte("content2"), 0644)
		fs.createDir("/workspace/subdir1")
		fs.createDir("/workspace/subdir2")

		tool := NewListDirectoryTool(fs, nil, config.DefaultConfig(), workspaceRoot)

		resp, err := tool.Run(context.Background(), ListDirectoryRequest{Path: "", MaxDepth: -1, Offset: 0, Limit: 1000})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if resp.DirectoryPath != "" {
			t.Errorf("expected DirectoryPath to be empty for workspace root, got %q", resp.DirectoryPath)
		}

		if len(resp.Entries) != 4 {
			t.Fatalf("expected 4 entries, got %d", len(resp.Entries))
		}

		// Verify sorting: directories first, then files alphabetically
		expectedOrder := []struct {
			name  string
			isDir bool
		}{
			{"subdir1", true},
			{"subdir2", true},
			{"file1.txt", false},
			{"file2.txt", false},
		}

		for i, expected := range expectedOrder {
			entry := resp.Entries[i]
			if entry.RelativePath != expected.name {
				t.Errorf("entry %d: expected RelativePath %q, got %q", i, expected.name, entry.RelativePath)
			}
			if entry.IsDir != expected.isDir {
				t.Errorf("entry %d: expected IsDir %v, got %v", i, expected.isDir, entry.IsDir)
			}
		}
	})

	t.Run("list nested directory", func(t *testing.T) {
		fs := newMockFileSystemForList()
		fs.createDir("/workspace")
		fs.createDir("/workspace/src")
		fs.createFile("/workspace/src/main.go", []byte("package main"), 0644)
		fs.createFile("/workspace/src/utils.go", []byte("package main"), 0644)
		fs.createDir("/workspace/src/internal")

		tool := NewListDirectoryTool(fs, nil, config.DefaultConfig(), workspaceRoot)

		resp, err := tool.Run(context.Background(), ListDirectoryRequest{Path: "src", MaxDepth: -1, Offset: 0, Limit: 1000})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if resp.DirectoryPath != "src" {
			t.Errorf("expected DirectoryPath 'src', got %q", resp.DirectoryPath)
		}

		if len(resp.Entries) != 3 {
			t.Fatalf("expected 3 entries, got %d", len(resp.Entries))
		}
	})

	t.Run("list empty directory", func(t *testing.T) {
		fs := newMockFileSystemForList()
		fs.createDir("/workspace")
		fs.createDir("/workspace/empty")

		tool := NewListDirectoryTool(fs, nil, config.DefaultConfig(), workspaceRoot)

		resp, err := tool.Run(context.Background(), ListDirectoryRequest{Path: "empty", MaxDepth: -1, Offset: 0, Limit: 1000})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(resp.Entries) != 0 {
			t.Errorf("expected 0 entries for empty directory, got %d", len(resp.Entries))
		}
	})

	t.Run("path resolves to file not directory", func(t *testing.T) {
		fs := newMockFileSystemForList()
		fs.createDir("/workspace")
		fs.createFile("/workspace/file.txt", []byte("content"), 0644)

		tool := NewListDirectoryTool(fs, nil, config.DefaultConfig(), workspaceRoot)

		_, err := tool.Run(context.Background(), ListDirectoryRequest{Path: "file.txt", MaxDepth: -1, Offset: 0, Limit: 1000})
		if err == nil {
			t.Fatal("expected error for file path, got nil")
		}
	})

	t.Run("path outside workspace", func(t *testing.T) {
		fs := newMockFileSystemForList()
		fs.createDir("/workspace")
		fs.createDir("/tmp/outside")

		tool := NewListDirectoryTool(fs, nil, config.DefaultConfig(), workspaceRoot)

		_, err := tool.Run(context.Background(), ListDirectoryRequest{Path: "../tmp/outside", MaxDepth: -1, Offset: 0, Limit: 1000})
		if err != toolserrors.ErrOutsideWorkspace {
			t.Errorf("expected ErrOutsideWorkspace, got %v", err)
		}
	})

	t.Run("directory does not exist", func(t *testing.T) {
		fs := newMockFileSystemForList()
		fs.createDir("/workspace")

		tool := NewListDirectoryTool(fs, nil, config.DefaultConfig(), workspaceRoot)

		_, err := tool.Run(context.Background(), ListDirectoryRequest{Path: "nonexistent", MaxDepth: -1, Offset: 0, Limit: 1000})
		if err == nil {
			t.Fatal("expected error for nonexistent directory, got nil")
		}
	})

	t.Run("relative path input", func(t *testing.T) {
		fs := newMockFileSystemForList()
		fs.createDir("/workspace")
		fs.createDir("/workspace/src")
		fs.createFile("/workspace/src/main.go", []byte("package main"), 0644)

		tool := NewListDirectoryTool(fs, nil, config.DefaultConfig(), workspaceRoot)

		resp, err := tool.Run(context.Background(), ListDirectoryRequest{Path: "src", MaxDepth: -1, Offset: 0, Limit: 1000})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if resp.DirectoryPath != "src" {
			t.Errorf("expected DirectoryPath 'src', got %q", resp.DirectoryPath)
		}
	})

	t.Run("absolute path input", func(t *testing.T) {
		fs := newMockFileSystemForList()
		fs.createDir("/workspace")
		fs.createDir("/workspace/src")
		fs.createFile("/workspace/src/main.go", []byte("package main"), 0644)

		tool := NewListDirectoryTool(fs, nil, config.DefaultConfig(), workspaceRoot)

		resp, err := tool.Run(context.Background(), ListDirectoryRequest{Path: "/workspace/src", MaxDepth: -1, Offset: 0, Limit: 1000})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if resp.DirectoryPath != "src" {
			t.Errorf("expected DirectoryPath 'src', got %q", resp.DirectoryPath)
		}
	})

	t.Run("dot path alias for workspace root", func(t *testing.T) {
		fs := newMockFileSystemForList()
		fs.createDir("/workspace")
		fs.createFile("/workspace/file.txt", []byte("content"), 0644)

		tool := NewListDirectoryTool(fs, nil, config.DefaultConfig(), workspaceRoot)

		resp, err := tool.Run(context.Background(), ListDirectoryRequest{Path: ".", MaxDepth: -1, Offset: 0, Limit: 1000})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if resp.DirectoryPath != "" {
			t.Errorf("expected DirectoryPath to be empty for '.', got %q", resp.DirectoryPath)
		}
	})
}

func TestListDirectory_Pagination(t *testing.T) {
	workspaceRoot := "/workspace"

	t.Run("pagination with offset and limit", func(t *testing.T) {
		fs := newMockFileSystemForList()
		fs.createDir("/workspace")
		// Create 10 files
		for i := 1; i <= 10; i++ {
			fs.createFile(fmt.Sprintf("/workspace/file%02d.txt", i), []byte("content"), 0644)
		}

		tool := NewListDirectoryTool(fs, nil, config.DefaultConfig(), workspaceRoot)

		// Get first 5
		resp, err := tool.Run(context.Background(), ListDirectoryRequest{Path: "", MaxDepth: -1, Offset: 0, Limit: 5})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(resp.Entries) != 5 {
			t.Errorf("expected 5 entries, got %d", len(resp.Entries))
		}

		if resp.TotalCount != 10 {
			t.Errorf("expected TotalCount 10, got %d", resp.TotalCount)
		}

		// Get next 5
		resp2, err := tool.Run(context.Background(), ListDirectoryRequest{Path: "", MaxDepth: -1, Offset: 5, Limit: 5})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(resp2.Entries) != 5 {
			t.Errorf("expected 5 entries in second page, got %d", len(resp2.Entries))
		}
	})
}

func TestListDirectory_WithSymlinks(t *testing.T) {
	workspaceRoot := "/workspace"

	t.Run("directory with symlinks", func(t *testing.T) {
		fs := newMockFileSystemForList()
		fs.createDir("/workspace")
		fs.createFile("/workspace/file.txt", []byte("content"), 0644)
		fs.createSymlink("/workspace/link.txt", "/workspace/file.txt")
		fs.createDir("/workspace/dir")
		fs.createSymlink("/workspace/linkdir", "/workspace/dir")

		tool := NewListDirectoryTool(fs, nil, config.DefaultConfig(), workspaceRoot)

		resp, err := tool.Run(context.Background(), ListDirectoryRequest{Path: "", MaxDepth: -1, Offset: 0, Limit: 1000})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Should have 4 entries: dir, linkdir, file.txt, link.txt
		if len(resp.Entries) != 4 {
			t.Fatalf("expected 4 entries, got %d", len(resp.Entries))
		}
	})
}

func TestListDirectory_UnicodeFilenames(t *testing.T) {
	workspaceRoot := "/workspace"

	t.Run("unicode and special characters", func(t *testing.T) {
		fs := newMockFileSystemForList()
		fs.createDir("/workspace")
		fs.createFile("/workspace/文件.txt", []byte("content"), 0644)
		fs.createFile("/workspace/файл.txt", []byte("content"), 0644)
		fs.createFile("/workspace/ファイル.txt", []byte("content"), 0644)

		tool := NewListDirectoryTool(fs, nil, config.DefaultConfig(), workspaceRoot)

		resp, err := tool.Run(context.Background(), ListDirectoryRequest{Path: "", MaxDepth: -1, Offset: 0, Limit: 1000})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(resp.Entries) != 3 {
			t.Fatalf("expected 3 entries, got %d", len(resp.Entries))
		}
	})
}

func TestListDirectory_DotfilesWithGitignore(t *testing.T) {
	workspaceRoot := "/workspace"

	t.Run("dotfiles filtered by gitignore", func(t *testing.T) {
		fs := newMockFileSystemForList()
		fs.createDir("/workspace")
		fs.createFile("/workspace/file.txt", []byte("content"), 0644)
		fs.createFile("/workspace/.hidden", []byte("content"), 0644)
		fs.createFile("/workspace/.gitignore", []byte("content"), 0644)

		gitignore := newMockGitignoreService()
		gitignore.shouldIgnore = func(path string) bool {
			return strings.HasPrefix(filepath.Base(path), ".")
		}

		tool := NewListDirectoryTool(fs, gitignore, config.DefaultConfig(), workspaceRoot)

		resp, err := tool.Run(context.Background(), ListDirectoryRequest{Path: "", MaxDepth: -1, Offset: 0, Limit: 1000})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Should only have file.txt, dotfiles filtered
		if len(resp.Entries) != 1 {
			t.Fatalf("expected 1 entry (dotfiles filtered), got %d", len(resp.Entries))
		}

		if resp.Entries[0].RelativePath != "file.txt" {
			t.Errorf("expected file.txt, got %s", resp.Entries[0].RelativePath)
		}
	})
}

func TestListDirectory_DotfilesWithoutGitignore(t *testing.T) {
	workspaceRoot := "/workspace"

	t.Run("all dotfiles included when gitignore service is nil", func(t *testing.T) {
		fs := newMockFileSystemForList()
		fs.createDir("/workspace")
		fs.createFile("/workspace/file.txt", []byte("content"), 0644)
		fs.createFile("/workspace/.hidden", []byte("content"), 0644)
		fs.createFile("/workspace/.gitignore", []byte("content"), 0644)

		tool := NewListDirectoryTool(fs, nil, config.DefaultConfig(), workspaceRoot)

		resp, err := tool.Run(context.Background(), ListDirectoryRequest{Path: "", MaxDepth: -1, Offset: 0, Limit: 1000})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Should have all 3 files
		if len(resp.Entries) != 3 {
			t.Fatalf("expected 3 entries, got %d", len(resp.Entries))
		}
	})
}

func TestListDirectory_LargeDirectory(t *testing.T) {
	workspaceRoot := "/workspace"

	t.Run("large directory pagination", func(t *testing.T) {
		fs := newMockFileSystemForList()
		fs.createDir("/workspace")
		// Create 100 files
		for i := 1; i <= 100; i++ {
			fs.createFile(fmt.Sprintf("/workspace/file%03d.txt", i), []byte("content"), 0644)
		}

		tool := NewListDirectoryTool(fs, nil, config.DefaultConfig(), workspaceRoot)

		resp, err := tool.Run(context.Background(), ListDirectoryRequest{Path: "", MaxDepth: -1, Offset: 0, Limit: 50})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(resp.Entries) != 50 {
			t.Errorf("expected 50 entries, got %d", len(resp.Entries))
		}

		if resp.TotalCount != 100 {
			t.Errorf("expected TotalCount 100, got %d", resp.TotalCount)
		}
	})
}

func TestListDirectory_OffsetBeyondEnd(t *testing.T) {
	workspaceRoot := "/workspace"

	t.Run("offset beyond end returns empty", func(t *testing.T) {
		fs := newMockFileSystemForList()
		fs.createDir("/workspace")
		fs.createFile("/workspace/file.txt", []byte("content"), 0644)

		tool := NewListDirectoryTool(fs, nil, config.DefaultConfig(), workspaceRoot)

		resp, err := tool.Run(context.Background(), ListDirectoryRequest{Path: "", MaxDepth: -1, Offset: 100, Limit: 10})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(resp.Entries) != 0 {
			t.Errorf("expected 0 entries for offset beyond end, got %d", len(resp.Entries))
		}

		if resp.TotalCount != 1 {
			t.Errorf("expected TotalCount 1, got %d", resp.TotalCount)
		}
	})

	t.Run("filesystem error propagation", func(t *testing.T) {
		fs := newMockFileSystemForList()
		fs.createDir("/workspace")
		fs.createDir("/workspace/testdir")
		fs.setError("/workspace/testdir", os.ErrPermission)

		tool := NewListDirectoryTool(fs, nil, config.DefaultConfig(), workspaceRoot)

		_, err := tool.Run(context.Background(), ListDirectoryRequest{Path: "testdir", MaxDepth: -1, Offset: 0, Limit: 1000})
		if err == nil {
			t.Fatal("expected error from filesystem, got nil")
		}
		if !strings.Contains(err.Error(), "permission") {
			t.Errorf("expected permission-related error, got: %v", err)
		}
	})

	t.Run("verify entry metadata correctness", func(t *testing.T) {
		fs := newMockFileSystemForList()
		fs.createDir("/workspace")
		fs.createFile("/workspace/file.txt", []byte("hello world"), 0644)
		fs.createDir("/workspace/subdir")

		tool := NewListDirectoryTool(fs, nil, config.DefaultConfig(), workspaceRoot)

		resp, err := tool.Run(context.Background(), ListDirectoryRequest{Path: "", MaxDepth: -1, Offset: 0, Limit: 1000})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Find file and directory entries
		var fileEntry, dirEntry *DirectoryEntry
		for i := range resp.Entries {
			switch resp.Entries[i].RelativePath {
			case "file.txt":
				fileEntry = &resp.Entries[i]
			case "subdir":
				dirEntry = &resp.Entries[i]
			}
		}

		if fileEntry == nil {
			t.Fatal("expected to find file.txt entry")
		}

		if dirEntry == nil {
			t.Fatal("expected to find subdir entry")
		}

		// Verify file entry
		if fileEntry.IsDir {
			t.Error("file.txt should not be marked as directory")
		}

		if fileEntry.RelativePath != "file.txt" {
			t.Errorf("expected RelativePath 'file.txt', got %q", fileEntry.RelativePath)
		}

		// Verify directory entry
		if !dirEntry.IsDir {
			t.Error("subdir should be marked as directory")
		}

		if dirEntry.RelativePath != "subdir" {
			t.Errorf("expected RelativePath 'subdir', got %q", dirEntry.RelativePath)
		}
	})

	t.Run("sorting: directories before files, alphabetical within each group", func(t *testing.T) {
		fs := newMockFileSystemForList()
		fs.createDir("/workspace")
		fs.createFile("/workspace/zebra.txt", []byte("z"), 0644)
		fs.createFile("/workspace/alpha.txt", []byte("a"), 0644)
		fs.createDir("/workspace/zulu")
		fs.createDir("/workspace/alpha")

		tool := NewListDirectoryTool(fs, nil, config.DefaultConfig(), workspaceRoot)

		resp, err := tool.Run(context.Background(), ListDirectoryRequest{Path: "", MaxDepth: -1, Offset: 0, Limit: 1000})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(resp.Entries) != 4 {
			t.Fatalf("expected 4 entries, got %d", len(resp.Entries))
		}

		// Verify order: alpha (dir), zulu (dir), alpha.txt (file), zebra.txt (file)
		expectedOrder := []string{"alpha", "zulu", "alpha.txt", "zebra.txt"}
		for i, expected := range expectedOrder {
			if resp.Entries[i].RelativePath != expected {
				t.Errorf("entry %d: expected %q, got %q", i, expected, resp.Entries[i].RelativePath)
			}
		}

		// Verify directories come first
		if !resp.Entries[0].IsDir || !resp.Entries[1].IsDir {
			t.Error("directories should come before files")
		}

		if resp.Entries[2].IsDir || resp.Entries[3].IsDir {
			t.Error("files should come after directories")
		}
	})

	t.Run("nested directory with relative path", func(t *testing.T) {
		fs := newMockFileSystemForList()
		fs.createDir("/workspace")
		fs.createDir("/workspace/src")
		fs.createDir("/workspace/src/app")
		fs.createFile("/workspace/src/app/main.go", []byte("package main"), 0644)

		tool := NewListDirectoryTool(fs, nil, config.DefaultConfig(), workspaceRoot)

		resp, err := tool.Run(context.Background(), ListDirectoryRequest{Path: "src/app", MaxDepth: -1, Offset: 0, Limit: 1000})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if resp.DirectoryPath != "src/app" {
			t.Errorf("expected DirectoryPath 'src/app', got %q", resp.DirectoryPath)
		}

		if len(resp.Entries) != 1 {
			t.Fatalf("expected 1 entry, got %d", len(resp.Entries))
		}

		if resp.Entries[0].RelativePath != "src/app/main.go" {
			t.Errorf("expected RelativePath 'src/app/main.go', got %q", resp.Entries[0].RelativePath)
		}
	})
}

func TestListDirectory_InvalidPagination(t *testing.T) {
	workspaceRoot := "/workspace"

	t.Run("zero limit uses default", func(t *testing.T) {
		fs := newMockFileSystemForList()
		fs.createDir("/workspace")
		fs.createFile("/workspace/file.txt", []byte("content"), 0644)

		cfg := config.DefaultConfig()
		tool := NewListDirectoryTool(fs, nil, cfg, workspaceRoot)

		// Limit=0 should use default
		resp, err := tool.Run(context.Background(), ListDirectoryRequest{Path: "", MaxDepth: -1, Offset: 0, Limit: 0})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Limit != cfg.Tools.DefaultListDirectoryLimit {
			t.Errorf("expected default limit %d, got %d", cfg.Tools.DefaultListDirectoryLimit, resp.Limit)
		}
	})
}
