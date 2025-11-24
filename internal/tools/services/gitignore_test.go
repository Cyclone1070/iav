package services

import (
	"testing"
)

func TestLoadGitignore(t *testing.T) {
	maxFileSize := int64(1024 * 1024)
	workspaceRoot := "/workspace"

	t.Run("load gitignore from workspace root", func(t *testing.T) {
		fs := NewMockFileSystem(maxFileSize)
		fs.CreateDir("/workspace")
		fs.CreateFile("/workspace/.gitignore", []byte("*.log\n*.tmp\n"), 0644)

		service, err := NewGitignoreService(workspaceRoot, fs)
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
		fs := NewMockFileSystem(maxFileSize)
		fs.CreateDir("/workspace")

		service, err := NewGitignoreService(workspaceRoot, fs)
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
		fs := NewMockFileSystem(maxFileSize)
		fs.CreateDir("/workspace")
		fs.CreateFile("/workspace/.gitignore", []byte("*.log\n"), 0644)

		service, err := NewGitignoreService(workspaceRoot, fs)
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
