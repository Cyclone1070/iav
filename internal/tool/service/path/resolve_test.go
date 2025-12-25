package path

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestAbs(t *testing.T) {
	workspaceRoot := "/workspace"
	resolver := NewResolver(workspaceRoot)

	tests := []struct {
		name     string
		input    string
		expected string
		err      error
	}{
		{
			name:     "relative path within workspace",
			input:    "src/main.go",
			expected: "/workspace/src/main.go",
			err:      nil,
		},
		{
			name:     "absolute path within workspace",
			input:    "/workspace/src/main.go",
			expected: "/workspace/src/main.go",
			err:      nil,
		},
		{
			name:     "path with dots within workspace",
			input:    "src/../src/main.go",
			expected: "/workspace/src/main.go",
			err:      nil,
		},
		{
			name:     "workspace root",
			input:    ".",
			expected: "/workspace",
			err:      nil,
		},
		{
			name:     "absolute workspace root",
			input:    "/workspace",
			expected: "/workspace",
			err:      nil,
		},
		{
			name:     "escape attempt via parent dots",
			input:    "../../../etc/passwd",
			expected: "",
			err:      ErrOutsideWorkspace,
		},
		{
			name:     "absolute path outside workspace",
			input:    "/etc/passwd",
			expected: "",
			err:      ErrOutsideWorkspace,
		},
		{
			name:     "prefix match but not child",
			input:    "/workspacefoo/bar",
			expected: "",
			err:      ErrOutsideWorkspace,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			abs, err := resolver.Abs(tt.input)
			if !errors.Is(err, tt.err) {
				t.Fatalf("expected error %v, got %v", tt.err, err)
			}
			if abs != tt.expected {
				t.Errorf("expected abs %q, got %q", tt.expected, abs)
			}
		})
	}
}

func TestRel(t *testing.T) {
	workspaceRoot := "/workspace"
	resolver := NewResolver(workspaceRoot)

	tests := []struct {
		name     string
		input    string
		expected string
		err      error
	}{
		{
			name:     "relative path within workspace",
			input:    "src/main.go",
			expected: "src/main.go",
			err:      nil,
		},
		{
			name:     "absolute path within workspace",
			input:    "/workspace/src/main.go",
			expected: "src/main.go",
			err:      nil,
		},
		{
			name:     "workspace root",
			input:    "/workspace",
			expected: "",
			err:      nil,
		},
		{
			name:     "escape attempt",
			input:    "/etc/passwd",
			expected: "",
			err:      ErrOutsideWorkspace,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rel, err := resolver.Rel(tt.input)
			if !errors.Is(err, tt.err) {
				t.Fatalf("expected error %v, got %v", tt.err, err)
			}
			if rel != tt.expected {
				t.Errorf("expected rel %q, got %q", tt.expected, rel)
			}
		})
	}
}

func TestCanonicaliseRoot(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pathutil-test")
	if err != nil {
		t.Fatalf("failed to create tmp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	resolvedTmpDir, err := filepath.EvalSymlinks(tmpDir)
	if err != nil {
		t.Fatalf("failed to resolve tmp dir: %v", err)
	}

	t.Run("valid directory", func(t *testing.T) {
		got, err := CanonicaliseRoot(resolvedTmpDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != resolvedTmpDir {
			t.Errorf("expected %q, got %q", resolvedTmpDir, got)
		}
	})

	t.Run("non-existent path", func(t *testing.T) {
		_, err := CanonicaliseRoot(filepath.Join(resolvedTmpDir, "non-existent"))
		if err == nil {
			t.Fatal("expected error for non-existent path")
		}
	})

	t.Run("file instead of directory", func(t *testing.T) {
		tmpFile := filepath.Join(resolvedTmpDir, "file.txt")
		if err := os.WriteFile(tmpFile, []byte("test"), 0o644); err != nil {
			t.Fatalf("failed to create tmp file: %v", err)
		}
		_, err := CanonicaliseRoot(tmpFile)
		if err == nil {
			t.Fatal("expected error for file instead of directory")
		}
	})
}
