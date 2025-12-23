package pathutil

// Path resolution tests

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Local mocks for directory listing tests

type mockFileInfoForPath struct {
	name  string
	size  int64
	mode  os.FileMode
	isDir bool
}

func (m *mockFileInfoForPath) Name() string       { return m.name }
func (m *mockFileInfoForPath) Size() int64        { return m.size }
func (m *mockFileInfoForPath) Mode() os.FileMode  { return m.mode }
func (m *mockFileInfoForPath) ModTime() time.Time { return time.Time{} }
func (m *mockFileInfoForPath) IsDir() bool        { return m.isDir }
func (m *mockFileInfoForPath) Sys() any           { return nil }

type mockFileSystemForPath struct {
	files    map[string][]byte
	dirs     map[string]bool
	symlinks map[string]string
	errors   map[string]error
}

func newMockFileSystemForPath() *mockFileSystemForPath {
	return &mockFileSystemForPath{
		files:    make(map[string][]byte),
		dirs:     make(map[string]bool),
		symlinks: make(map[string]string),
		errors:   make(map[string]error),
	}
}

func (m *mockFileSystemForPath) createFile(path string, content []byte, mode os.FileMode) {
	m.files[path] = content
	m.dirs[path] = false
}

func (m *mockFileSystemForPath) createDir(path string) {
	m.dirs[path] = true
}

func (m *mockFileSystemForPath) createSymlink(symlinkPath, targetPath string) {
	m.symlinks[symlinkPath] = targetPath
}

func (m *mockFileSystemForPath) setError(path string, err error) {
	m.errors[path] = err
}

func (m *mockFileSystemForPath) Stat(path string) (os.FileInfo, error) {
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
		return &mockFileInfoForPath{
			name:  filepath.Base(finalPath),
			size:  0,
			mode:  mode,
			isDir: isDir,
		}, nil
	}

	if content, ok := m.files[finalPath]; ok {
		return &mockFileInfoForPath{
			name:  filepath.Base(finalPath),
			size:  int64(len(content)),
			mode:  0644,
			isDir: false,
		}, nil
	}

	return nil, os.ErrNotExist
}

func (m *mockFileSystemForPath) Lstat(path string) (os.FileInfo, error) {
	if err, ok := m.errors[path]; ok {
		return nil, err
	}

	// Don't follow symlinks
	if _, ok := m.symlinks[path]; ok {
		return &mockFileInfoForPath{
			name:  filepath.Base(path),
			size:  0,
			mode:  os.ModeSymlink | 0777,
			isDir: false,
		}, nil
	}

	return m.Stat(path)
}

func (m *mockFileSystemForPath) Readlink(path string) (string, error) {
	if target, ok := m.symlinks[path]; ok {
		return target, nil
	}
	return "", fmt.Errorf("not a symlink")
}

func (m *mockFileSystemForPath) UserHomeDir() (string, error) {
	return "/home/user", nil
}

