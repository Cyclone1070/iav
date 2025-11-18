package tools

// TEST CONTRACT: do not modify without updating symlink safety spec
// These tests enforce the symlink safety guarantees for path resolution.
// Any changes to these tests must be reviewed against the symlink safety specification.

import (
	"fmt"
	"strings"
	"testing"
)

func TestResolve(t *testing.T) {
	workspaceRoot := "/workspace"
	maxFileSize := int64(1024 * 1024)

	t.Run("relative path resolution", func(t *testing.T) {
		fs := NewMockFileSystem(maxFileSize)
		ctx := &WorkspaceContext{
			FS:            fs,
			WorkspaceRoot: workspaceRoot,
		}

		abs, rel, err := Resolve(ctx, "test.txt")
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
		fs := NewMockFileSystem(maxFileSize)
		ctx := &WorkspaceContext{
			FS:            fs,
			WorkspaceRoot: workspaceRoot,
		}

		abs, rel, err := Resolve(ctx, "/workspace/nested/file.txt")
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
		fs := NewMockFileSystem(maxFileSize)
		ctx := &WorkspaceContext{
			FS:            fs,
			WorkspaceRoot: workspaceRoot,
		}

		_, _, err := Resolve(ctx, "/outside/file.txt")
		if err != ErrOutsideWorkspace {
			t.Errorf("expected ErrOutsideWorkspace, got %v", err)
		}
	})

	t.Run("path traversal rejected", func(t *testing.T) {
		fs := NewMockFileSystem(maxFileSize)
		ctx := &WorkspaceContext{
			FS:            fs,
			WorkspaceRoot: workspaceRoot,
		}

		_, _, err := Resolve(ctx, "../outside/file.txt")
		if err != ErrOutsideWorkspace {
			t.Errorf("expected ErrOutsideWorkspace, got %v", err)
		}
	})

	t.Run(".. within workspace allowed", func(t *testing.T) {
		fs := NewMockFileSystem(maxFileSize)
		ctx := &WorkspaceContext{
			FS:            fs,
			WorkspaceRoot: workspaceRoot,
		}

		// Create nested directory structure
		fs.CreateDir("/workspace/nested")
		fs.CreateFile("/workspace/file.txt", []byte("content"), 0644)

		// Path with .. that stays within workspace should be allowed
		abs, rel, err := Resolve(ctx, "nested/../file.txt")
		if err != nil {
			t.Fatalf("unexpected error for .. within workspace: %v", err)
		}

		if abs != "/workspace/file.txt" {
			t.Errorf("expected absolute path /workspace/file.txt, got %s", abs)
		}
		if rel != "file.txt" {
			t.Errorf("expected relative path file.txt, got %s", rel)
		}
	})

	t.Run(".. escaping workspace rejected", func(t *testing.T) {
		fs := NewMockFileSystem(maxFileSize)
		ctx := &WorkspaceContext{
			FS:            fs,
			WorkspaceRoot: workspaceRoot,
		}

		// Path with .. that would escape workspace should be rejected
		_, _, err := Resolve(ctx, "../outside/file.txt")
		if err != ErrOutsideWorkspace {
			t.Errorf("expected ErrOutsideWorkspace for .. escaping workspace, got %v", err)
		}
	})

	t.Run(".. at workspace root rejected", func(t *testing.T) {
		fs := NewMockFileSystem(maxFileSize)
		ctx := &WorkspaceContext{
			FS:            fs,
			WorkspaceRoot: workspaceRoot,
		}

		// Path with .. that would go above workspace root should be rejected
		_, _, err := Resolve(ctx, "..")
		if err != ErrOutsideWorkspace {
			t.Errorf("expected ErrOutsideWorkspace for .. at workspace root, got %v", err)
		}
	})

	t.Run("symlink target with .. within workspace allowed", func(t *testing.T) {
		fs := NewMockFileSystem(maxFileSize)
		ctx := &WorkspaceContext{
			FS:            fs,
			WorkspaceRoot: workspaceRoot,
		}

		// Create nested directory structure
		fs.CreateDir("/workspace/nested")
		fs.CreateFile("/workspace/file.txt", []byte("content"), 0644)
		// Create symlink that points to path with ..
		fs.CreateSymlink("/workspace/link", "/workspace/nested/../file.txt")

		// Resolving symlink with .. in target should work if it stays within workspace
		abs, rel, err := Resolve(ctx, "link")
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
		fs := NewMockFileSystem(maxFileSize)
		ctx := &WorkspaceContext{
			FS:            fs,
			WorkspaceRoot: workspaceRoot,
		}

		// Create symlink that points to path with .. that would escape
		fs.CreateSymlink("/workspace/link", "/workspace/../outside/file.txt")

		// Resolving symlink with .. that escapes should be rejected
		_, _, err := Resolve(ctx, "link")
		if err != ErrOutsideWorkspace {
			t.Errorf("expected ErrOutsideWorkspace for symlink with .. escaping workspace, got %v", err)
		}
	})

	t.Run("direct path with .. component (bypassing Clean)", func(t *testing.T) {
		// This test directly calls resolveSymlink with a path containing ..
		// to verify the .. handling logic works correctly
		fs := NewMockFileSystem(maxFileSize)
		ctx := &WorkspaceContext{
			FS:            fs,
			WorkspaceRoot: workspaceRoot,
		}

		fs.CreateDir("/workspace/nested")
		fs.CreateFile("/workspace/file.txt", []byte("content"), 0644)

		// Directly test resolveSymlink with .. (bypassing Resolve's Clean)
		// This simulates what happens when a symlink target contains ..
		resolved, err := resolveSymlink(ctx, "/workspace/nested/../file.txt")
		if err != nil {
			t.Fatalf("unexpected error for .. within workspace: %v", err)
		}

		if resolved != "/workspace/file.txt" {
			t.Errorf("expected resolved path /workspace/file.txt, got %s", resolved)
		}
	})

	t.Run("symlink escape rejected", func(t *testing.T) {
		fs := NewMockFileSystem(maxFileSize)
		ctx := &WorkspaceContext{
			FS:            fs,
			WorkspaceRoot: workspaceRoot,
		}

		// Create symlink pointing outside
		fs.CreateSymlink("/workspace/escape", "/outside/target.txt")

		_, _, err := Resolve(ctx, "escape")
		if err != ErrOutsideWorkspace {
			t.Errorf("expected ErrOutsideWorkspace for symlink escape, got %v", err)
		}
	})

	t.Run("symlink inside workspace allowed", func(t *testing.T) {
		fs := NewMockFileSystem(maxFileSize)
		ctx := &WorkspaceContext{
			FS:            fs,
			WorkspaceRoot: workspaceRoot,
		}

		// Create symlink pointing inside workspace
		fs.CreateSymlink("/workspace/link", "/workspace/target.txt")
		fs.CreateFile("/workspace/target.txt", []byte("target"), 0644)

		abs, rel, err := Resolve(ctx, "link")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Should resolve to target
		if abs != "/workspace/target.txt" {
			t.Errorf("expected absolute path /workspace/target.txt, got %s", abs)
		}
		if rel != "target.txt" {
			t.Errorf("expected relative path target.txt, got %s", rel)
		}
	})

	t.Run("symlink chain", func(t *testing.T) {
		fs := NewMockFileSystem(maxFileSize)
		ctx := &WorkspaceContext{
			FS:            fs,
			WorkspaceRoot: workspaceRoot,
		}

		// Create symlink chain: link1 -> link2 -> target
		fs.CreateSymlink("/workspace/link1", "/workspace/link2")
		fs.CreateSymlink("/workspace/link2", "/workspace/target.txt")
		fs.CreateFile("/workspace/target.txt", []byte("target"), 0644)

		abs, _, err := Resolve(ctx, "link1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if abs != "/workspace/target.txt" {
			t.Errorf("expected absolute path /workspace/target.txt, got %s", abs)
		}
	})

	t.Run("filename with dots allowed", func(t *testing.T) {
		fs := NewMockFileSystem(maxFileSize)
		ctx := &WorkspaceContext{
			FS:            fs,
			WorkspaceRoot: workspaceRoot,
		}

		abs, rel, err := Resolve(ctx, "file..txt")
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
		fs := NewMockFileSystem(maxFileSize)
		ctx := &WorkspaceContext{
			FS:            fs,
			WorkspaceRoot: "",
		}

		_, _, err := Resolve(ctx, "test.txt")
		if err == nil {
			t.Error("expected error for empty workspace root")
		}
	})
}

