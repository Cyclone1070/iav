package tools

import (
	"testing"
)

func TestReadFile(t *testing.T) {
	workspaceRoot := "/workspace"
	maxFileSize := int64(1024 * 1024) // 1MB

	t.Run("full read caches checksum", func(t *testing.T) {
		fs := NewMockFileSystem(maxFileSize)
		checksumManager := NewMockChecksumManager()
		content := []byte("test content")
		fs.CreateFile("/workspace/test.txt", content, 0644)

		ctx := &WorkspaceContext{
			FS:              fs,
			BinaryDetector:  NewMockBinaryDetector(),
			ChecksumManager: checksumManager,
			MaxFileSize:      maxFileSize,
			WorkspaceRoot:    workspaceRoot,
		}

		resp, err := ReadFile(ctx, "test.txt", nil, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if resp.Content != string(content) {
			t.Errorf("expected content %q, got %q", string(content), resp.Content)
		}

		// Verify cache was updated
		checksum, ok := ctx.ChecksumManager.Get(resp.AbsolutePath)
		if !ok {
			t.Error("expected cache to be updated after full read")
		}
		if checksum == "" {
			t.Error("expected non-empty checksum in cache")
		}
	})

	t.Run("partial read skips cache update", func(t *testing.T) {
		fs := NewMockFileSystem(maxFileSize)
		checksumManager := NewMockChecksumManager()
		content := []byte("test content")
		fs.CreateFile("/workspace/test.txt", content, 0644)

		ctx := &WorkspaceContext{
			FS:              fs,
			BinaryDetector:  NewMockBinaryDetector(),
			ChecksumManager: checksumManager,
			MaxFileSize:      maxFileSize,
			WorkspaceRoot:    workspaceRoot,
		}

		offset := int64(5)
		resp, err := ReadFile(ctx, "test.txt", &offset, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expected := string(content[5:])
		if resp.Content != expected {
			t.Errorf("expected content %q, got %q", expected, resp.Content)
		}

		// Verify cache was NOT updated
		_, ok := ctx.ChecksumManager.Get(resp.AbsolutePath)
		if ok {
			t.Error("expected cache to NOT be updated after partial read")
		}
	})

	t.Run("binary detection rejection", func(t *testing.T) {
		fs := NewMockFileSystem(maxFileSize)
		checksumManager := NewMockChecksumManager()
		detector := NewMockBinaryDetector()

		content := []byte("test content")
		fs.CreateFile("/workspace/binary.bin", content, 0644)
		detector.SetBinaryPath("/workspace/binary.bin", true)

		ctx := &WorkspaceContext{
			FS:               fs,
			BinaryDetector:   detector,
			ChecksumManager: checksumManager,
			MaxFileSize:      maxFileSize,
			WorkspaceRoot:    workspaceRoot,
		}

		_, err := ReadFile(ctx, "binary.bin", nil, nil)
		if err != ErrBinaryFile {
			t.Errorf("expected ErrBinaryFile, got %v", err)
		}
	})

	t.Run("size limit enforcement", func(t *testing.T) {
		fs := NewMockFileSystem(maxFileSize)
		checksumManager := NewMockChecksumManager()
		// Create file larger than limit
		largeContent := make([]byte, maxFileSize+1)
		fs.CreateFile("/workspace/large.txt", largeContent, 0644)

		ctx := &WorkspaceContext{
			FS:               fs,
			BinaryDetector:   NewMockBinaryDetector(),
			ChecksumManager: checksumManager,
			MaxFileSize:      maxFileSize,
			WorkspaceRoot:    workspaceRoot,
		}

		_, err := ReadFile(ctx, "large.txt", nil, nil)
		if err != ErrTooLarge {
			t.Errorf("expected ErrTooLarge, got %v", err)
		}
	})

	t.Run("negative offset", func(t *testing.T) {
		fs := NewMockFileSystem(maxFileSize)
		checksumManager := NewMockChecksumManager()
		content := []byte("test content")
		fs.CreateFile("/workspace/test.txt", content, 0644)

		ctx := &WorkspaceContext{
			FS:              fs,
			BinaryDetector:  NewMockBinaryDetector(),
			ChecksumManager: checksumManager,
			MaxFileSize:      maxFileSize,
			WorkspaceRoot:    workspaceRoot,
		}

		offset := int64(-1)
		_, err := ReadFile(ctx, "test.txt", &offset, nil)
		if err != ErrInvalidOffset {
			t.Errorf("expected ErrInvalidOffset, got %v", err)
		}
	})

	t.Run("negative limit", func(t *testing.T) {
		fs := NewMockFileSystem(maxFileSize)
		checksumManager := NewMockChecksumManager()
		content := []byte("test content")
		fs.CreateFile("/workspace/test.txt", content, 0644)

		ctx := &WorkspaceContext{
			FS:              fs,
			BinaryDetector:  NewMockBinaryDetector(),
			ChecksumManager: checksumManager,
			MaxFileSize:      maxFileSize,
			WorkspaceRoot:    workspaceRoot,
		}

		limit := int64(-1)
		_, err := ReadFile(ctx, "test.txt", nil, &limit)
		if err != ErrInvalidLimit {
			t.Errorf("expected ErrInvalidLimit, got %v", err)
		}
	})

	t.Run("offset beyond EOF", func(t *testing.T) {
		fs := NewMockFileSystem(maxFileSize)
		checksumManager := NewMockChecksumManager()
		content := []byte("test")
		fs.CreateFile("/workspace/test.txt", content, 0644)

		ctx := &WorkspaceContext{
			FS:               fs,
			BinaryDetector:   NewMockBinaryDetector(),
			ChecksumManager: checksumManager,
			MaxFileSize:      maxFileSize,
			WorkspaceRoot:    workspaceRoot,
		}

		offset := int64(10000)
		resp, err := ReadFile(ctx, "test.txt", &offset, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Content != "" {
			t.Errorf("expected empty content for offset beyond EOF, got %q", resp.Content)
		}
	})

	t.Run("directory rejection", func(t *testing.T) {
		fs := NewMockFileSystem(maxFileSize)
		checksumManager := NewMockChecksumManager()
		fs.CreateDir("/workspace/subdir")

		ctx := &WorkspaceContext{
			FS:               fs,
			BinaryDetector:   NewMockBinaryDetector(),
			ChecksumManager: checksumManager,
			MaxFileSize:      maxFileSize,
			WorkspaceRoot:    workspaceRoot,
		}

		_, err := ReadFile(ctx, "subdir", nil, nil)
		if err == nil {
			t.Error("expected error when reading directory")
		}
	})

	t.Run("file not found", func(t *testing.T) {
		fs := NewMockFileSystem(maxFileSize)
		checksumManager := NewMockChecksumManager()
		ctx := &WorkspaceContext{
			FS:              fs,
			BinaryDetector:  NewMockBinaryDetector(),
			ChecksumManager: checksumManager,
			MaxFileSize:      maxFileSize,
			WorkspaceRoot:    workspaceRoot,
		}

		_, err := ReadFile(ctx, "nonexistent.txt", nil, nil)
		if err == nil {
			t.Error("expected error for nonexistent file")
		}
	})

	t.Run("limit truncation", func(t *testing.T) {
		fs := NewMockFileSystem(maxFileSize)
		checksumManager := NewMockChecksumManager()
		content := []byte("test content")
		fs.CreateFile("/workspace/test.txt", content, 0644)

		ctx := &WorkspaceContext{
			FS:              fs,
			BinaryDetector:  NewMockBinaryDetector(),
			ChecksumManager: checksumManager,
			MaxFileSize:      maxFileSize,
			WorkspaceRoot:    workspaceRoot,
		}

		limit := int64(4)
		resp, err := ReadFile(ctx, "test.txt", nil, &limit)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expected := string(content[:4])
		if resp.Content != expected {
			t.Errorf("expected content %q, got %q", expected, resp.Content)
		}
	})
}
