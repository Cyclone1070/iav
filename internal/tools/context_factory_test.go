package tools

import (
	"testing"

	"github.com/Cyclone1070/deployforme/internal/tools/models"
	"github.com/Cyclone1070/deployforme/internal/tools/services"
)

func TestMultiContextIsolation(t *testing.T) {
	maxFileSize := int64(1024 * 1024) // 1MB

	// Create two separate contexts with different workspace roots
	fs1 := services.NewMockFileSystem(maxFileSize)
	checksumManager1 := services.NewChecksumManager()

	fs2 := services.NewMockFileSystem(maxFileSize)
	checksumManager2 := services.NewChecksumManager()

	ctx1 := &models.WorkspaceContext{
		FS:              fs1,
		BinaryDetector:  services.NewMockBinaryDetector(),
		ChecksumManager: checksumManager1,
		MaxFileSize:     maxFileSize,
		WorkspaceRoot:   "/workspace1",
	}

	ctx2 := &models.WorkspaceContext{
		FS:              fs2,
		BinaryDetector:  services.NewMockBinaryDetector(),
		ChecksumManager: checksumManager2,
		MaxFileSize:     maxFileSize,
		WorkspaceRoot:   "/workspace2",
	}

	// Create files in both contexts
	content1 := "content1"
	content2 := "content2"

	resp1, err := WriteFile(ctx1, "file.txt", content1, nil)
	if err != nil {
		t.Fatalf("failed to write file in ctx1: %v", err)
	}

	resp2, err := WriteFile(ctx2, "file.txt", content2, nil)
	if err != nil {
		t.Fatalf("failed to write file in ctx2: %v", err)
	}

	// Verify caches are isolated
	checksum1, ok1 := ctx1.ChecksumManager.Get(resp1.AbsolutePath)
	if !ok1 {
		t.Error("ctx1 cache should contain file1")
	}
	if checksum1 == "" {
		t.Error("ctx1 cache checksum should not be empty")
	}

	checksum2, ok2 := ctx2.ChecksumManager.Get(resp2.AbsolutePath)
	if !ok2 {
		t.Error("ctx2 cache should contain file2")
	}
	if checksum2 == "" {
		t.Error("ctx2 cache checksum should not be empty")
	}

	// Verify ctx1 cache doesn't contain ctx2's file
	_, ok := ctx1.ChecksumManager.Get(resp2.AbsolutePath)
	if ok {
		t.Error("ctx1 cache should not contain ctx2's file")
	}

	// Verify ctx2 cache doesn't contain ctx1's file
	_, ok = ctx2.ChecksumManager.Get(resp1.AbsolutePath)
	if ok {
		t.Error("ctx2 cache should not contain ctx1's file")
	}

	// Verify filesystems are isolated
	read1, err := ReadFile(ctx1, "file.txt", nil, nil)
	if err != nil {
		t.Fatalf("failed to read file from ctx1: %v", err)
	}
	if read1.Content != content1 {
		t.Errorf("ctx1 should read its own content, got %q", read1.Content)
	}

	read2, err := ReadFile(ctx2, "file.txt", nil, nil)
	if err != nil {
		t.Fatalf("failed to read file from ctx2: %v", err)
	}
	if read2.Content != content2 {
		t.Errorf("ctx2 should read its own content, got %q", read2.Content)
	}
}

func TestCustomFileSizeLimit(t *testing.T) {
	workspaceRoot := "/workspace"
	smallLimit := int64(100)              // 100 bytes
	largeLimit := int64(10 * 1024 * 1024) // 10MB

	t.Run("small limit enforced", func(t *testing.T) {
		fs := services.NewMockFileSystem(smallLimit)
		checksumManager := services.NewChecksumManager()

		ctx := &models.WorkspaceContext{
			FS:              fs,
			BinaryDetector:  services.NewMockBinaryDetector(),
			ChecksumManager: checksumManager,
			MaxFileSize:     smallLimit,
			WorkspaceRoot:   workspaceRoot,
		}

		// Create content that exceeds the limit
		largeContent := make([]byte, smallLimit+1)
		for i := range largeContent {
			largeContent[i] = 'A'
		}

		_, err := WriteFile(ctx, "large.txt", string(largeContent), nil)
		if err != models.ErrTooLarge {
			t.Errorf("expected ErrTooLarge for content exceeding limit, got %v", err)
		}
	})

	t.Run("large limit allows bigger files", func(t *testing.T) {
		fs := services.NewMockFileSystem(largeLimit)
		checksumManager := services.NewChecksumManager()

		ctx := &models.WorkspaceContext{
			FS:              fs,
			BinaryDetector:  services.NewMockBinaryDetector(),
			ChecksumManager: checksumManager,
			MaxFileSize:     largeLimit,
			WorkspaceRoot:   workspaceRoot,
		}

		// Create content within the large limit but exceeding default
		content := make([]byte, 6*1024*1024) // 6MB, exceeds default 5MB
		for i := range content {
			content[i] = 'A'
		}

		_, err := WriteFile(ctx, "large.txt", string(content), nil)
		if err != nil {
			t.Errorf("expected success with large limit, got %v", err)
		}
	})

	t.Run("different limits in different contexts", func(t *testing.T) {
		fs1 := services.NewMockFileSystem(smallLimit)
		checksumManager1 := services.NewChecksumManager()

		fs2 := services.NewMockFileSystem(largeLimit)
		checksumManager2 := services.NewChecksumManager()

		ctx1 := &models.WorkspaceContext{
			FS:              fs1,
			BinaryDetector:  services.NewMockBinaryDetector(),
			ChecksumManager: checksumManager1,
			MaxFileSize:     smallLimit,
			WorkspaceRoot:   workspaceRoot,
		}

		ctx2 := &models.WorkspaceContext{
			FS:              fs2,
			BinaryDetector:  services.NewMockBinaryDetector(),
			ChecksumManager: checksumManager2,
			MaxFileSize:     largeLimit,
			WorkspaceRoot:   workspaceRoot,
		}

		// Content that fits in ctx2 but not ctx1
		content := make([]byte, smallLimit+50)
		for i := range content {
			content[i] = 'A'
		}

		// Should fail in ctx1
		_, err := WriteFile(ctx1, "file.txt", string(content), nil)
		if err != models.ErrTooLarge {
			t.Errorf("expected ErrTooLarge in ctx1, got %v", err)
		}

		// Should succeed in ctx2
		_, err = WriteFile(ctx2, "file.txt", string(content), nil)
		if err != nil {
			t.Errorf("expected success in ctx2, got %v", err)
		}
	})
}
