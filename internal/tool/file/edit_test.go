package file

// This file contains edit file tests.
// Mocks are defined in write_test.go and shared across all test files in this package.

import (
	"context"
	"errors"
	"testing"

	"github.com/Cyclone1070/iav/internal/config"
	"github.com/Cyclone1070/iav/internal/tool/service/path"
)

func TestEditFile(t *testing.T) {
	workspaceRoot := "/workspace"
	maxFileSize := int64(1024 * 1024) // 1MB

	t.Run("conflict detection when cache checksum differs", func(t *testing.T) {
		fs := newMockFileSystemForWrite()
		checksumManager := newMockChecksumManagerForWrite()
		originalContent := []byte("original content")
		fs.createFile("/workspace/test.txt", originalContent, 0644)

		cfg := config.DefaultConfig()
		cfg.Tools.MaxFileSize = maxFileSize

		readTool := NewReadFileTool(fs, checksumManager, cfg, path.NewResolver(workspaceRoot))
		editTool := NewEditFileTool(fs, checksumManager, cfg, path.NewResolver(workspaceRoot))

		// Read file to populate cache
		readReq := &ReadFileRequest{Path: "test.txt"}
		_, err := readTool.Run(context.Background(), readReq)
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}

		// Modify file externally (simulate external change)
		modifiedContent := []byte("modified externally")
		fs.createFile("/workspace/test.txt", modifiedContent, 0644)

		// Try to edit - should fail with conflict
		ops := []EditOperation{
			{
				Before:               "original content",
				After:                "new content",
				ExpectedReplacements: 1,
			},
		}

		editReq := &EditFileRequest{Path: "test.txt", Operations: ops}
		_, err = editTool.Run(context.Background(), editReq)
		if err == nil || !errors.Is(err, ErrEditConflict) {
			t.Errorf("expected ErrEditConflict, got %v", err)
		}
	})

	t.Run("no cached checksum skips revalidation", func(t *testing.T) {
		fs := newMockFileSystemForWrite()
		checksumManager := newMockChecksumManagerForWrite()
		// File content
		content := []byte("some content")
		fs.createFile("/workspace/test.txt", content, 0644)

		cfg := config.DefaultConfig()

		// Skip reading first, so no cache
		editTool := NewEditFileTool(fs, checksumManager, cfg, path.NewResolver(workspaceRoot))

		ops := []EditOperation{
			{
				Before:               "some",
				After:                "new",
				ExpectedReplacements: 1,
			},
		}

		editReq := &EditFileRequest{Path: "test.txt", Operations: ops}
		resp, err := editTool.Run(context.Background(), editReq)
		if err != nil {
			t.Fatalf("Run failed: %v", err)
		}
		if resp.OperationsApplied != 1 {
			t.Errorf("expected 1 op applied, got %d", resp.OperationsApplied)
		}
	})

	t.Run("multiple operations", func(t *testing.T) {
		fs := newMockFileSystemForWrite()
		checksumManager := newMockChecksumManagerForWrite()
		content := []byte("line1\nline2\nline3")
		fs.createFile("/workspace/test.txt", content, 0644)

		cfg := config.DefaultConfig()
		cfg.Tools.MaxFileSize = maxFileSize

		readTool := NewReadFileTool(fs, checksumManager, cfg, path.NewResolver(workspaceRoot))
		editTool := NewEditFileTool(fs, checksumManager, cfg, path.NewResolver(workspaceRoot))

		// Read first to populate cache
		readReq := &ReadFileRequest{Path: "test.txt"}
		_, err := readTool.Run(context.Background(), readReq)
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}

		ops := []EditOperation{
			{
				Before:               "line1",
				After:                "modified1",
				ExpectedReplacements: 1,
			},
			{
				Before:               "line2",
				After:                "modified2",
				ExpectedReplacements: 1,
			},
		}

		editReq := &EditFileRequest{Path: "test.txt", Operations: ops}
		resp, err := editTool.Run(context.Background(), editReq)
		if err != nil {
			t.Fatalf("Run failed: %v", err)
		}

		if resp.OperationsApplied != 2 {
			t.Errorf("expected 2 operations applied, got %d", resp.OperationsApplied)
		}

		// Verify final content
		final, _ := fs.ReadFileRange("/workspace/test.txt", 0, 0)
		expected := "modified1\nmodified2\nline3"
		if string(final) != expected {
			t.Errorf("expected content %q, got %q", expected, string(final))
		}
	})

	t.Run("mismatch in expected replacements", func(t *testing.T) {
		fs := newMockFileSystemForWrite()
		checksumManager := newMockChecksumManagerForWrite()
		content := []byte("line1\nline1\nline3")
		fs.createFile("/workspace/test.txt", content, 0644)

		cfg := config.DefaultConfig()
		cfg.Tools.MaxFileSize = maxFileSize

		readTool := NewReadFileTool(fs, checksumManager, cfg, path.NewResolver(workspaceRoot))
		editTool := NewEditFileTool(fs, checksumManager, cfg, path.NewResolver(workspaceRoot))

		// Read first to populate cache
		readReq := &ReadFileRequest{Path: "test.txt"}
		_, err := readTool.Run(context.Background(), readReq)
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}

		ops := []EditOperation{
			{
				Before:               "line1",
				After:                "modified1",
				ExpectedReplacements: 1, // But there are 2 occurrences
			},
		}

		editReq := &EditFileRequest{Path: "test.txt", Operations: ops}
		_, err = editTool.Run(context.Background(), editReq)
		if err == nil || !errors.Is(err, ErrReplacementCountMismatch) {
			t.Errorf("expected ErrReplacementCountMismatch, got %v", err)
		}
	})

	t.Run("replacement when snippet appears multiple times but ExpectedReplacements matches", func(t *testing.T) {
		fs := newMockFileSystemForWrite()
		checksumManager := newMockChecksumManagerForWrite()
		content := []byte("foo\nfoo\nbar")
		fs.createFile("/workspace/test.txt", content, 0644)

		cfg := config.DefaultConfig()

		readTool := NewReadFileTool(fs, checksumManager, cfg, path.NewResolver(workspaceRoot))
		editTool := NewEditFileTool(fs, checksumManager, cfg, path.NewResolver(workspaceRoot))

		// Read first to populate cache
		readReq := &ReadFileRequest{Path: "test.txt"}
		_, _ = readTool.Run(context.Background(), readReq)

		ops := []EditOperation{
			{
				Before:               "foo",
				After:                "baz",
				ExpectedReplacements: 2,
			},
		}

		editReq := &EditFileRequest{Path: "test.txt", Operations: ops}
		resp, err := editTool.Run(context.Background(), editReq)
		if err != nil {
			t.Fatalf("Run failed: %v", err)
		}

		if resp.OperationsApplied != 1 {
			t.Errorf("expected 1 operation applied, got %d", resp.OperationsApplied)
		}

		final, _ := fs.ReadFileRange("/workspace/test.txt", 0, 0)
		expected := "baz\nbaz\nbar"
		if string(final) != expected {
			t.Errorf("expected %q, got %q", expected, string(final))
		}
	})

	t.Run("zero expected replacements defaults to 1", func(t *testing.T) {
		fs := newMockFileSystemForWrite()
		checksumManager := newMockChecksumManagerForWrite()
		content := []byte("foo\nfoo\nbar")
		fs.createFile("/workspace/test.txt", content, 0644)

		cfg := config.DefaultConfig()

		readTool := NewReadFileTool(fs, checksumManager, cfg, path.NewResolver(workspaceRoot))
		editTool := NewEditFileTool(fs, checksumManager, cfg, path.NewResolver(workspaceRoot))

		// Read first to populate cache
		readReq := &ReadFileRequest{Path: "test.txt"}
		_, _ = readTool.Run(context.Background(), readReq)

		ops := []EditOperation{
			{
				Before:               "foo",
				After:                "baz",
				ExpectedReplacements: 0, // Defaults to 1, but there are 2
			},
		}

		editReq := &EditFileRequest{Path: "test.txt", Operations: ops}
		_, err := editTool.Run(context.Background(), editReq)
		if err == nil || !errors.Is(err, ErrReplacementCountMismatch) {
			t.Errorf("expected ErrReplacementCountMismatch, got %v", err)
		}
	})

	t.Run("snippet not found", func(t *testing.T) {
		fs := newMockFileSystemForWrite()
		checksumManager := newMockChecksumManagerForWrite()
		fs.createFile("/workspace/test.txt", []byte("content"), 0644)

		cfg := config.DefaultConfig()

		readTool := NewReadFileTool(fs, checksumManager, cfg, path.NewResolver(workspaceRoot))
		editTool := NewEditFileTool(fs, checksumManager, cfg, path.NewResolver(workspaceRoot))

		readReq := &ReadFileRequest{Path: "test.txt"}
		_, _ = readTool.Run(context.Background(), readReq)

		ops := []EditOperation{
			{
				Before: "nonexistent",
				After:  "new",
			},
		}

		editReq := &EditFileRequest{Path: "test.txt", Operations: ops}
		_, err := editTool.Run(context.Background(), editReq)
		if err == nil || !errors.Is(err, ErrSnippetNotFound) {
			t.Errorf("expected ErrSnippetNotFound, got %v", err)
		}
	})

	t.Run("append to non-empty file", func(t *testing.T) {
		fs := newMockFileSystemForWrite()
		checksumManager := newMockChecksumManagerForWrite()
		fs.createFile("/workspace/test.txt", []byte("existing"), 0644)

		cfg := config.DefaultConfig()
		editTool := NewEditFileTool(fs, checksumManager, cfg, path.NewResolver(workspaceRoot))

		ops := []EditOperation{
			{
				Before: "",
				After:  "\nnew line",
			},
		}

		editReq := &EditFileRequest{Path: "test.txt", Operations: ops}
		resp, err := editTool.Run(context.Background(), editReq)
		if err != nil {
			t.Fatalf("Run failed: %v", err)
		}

		if resp.OperationsApplied != 1 {
			t.Errorf("expected 1 op applied, got %d", resp.OperationsApplied)
		}

		final, _ := fs.ReadFileRange("/workspace/test.txt", 0, 0)
		expected := "existing\nnew line"
		if string(final) != expected {
			t.Errorf("expected content %q, got %q", expected, string(final))
		}
	})

	t.Run("append to empty file", func(t *testing.T) {
		fs := newMockFileSystemForWrite()
		checksumManager := newMockChecksumManagerForWrite()
		fs.createFile("/workspace/test.txt", []byte(""), 0644)

		cfg := config.DefaultConfig()
		editTool := NewEditFileTool(fs, checksumManager, cfg, path.NewResolver(workspaceRoot))

		ops := []EditOperation{
			{
				Before: "",
				After:  "first content",
			},
		}

		editReq := &EditFileRequest{Path: "test.txt", Operations: ops}
		resp, err := editTool.Run(context.Background(), editReq)
		if err != nil {
			t.Fatalf("Run failed: %v", err)
		}

		if resp.OperationsApplied != 1 {
			t.Errorf("expected 1 op applied, got %d", resp.OperationsApplied)
		}

		final, _ := fs.ReadFileRange("/workspace/test.txt", 0, 0)
		expected := "first content"
		if string(final) != expected {
			t.Errorf("expected content %q, got %q", expected, string(final))
		}
	})

	t.Run("multiple appends in one request", func(t *testing.T) {
		fs := newMockFileSystemForWrite()
		checksumManager := newMockChecksumManagerForWrite()
		fs.createFile("/workspace/test.txt", []byte("start"), 0644)

		cfg := config.DefaultConfig()
		editTool := NewEditFileTool(fs, checksumManager, cfg, path.NewResolver(workspaceRoot))

		ops := []EditOperation{
			{
				Before: "",
				After:  "1",
			},
			{
				Before: "",
				After:  "2",
			},
		}

		editReq := &EditFileRequest{Path: "test.txt", Operations: ops}
		resp, err := editTool.Run(context.Background(), editReq)
		if err != nil {
			t.Fatalf("Run failed: %v", err)
		}

		if resp.OperationsApplied != 2 {
			t.Errorf("expected 2 ops applied, got %d", resp.OperationsApplied)
		}

		final, _ := fs.ReadFileRange("/workspace/test.txt", 0, 0)
		expected := "start12"
		if string(final) != expected {
			t.Errorf("expected content %q, got %q", expected, string(final))
		}
	})

	t.Run("mixed replace and append", func(t *testing.T) {
		fs := newMockFileSystemForWrite()
		checksumManager := newMockChecksumManagerForWrite()
		fs.createFile("/workspace/test.txt", []byte("foo\nbar"), 0644)

		cfg := config.DefaultConfig()
		editTool := NewEditFileTool(fs, checksumManager, cfg, path.NewResolver(workspaceRoot))

		ops := []EditOperation{
			{
				Before:               "foo",
				After:                "baz",
				ExpectedReplacements: 1,
			},
			{
				Before: "",
				After:  "\nend",
			},
		}

		editReq := &EditFileRequest{Path: "test.txt", Operations: ops}
		resp, err := editTool.Run(context.Background(), editReq)
		if err != nil {
			t.Fatalf("Run failed: %v", err)
		}

		if resp.OperationsApplied != 2 {
			t.Errorf("expected 2 ops applied, got %d", resp.OperationsApplied)
		}

		final, _ := fs.ReadFileRange("/workspace/test.txt", 0, 0)
		expected := "baz\nbar\nend"
		if string(final) != expected {
			t.Errorf("expected content %q, got %q", expected, string(final))
		}
	})

	t.Run("append with count > 1 errors", func(t *testing.T) {
		fs := newMockFileSystemForWrite()
		checksumManager := newMockChecksumManagerForWrite()
		fs.createFile("/workspace/test.txt", []byte("start"), 0644)

		cfg := config.DefaultConfig()
		editTool := NewEditFileTool(fs, checksumManager, cfg, path.NewResolver(workspaceRoot))

		ops := []EditOperation{
			{
				Before:               "",
				After:                "tail",
				ExpectedReplacements: 2, // Should fail since only 1 place to append
			},
		}

		editReq := &EditFileRequest{Path: "test.txt", Operations: ops}
		_, err := editTool.Run(context.Background(), editReq)
		if err == nil || !errors.Is(err, ErrReplacementCountMismatch) {
			t.Errorf("expected ErrReplacementCountMismatch, got %v", err)
		}
	})
}
