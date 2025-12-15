package gitignore

import (
	"os"
	"testing"
)

// mockFileSystem is a local mock implementing gitignore.FileSystem for testing
type mockFileSystem struct {
	files map[string][]byte
}

func newMockFileSystem() *mockFileSystem {
	return &mockFileSystem{
		files: make(map[string][]byte),
	}
}

func (m *mockFileSystem) createFile(path string, content []byte) {
	m.files[path] = content
}

func (m *mockFileSystem) Stat(path string) (os.FileInfo, error) {
	if _, ok := m.files[path]; ok {
		return nil, nil // File exists (we don't need full FileInfo for this test)
	}
	return nil, os.ErrNotExist
}

func (m *mockFileSystem) ReadFileRange(path string, offset, limit int64) ([]byte, error) {
	content, ok := m.files[path]
	if !ok {
		return nil, os.ErrNotExist
	}
	return content, nil
}

func TestLoadGitignore(t *testing.T) {
	workspaceRoot := "/workspace"

	t.Run("load gitignore from workspace root", func(t *testing.T) {
		fs := newMockFileSystem()
		fs.createFile("/workspace/.gitignore", []byte("*.log\n*.tmp\n"))

		service, err := NewService(workspaceRoot, fs)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if service == nil {
			t.Fatal("expected non-nil service")
		}

		// Test matching patterns
		if !service.ShouldIgnore("test.log") {
			t.Error("expected test.log to be ignored")
		}
		if !service.ShouldIgnore("file.tmp") {
			t.Error("expected file.tmp to be ignored")
		}
		if service.ShouldIgnore("test.txt") {
			t.Error("expected test.txt not to be ignored")
		}
	})

	t.Run("non-existent gitignore should not error", func(t *testing.T) {
		fs := newMockFileSystem()

		service, err := NewService(workspaceRoot, fs)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if service == nil {
			t.Fatal("expected non-nil service")
		}

		// Should not ignore anything if no .gitignore
		if service.ShouldIgnore("test.log") {
			t.Error("expected no files to be ignored without .gitignore")
		}
	})

	t.Run("dotfiles matching gitignore patterns", func(t *testing.T) {
		fs := newMockFileSystem()
		fs.createFile("/workspace/.gitignore", []byte("*.log\n"))

		service, err := NewService(workspaceRoot, fs)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Dotfile matching pattern should be ignored
		if !service.ShouldIgnore(".test.log") {
			t.Error("expected .test.log to be ignored")
		}

		// Dotfile not matching pattern should not be ignored
		if service.ShouldIgnore(".keep") {
			t.Error("expected .keep not to be ignored")
		}
	})
}