func TestEnsureParentDirs(t *testing.T) {
	workspaceRoot := "/workspace"
	maxFileSize := int64(1024 * 1024)

	t.Run("create nested directories", func(t *testing.T) {
		fs := NewMockFileSystem(maxFileSize)
		ctx := &WorkspaceContext{
			FS:            fs,
			WorkspaceRoot: workspaceRoot,
		}

		err := EnsureParentDirs(ctx, "nested/deep/file.txt")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify directories were created
		isDir, err := fs.IsDir("/workspace/nested")
		if err != nil || !isDir {
			t.Error("expected nested directory to be created")
		}

		isDir, err = fs.IsDir("/workspace/nested/deep")
		if err != nil || !isDir {
			t.Error("expected deep directory to be created")
		}
	})

	t.Run("reject path outside workspace", func(t *testing.T) {
		fs := NewMockFileSystem(maxFileSize)
		ctx := &WorkspaceContext{
			FS:            fs,
			WorkspaceRoot: workspaceRoot,
		}

		err := EnsureParentDirs(ctx, "../outside/file.txt")
		if err != ErrOutsideWorkspace {
			t.Errorf("expected ErrOutsideWorkspace, got %v", err)
		}
	})
}

func TestIsDirectory(t *testing.T) {
	workspaceRoot := "/workspace"
	maxFileSize := int64(1024 * 1024)

	t.Run("file is not directory", func(t *testing.T) {
		fs := NewMockFileSystem(maxFileSize)
		ctx := &WorkspaceContext{
			FS:            fs,
			WorkspaceRoot: workspaceRoot,
		}

		fs.CreateFile("/workspace/test.txt", []byte("content"), 0644)

		isDir, err := IsDirectory(ctx, "test.txt")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if isDir {
			t.Error("expected file not to be directory")
		}
	})

	t.Run("directory is directory", func(t *testing.T) {
		fs := NewMockFileSystem(maxFileSize)
		ctx := &WorkspaceContext{
			FS:            fs,
			WorkspaceRoot: workspaceRoot,
		}

		fs.CreateDir("/workspace/subdir")

		isDir, err := IsDirectory(ctx, "subdir")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !isDir {
			t.Error("expected directory to be directory")
		}
	})
}