func (m *mockFileSystemForPath) ListDir(path string) ([]os.FileInfo, error) {
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
		if after, ok := strings.CutPrefix(p, prefix); ok {
			rel := after
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
		if after, ok := strings.CutPrefix(p, prefix); ok {
			rel := after
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
		if after, ok := strings.CutPrefix(p, prefix); ok {
			rel := after
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

// Test functions

func TestResolve(t *testing.T) {
	workspaceRoot := "/workspace"

	t.Run("relative path resolution", func(t *testing.T) {
		fs := newMockFileSystemForPath()

		abs, rel, err := NewResolver(workspaceRoot, fs).Resolve("test.txt")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if abs != "/workspace/test.txt" {
			t.Errorf("expected absolute path /workspace/test.txt, got %s", abs)
		}
		if rel != "test.txt" {
			t.Errorf("expected relative path test.txt, got %s", rel)
		}
	})

	t.Run("absolute path within workspace", func(t *testing.T) {
		fs := newMockFileSystemForPath()

		abs, rel, err := NewResolver(workspaceRoot, fs).Resolve("/workspace/nested/file.txt")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if abs != "/workspace/nested/file.txt" {
			t.Errorf("expected absolute path /workspace/nested/file.txt, got %s", abs)
		}
		if rel != "nested/file.txt" {
			t.Errorf("expected relative path nested/file.txt, got %s", rel)
		}
	})

	t.Run("path outside workspace rejected", func(t *testing.T) {
		fs := newMockFileSystemForPath()

		_, _, err := NewResolver(workspaceRoot, fs).Resolve("/outside/file.txt")
		if err != ErrOutsideWorkspace {
			t.Errorf("expected ErrOutsideWorkspace, got %v", err)
		}
	})

	t.Run("path traversal rejected", func(t *testing.T) {
		fs := newMockFileSystemForPath()

		_, _, err := NewResolver(workspaceRoot, fs).Resolve("../outside/file.txt")
		if err != ErrOutsideWorkspace {
			t.Errorf("expected ErrOutsideWorkspace, got %v", err)
		}
	})

	t.Run(".. within workspace allowed", func(t *testing.T) {
		fs := newMockFileSystemForPath()
		fs.createDir("/workspace/nested")
		fs.createFile("/workspace/file.txt", []byte("content"), 0644)

		abs, rel, err := NewResolver(workspaceRoot, fs).Resolve("nested/../file.txt")
		if err != nil {
			t.Fatalf("unexpected error for .. within workspace: %v", err)
		}

		if abs != "/workspace/file.txt" {
			t.Errorf("expected /workspace/file.txt, got %s", abs)
		}
		if rel != "file.txt" {
			t.Errorf("expected file.txt, got %s", rel)
		}
	})

	// Symlink tests
	t.Run("symlink inside workspace", func(t *testing.T) {
		fs := newMockFileSystemForPath()
		fs.createFile("/workspace/target.txt", []byte("content"), 0644)
		fs.createSymlink("/workspace/link.txt", "/workspace/target.txt")

		abs, rel, err := NewResolver(workspaceRoot, fs).Resolve("link.txt")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if abs != "/workspace/target.txt" {
			t.Errorf("expected /workspace/target.txt, got %s", abs)
		}
		if rel != "target.txt" {
			t.Errorf("expected target.txt, got %s", rel)
		}
	})

	t.Run("symlink escaping workspace rejected", func(t *testing.T) {
		fs := newMockFileSystemForPath()
		fs.createFile("/tmp/outside.txt", []byte("content"), 0644)
		fs.createSymlink("/workspace/link.txt", "/tmp/outside.txt")

		_, _, err := NewResolver(workspaceRoot, fs).Resolve("link.txt")
		if err != ErrOutsideWorkspace {
			t.Errorf("expected ErrOutsideWorkspace for escaping symlink, got %v", err)
		}
	})

	t.Run("symlink chain inside workspace", func(t *testing.T) {
		fs := newMockFileSystemForPath()
		fs.createFile("/workspace/target.txt", []byte("content"), 0644)
		fs.createSymlink("/workspace/link1.txt", "/workspace/link2.txt")
		fs.createSymlink("/workspace/link2.txt", "/workspace/target.txt")

		abs, _, err := NewResolver(workspaceRoot, fs).Resolve("link1.txt")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if abs != "/workspace/target.txt" {
			t.Errorf("expected /workspace/target.txt, got %s", abs)
		}
	})

	t.Run("symlink chain escaping workspace rejected", func(t *testing.T) {
		fs := newMockFileSystemForPath()
		fs.createFile("/tmp/outside.txt", []byte("content"), 0644)
		fs.createSymlink("/workspace/link1.txt", "/workspace/link2.txt")
		fs.createSymlink("/workspace/link2.txt", "/tmp/outside.txt")

		_, _, err := NewResolver(workspaceRoot, fs).Resolve("link1.txt")
		if err != ErrOutsideWorkspace {
			t.Errorf("expected ErrOutsideWorkspace for escaping symlink chain, got %v", err)
		}
	})

	t.Run(".. escaping workspace rejected", func(t *testing.T) {
		fs := newMockFileSystemForPath()

		_, _, err := NewResolver(workspaceRoot, fs).Resolve("../outside/file.txt")
		if err != ErrOutsideWorkspace {
			t.Errorf("expected ErrOutsideWorkspace for .. escaping workspace, got %v", err)
		}
	})

	t.Run(".. at workspace root rejected", func(t *testing.T) {
		fs := newMockFileSystemForPath()

		_, _, err := NewResolver(workspaceRoot, fs).Resolve("..")
		if err != ErrOutsideWorkspace {
			t.Errorf("expected ErrOutsideWorkspace for .. at workspace root, got %v", err)
		}
	})

	t.Run("symlink target with .. within workspace allowed", func(t *testing.T) {
		fs := newMockFileSystemForPath()
		fs.createDir("/workspace/nested")
		fs.createFile("/workspace/file.txt", []byte("content"), 0644)
		fs.createSymlink("/workspace/link", "/workspace/nested/../file.txt")

		abs, rel, err := NewResolver(workspaceRoot, fs).Resolve("link")
		if err != nil {
			t.Fatalf("unexpected error for symlink with .. in target: %v", err)
		}

		if abs != "/workspace/file.txt" {
			t.Errorf("expected absolute path /workspace/file.txt, got %s", abs)
		}
		if rel != "file.txt" {
			t.Errorf("expected relative path file.txt, got %s", rel)
		}
	})

	t.Run("symlink target with .. escaping workspace rejected", func(t *testing.T) {
		fs := newMockFileSystemForPath()
		fs.createSymlink("/workspace/link", "/workspace/../outside/file.txt")

		_, _, err := NewResolver(workspaceRoot, fs).Resolve("link")
		if err != ErrOutsideWorkspace {
			t.Errorf("expected ErrOutsideWorkspace for symlink with .. escaping workspace, got %v", err)
		}
	})

	t.Run("symlink chain", func(t *testing.T) {
		fs := newMockFileSystemForPath()
		fs.createSymlink("/workspace/link1", "/workspace/link2")
		fs.createSymlink("/workspace/link2", "/workspace/target.txt")
		fs.createFile("/workspace/target.txt", []byte("target"), 0644)

		abs, _, err := NewResolver(workspaceRoot, fs).Resolve("link1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if abs != "/workspace/target.txt" {
			t.Errorf("expected absolute path /workspace/target.txt, got %s", abs)
		}
	})

	t.Run("filename with dots allowed", func(t *testing.T) {
		fs := newMockFileSystemForPath()

		abs, rel, err := NewResolver(workspaceRoot, fs).Resolve("file..txt")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if abs != "/workspace/file..txt" {
			t.Errorf("expected absolute path /workspace/file..txt, got %s", abs)
		}
		if rel != "file..txt" {
			t.Errorf("expected relative path file..txt, got %s", rel)
		}
	})

	t.Run("empty workspace root error", func(t *testing.T) {
		fs := newMockFileSystemForPath()

		_, _, err := NewResolver("", fs).Resolve("test.txt")
		if !errors.Is(err, ErrWorkspaceRootNotSet) {
			t.Errorf("expected ErrWorkspaceRootNotSet, got %v", err)
		}
	})
}

func TestResolveSymlinkEscapePrevention(t *testing.T) {
	workspaceRoot := "/workspace"

	t.Run("symlink directory escape rejected", func(t *testing.T) {
		fs := newMockFileSystemForPath()
		fs.createSymlink("/workspace/link", "/outside")
		fs.createDir("/outside")

		_, _, err := NewResolver(workspaceRoot, fs).Resolve("link/escape.txt")
		if err != ErrOutsideWorkspace {
			t.Errorf("expected ErrOutsideWorkspace for symlink directory escape, got %v", err)
		}
	})

	t.Run("symlink directory inside workspace allowed", func(t *testing.T) {
		fs := newMockFileSystemForPath()
		fs.createSymlink("/workspace/link", "/workspace/target")
		fs.createDir("/workspace/target")
		fs.createFile("/workspace/target/file.txt", []byte("content"), 0644)

		abs, rel, err := NewResolver(workspaceRoot, fs).Resolve("link/file.txt")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if abs != "/workspace/target/file.txt" {
			t.Errorf("expected absolute path /workspace/target/file.txt, got %s", abs)
		}
		if rel != "target/file.txt" {
			t.Errorf("expected relative path target/file.txt, got %s", rel)
		}
	})

	t.Run("nested symlink escape rejected", func(t *testing.T) {
		fs := newMockFileSystemForPath()
		fs.createDir("/workspace/nested")
		fs.createSymlink("/workspace/nested/link", "/outside")
		fs.createDir("/outside")

		_, _, err := NewResolver(workspaceRoot, fs).Resolve("nested/link/escape.txt")
		if err != ErrOutsideWorkspace {
			t.Errorf("expected ErrOutsideWorkspace for nested symlink escape, got %v", err)
		}
	})
}

func TestResolveMissingDirectories(t *testing.T) {
	workspaceRoot := "/workspace"

	t.Run("missing intermediate directories resolve successfully", func(t *testing.T) {
		fs := newMockFileSystemForPath()

		abs, rel, err := NewResolver(workspaceRoot, fs).Resolve("nested/new/file.txt")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expectedAbs := "/workspace/nested/new/file.txt"
		if abs != expectedAbs {
			t.Errorf("expected absolute path %s, got %s", expectedAbs, abs)
		}
		if rel != "nested/new/file.txt" {
			t.Errorf("expected relative path nested/new/file.txt, got %s", rel)
		}
	})

	t.Run("missing directories with symlink parent", func(t *testing.T) {
		fs := newMockFileSystemForPath()
		fs.createSymlink("/workspace/link", "/workspace/target")
		fs.createDir("/workspace/target")

		abs, rel, err := NewResolver(workspaceRoot, fs).Resolve("link/missing/sub/file.txt")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expectedAbs := "/workspace/target/missing/sub/file.txt"
		if abs != expectedAbs {
			t.Errorf("expected absolute path %s, got %s", expectedAbs, abs)
		}
		if rel != "target/missing/sub/file.txt" {
			t.Errorf("expected relative path target/missing/sub/file.txt, got %s", rel)
		}
	})
}

func TestResolveSymlinkChains(t *testing.T) {
	workspaceRoot := "/workspace"

	t.Run("symlink chain entirely inside workspace", func(t *testing.T) {
		fs := newMockFileSystemForPath()
		fs.createSymlink("/workspace/link1", "/workspace/link2")
		fs.createSymlink("/workspace/link2", "/workspace/target.txt")
		fs.createFile("/workspace/target.txt", []byte("target"), 0644)

		abs, rel, err := NewResolver(workspaceRoot, fs).Resolve("link1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if abs != "/workspace/target.txt" {
			t.Errorf("expected absolute path /workspace/target.txt, got %s", abs)
		}
		if rel != "target.txt" {
			t.Errorf("expected relative path target.txt, got %s", rel)
		}
	})

	t.Run("symlink chain escaping workspace at first hop", func(t *testing.T) {
		fs := newMockFileSystemForPath()
		fs.createSymlink("/workspace/link1", "/tmp/outside")
		fs.createDir("/tmp/outside")

		_, _, err := NewResolver(workspaceRoot, fs).Resolve("link1")
		if err != ErrOutsideWorkspace {
			t.Errorf("expected ErrOutsideWorkspace for escaping chain, got %v", err)
		}
	})

	t.Run("symlink chain escaping workspace at second hop", func(t *testing.T) {
		fs := newMockFileSystemForPath()
		fs.createSymlink("/workspace/link1", "/workspace/link2")
		fs.createSymlink("/workspace/link2", "/tmp/outside")
		fs.createDir("/tmp/outside")

		_, _, err := NewResolver(workspaceRoot, fs).Resolve("link1")
		if err != ErrOutsideWorkspace {
			t.Errorf("expected ErrOutsideWorkspace for escaping chain at second hop, got %v", err)
		}
	})

	t.Run("symlink loop detection", func(t *testing.T) {
		fs := newMockFileSystemForPath()
		fs.createSymlink("/workspace/loop1", "/workspace/loop2")
		fs.createSymlink("/workspace/loop2", "/workspace/loop1")

		_, _, err := NewResolver(workspaceRoot, fs).Resolve("loop1")
		if !errors.Is(err, ErrSymlinkLoop) {
			t.Errorf("expected ErrSymlinkLoop, got %v", err)
		}
	})

	t.Run("dangling symlink pointing inside workspace", func(t *testing.T) {
		fs := newMockFileSystemForPath()
		fs.createSymlink("/workspace/dangling", "/workspace/nonexistent/file.txt")

		abs, rel, err := NewResolver(workspaceRoot, fs).Resolve("dangling")
		if err != nil {
			t.Fatalf("unexpected error for dangling symlink inside workspace: %v", err)
		}

		expectedAbs := "/workspace/nonexistent/file.txt"
		if abs != expectedAbs {
			t.Errorf("expected absolute path %s, got %s", expectedAbs, abs)
		}
		if rel != "nonexistent/file.txt" {
			t.Errorf("expected relative path nonexistent/file.txt, got %s", rel)
		}
	})

	t.Run("dangling symlink pointing outside workspace", func(t *testing.T) {
		fs := newMockFileSystemForPath()
		fs.createSymlink("/workspace/dangling", "/tmp/outside/file.txt")

		_, _, err := NewResolver(workspaceRoot, fs).Resolve("dangling")
		if err != ErrOutsideWorkspace {
			t.Errorf("expected ErrOutsideWorkspace for dangling symlink outside workspace, got %v", err)
		}
	})

	t.Run("symlink chain exceeding max hops limit (65 hops)", func(t *testing.T) {
		fs := newMockFileSystemForPath()
		const maxHops = 64
		const chainLength = maxHops + 1 // 65 hops

		fs.createFile("/workspace/target.txt", []byte("target"), 0644)

		for i := range chainLength {
			var target string
			if i == chainLength-1 {
				target = "/workspace/target.txt"
			} else {
				target = fmt.Sprintf("/workspace/link%d", i+1)
			}
			fs.createSymlink(fmt.Sprintf("/workspace/link%d", i), target)
		}

		_, _, err := NewResolver(workspaceRoot, fs).Resolve("link0")
		if !errors.Is(err, ErrSymlinkChainTooLong) {
			t.Errorf("expected ErrSymlinkChainTooLong, got %v", err)
		}
	})

	t.Run("symlink chain at max hops limit (64 hops)", func(t *testing.T) {
		fs := newMockFileSystemForPath()
		const maxHops = 64
		const chainLength = maxHops // 64 hops

		fs.createFile("/workspace/target.txt", []byte("target"), 0644)

		for i := range chainLength {
			var target string
			if i == chainLength-1 {
				target = "/workspace/target.txt"
			} else {
				target = fmt.Sprintf("/workspace/link%d", i+1)
			}
			fs.createSymlink(fmt.Sprintf("/workspace/link%d", i), target)
		}

		abs, rel, err := NewResolver(workspaceRoot, fs).Resolve("link0")
		if err != nil {
			t.Fatalf("unexpected error for symlink chain at max hops: %v", err)
		}

		if abs != "/workspace/target.txt" {
			t.Errorf("expected absolute path /workspace/target.txt, got %s", abs)
		}
		if rel != "target.txt" {
			t.Errorf("expected relative path target.txt, got %s", rel)
		}
	})
}

func TestResolveTildeExpansion(t *testing.T) {
	workspaceRoot := "/workspace"

	t.Run("tilde expansion outside workspace rejected", func(t *testing.T) {
		fs := newMockFileSystemForPath()

		// MockFileSystem.UserHomeDir returns "/home/user"
		// Since /home/user/file.txt is outside /workspace, this should fail
		_, _, err := NewResolver(workspaceRoot, fs).Resolve("~/file.txt")
		if err != ErrOutsideWorkspace {
			t.Errorf("expected ErrOutsideWorkspace for path outside workspace after tilde expansion, got %v", err)
		}
	})
}

func TestResolveAbsoluteVsRelative(t *testing.T) {
	workspaceRoot := "/workspace"

	t.Run("absolute path within workspace", func(t *testing.T) {
		fs := newMockFileSystemForPath()

		abs, rel, err := NewResolver(workspaceRoot, fs).Resolve("/workspace/nested/file.txt")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if abs != "/workspace/nested/file.txt" {
			t.Errorf("expected absolute path /workspace/nested/file.txt, got %s", abs)
		}
		if rel != "nested/file.txt" {
			t.Errorf("expected relative path nested/file.txt, got %s", rel)
		}
	})

	t.Run("relative path resolves correctly", func(t *testing.T) {
		fs := newMockFileSystemForPath()

		abs, rel, err := NewResolver(workspaceRoot, fs).Resolve("nested/file.txt")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if abs != "/workspace/nested/file.txt" {
			t.Errorf("expected absolute path /workspace/nested/file.txt, got %s", abs)
		}
		if rel != "nested/file.txt" {
			t.Errorf("expected relative path nested/file.txt, got %s", rel)
		}
	})
}
