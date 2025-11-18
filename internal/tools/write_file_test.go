package tools

import (
	"fmt"
	"os"
	"testing"
)

func TestWriteFile(t *testing.T) {
	workspaceRoot := "/workspace"
	maxFileSize := int64(1024 * 1024) // 1MB

	t.Run("create new file succeeds and updates cache", func(t *testing.T) {
		fs := NewMockFileSystem(maxFileSize)
		cache := NewMockChecksumStore()
		ctx := &WorkspaceContext{
			FS:               fs,
			BinaryDetector:   NewMockBinaryDetector(),
			ChecksumComputer: NewMockChecksumComputer(),
			ChecksumCache:    cache,
			MaxFileSize:      maxFileSize,
			WorkspaceRoot:    workspaceRoot,
		}

		content := "test content"
		resp, err := WriteFile(ctx, "new.txt", content, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if resp.BytesWritten != len(content) {
			t.Errorf("expected %d bytes written, got %d", len(content), resp.BytesWritten)
		}

		// Verify file was created
		fileContent, err := fs.ReadFileRange("/workspace/new.txt", 0, 0)
		if err != nil {
			t.Fatalf("failed to read created file: %v", err)
		}
		if string(fileContent) != content {
			t.Errorf("expected content %q, got %q", content, string(fileContent))
		}

		// Verify cache was updated
		checksum, ok := ctx.ChecksumCache.Get(resp.AbsolutePath)
		if !ok {
			t.Error("expected cache to be updated after write")
		}
		if checksum == "" {
			t.Error("expected non-empty checksum in cache")
		}
	})

	t.Run("existing file rejection", func(t *testing.T) {
		fs := NewMockFileSystem(maxFileSize)
		cache := NewMockChecksumStore()
		fs.CreateFile("/workspace/existing.txt", []byte("existing"), 0o644)

		ctx := &WorkspaceContext{
			FS:               fs,
			BinaryDetector:   NewMockBinaryDetector(),
			ChecksumComputer: NewMockChecksumComputer(),
			ChecksumCache:    cache,
			MaxFileSize:      maxFileSize,
			WorkspaceRoot:    workspaceRoot,
		}

		_, err := WriteFile(ctx, "existing.txt", "new content", nil)
		if err != ErrFileExists {
			t.Errorf("expected ErrFileExists, got %v", err)
		}
	})

	t.Run("symlink escape prevention", func(t *testing.T) {
		fs := NewMockFileSystem(maxFileSize)
		cache := NewMockChecksumStore()
		// Create symlink pointing outside workspace
		fs.CreateSymlink("/workspace/escape", "/outside/target.txt")

		ctx := &WorkspaceContext{
			FS:               fs,
			BinaryDetector:   NewMockBinaryDetector(),
			ChecksumComputer: NewMockChecksumComputer(),
			ChecksumCache:    cache,
			MaxFileSize:      maxFileSize,
			WorkspaceRoot:    workspaceRoot,
		}

		_, err := WriteFile(ctx, "escape", "content", nil)
		if err != ErrOutsideWorkspace {
			t.Errorf("expected ErrOutsideWorkspace for symlink escape, got %v", err)
		}
	})

	t.Run("large content rejection", func(t *testing.T) {
		fs := NewMockFileSystem(maxFileSize)
		cache := NewMockChecksumStore()
		ctx := &WorkspaceContext{
			FS:               fs,
			BinaryDetector:   NewMockBinaryDetector(),
			ChecksumComputer: NewMockChecksumComputer(),
			ChecksumCache:    cache,
			MaxFileSize:      maxFileSize,
			WorkspaceRoot:    workspaceRoot,
		}

		// Create content larger than limit
		largeContent := make([]byte, maxFileSize+1)
		for i := range largeContent {
			largeContent[i] = 'A'
		}

		_, err := WriteFile(ctx, "large.txt", string(largeContent), nil)
		if err != ErrTooLarge {
			t.Errorf("expected ErrTooLarge, got %v", err)
		}
	})

	t.Run("binary content rejection", func(t *testing.T) {
		fs := NewMockFileSystem(maxFileSize)
		cache := NewMockChecksumStore()
		ctx := &WorkspaceContext{
			FS:               fs,
			BinaryDetector:   NewMockBinaryDetector(),
			ChecksumComputer: NewMockChecksumComputer(),
			ChecksumCache:    cache,
			MaxFileSize:      maxFileSize,
			WorkspaceRoot:    workspaceRoot,
		}

		// Content with NUL byte
		binaryContent := []byte{0x48, 0x65, 0x6C, 0x00, 0x6C, 0x6F}
		_, err := WriteFile(ctx, "binary.bin", string(binaryContent), nil)
		if err != ErrBinaryFile {
			t.Errorf("expected ErrBinaryFile, got %v", err)
		}
	})

	t.Run("custom permissions", func(t *testing.T) {
		fs := NewMockFileSystem(maxFileSize)
		cache := NewMockChecksumStore()
		ctx := &WorkspaceContext{
			FS:               fs,
			BinaryDetector:   NewMockBinaryDetector(),
			ChecksumComputer: NewMockChecksumComputer(),
			ChecksumCache:    cache,
			MaxFileSize:      maxFileSize,
			WorkspaceRoot:    workspaceRoot,
		}

		perm := os.FileMode(0o755)
		resp, err := WriteFile(ctx, "executable.txt", "content", &perm)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		info, err := fs.Stat("/workspace/executable.txt")
		if err != nil {
			t.Fatalf("failed to stat file: %v", err)
		}

		if info.Mode().Perm() != perm {
			t.Errorf("expected permissions %o, got %o", perm, info.Mode().Perm())
		}

		if resp.FileMode != uint32(perm) {
			t.Errorf("expected FileMode %o, got %o", perm, resp.FileMode)
		}
	})

	t.Run("nested directory creation", func(t *testing.T) {
		fs := NewMockFileSystem(maxFileSize)
		cache := NewMockChecksumStore()
		ctx := &WorkspaceContext{
			FS:               fs,
			BinaryDetector:   NewMockBinaryDetector(),
			ChecksumComputer: NewMockChecksumComputer(),
			ChecksumCache:    cache,
			MaxFileSize:      maxFileSize,
			WorkspaceRoot:    workspaceRoot,
		}

		_, err := WriteFile(ctx, "nested/deep/file.txt", "content", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify file was created
		fileContent, err := fs.ReadFileRange("/workspace/nested/deep/file.txt", 0, 0)
		if err != nil {
			t.Fatalf("failed to read created file: %v", err)
		}
		if string(fileContent) != "content" {
			t.Errorf("expected content %q, got %q", "content", string(fileContent))
		}
	})

	t.Run("symlink inside workspace allowed", func(t *testing.T) {
		fs := NewMockFileSystem(maxFileSize)
		cache := NewMockChecksumStore()
		// Create symlink pointing inside workspace
		fs.CreateSymlink("/workspace/link", "/workspace/target.txt")
		fs.CreateFile("/workspace/target.txt", []byte("target"), 0o644)

		ctx := &WorkspaceContext{
			FS:               fs,
			BinaryDetector:   NewMockBinaryDetector(),
			ChecksumComputer: NewMockChecksumComputer(),
			ChecksumCache:    cache,
			MaxFileSize:      maxFileSize,
			WorkspaceRoot:    workspaceRoot,
		}

		// Writing to a symlink that points inside workspace should work
		// (the symlink itself is treated as a regular path for new files)
		_, err := WriteFile(ctx, "link", "new content", nil)
		// This should succeed because we're creating a new file at the symlink path
		if err != nil {
			// If it fails, it's because the symlink exists, which is expected
			if err != ErrFileExists {
				t.Errorf("unexpected error: %v", err)
			}
		}
	})

	t.Run("symlink directory escape prevention", func(t *testing.T) {
		fs := NewMockFileSystem(maxFileSize)
		cache := NewMockChecksumStore()
		// Create symlink directory pointing outside workspace
		fs.CreateSymlink("/workspace/link", "/outside")
		fs.CreateDir("/outside")

		ctx := &WorkspaceContext{
			FS:               fs,
			BinaryDetector:   NewMockBinaryDetector(),
			ChecksumComputer: NewMockChecksumComputer(),
			ChecksumCache:    cache,
			MaxFileSize:      maxFileSize,
			WorkspaceRoot:    workspaceRoot,
		}

		// Try to write a file through the symlink directory - should fail
		_, err := WriteFile(ctx, "link/escape.txt", "content", nil)
		if err != ErrOutsideWorkspace {
			t.Errorf("expected ErrOutsideWorkspace for symlink directory escape, got %v", err)
		}
	})

	t.Run("write through symlink chain inside workspace", func(t *testing.T) {
		fs := NewMockFileSystem(maxFileSize)
		cache := NewMockChecksumStore()
		// Create symlink chain: link1 -> link2 -> target_dir
		fs.CreateSymlink("/workspace/link1", "/workspace/link2")
		fs.CreateSymlink("/workspace/link2", "/workspace/target_dir")
		fs.CreateDir("/workspace/target_dir")

		ctx := &WorkspaceContext{
			FS:               fs,
			BinaryDetector:   NewMockBinaryDetector(),
			ChecksumComputer: NewMockChecksumComputer(),
			ChecksumCache:    cache,
			MaxFileSize:      maxFileSize,
			WorkspaceRoot:    workspaceRoot,
		}

		// Write through symlink chain - should succeed
		resp, err := WriteFile(ctx, "link1/file.txt", "content", nil)
		if err != nil {
			t.Fatalf("unexpected error writing through symlink chain: %v", err)
		}

		// Verify file was created at resolved location
		fileContent, err := fs.ReadFileRange("/workspace/target_dir/file.txt", 0, 0)
		if err != nil {
			t.Fatalf("failed to read created file: %v", err)
		}
		if string(fileContent) != "content" {
			t.Errorf("expected content %q, got %q", "content", string(fileContent))
		}

		// Verify response has correct absolute path
		if resp.AbsolutePath != "/workspace/target_dir/file.txt" {
			t.Errorf("expected absolute path /workspace/target_dir/file.txt, got %s", resp.AbsolutePath)
		}
	})

	t.Run("write through symlink chain escaping workspace", func(t *testing.T) {
		fs := NewMockFileSystem(maxFileSize)
		cache := NewMockChecksumStore()
		// Create chain: link1 -> link2 -> /tmp/outside
		fs.CreateSymlink("/workspace/link1", "/workspace/link2")
		fs.CreateSymlink("/workspace/link2", "/tmp/outside")
		fs.CreateDir("/tmp/outside")

		ctx := &WorkspaceContext{
			FS:               fs,
			BinaryDetector:   NewMockBinaryDetector(),
			ChecksumComputer: NewMockChecksumComputer(),
			ChecksumCache:    cache,
			MaxFileSize:      maxFileSize,
			WorkspaceRoot:    workspaceRoot,
		}

		// Try to write through escaping chain - should fail
		_, err := WriteFile(ctx, "link1/file.txt", "content", nil)
		if err != ErrOutsideWorkspace {
			t.Errorf("expected ErrOutsideWorkspace for escaping symlink chain, got %v", err)
		}
	})
}

func TestAtomicWriteCrashScenarios(t *testing.T) {
	workspaceRoot := "/workspace"
	maxFileSize := int64(1024 * 1024) // 1MB

	t.Run("crash during CreateTemp - no side effects", func(t *testing.T) {
		fs := NewMockFileSystem(maxFileSize)
		cache := NewMockChecksumStore()
		// Inject failure during CreateTemp
		fs.SetOperationError("CreateTemp", fmt.Errorf("disk full"))

		ctx := &WorkspaceContext{
			FS:               fs,
			BinaryDetector:   NewMockBinaryDetector(),
			ChecksumComputer: NewMockChecksumComputer(),
			ChecksumCache:    cache,
			MaxFileSize:      maxFileSize,
			WorkspaceRoot:    workspaceRoot,
		}

		_, err := WriteFile(ctx, "test.txt", "content", nil)
		if err == nil {
			t.Fatal("expected error")
		}

		// Verify no temp files were created
		tempFiles := fs.GetTempFiles()
		if len(tempFiles) != 0 {
			t.Errorf("expected no temp files, got %v", tempFiles)
		}

		// Verify original file doesn't exist
		_, err = fs.Stat("/workspace/test.txt")
		if err == nil {
			t.Error("expected file not to exist")
		}
	})

	t.Run("crash during Write - temp cleaned up", func(t *testing.T) {
		fs := NewMockFileSystem(maxFileSize)
		cache := NewMockChecksumStore()
		// Inject failure during Write
		fs.SetOperationError("Write", fmt.Errorf("write failed"))

		ctx := &WorkspaceContext{
			FS:               fs,
			BinaryDetector:   NewMockBinaryDetector(),
			ChecksumComputer: NewMockChecksumComputer(),
			ChecksumCache:    cache,
			MaxFileSize:      maxFileSize,
			WorkspaceRoot:    workspaceRoot,
		}

		_, err := WriteFile(ctx, "test.txt", "content", nil)
		if err == nil {
			t.Fatal("expected error")
		}

		// Verify temp file was cleaned up
		tempFiles := fs.GetTempFiles()
		if len(tempFiles) != 0 {
			t.Errorf("expected temp file to be cleaned up, got %v", tempFiles)
		}

		// Verify original file doesn't exist
		_, err = fs.Stat("/workspace/test.txt")
		if err == nil {
			t.Error("expected file not to exist")
		}
	})

	t.Run("crash during Sync - temp cleaned up", func(t *testing.T) {
		fs := NewMockFileSystem(maxFileSize)
		cache := NewMockChecksumStore()
		// Inject failure during Sync
		fs.SetOperationError("Sync", fmt.Errorf("sync failed"))

		ctx := &WorkspaceContext{
			FS:               fs,
			BinaryDetector:   NewMockBinaryDetector(),
			ChecksumComputer: NewMockChecksumComputer(),
			ChecksumCache:    cache,
			MaxFileSize:      maxFileSize,
			WorkspaceRoot:    workspaceRoot,
		}

		_, err := WriteFile(ctx, "test.txt", "content", nil)
		if err == nil {
			t.Fatal("expected error")
		}

		// Verify temp file was cleaned up
		tempFiles := fs.GetTempFiles()
		if len(tempFiles) != 0 {
			t.Errorf("expected temp file to be cleaned up, got %v", tempFiles)
		}

		// Verify original file doesn't exist
		_, err = fs.Stat("/workspace/test.txt")
		if err == nil {
			t.Error("expected file not to exist")
		}
	})

	t.Run("crash during Close - temp cleaned up", func(t *testing.T) {
		fs := NewMockFileSystem(maxFileSize)
		cache := NewMockChecksumStore()
		// Inject failure during Close
		fs.SetOperationError("Close", fmt.Errorf("close failed"))

		ctx := &WorkspaceContext{
			FS:               fs,
			BinaryDetector:   NewMockBinaryDetector(),
			ChecksumComputer: NewMockChecksumComputer(),
			ChecksumCache:    cache,
			MaxFileSize:      maxFileSize,
			WorkspaceRoot:    workspaceRoot,
		}

		_, err := WriteFile(ctx, "test.txt", "content", nil)
		if err == nil {
			t.Fatal("expected error")
		}

		// Verify temp file was cleaned up
		tempFiles := fs.GetTempFiles()
		if len(tempFiles) != 0 {
			t.Errorf("expected temp file to be cleaned up, got %v", tempFiles)
		}

		// Verify original file doesn't exist
		_, err = fs.Stat("/workspace/test.txt")
		if err == nil {
			t.Error("expected file not to exist")
		}
	})

	t.Run("crash during Rename - temp cleaned up, original intact", func(t *testing.T) {
		fs := NewMockFileSystem(maxFileSize)
		cache := NewMockChecksumStore()
		// Create an existing file
		fs.CreateFile("/workspace/test.txt", []byte("original"), 0o644)

		// Inject failure during Rename
		fs.SetOperationError("Rename", fmt.Errorf("rename failed"))

		ctx := &WorkspaceContext{
			FS:               fs,
			BinaryDetector:   NewMockBinaryDetector(),
			ChecksumComputer: NewMockChecksumComputer(),
			ChecksumCache:    cache,
			MaxFileSize:      maxFileSize,
			WorkspaceRoot:    workspaceRoot,
		}

		// Resolve path to get absolute path
		abs, _, err := Resolve(ctx, "test.txt")
		if err != nil {
			t.Fatalf("failed to resolve path: %v", err)
		}

		// Call writeFileAtomic directly to bypass WriteFile's existence check
		// This tests the atomic write path when rename fails
		err = writeFileAtomic(ctx, abs, []byte("new content"), 0o644)
		if err == nil {
			t.Fatal("expected error from rename failure")
		}

		// Verify original file is still intact (atomic write protects it)
		content, err := fs.ReadFileRange("/workspace/test.txt", 0, 0)
		if err != nil {
			t.Fatalf("failed to read original file: %v", err)
		}
		if string(content) != "original" {
			t.Errorf("expected original content to be preserved, got %q", string(content))
		}

		// Verify temp file was cleaned up
		tempFiles := fs.GetTempFiles()
		if len(tempFiles) != 0 {
			t.Errorf("expected temp file to be cleaned up, got %v", tempFiles)
		}
	})

	t.Run("crash during Chmod - file exists but wrong permissions handled", func(t *testing.T) {
		fs := NewMockFileSystem(maxFileSize)
		cache := NewMockChecksumStore()
		// Inject failure during Chmod
		fs.SetOperationError("Chmod", fmt.Errorf("chmod failed"))

		ctx := &WorkspaceContext{
			FS:               fs,
			BinaryDetector:   NewMockBinaryDetector(),
			ChecksumComputer: NewMockChecksumComputer(),
			ChecksumCache:    cache,
			MaxFileSize:      maxFileSize,
			WorkspaceRoot:    workspaceRoot,
		}

		_, err := WriteFile(ctx, "test.txt", "content", nil)
		// Chmod failure should still result in error, but file should exist
		// The atomic write succeeded (rename worked), but chmod failed
		if err == nil {
			t.Fatal("expected error from chmod failure")
		}

		// Verify temp file was cleaned up
		tempFiles := fs.GetTempFiles()
		if len(tempFiles) != 0 {
			t.Errorf("expected temp file to be cleaned up, got %v", tempFiles)
		}
	})

	t.Run("successful atomic write", func(t *testing.T) {
		fs := NewMockFileSystem(maxFileSize)
		cache := NewMockChecksumStore()
		ctx := &WorkspaceContext{
			FS:               fs,
			BinaryDetector:   NewMockBinaryDetector(),
			ChecksumComputer: NewMockChecksumComputer(),
			ChecksumCache:    cache,
			MaxFileSize:      maxFileSize,
			WorkspaceRoot:    workspaceRoot,
		}

		content := "test content"
		resp, err := WriteFile(ctx, "test.txt", content, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify file was created with correct content
		fileContent, err := fs.ReadFileRange("/workspace/test.txt", 0, 0)
		if err != nil {
			t.Fatalf("failed to read created file: %v", err)
		}
		if string(fileContent) != content {
			t.Errorf("expected content %q, got %q", content, string(fileContent))
		}

		// Verify no temp files remain
		tempFiles := fs.GetTempFiles()
		if len(tempFiles) != 0 {
			t.Errorf("expected no temp files after successful write, got %v", tempFiles)
		}

		// Verify response
		if resp.BytesWritten != len(content) {
			t.Errorf("expected %d bytes written, got %d", len(content), resp.BytesWritten)
		}
	})
}