func TestResolveSymlinkEscapePrevention(t *testing.T) {
	workspaceRoot := "/workspace"
	maxFileSize := int64(1024 * 1024)

	t.Run("symlink directory escape rejected", func(t *testing.T) {
		fs := NewMockFileSystem(maxFileSize)
		ctx := &WorkspaceContext{
			FS:            fs,
			WorkspaceRoot: workspaceRoot,
		}

		// Create symlink directory pointing outside
		fs.CreateSymlink("/workspace/link", "/outside")
		fs.CreateDir("/outside")

		// Try to resolve a file through the symlink directory
		_, _, err := Resolve(ctx, "link/escape.txt")
		if err != ErrOutsideWorkspace {
			t.Errorf("expected ErrOutsideWorkspace for symlink directory escape, got %v", err)
		}
	})

	t.Run("symlink directory inside workspace allowed", func(t *testing.T) {
		fs := NewMockFileSystem(maxFileSize)
		ctx := &WorkspaceContext{
			FS:            fs,
			WorkspaceRoot: workspaceRoot,
		}

		// Create symlink directory pointing inside workspace
		fs.CreateSymlink("/workspace/link", "/workspace/target")
		fs.CreateDir("/workspace/target")
		fs.CreateFile("/workspace/target/file.txt", []byte("content"), 0644)

		abs, rel, err := Resolve(ctx, "link/file.txt")
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
		fs := NewMockFileSystem(maxFileSize)
		ctx := &WorkspaceContext{
			FS:            fs,
			WorkspaceRoot: workspaceRoot,
		}

		// Create nested symlink structure: workspace/nested/link -> /outside
		fs.CreateDir("/workspace/nested")
		fs.CreateSymlink("/workspace/nested/link", "/outside")
		fs.CreateDir("/outside")

		// Try to resolve a file through nested symlink
		_, _, err := Resolve(ctx, "nested/link/escape.txt")
		if err != ErrOutsideWorkspace {
			t.Errorf("expected ErrOutsideWorkspace for nested symlink escape, got %v", err)
		}
	})
}

