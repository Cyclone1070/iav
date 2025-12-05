package tools

// TEST CONTRACT: do not modify without updating symlink safety spec
// These tests enforce the symlink safety guarantees for file operations.
// Any changes to these tests must be reviewed against the symlink safety specification.

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/Cyclone1070/iav/internal/tools/models"
	"github.com/Cyclone1070/iav/internal/tools/services"
)

func TestEditFile(t *testing.T) {
	workspaceRoot := "/workspace"
	maxFileSize := int64(1024 * 1024) // 1MB

	t.Run("conflict detection when cache checksum differs", func(t *testing.T) {
		fs := services.NewMockFileSystem(maxFileSize)
		checksumManager := services.NewChecksumManager()
		originalContent := []byte("original content")
		fs.CreateFile("/workspace/test.txt", originalContent, 0o644)

		// Read file to populate cache
		ctx := &models.WorkspaceContext{
			FS:              fs,
			BinaryDetector:  services.NewMockBinaryDetector(),
			ChecksumManager: checksumManager,
			MaxFileSize:     maxFileSize,
			WorkspaceRoot:   workspaceRoot,
		}

		_, err := ReadFile(context.Background(), ctx, models.ReadFileRequest{Path: "test.txt"})
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}

		// Modify file externally (simulate external change)
		modifiedContent := []byte("modified externally")
		fs.CreateFile("/workspace/test.txt", modifiedContent, 0o644)

		// Try to edit - should fail with conflict
		ops := []models.Operation{
			{
				Before:               "original content",
				After:                "new content",
				ExpectedReplacements: 1,
			},
		}

		_, err = EditFile(context.Background(), ctx, models.EditFileRequest{Path: "test.txt", Operations: ops})
		if err != models.ErrEditConflict {
			t.Errorf("expected ErrEditConflict, got %v", err)
		}
	})

	t.Run("multiple operations", func(t *testing.T) {
		fs := services.NewMockFileSystem(maxFileSize)
		checksumManager := services.NewChecksumManager()
		content := []byte("line1\nline2\nline3")
		fs.CreateFile("/workspace/test.txt", content, 0644)

		ctx := &models.WorkspaceContext{
			FS:              fs,
			BinaryDetector:  services.NewMockBinaryDetector(),
			ChecksumManager: checksumManager,
			MaxFileSize:     maxFileSize,
			WorkspaceRoot:   workspaceRoot,
		}

		// Read first to populate cache
		_, err := ReadFile(context.Background(), ctx, models.ReadFileRequest{Path: "test.txt"})
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}

		ops := []models.Operation{
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

		resp, err := EditFile(context.Background(), ctx, models.EditFileRequest{Path: "test.txt", Operations: ops})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if resp.OperationsApplied != 2 {
			t.Errorf("expected 2 operations applied, got %d", resp.OperationsApplied)
		}

		// Verify edits were applied
		fileContent, err := fs.ReadFileRange("/workspace/test.txt", 0, 0)
		if err != nil {
			t.Fatalf("failed to read edited file: %v", err)
		}

		result := string(fileContent)
		if !strings.Contains(result, "modified1") || !strings.Contains(result, "modified2") {
			t.Errorf("expected edits to be applied, got: %q", result)
		}
	})

	t.Run("snippet not found", func(t *testing.T) {
		fs := services.NewMockFileSystem(maxFileSize)
		checksumManager := services.NewChecksumManager()
		content := []byte("test content")
		fs.CreateFile("/workspace/test.txt", content, 0644)

		ctx := &models.WorkspaceContext{
			FS:              fs,
			BinaryDetector:  services.NewMockBinaryDetector(),
			ChecksumManager: checksumManager,
			MaxFileSize:     maxFileSize,
			WorkspaceRoot:   workspaceRoot,
		}

		_, err := ReadFile(context.Background(), ctx, models.ReadFileRequest{Path: "test.txt"})
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}

		ops := []models.Operation{
			{
				Before:               "nonexistent",
				After:                "replacement",
				ExpectedReplacements: 1,
			},
		}

		_, err = EditFile(context.Background(), ctx, models.EditFileRequest{Path: "test.txt", Operations: ops})
		if err != models.ErrSnippetNotFound {
			t.Errorf("expected ErrSnippetNotFound, got %v", err)
		}
	})

	t.Run("expected replacements mismatch", func(t *testing.T) {
		fs := services.NewMockFileSystem(maxFileSize)
		checksumManager := services.NewChecksumManager()
		content := []byte("test test test")
		fs.CreateFile("/workspace/test.txt", content, 0644)

		ctx := &models.WorkspaceContext{
			FS:              fs,
			BinaryDetector:  services.NewMockBinaryDetector(),
			ChecksumManager: checksumManager,
			MaxFileSize:     maxFileSize,
			WorkspaceRoot:   workspaceRoot,
		}

		_, err := ReadFile(context.Background(), ctx, models.ReadFileRequest{Path: "test.txt"})
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}

		ops := []models.Operation{
			{
				Before:               "test",
				After:                "replaced",
				ExpectedReplacements: 1, // But there are 3 occurrences
			},
		}

		_, err = EditFile(context.Background(), ctx, models.EditFileRequest{Path: "test.txt", Operations: ops})
		if err != models.ErrExpectedReplacementsMismatch {
			t.Errorf("expected ErrExpectedReplacementsMismatch, got %v", err)
		}
	})

	t.Run("default ExpectedReplacements to 1 when omitted", func(t *testing.T) {
		fs := services.NewMockFileSystem(maxFileSize)
		checksumManager := services.NewChecksumManager()
		content := []byte("replace me")
		fs.CreateFile("/workspace/test.txt", content, 0644)

		ctx := &models.WorkspaceContext{
			FS:              fs,
			BinaryDetector:  services.NewMockBinaryDetector(),
			ChecksumManager: checksumManager,
			MaxFileSize:     maxFileSize,
			WorkspaceRoot:   workspaceRoot,
		}

		_, err := ReadFile(context.Background(), ctx, models.ReadFileRequest{Path: "test.txt"})
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}

		ops := []models.Operation{
			{
				Before:               "replace me",
				After:                "replaced",
				ExpectedReplacements: 0, // Should default to 1
			},
		}

		resp, err := EditFile(context.Background(), ctx, models.EditFileRequest{Path: "test.txt", Operations: ops})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if resp.OperationsApplied != 1 {
			t.Errorf("expected 1 operation applied, got %d", resp.OperationsApplied)
		}

		// Verify edit was applied
		fileContent, err := fs.ReadFileRange("/workspace/test.txt", 0, 0)
		if err != nil {
			t.Fatalf("failed to read edited file: %v", err)
		}

		result := string(fileContent)
		if result != "replaced" {
			t.Errorf("expected content %q, got %q", "replaced", result)
		}
	})

	t.Run("default ExpectedReplacements fails on multiple matches", func(t *testing.T) {
		fs := services.NewMockFileSystem(maxFileSize)
		checksumManager := services.NewChecksumManager()
		content := []byte("test test")
		fs.CreateFile("/workspace/test.txt", content, 0644)

		ctx := &models.WorkspaceContext{
			FS:              fs,
			BinaryDetector:  services.NewMockBinaryDetector(),
			ChecksumManager: checksumManager,
			MaxFileSize:     maxFileSize,
			WorkspaceRoot:   workspaceRoot,
		}

		_, err := ReadFile(context.Background(), ctx, models.ReadFileRequest{Path: "test.txt"})
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}

		ops := []models.Operation{
			{
				Before:               "test",
				After:                "replaced",
				ExpectedReplacements: 0, // Defaults to 1, but there are 2 occurrences
			},
		}

		_, err = EditFile(context.Background(), ctx, models.EditFileRequest{Path: "test.txt", Operations: ops})
		if err != models.ErrExpectedReplacementsMismatch {
			t.Errorf("expected ErrExpectedReplacementsMismatch, got %v", err)
		}
	})

	t.Run("binary file rejection", func(t *testing.T) {
		fs := services.NewMockFileSystem(maxFileSize)
		checksumManager := services.NewChecksumManager()
		detector := services.NewMockBinaryDetector()

		// Create file with null bytes (actual binary content)
		content := []byte{0x00, 0x01, 0x02, 't', 'e', 's', 't'}
		fs.CreateFile("/workspace/binary.bin", content, 0644)

		ctx := &models.WorkspaceContext{
			FS:              fs,
			BinaryDetector:  detector,
			ChecksumManager: checksumManager,
			MaxFileSize:     maxFileSize,
			WorkspaceRoot:   workspaceRoot,
		}

		ops := []models.Operation{
			{
				Before:               "test",
				After:                "replaced",
				ExpectedReplacements: 1,
			},
		}

		_, err := EditFile(context.Background(), ctx, models.EditFileRequest{Path: "binary.bin", Operations: ops})
		if err != models.ErrBinaryFile {
			t.Errorf("expected ErrBinaryFile, got %v", err)
		}
	})

	t.Run("permission preservation", func(t *testing.T) {
		fs := services.NewMockFileSystem(maxFileSize)
		checksumManager := services.NewChecksumManager()
		originalPerm := os.FileMode(0755)
		content := []byte("test content")
		fs.CreateFile("/workspace/test.txt", content, originalPerm)

		ctx := &models.WorkspaceContext{
			FS:              fs,
			BinaryDetector:  services.NewMockBinaryDetector(),
			ChecksumManager: checksumManager,
			MaxFileSize:     maxFileSize,
			WorkspaceRoot:   workspaceRoot,
		}

		_, err := ReadFile(context.Background(), ctx, models.ReadFileRequest{Path: "test.txt"})
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}

		ops := []models.Operation{
			{
				Before:               "test",
				After:                "modified",
				ExpectedReplacements: 1,
			},
		}

		_, err = EditFile(context.Background(), ctx, models.EditFileRequest{Path: "test.txt", Operations: ops})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify permissions preserved
		info, err := fs.Stat("/workspace/test.txt")
		if err != nil {
			t.Fatalf("failed to stat file: %v", err)
		}

		if info.Mode().Perm() != originalPerm {
			t.Errorf("expected permissions %o, got %o", originalPerm, info.Mode().Perm())
		}
	})

	t.Run("empty Before string", func(t *testing.T) {
		fs := services.NewMockFileSystem(maxFileSize)
		checksumManager := services.NewChecksumManager()
		content := []byte("test content")
		fs.CreateFile("/workspace/test.txt", content, 0644)

		ctx := &models.WorkspaceContext{
			FS:              fs,
			BinaryDetector:  services.NewMockBinaryDetector(),
			ChecksumManager: checksumManager,
			MaxFileSize:     maxFileSize,
			WorkspaceRoot:   workspaceRoot,
		}

		_, err := ReadFile(context.Background(), ctx, models.ReadFileRequest{Path: "test.txt"})
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}

		ops := []models.Operation{
			{
				Before:               "",
				After:                "replacement",
				ExpectedReplacements: 1,
			},
		}

		_, err = EditFile(context.Background(), ctx, models.EditFileRequest{Path: "test.txt", Operations: ops})
		if err == nil {
			t.Error("expected error for empty Before string")
		}
	})

	t.Run("file not found", func(t *testing.T) {
		fs := services.NewMockFileSystem(maxFileSize)
		checksumManager := services.NewChecksumManager()
		ctx := &models.WorkspaceContext{
			FS:              fs,
			BinaryDetector:  services.NewMockBinaryDetector(),
			ChecksumManager: checksumManager,
			MaxFileSize:     maxFileSize,
			WorkspaceRoot:   workspaceRoot,
		}

		ops := []models.Operation{
			{
				Before:               "test",
				After:                "replacement",
				ExpectedReplacements: 1,
			},
		}

		_, err := EditFile(context.Background(), ctx, models.EditFileRequest{Path: "nonexistent.txt", Operations: ops})
		if err != models.ErrFileMissing {
			t.Errorf("expected ErrFileMissing, got %v", err)
		}
	})

	t.Run("large content after edit", func(t *testing.T) {
		fs := services.NewMockFileSystem(maxFileSize)
		checksumManager := services.NewChecksumManager()
		// Create file with unique marker just under limit
		prefix := []byte("UNIQUE_MARKER_12345")
		middle := make([]byte, int(maxFileSize)-100-len(prefix))
		for i := range middle {
			middle[i] = 'A'
		}
		content := append(prefix, middle...)
		fs.CreateFile("/workspace/test.txt", content, 0644)

		ctx := &models.WorkspaceContext{
			FS:              fs,
			BinaryDetector:  services.NewMockBinaryDetector(),
			ChecksumManager: checksumManager,
			MaxFileSize:     maxFileSize,
			WorkspaceRoot:   workspaceRoot,
		}

		_, err := ReadFile(context.Background(), ctx, models.ReadFileRequest{Path: "test.txt"})
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}

		// Replace unique marker with content that exceeds limit
		largeReplacement := make([]byte, 200)
		for i := range largeReplacement {
			largeReplacement[i] = 'B'
		}

		ops := []models.Operation{
			{
				Before:               string(prefix),
				After:                string(largeReplacement),
				ExpectedReplacements: 1,
			},
		}

		_, err = EditFile(context.Background(), ctx, models.EditFileRequest{Path: "test.txt", Operations: ops})
		if err != models.ErrTooLarge {
			t.Errorf("expected ErrTooLarge, got %v", err)
		}
	})

	t.Run("race condition detection - file modified between read and write", func(t *testing.T) {
		fs := services.NewMockFileSystem(maxFileSize)
		checksumManager := services.NewChecksumManager()
		originalContent := []byte("original content")
		fs.CreateFile("/workspace/test.txt", originalContent, 0o644)

		ctx := &models.WorkspaceContext{
			FS:              fs,
			BinaryDetector:  services.NewMockBinaryDetector(),
			ChecksumManager: checksumManager,
			MaxFileSize:     maxFileSize,
			WorkspaceRoot:   workspaceRoot,
		}

		// Read file to populate cache
		_, err := ReadFile(context.Background(), ctx, models.ReadFileRequest{Path: "test.txt"})
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}

		// Modify file externally (simulate concurrent modification)
		modifiedContent := []byte("modified externally")
		fs.CreateFile("/workspace/test.txt", modifiedContent, 0o644)

		// Try to edit - should fail with conflict due to revalidation
		ops := []models.Operation{
			{
				Before:               "original content",
				After:                "new content",
				ExpectedReplacements: 1,
			},
		}

		_, err = EditFile(context.Background(), ctx, models.EditFileRequest{Path: "test.txt", Operations: ops})
		if err != models.ErrEditConflict {
			t.Errorf("expected ErrEditConflict due to race condition, got %v", err)
		}
	})

	t.Run("edit through symlink chain inside workspace", func(t *testing.T) {
		fs := services.NewMockFileSystem(maxFileSize)
		checksumManager := services.NewChecksumManager()
		// Create symlink chain: link1 -> link2 -> target.txt
		fs.CreateSymlink("/workspace/link1", "/workspace/link2")
		fs.CreateSymlink("/workspace/link2", "/workspace/target.txt")
		content := []byte("original content")
		fs.CreateFile("/workspace/target.txt", content, 0644)

		ctx := &models.WorkspaceContext{
			FS:              fs,
			BinaryDetector:  services.NewMockBinaryDetector(),
			ChecksumManager: checksumManager,
			MaxFileSize:     maxFileSize,
			WorkspaceRoot:   workspaceRoot,
		}

		// Read first to populate cache
		_, err := ReadFile(context.Background(), ctx, models.ReadFileRequest{Path: "link1"})
		if err != nil {
			t.Fatalf("failed to read file through symlink chain: %v", err)
		}

		ops := []models.Operation{
			{
				Before:               "original content",
				After:                "modified content",
				ExpectedReplacements: 1,
			},
		}

		resp, err := EditFile(context.Background(), ctx, models.EditFileRequest{Path: "link1", Operations: ops})
		if err != nil {
			t.Fatalf("unexpected error editing through symlink chain: %v", err)
		}

		// Verify edit was applied at resolved location
		fileContent, err := fs.ReadFileRange("/workspace/target.txt", 0, 0)
		if err != nil {
			t.Fatalf("failed to read edited file: %v", err)
		}
		if string(fileContent) != "modified content" {
			t.Errorf("expected content %q, got %q", "modified content", string(fileContent))
		}

		// Verify response has correct absolute path
		if resp.AbsolutePath != "/workspace/target.txt" {
			t.Errorf("expected absolute path /workspace/target.txt, got %s", resp.AbsolutePath)
		}
	})

	t.Run("edit through symlink chain escaping workspace", func(t *testing.T) {
		fs := services.NewMockFileSystem(maxFileSize)
		checksumManager := services.NewChecksumManager()
		// Create chain: link1 -> link2 -> /tmp/outside/file.txt
		fs.CreateSymlink("/workspace/link1", "/workspace/link2")
		fs.CreateSymlink("/workspace/link2", "/tmp/outside/file.txt")
		fs.CreateDir("/tmp/outside")
		fs.CreateFile("/tmp/outside/file.txt", []byte("content"), 0644)

		ctx := &models.WorkspaceContext{
			FS:              fs,
			BinaryDetector:  services.NewMockBinaryDetector(),
			ChecksumManager: checksumManager,
			MaxFileSize:     maxFileSize,
			WorkspaceRoot:   workspaceRoot,
		}

		ops := []models.Operation{
			{
				Before:               "content",
				After:                "modified",
				ExpectedReplacements: 1,
			},
		}

		// Try to edit through escaping chain - should fail
		_, err := EditFile(context.Background(), ctx, models.EditFileRequest{Path: "link1", Operations: ops})
		if err != models.ErrOutsideWorkspace {
			t.Errorf("expected ErrOutsideWorkspace for escaping symlink chain, got %v", err)
		}
	})
}
