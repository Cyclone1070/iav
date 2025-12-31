package directory

// List directory tests - mocks shared from file package write_test.go pattern

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
	"github.com/Cyclone1070/iav/internal/tool/service/path"
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
func (m *mockFileInfoForList) Sys() any           { return nil }

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
			mode = 0o644
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
			mode:  0o644,
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

	// Follow symlinks for the directory check
	finalPath := path
	for {
		if target, ok := m.symlinks[finalPath]; ok {
			finalPath = target
		} else {
			break
		}
	}

	if isDir, ok := m.dirs[finalPath]; !ok || !isDir {
		return nil, fmt.Errorf("not a directory")
	}

	var entries []os.FileInfo
	prefix := finalPath
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	// Find all direct children
	seen := make(map[string]bool)

	// Helper to collect direct children
	collectChildren := func(pathStr string) {
		if after, ok := strings.CutPrefix(pathStr, prefix); ok {
			rel := after
			parts := strings.Split(rel, "/")
			if len(parts) > 0 && parts[0] != "" && !seen[parts[0]] {
				seen[parts[0]] = true
				childPath := filepath.Join(finalPath, parts[0])
				info, _ := m.Lstat(childPath)
				if info != nil {
					entries = append(entries, info)
				}
			}
		}
	}

	// Check files
	for p := range m.files {
		collectChildren(p)
	}
	// Also check dirs
	for p, isDir := range m.dirs {
		if isDir {
			collectChildren(p)
		}
	}
	// Also check symlinks
	for p := range m.symlinks {
		collectChildren(p)
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
		fs.createFile("/workspace/file1.txt", []byte("content1"), 0o644)
		fs.createFile("/workspace/file2.txt", []byte("content2"), 0o644)
		fs.createDir("/workspace/subdir1")
		fs.createDir("/workspace/subdir2")

		cfg := config.DefaultConfig()
		listTool := NewListDirectoryTool(fs, nil, cfg, path.NewResolver(workspaceRoot))

		req := &ListDirectoryRequest{Path: ".", MaxDepth: -1, Offset: 0, Limit: 1000}
		resp, err := listTool.Run(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if resp.DirectoryPath != "" {
			t.Errorf("expected DirectoryPath to be empty for workspace root, got %q", resp.DirectoryPath)
		}

		if resp.TotalCount != 4 {
			t.Fatalf("expected 4 entries, got %d", resp.TotalCount)
		}

		// Verify formatting and sorting: directories first (with /), then files alphabetically
		expected := "subdir1/\nsubdir2/\nfile1.txt\nfile2.txt\n"
		if resp.FormattedEntries != expected {
			t.Errorf("expected FormattedEntries:\n%q\ngot:\n%q", expected, resp.FormattedEntries)
		}
	})

	t.Run("list nested directory", func(t *testing.T) {
		fs := newMockFileSystemForList()
		fs.createDir("/workspace")
		fs.createDir("/workspace/src")
		fs.createFile("/workspace/src/main.go", []byte("package main"), 0o644)
		fs.createFile("/workspace/src/utils.go", []byte("package main"), 0o644)
		fs.createDir("/workspace/src/internal")

		cfg := config.DefaultConfig()
		listTool := NewListDirectoryTool(fs, nil, cfg, path.NewResolver(workspaceRoot))

		req := &ListDirectoryRequest{Path: "src", MaxDepth: -1, Offset: 0, Limit: 1000}
		resp, err := listTool.Run(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if resp.DirectoryPath != "src" {
			t.Errorf("expected DirectoryPath 'src', got %q", resp.DirectoryPath)
		}

		if resp.TotalCount != 3 {
			t.Fatalf("expected 3 entries, got %d", resp.TotalCount)
		}
		expected := "src/internal/\nsrc/main.go\nsrc/utils.go\n"
		if resp.FormattedEntries != expected {
			t.Errorf("expected FormattedEntries:\n%q\ngot:\n%q", expected, resp.FormattedEntries)
		}
	})

	t.Run("list empty directory", func(t *testing.T) {
		fs := newMockFileSystemForList()
		fs.createDir("/workspace")
		fs.createDir("/workspace/empty")

		cfg := config.DefaultConfig()
		listTool := NewListDirectoryTool(fs, nil, cfg, path.NewResolver(workspaceRoot))

		req := &ListDirectoryRequest{Path: "empty", MaxDepth: -1, Offset: 0, Limit: 1000}
		resp, err := listTool.Run(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if resp.FormattedEntries != "" {
			t.Errorf("expected empty string for empty directory, got %q", resp.FormattedEntries)
		}
	})

	t.Run("path resolves to file not directory", func(t *testing.T) {
		fs := newMockFileSystemForList()
		fs.createDir("/workspace")
		fs.createFile("/workspace/file.txt", []byte("content"), 0o644)
		cfg := config.DefaultConfig()

		req := &ListDirectoryRequest{Path: "file.txt", MaxDepth: -1, Offset: 0, Limit: 1000}
		listTool := NewListDirectoryTool(fs, nil, cfg, path.NewResolver(workspaceRoot))
		_, err := listTool.Run(context.Background(), req)

		if err == nil {
			t.Fatalf("expected error for file instead of directory, got nil")
		}
	})

	t.Run("path outside workspace", func(t *testing.T) {
		fs := newMockFileSystemForList()
		fs.createDir("/workspace")
		fs.createDir("/tmp/outside")

		cfg := config.DefaultConfig()
		listTool := NewListDirectoryTool(fs, nil, cfg, path.NewResolver(workspaceRoot))

		req := &ListDirectoryRequest{Path: "../tmp/outside", MaxDepth: -1, Offset: 0, Limit: 1000}
		_, err := listTool.Run(context.Background(), req)

		if err == nil {
			t.Fatal("expected error, got nil")
		}

		if !errors.Is(err, path.ErrOutsideWorkspace) {
			t.Errorf("expected ErrOutsideWorkspace, got %v", err)
		}
	})

	t.Run("directory does not exist", func(t *testing.T) {
		fs := newMockFileSystemForList()
		fs.createDir("/workspace")
		cfg := config.DefaultConfig()

		req := &ListDirectoryRequest{Path: "nonexistent", MaxDepth: -1, Offset: 0, Limit: 1000}
		listTool := NewListDirectoryTool(fs, nil, cfg, path.NewResolver(workspaceRoot))
		_, err := listTool.Run(context.Background(), req)
		if err == nil {
			t.Errorf("expected error for non-existent directory, got nil")
		}
	})

	t.Run("relative path input", func(t *testing.T) {
		fs := newMockFileSystemForList()
		fs.createDir("/workspace")
		fs.createDir("/workspace/src")
		fs.createFile("/workspace/src/main.go", []byte("package main"), 0o644)

		cfg := config.DefaultConfig()
		listTool := NewListDirectoryTool(fs, nil, cfg, path.NewResolver(workspaceRoot))

		req := &ListDirectoryRequest{Path: "src", MaxDepth: -1, Offset: 0, Limit: 1000}
		resp, err := listTool.Run(context.Background(), req)
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
		fs.createFile("/workspace/src/main.go", []byte("package main"), 0o644)

		cfg := config.DefaultConfig()
		listTool := NewListDirectoryTool(fs, nil, cfg, path.NewResolver(workspaceRoot))

		req := &ListDirectoryRequest{Path: "/workspace/src", MaxDepth: -1, Offset: 0, Limit: 1000}
		resp, err := listTool.Run(context.Background(), req)
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
		fs.createFile("/workspace/file.txt", []byte("content"), 0o644)

		cfg := config.DefaultConfig()
		listTool := NewListDirectoryTool(fs, nil, cfg, path.NewResolver(workspaceRoot))

		req := &ListDirectoryRequest{Path: ".", MaxDepth: -1, Offset: 0, Limit: 1000}
		resp, err := listTool.Run(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if resp.DirectoryPath != "" {
			t.Errorf("expected DirectoryPath to be empty for '.', got %q", resp.DirectoryPath)
		}
	})
	t.Run("empty path defaults to workspace root", func(t *testing.T) {
		fs := newMockFileSystemForList()
		fs.createDir("/workspace")
		fs.createFile("/workspace/file.txt", []byte("content"), 0o644)

		cfg := config.DefaultConfig()
		listTool := NewListDirectoryTool(fs, nil, cfg, path.NewResolver(workspaceRoot))

		req := &ListDirectoryRequest{Path: "", MaxDepth: -1, Offset: 0, Limit: 1000}
		resp, err := listTool.Run(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if resp.DirectoryPath != "" {
			t.Errorf("expected DirectoryPath to be empty for '', got %q", resp.DirectoryPath)
		}
		if resp.FormattedEntries == "" {
			t.Error("expected non-empty FormattedEntries for root")
		}
		if resp.TotalCount != 1 {
			t.Errorf("expected 1 entry, got %d", resp.TotalCount)
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
			fs.createFile(fmt.Sprintf("/workspace/file%02d.txt", i), []byte("content"), 0o644)
		}

		cfg := config.DefaultConfig()
		listTool := NewListDirectoryTool(fs, nil, cfg, path.NewResolver(workspaceRoot))

		// Get first 5
		req1 := &ListDirectoryRequest{Path: ".", MaxDepth: -1, Offset: 0, Limit: 5}
		resp, err := listTool.Run(context.Background(), req1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if resp.TotalCount != 10 {
			t.Errorf("expected TotalCount 10, got %d", resp.TotalCount)
		}

		// Get next 5
		req2 := &ListDirectoryRequest{Path: ".", MaxDepth: -1, Offset: 5, Limit: 5}
		resp2, err := listTool.Run(context.Background(), req2)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if resp2.TotalCount != 10 {
			t.Errorf("expected TotalCount 10 in second page, got %d", resp2.TotalCount)
		}
	})
}

func TestListDirectory_WithSymlinks(t *testing.T) {
	workspaceRoot := "/workspace"

	t.Run("directory with symlinks", func(t *testing.T) {
		fs := newMockFileSystemForList()
		fs.createDir("/workspace")
		fs.createFile("/workspace/file.txt", []byte("content"), 0o644)
		fs.createSymlink("/workspace/link.txt", "/workspace/file.txt")
		fs.createDir("/workspace/dir")
		fs.createSymlink("/workspace/linkdir", "/workspace/dir")

		cfg := config.DefaultConfig()
		listTool := NewListDirectoryTool(fs, nil, cfg, path.NewResolver(workspaceRoot))

		req := &ListDirectoryRequest{Path: ".", MaxDepth: -1, Offset: 0, Limit: 1000}
		resp, err := listTool.Run(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Should have 4 entries: dir, linkdir, file.txt, link.txt
		if resp.TotalCount != 4 {
			t.Fatalf("expected 4 entries, got %d", resp.TotalCount)
		}
		expected := "dir/\nlinkdir/\nfile.txt\nlink.txt\n" // Assuming linkdir points to a dir
		if resp.FormattedEntries != expected {
			t.Errorf("expected:\n%q\ngot:\n%q", expected, resp.FormattedEntries)
		}
	})
}

func TestListDirectory_UnicodeFilenames(t *testing.T) {
	workspaceRoot := "/workspace"

	t.Run("unicode and special characters", func(t *testing.T) {
		fs := newMockFileSystemForList()
		fs.createDir("/workspace")
		fs.createFile("/workspace/文件.txt", []byte("content"), 0o644)
		fs.createFile("/workspace/файл.txt", []byte("content"), 0o644)
		fs.createFile("/workspace/ファイル.txt", []byte("content"), 0o644)

		cfg := config.DefaultConfig()
		listTool := NewListDirectoryTool(fs, nil, cfg, path.NewResolver(workspaceRoot))

		req := &ListDirectoryRequest{Path: ".", MaxDepth: -1, Offset: 0, Limit: 1000}
		resp, err := listTool.Run(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if resp.TotalCount != 3 {
			t.Fatalf("expected 3 entries, got %d", resp.TotalCount)
		}
	})
}

func TestListDirectory_DotfilesWithGitignore(t *testing.T) {
	workspaceRoot := "/workspace"

	t.Run("dotfiles filtered by gitignore", func(t *testing.T) {
		fs := newMockFileSystemForList()
		fs.createDir("/workspace")
		fs.createFile("/workspace/file.txt", []byte("content"), 0o644)
		fs.createFile("/workspace/.hidden", []byte("content"), 0o644)
		fs.createFile("/workspace/.gitignore", []byte("content"), 0o644)

		gitignore := newMockGitignoreService()
		gitignore.shouldIgnore = func(path string) bool {
			return strings.HasPrefix(filepath.Base(path), ".")
		}

		cfg := config.DefaultConfig()
		listTool := NewListDirectoryTool(fs, gitignore, cfg, path.NewResolver(workspaceRoot))

		req := &ListDirectoryRequest{Path: ".", MaxDepth: -1, Offset: 0, Limit: 1000}
		resp, err := listTool.Run(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Should only have file.txt, dotfiles filtered
		if resp.TotalCount != 1 {
			t.Fatalf("expected 1 entry (dotfiles filtered), got %d", resp.TotalCount)
		}

		if resp.FormattedEntries != "file.txt\n" {
			t.Errorf("expected 'file.txt\\n', got %q", resp.FormattedEntries)
		}
	})
}

func TestListDirectory_DotfilesWithoutGitignore(t *testing.T) {
	workspaceRoot := "/workspace"

	t.Run("all dotfiles included when gitignore service is nil", func(t *testing.T) {
		fs := newMockFileSystemForList()
		fs.createDir("/workspace")
		fs.createFile("/workspace/file.txt", []byte("content"), 0o644)
		fs.createFile("/workspace/.hidden", []byte("content"), 0o644)
		fs.createFile("/workspace/.gitignore", []byte("content"), 0o644)

		cfg := config.DefaultConfig()
		listTool := NewListDirectoryTool(fs, nil, cfg, path.NewResolver(workspaceRoot))

		req := &ListDirectoryRequest{Path: ".", MaxDepth: -1, Offset: 0, Limit: 1000}
		resp, err := listTool.Run(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Should have all 3 files
		if resp.TotalCount != 3 {
			t.Fatalf("expected 3 entries, got %d", resp.TotalCount)
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
			fs.createFile(fmt.Sprintf("/workspace/file%03d.txt", i), []byte("content"), 0o644)
		}

		cfg := config.DefaultConfig()
		listTool := NewListDirectoryTool(fs, nil, cfg, path.NewResolver(workspaceRoot))

		req := &ListDirectoryRequest{Path: ".", MaxDepth: -1, Offset: 0, Limit: 50}
		resp, err := listTool.Run(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
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
		fs.createFile("/workspace/file.txt", []byte("content"), 0o644)

		cfg := config.DefaultConfig()
		listTool := NewListDirectoryTool(fs, nil, cfg, path.NewResolver(workspaceRoot))

		req := &ListDirectoryRequest{Path: ".", MaxDepth: -1, Offset: 100, Limit: 10}
		resp, err := listTool.Run(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if resp.FormattedEntries != "" {
			t.Errorf("expected empty string for offset beyond end, got %q", resp.FormattedEntries)
		}

		if resp.TotalCount != 1 {
			t.Errorf("expected TotalCount 1, got %d", resp.TotalCount)
		}
	})
}

func TestListDirectory_FilesystemErrorPropagation(t *testing.T) {
	workspaceRoot := "/workspace"

	t.Run("filesystem error propagation", func(t *testing.T) {
		fs := newMockFileSystemForList()
		fs.createDir("/workspace")
		fs.createDir("/workspace/testdir")
		fs.setError("/workspace/testdir", os.ErrPermission)
		cfg := config.DefaultConfig()

		req := &ListDirectoryRequest{Path: "testdir", MaxDepth: -1, Offset: 0, Limit: 1000}
		listTool := NewListDirectoryTool(fs, nil, cfg, path.NewResolver(workspaceRoot))
		_, err := listTool.Run(context.Background(), req)
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		if !strings.Contains(err.Error(), "permission") {
			t.Errorf("expected permission-related error, got: %v", err)
		}
	})
}

func TestListDirectory_EntryMetadata(t *testing.T) {
	workspaceRoot := "/workspace"

	t.Run("verify entry metadata correctness", func(t *testing.T) {
		fs := newMockFileSystemForList()
		fs.createDir("/workspace")
		fs.createFile("/workspace/file.txt", []byte("hello world"), 0o644)
		fs.createDir("/workspace/subdir")

		cfg := config.DefaultConfig()
		listTool := NewListDirectoryTool(fs, nil, cfg, path.NewResolver(workspaceRoot))

		req := &ListDirectoryRequest{Path: ".", MaxDepth: -1, Offset: 0, Limit: 1000}
		resp, err := listTool.Run(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if resp.TotalCount != 2 {
			t.Fatalf("expected 2 entries, got %d", resp.TotalCount)
		}

		if !strings.Contains(resp.FormattedEntries, "file.txt\n") {
			t.Error("expected FormattedEntries to contain file.txt")
		}
		if !strings.Contains(resp.FormattedEntries, "subdir/\n") {
			t.Error("expected FormattedEntries to contain subdir/")
		}
	})
}

func TestListDirectory_Sorting(t *testing.T) {
	workspaceRoot := "/workspace"

	t.Run("sorting: directories before files, alphabetical within each group", func(t *testing.T) {
		fs := newMockFileSystemForList()
		fs.createDir("/workspace")
		fs.createFile("/workspace/zebra.txt", []byte("z"), 0o644)
		fs.createFile("/workspace/alpha.txt", []byte("a"), 0o644)
		fs.createDir("/workspace/zulu")
		fs.createDir("/workspace/alpha")

		cfg := config.DefaultConfig()
		listTool := NewListDirectoryTool(fs, nil, cfg, path.NewResolver(workspaceRoot))

		req := &ListDirectoryRequest{Path: ".", MaxDepth: -1, Offset: 0, Limit: 1000}
		resp, err := listTool.Run(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if resp.TotalCount != 4 {
			t.Fatalf("expected 4 entries, got %d", resp.TotalCount)
		}

		// Verify order: alpha (dir), zulu (dir), alpha.txt (file), zebra.txt (file)
		expected := "alpha/\nzulu/\nalpha.txt\nzebra.txt\n"
		if resp.FormattedEntries != expected {
			t.Errorf("expected sorted output:\n%q\ngot:\n%q", expected, resp.FormattedEntries)
		}
	})
}

func TestListDirectory_NestedRelativePath(t *testing.T) {
	workspaceRoot := "/workspace"

	t.Run("nested directory with relative path", func(t *testing.T) {
		fs := newMockFileSystemForList()
		fs.createDir("/workspace")
		fs.createDir("/workspace/src")
		fs.createDir("/workspace/src/app")
		fs.createFile("/workspace/src/app/main.go", []byte("package main"), 0o644)

		cfg := config.DefaultConfig()
		listTool := NewListDirectoryTool(fs, nil, cfg, path.NewResolver(workspaceRoot))

		req := &ListDirectoryRequest{Path: "src/app", MaxDepth: -1, Offset: 0, Limit: 1000}
		resp, err := listTool.Run(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if resp.DirectoryPath != "src/app" {
			t.Errorf("expected DirectoryPath 'src/app', got %q", resp.DirectoryPath)
		}

		if resp.TotalCount != 1 {
			t.Fatalf("expected 1 entry, got %d", resp.TotalCount)
		}

		if resp.FormattedEntries != "src/app/main.go\n" {
			t.Errorf("expected 'src/app/main.go\\n', got %q", resp.FormattedEntries)
		}
	})
}

func TestListDirectory_NegativeOffset_Clamps(t *testing.T) {
	workspaceRoot := "/workspace"
	t.Run("negative offset", func(t *testing.T) {
		fs := newMockFileSystemForList()
		fs.createDir("/workspace")
		cfg := config.DefaultConfig()
		listTool := NewListDirectoryTool(fs, nil, cfg, path.NewResolver(workspaceRoot))

		req := &ListDirectoryRequest{Path: ".", MaxDepth: -1, Offset: -1, Limit: 10}
		_, err := listTool.Run(context.Background(), req)
		if err != nil {
			t.Errorf("unexpected error for negative offset: %v", err)
		}
	})
}

func TestListDirectory_ContextCancellation(t *testing.T) {
	workspaceRoot := "/workspace"

	t.Run("context cancellation stops listing", func(t *testing.T) {
		fs := newMockFileSystemForList()
		fs.createDir("/workspace")
		fs.createFile("/workspace/files.txt", []byte("content"), 0o644)

		cfg := config.DefaultConfig()
		listTool := NewListDirectoryTool(fs, nil, cfg, path.NewResolver(workspaceRoot))

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		req := &ListDirectoryRequest{Path: ".", MaxDepth: -1, Offset: 0, Limit: 1000}
		_, err := listTool.Run(ctx, req)
		if err == nil {
			t.Error("expected error for cancelled context")
		}
		if !errors.Is(err, context.Canceled) {
			t.Errorf("expected context.Canceled, got %v", err)
		}
	})
}

func TestListDirectory_HitMaxResults(t *testing.T) {
	workspaceRoot := "/workspace"
	fs := newMockFileSystemForList()
	fs.createDir("/workspace")
	// Create 10 files
	for i := 1; i <= 10; i++ {
		fs.createFile(fmt.Sprintf("/workspace/file%d.txt", i), []byte("content"), 0o644)
	}

	cfg := config.DefaultConfig()
	// Set a very low limit to trigger cap
	cfg.Tools.MaxListDirectoryResults = 5
	listTool := NewListDirectoryTool(fs, nil, cfg, path.NewResolver(workspaceRoot))

	req := &ListDirectoryRequest{Path: ".", MaxDepth: -1, Offset: 0, Limit: 100}
	resp, err := listTool.Run(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !resp.HitMaxResults {
		t.Error("expected HitMaxResults to be true")
	}

	if resp.TotalCount != 5 {
		t.Errorf("expected TotalCount 5 (capped), got %d", resp.TotalCount)
	}
}
