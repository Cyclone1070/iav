package tools

import (
	"context"
	"testing"

	"github.com/Cyclone1070/iav/internal/config"
	"github.com/Cyclone1070/iav/internal/testing/mocks"
	"github.com/Cyclone1070/iav/internal/tools/models"
	"github.com/Cyclone1070/iav/internal/tools/services"
)

func TestReadFile(t *testing.T) {
	workspaceRoot := "/workspace"
	maxFileSize := int64(1024 * 1024) // 1MB

	t.Run("full read caches checksum", func(t *testing.T) {
		fs := mocks.NewMockFileSystem(maxFileSize)
		checksumManager := services.NewChecksumManager()
		content := []byte("test content")
		fs.CreateFile("/workspace/test.txt", content, 0644)

		cfg := config.DefaultConfig()
		cfg.Tools.MaxFileSize = maxFileSize
		ctx := &models.WorkspaceContext{
			FS:              fs,
			BinaryDetector:  mocks.NewMockBinaryDetector(),
			ChecksumManager: checksumManager,
			WorkspaceRoot:   workspaceRoot,
			Config:          *cfg,
		}

		resp, err := ReadFile(context.Background(), ctx, models.ReadFileRequest{Path: "test.txt"})
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
		fs := mocks.NewMockFileSystem(maxFileSize)
		checksumManager := services.NewChecksumManager()
		content := []byte("test content")
		fs.CreateFile("/workspace/test.txt", content, 0644)

		ctx := &models.WorkspaceContext{
			FS:              fs,
			BinaryDetector:  mocks.NewMockBinaryDetector(),
			ChecksumManager: checksumManager,
			WorkspaceRoot:   workspaceRoot,
			Config:          *config.DefaultConfig(),
		}

		offset := int64(5)
		resp, err := ReadFile(context.Background(), ctx, models.ReadFileRequest{Path: "test.txt", Offset: &offset})
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
		fs := mocks.NewMockFileSystem(maxFileSize)
		checksumManager := services.NewChecksumManager()
		detector := mocks.NewMockBinaryDetector()
		detector.IsBinaryContentFunc = func(content []byte) bool {
			return true
		}

		// Create file with null bytes (actual binary content)
		content := []byte{0x00, 0x01, 0x02, 't', 'e', 's', 't'}
		fs.CreateFile("/workspace/binary.bin", content, 0644)

		cfg := config.DefaultConfig()
		cfg.Tools.MaxFileSize = maxFileSize
		ctx := &models.WorkspaceContext{
			FS:              fs,
			BinaryDetector:  detector,
			ChecksumManager: checksumManager,
			WorkspaceRoot:   workspaceRoot,
			Config:          *cfg,
		}

		_, err := ReadFile(context.Background(), ctx, models.ReadFileRequest{Path: "binary.bin"})
		if err != models.ErrBinaryFile {
			t.Errorf("expected ErrBinaryFile, got %v", err)
		}
	})

	t.Run("size limit enforcement", func(t *testing.T) {
		fs := mocks.NewMockFileSystem(maxFileSize)
		checksumManager := services.NewChecksumManager()
		// Create file larger than limit
		largeContent := make([]byte, maxFileSize+1)
		fs.CreateFile("/workspace/large.txt", largeContent, 0644)

		cfg := config.DefaultConfig()
		cfg.Tools.MaxFileSize = maxFileSize
		ctx := &models.WorkspaceContext{
			FS:              fs,
			BinaryDetector:  mocks.NewMockBinaryDetector(),
			ChecksumManager: checksumManager,
			WorkspaceRoot:   workspaceRoot,
			Config:          *cfg,
		}

		_, err := ReadFile(context.Background(), ctx, models.ReadFileRequest{Path: "large.txt"})
		if err != models.ErrTooLarge {
			t.Errorf("expected ErrTooLarge, got %v", err)
		}
	})

	t.Run("offset beyond EOF", func(t *testing.T) {
		fs := mocks.NewMockFileSystem(maxFileSize)
		checksumManager := services.NewChecksumManager()
		content := []byte("test")
		fs.CreateFile("/workspace/test.txt", content, 0644)

		ctx := &models.WorkspaceContext{
			FS:              fs,
			BinaryDetector:  mocks.NewMockBinaryDetector(),
			ChecksumManager: checksumManager,
			WorkspaceRoot:   workspaceRoot,
			Config:          *config.DefaultConfig(),
		}

		offset := int64(10000)
		resp, err := ReadFile(context.Background(), ctx, models.ReadFileRequest{Path: "test.txt", Offset: &offset})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Content != "" {
			t.Errorf("expected empty content for offset beyond EOF, got %q", resp.Content)
		}
	})

	t.Run("directory rejection", func(t *testing.T) {
		fs := mocks.NewMockFileSystem(maxFileSize)
		checksumManager := services.NewChecksumManager()
		fs.CreateDir("/workspace/subdir")

		ctx := &models.WorkspaceContext{
			FS:              fs,
			BinaryDetector:  mocks.NewMockBinaryDetector(),
			ChecksumManager: checksumManager,
			WorkspaceRoot:   workspaceRoot,
			Config:          *config.DefaultConfig(),
		}

		_, err := ReadFile(context.Background(), ctx, models.ReadFileRequest{Path: "subdir"})
		if err == nil {
			t.Error("expected error when reading directory")
		}
	})

	t.Run("file not found", func(t *testing.T) {
		fs := mocks.NewMockFileSystem(maxFileSize)
		checksumManager := services.NewChecksumManager()
		ctx := &models.WorkspaceContext{
			FS:              fs,
			BinaryDetector:  mocks.NewMockBinaryDetector(),
			ChecksumManager: checksumManager,
			WorkspaceRoot:   workspaceRoot,
			Config:          *config.DefaultConfig(),
		}

		_, err := ReadFile(context.Background(), ctx, models.ReadFileRequest{Path: "nonexistent.txt"})
		if err == nil {
			t.Error("expected error for nonexistent file")
		}
	})

	t.Run("limit truncation", func(t *testing.T) {
		fs := mocks.NewMockFileSystem(maxFileSize)
		checksumManager := services.NewChecksumManager()
		content := []byte("test content")
		fs.CreateFile("/workspace/test.txt", content, 0644)

		ctx := &models.WorkspaceContext{
			FS:              fs,
			BinaryDetector:  mocks.NewMockBinaryDetector(),
			ChecksumManager: checksumManager,
			WorkspaceRoot:   workspaceRoot,
			Config:          *config.DefaultConfig(),
		}

		limit := int64(4)
		resp, err := ReadFile(context.Background(), ctx, models.ReadFileRequest{Path: "test.txt", Limit: &limit})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expected := string(content[:4])
		if resp.Content != expected {
			t.Errorf("expected content %q, got %q", expected, resp.Content)
		}
	})
}