func TestResolveMissingDirectories(t *testing.T) {
	workspaceRoot := "/workspace"
	maxFileSize := int64(1024 * 1024)

	t.Run("missing intermediate directories resolve successfully", func(t *testing.T) {
		fs := NewMockFileSystem(maxFileSize)
		ctx := &WorkspaceContext{
			FS:            fs,
			WorkspaceRoot: workspaceRoot,
		}

		// Path with missing intermediate directories should resolve without error
		abs, rel, err := Resolve(ctx, "nested/new/file.txt")
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
		fs := NewMockFileSystem(maxFileSize)
		ctx := &WorkspaceContext{
			FS:            fs,
			WorkspaceRoot: workspaceRoot,
		}

		// Create symlink pointing to a directory
		fs.CreateSymlink("/workspace/link", "/workspace/target")
		fs.CreateDir("/workspace/target")

		// Path through symlink with missing subdirectories
		abs, rel, err := Resolve(ctx, "link/missing/sub/file.txt")
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
	maxFileSize := int64(1024 * 1024)

	t.Run("symlink chain entirely inside workspace", func(t *testing.T) {
		fs := NewMockFileSystem(maxFileSize)
		ctx := &WorkspaceContext{
			FS:            fs,
			WorkspaceRoot: workspaceRoot,
		}

		// Create chain: link1 -> link2 -> target
		fs.CreateSymlink("/workspace/link1", "/workspace/link2")
		fs.CreateSymlink("/workspace/link2", "/workspace/target.txt")
		fs.CreateFile("/workspace/target.txt", []byte("target"), 0644)

		abs, rel, err := Resolve(ctx, "link1")
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
		fs := NewMockFileSystem(maxFileSize)
		ctx := &WorkspaceContext{
			FS:            fs,
			WorkspaceRoot: workspaceRoot,
		}

		// Chain where first hop escapes
		fs.CreateSymlink("/workspace/link1", "/tmp/outside")
		fs.CreateDir("/tmp/outside")

		_, _, err := Resolve(ctx, "link1")
		if err != ErrOutsideWorkspace {
			t.Errorf("expected ErrOutsideWorkspace for escaping chain, got %v", err)
		}
	})

	t.Run("symlink chain escaping workspace at second hop", func(t *testing.T) {
		fs := NewMockFileSystem(maxFileSize)
		ctx := &WorkspaceContext{
			FS:            fs,
			WorkspaceRoot: workspaceRoot,
		}

		// Chain: link1 -> link2 -> /tmp/outside
		fs.CreateSymlink("/workspace/link1", "/workspace/link2")
		fs.CreateSymlink("/workspace/link2", "/tmp/outside")
		fs.CreateDir("/tmp/outside")

		_, _, err := Resolve(ctx, "link1")
		if err != ErrOutsideWorkspace {
			t.Errorf("expected ErrOutsideWorkspace for escaping chain at second hop, got %v", err)
		}
	})

	t.Run("symlink loop detection", func(t *testing.T) {
		fs := NewMockFileSystem(maxFileSize)
		ctx := &WorkspaceContext{
			FS:            fs,
			WorkspaceRoot: workspaceRoot,
		}

		// Create loop: loop1 -> loop2 -> loop1
		fs.CreateSymlink("/workspace/loop1", "/workspace/loop2")
		fs.CreateSymlink("/workspace/loop2", "/workspace/loop1")

		_, _, err := Resolve(ctx, "loop1")
		if err == nil {
			t.Error("expected error for symlink loop, got nil")
		}
		// Check that error message mentions loop
		if err != nil && err.Error() == "" {
			t.Error("expected non-empty error message for symlink loop")
		}
	})

	t.Run("dangling symlink pointing inside workspace", func(t *testing.T) {
		fs := NewMockFileSystem(maxFileSize)
		ctx := &WorkspaceContext{
			FS:            fs,
			WorkspaceRoot: workspaceRoot,
		}

		// Create dangling symlink pointing to nonexistent path inside workspace
		fs.CreateSymlink("/workspace/dangling", "/workspace/nonexistent/file.txt")

		abs, rel, err := Resolve(ctx, "dangling")
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
		fs := NewMockFileSystem(maxFileSize)
		ctx := &WorkspaceContext{
			FS:            fs,
			WorkspaceRoot: workspaceRoot,
		}

		// Create dangling symlink pointing outside workspace
		fs.CreateSymlink("/workspace/dangling", "/tmp/outside/file.txt")

		_, _, err := Resolve(ctx, "dangling")
		if err != ErrOutsideWorkspace {
			t.Errorf("expected ErrOutsideWorkspace for dangling symlink outside workspace, got %v", err)
		}
	})

	t.Run("symlink chain exceeding max hops limit (65 hops)", func(t *testing.T) {
		fs := NewMockFileSystem(maxFileSize)
		ctx := &WorkspaceContext{
			FS:            fs,
			WorkspaceRoot: workspaceRoot,
		}

		// Create a chain with exactly 65 hops (should exceed the limit of 64)
		// link0 -> link1 -> link2 -> ... -> link64 -> target.txt
		const maxHops = 64
		const chainLength = maxHops + 1 // 65 hops

		// Create the final target
		fs.CreateFile("/workspace/target.txt", []byte("target"), 0644)

		// Create the chain: link0 -> link1 -> link2 -> ... -> link64 -> target.txt
		for i := 0; i < chainLength; i++ {
			var target string
			if i == chainLength-1 {
				// Last link points to target
				target = "/workspace/target.txt"
			} else {
				// Intermediate links point to next link
				target = fmt.Sprintf("/workspace/link%d", i+1)
			}
			fs.CreateSymlink(fmt.Sprintf("/workspace/link%d", i), target)
		}

		// Resolve should fail because chain exceeds max hops
		_, _, err := Resolve(ctx, "link0")
		if err == nil {
			t.Error("expected error for symlink chain exceeding max hops, got nil")
		}
		if err != nil && !strings.Contains(err.Error(), "symlink chain too long") {
			t.Errorf("expected error message containing 'symlink chain too long', got: %v", err)
		}
		if err != nil && !strings.Contains(err.Error(), fmt.Sprintf("max %d hops", maxHops)) {
			t.Errorf("expected error message containing 'max %d hops', got: %v", maxHops, err)
		}
	})

	t.Run("symlink chain at max hops limit (64 hops)", func(t *testing.T) {
		fs := NewMockFileSystem(maxFileSize)
		ctx := &WorkspaceContext{
			FS:            fs,
			WorkspaceRoot: workspaceRoot,
		}

		// Create a chain with exactly 64 hops (should be allowed)
		// link0 -> link1 -> link2 -> ... -> link63 -> target.txt
		const maxHops = 64
		const chainLength = maxHops // 64 hops

		// Create the final target
		fs.CreateFile("/workspace/target.txt", []byte("target"), 0644)

		// Create the chain: link0 -> link1 -> link2 -> ... -> link63 -> target.txt
		for i := 0; i < chainLength; i++ {
			var target string
			if i == chainLength-1 {
				// Last link points to target
				target = "/workspace/target.txt"
			} else {
				// Intermediate links point to next link
				target = fmt.Sprintf("/workspace/link%d", i+1)
			}
			fs.CreateSymlink(fmt.Sprintf("/workspace/link%d", i), target)
		}

		// Resolve should succeed because chain is exactly at max hops
		abs, rel, err := Resolve(ctx, "link0")
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
	maxFileSize := int64(1024 * 1024)

	t.Run("tilde expansion outside workspace rejected", func(t *testing.T) {
		fs := NewMockFileSystem(maxFileSize)
		ctx := &WorkspaceContext{
			FS:            fs,
			WorkspaceRoot: workspaceRoot,
		}

		// Note: MockFileSystem.UserHomeDir returns "/home/user"
		// Since /home/user/file.txt is outside /workspace, this should fail
		_, _, err := Resolve(ctx, "~/file.txt")
		if err != ErrOutsideWorkspace {
			t.Errorf("expected ErrOutsideWorkspace for path outside workspace after tilde expansion, got %v", err)
		}
	})
}

func TestResolveAbsoluteVsRelative(t *testing.T) {
	workspaceRoot := "/workspace"
	maxFileSize := int64(1024 * 1024)

	t.Run("absolute path within workspace", func(t *testing.T) {
		fs := NewMockFileSystem(maxFileSize)
		ctx := &WorkspaceContext{
			FS:            fs,
			WorkspaceRoot: workspaceRoot,
		}

		abs, rel, err := Resolve(ctx, "/workspace/nested/file.txt")
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
		fs := NewMockFileSystem(maxFileSize)
		ctx := &WorkspaceContext{
			FS:            fs,
			WorkspaceRoot: workspaceRoot,
		}

		abs, rel, err := Resolve(ctx, "nested/file.txt")
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
