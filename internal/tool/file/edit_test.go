package file

// This file contains edit file tests.
// Mocks are defined in write_test.go and shared across all test files in this package.

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/Cyclone1070/iav/internal/config"
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

		readTool := NewReadFileTool(fs, newMockBinaryDetectorForWrite(), checksumManager, cfg, workspaceRoot)
		editTool := NewEditFileTool(fs, newMockBinaryDetectorForWrite(), checksumManager, cfg, workspaceRoot)

		// Read file to populate cache
		readReq, _ := NewReadFileRequest(ReadFileDTO{Path: "test.txt"}, cfg, workspaceRoot, fs)
		_, err := readTool.Run(context.Background(), readReq)
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}

		// Modify file externally (simulate external change)
		modifiedContent := []byte("modified externally")
		fs.createFile("/workspace/test.txt", modifiedContent, 0644)

		// Try to edit - should fail with conflict
		ops := []Operation{
			{
				Before:               "original content",
				After:                "new content",
				ExpectedReplacements: 1,
			},
		}

		editReq, err := NewEditFileRequest(EditFileDTO{Path: "test.txt", Operations: ops}, cfg, workspaceRoot, fs)
		if err != nil {
			t.Fatalf("failed to create edit request: %v", err)
		}

		_, err = editTool.Run(context.Background(), editReq)
		var editConflictErr *EditConflictError
		if err == nil || !errors.As(err, &editConflictErr) {
			t.Errorf("expected EditConflictError, got %v", err)
		}
	})

	t.Run("multiple operations", func(t *testing.T) {
		fs := newMockFileSystemForWrite()
		checksumManager := newMockChecksumManagerForWrite()
		content := []byte("line1\nline2\nline3")
		fs.createFile("/workspace/test.txt", content, 0644)

		cfg := config.DefaultConfig()
		cfg.Tools.MaxFileSize = maxFileSize

		readTool := NewReadFileTool(fs, newMockBinaryDetectorForWrite(), checksumManager, cfg, workspaceRoot)
		editTool := NewEditFileTool(fs, newMockBinaryDetectorForWrite(), checksumManager, cfg, workspaceRoot)

		// Read first to populate cache
		readReq, _ := NewReadFileRequest(ReadFileDTO{Path: "test.txt"}, cfg, workspaceRoot, fs)
		_, err := readTool.Run(context.Background(), readReq)
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}

		ops := []Operation{
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

		editReq, err := NewEditFileRequest(EditFileDTO{Path: "test.txt", Operations: ops}, cfg, workspaceRoot, fs)
		if err != nil {
			t.Fatalf("failed to create edit request: %v", err)
		}

		resp, err := editTool.Run(context.Background(), editReq)
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
		fs := newMockFileSystemForWrite()
		checksumManager := newMockChecksumManagerForWrite()
		content := []byte("test content")
		fs.createFile("/workspace/test.txt", content, 0644)

		cfg := config.DefaultConfig()

		readTool := NewReadFileTool(fs, newMockBinaryDetectorForWrite(), checksumManager, cfg, workspaceRoot)
		editTool := NewEditFileTool(fs, newMockBinaryDetectorForWrite(), checksumManager, cfg, workspaceRoot)

		readReq, _ := NewReadFileRequest(ReadFileDTO{Path: "test.txt"}, cfg, workspaceRoot, fs)
		_, err := readTool.Run(context.Background(), readReq)
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}

		ops := []Operation{
			{
				Before:               "nonexistent",
				After:                "replacement",
				ExpectedReplacements: 1,
			},
		}

		editReq, err := NewEditFileRequest(EditFileDTO{Path: "test.txt", Operations: ops}, cfg, workspaceRoot, fs)
		if err != nil {
			t.Fatalf("failed to create edit request: %v", err)
		}

		_, err = editTool.Run(context.Background(), editReq)
		var snippetNotFoundErr *SnippetNotFoundError
		if err == nil || !errors.As(err, &snippetNotFoundErr) {
			t.Errorf("expected SnippetNotFoundError, got %v", err)
		}
	})

	t.Run("expected replacements mismatch", func(t *testing.T) {
		fs := newMockFileSystemForWrite()
		checksumManager := newMockChecksumManagerForWrite()
		content := []byte("test test test")
		fs.createFile("/workspace/test.txt", content, 0644)

		cfg := config.DefaultConfig()
		readTool := NewReadFileTool(fs, newMockBinaryDetectorForWrite(), checksumManager, cfg, workspaceRoot)
		editTool := NewEditFileTool(fs, newMockBinaryDetectorForWrite(), checksumManager, cfg, workspaceRoot)

		readReq, _ := NewReadFileRequest(ReadFileDTO{Path: "test.txt"}, cfg, workspaceRoot, fs)
		_, err := readTool.Run(context.Background(), readReq)
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}

		ops := []Operation{
			{
				Before:               "test",
				After:                "replaced",
				ExpectedReplacements: 1, // But there are 3 occurrences
			},
		}

		editReq, err := NewEditFileRequest(EditFileDTO{Path: "test.txt", Operations: ops}, cfg, workspaceRoot, fs)
		if err != nil {
			t.Fatalf("failed to create edit request: %v", err)
		}

		_, err = editTool.Run(context.Background(), editReq)
		var mismatchErr *ReplacementMismatchError
		if err == nil || !errors.As(err, &mismatchErr) {
			t.Errorf("expected ReplacementMismatchError, got %v", err)
		}
	})

	t.Run("binary file rejection", func(t *testing.T) {
		fs := newMockFileSystemForWrite()
		checksumManager := newMockChecksumManagerForWrite()
		detector := newMockBinaryDetectorForWrite()
		detector.isBinaryFunc = func(content []byte) bool {
			return true
		}

		// Create file with null bytes (actual binary content)
		content := []byte{0x00, 0x01, 0x02, 't', 'e', 's', 't'}
		fs.createFile("/workspace/binary.bin", content, 0644)

		cfg := config.DefaultConfig()
		editTool := NewEditFileTool(fs, detector, checksumManager, cfg, workspaceRoot)

		ops := []Operation{
			{
				Before:               "test",
				After:                "replaced",
				ExpectedReplacements: 1,
			},
		}

		editReq, err := NewEditFileRequest(EditFileDTO{Path: "binary.bin", Operations: ops}, cfg, workspaceRoot, fs)
		if err != nil {
			t.Fatalf("failed to create edit request: %v", err)
		}

		_, err = editTool.Run(context.Background(), editReq)
		var binaryErr *BinaryFileError
		if err == nil || !errors.As(err, &binaryErr) {
			t.Errorf("expected BinaryFileError, got %v", err)
		}
	})

	t.Run("permission preservation", func(t *testing.T) {
		fs := newMockFileSystemForWrite()
		checksumManager := newMockChecksumManagerForWrite()
		originalPerm := os.FileMode(0755)
		content := []byte("test content")
		fs.createFile("/workspace/test.txt", content, originalPerm)

		cfg := config.DefaultConfig()
		readTool := NewReadFileTool(fs, newMockBinaryDetectorForWrite(), checksumManager, cfg, workspaceRoot)
		editTool := NewEditFileTool(fs, newMockBinaryDetectorForWrite(), checksumManager, cfg, workspaceRoot)

		readReq, _ := NewReadFileRequest(ReadFileDTO{Path: "test.txt"}, cfg, workspaceRoot, fs)
		_, err := readTool.Run(context.Background(), readReq)
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}

		ops := []Operation{
			{
				Before:               "test",
				After:                "modified",
				ExpectedReplacements: 1,
			},
		}

		editReq, err := NewEditFileRequest(EditFileDTO{Path: "test.txt", Operations: ops}, cfg, workspaceRoot, fs)
		if err != nil {
			t.Fatalf("failed to create edit request: %v", err)
		}

		_, err = editTool.Run(context.Background(), editReq)
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

	t.Run("file not found", func(t *testing.T) {
		fs := newMockFileSystemForWrite()
		checksumManager := newMockChecksumManagerForWrite()

		cfg := config.DefaultConfig()
		editTool := NewEditFileTool(fs, newMockBinaryDetectorForWrite(), checksumManager, cfg, workspaceRoot)

		ops := []Operation{
			{
				Before:               "test",
				After:                "replacement",
				ExpectedReplacements: 1,
			},
		}

		editReq, err := NewEditFileRequest(EditFileDTO{Path: "nonexistent.txt", Operations: ops}, cfg, workspaceRoot, fs)
		if err != nil {
			t.Fatalf("failed to create edit request: %v", err)
		}

		_, err = editTool.Run(context.Background(), editReq)
		var fileMissingErr *FileMissingError
		if err == nil || !errors.As(err, &fileMissingErr) {
			t.Errorf("expected FileMissingError, got %v", err)
		}
	})

	t.Run("default ExpectedReplacements to 1 when omitted", func(t *testing.T) {
		fs := newMockFileSystemForWrite()
		checksumManager := newMockChecksumManagerForWrite()
		content := []byte("replace me")
		fs.createFile("/workspace/test.txt", content, 0644)

		cfg := config.DefaultConfig()
		readTool := NewReadFileTool(fs, newMockBinaryDetectorForWrite(), checksumManager, cfg, workspaceRoot)
		editTool := NewEditFileTool(fs, newMockBinaryDetectorForWrite(), checksumManager, cfg, workspaceRoot)

		readReq, _ := NewReadFileRequest(ReadFileDTO{Path: "test.txt"}, cfg, workspaceRoot, fs)
		_, err := readTool.Run(context.Background(), readReq)
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}

		ops := []Operation{
			{
				Before:               "replace me",
				After:                "replaced",
				ExpectedReplacements: 0, // Should default to 1
			},
		}

		editReq, err := NewEditFileRequest(EditFileDTO{Path: "test.txt", Operations: ops}, cfg, workspaceRoot, fs)
		if err != nil {
			t.Fatalf("failed to create edit request: %v", err)
		}

		resp, err := editTool.Run(context.Background(), editReq)
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
		fs := newMockFileSystemForWrite()
		checksumManager := newMockChecksumManagerForWrite()
		content := []byte("test test")
		fs.createFile("/workspace/test.txt", content, 0644)

		cfg := config.DefaultConfig()
		readTool := NewReadFileTool(fs, newMockBinaryDetectorForWrite(), checksumManager, cfg, workspaceRoot)
		editTool := NewEditFileTool(fs, newMockBinaryDetectorForWrite(), checksumManager, cfg, workspaceRoot)

		readReq, _ := NewReadFileRequest(ReadFileDTO{Path: "test.txt"}, cfg, workspaceRoot, fs)
		_, err := readTool.Run(context.Background(), readReq)
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}

		ops := []Operation{
			{
				Before:               "test",
				After:                "replaced",
				ExpectedReplacements: 0, // Defaults to 1, but there are 2 occurrences
			},
		}

		editReq, err := NewEditFileRequest(EditFileDTO{Path: "test.txt", Operations: ops}, cfg, workspaceRoot, fs)
		if err != nil {
			t.Fatalf("failed to create edit request: %v", err)
		}

		_, err = editTool.Run(context.Background(), editReq)
		var mismatchErr *ReplacementMismatchError
		if err == nil || !errors.As(err, &mismatchErr) {
			t.Errorf("expected ReplacementMismatchError, got %v", err)
		}
	})

	t.Run("large content after edit", func(t *testing.T) {
		fs := newMockFileSystemForWrite()
		checksumManager := newMockChecksumManagerForWrite()
		// Create file with unique marker just under limit
		prefix := []byte("UNIQUE_MARKER_12345")
		middle := make([]byte, int(maxFileSize)-100-len(prefix))
		for i := range middle {
			middle[i] = 'A'
		}
		content := append(prefix, middle...)
		fs.createFile("/workspace/test.txt", content, 0644)

		cfg := config.DefaultConfig()
		cfg.Tools.MaxFileSize = maxFileSize

		readTool := NewReadFileTool(fs, newMockBinaryDetectorForWrite(), checksumManager, cfg, workspaceRoot)
		editTool := NewEditFileTool(fs, newMockBinaryDetectorForWrite(), checksumManager, cfg, workspaceRoot)

		readReq, _ := NewReadFileRequest(ReadFileDTO{Path: "test.txt"}, cfg, workspaceRoot, fs)
		_, err := readTool.Run(context.Background(), readReq)
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}

		// Replace unique marker with content that exceeds limit
		largeReplacement := make([]byte, 200)
		for i := range largeReplacement {
			largeReplacement[i] = 'B'
		}

		ops := []Operation{
			{
				Before:               string(prefix),
				After:                string(largeReplacement),
				ExpectedReplacements: 1,
			},
		}

		editReq, err := NewEditFileRequest(EditFileDTO{Path: "test.txt", Operations: ops}, cfg, workspaceRoot, fs)
		if err != nil {
			t.Fatalf("failed to create edit request: %v", err)
		}

		_, err = editTool.Run(context.Background(), editReq)
		var tooLargeErr *TooLargeError
		if err == nil || !errors.As(err, &tooLargeErr) {
			t.Errorf("expected TooLargeError, got %v", err)
		}
	})

	t.Run("race condition detection - file modified between read and write", func(t *testing.T) {
		fs := newMockFileSystemForWrite()
		checksumManager := newMockChecksumManagerForWrite()
		originalContent := []byte("original content")
		fs.createFile("/workspace/test.txt", originalContent, 0644)

		cfg := config.DefaultConfig()
		readTool := NewReadFileTool(fs, newMockBinaryDetectorForWrite(), checksumManager, cfg, workspaceRoot)
		editTool := NewEditFileTool(fs, newMockBinaryDetectorForWrite(), checksumManager, cfg, workspaceRoot)

		// Read file to populate cache
		readReq, _ := NewReadFileRequest(ReadFileDTO{Path: "test.txt"}, cfg, workspaceRoot, fs)
		_, err := readTool.Run(context.Background(), readReq)
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}

		// Modify file externally (simulate concurrent modification)
		modifiedContent := []byte("modified externally")
		fs.createFile("/workspace/test.txt", modifiedContent, 0644)

		// Try to edit - should fail with conflict due to revalidation
		ops := []Operation{
			{
				Before:               "original content",
				After:                "new content",
				ExpectedReplacements: 1,
			},
		}

		editReq, err := NewEditFileRequest(EditFileDTO{Path: "test.txt", Operations: ops}, cfg, workspaceRoot, fs)
		if err != nil {
			t.Fatalf("failed to create edit request: %v", err)
		}

		_, err = editTool.Run(context.Background(), editReq)
		var editConflictErr *EditConflictError
		if err == nil || !errors.As(err, &editConflictErr) {
			t.Errorf("expected EditConflictError due to race condition, got %v", err)
		}
	})

	t.Run("edit through symlink chain inside workspace", func(t *testing.T) {
		fs := newMockFileSystemForWrite()
		checksumManager := newMockChecksumManagerForWrite()
		// Create symlink chain: link1 -> link2 -> target.txt
		fs.createSymlink("/workspace/link1", "/workspace/link2")
		fs.createSymlink("/workspace/link2", "/workspace/target.txt")
		content := []byte("original content")
		fs.createFile("/workspace/target.txt", content, 0644)

		cfg := config.DefaultConfig()
		readTool := NewReadFileTool(fs, newMockBinaryDetectorForWrite(), checksumManager, cfg, workspaceRoot)
		editTool := NewEditFileTool(fs, newMockBinaryDetectorForWrite(), checksumManager, cfg, workspaceRoot)

		// Read first to populate cache
		readReq, _ := NewReadFileRequest(ReadFileDTO{Path: "link1"}, cfg, workspaceRoot, fs)
		_, err := readTool.Run(context.Background(), readReq)
		if err != nil {
			t.Fatalf("failed to read file through symlink chain: %v", err)
		}

		ops := []Operation{
			{
				Before:               "original content",
				After:                "modified content",
				ExpectedReplacements: 1,
			},
		}

		editReq, err := NewEditFileRequest(EditFileDTO{Path: "link1", Operations: ops}, cfg, workspaceRoot, fs)
		if err != nil {
			t.Fatalf("failed to create edit request: %v", err)
		}

		resp, err := editTool.Run(context.Background(), editReq)
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
		fs := newMockFileSystemForWrite()
		checksumManager := newMockChecksumManagerForWrite()
		// Create chain: link1 -> link2 -> /tmp/outside/file.txt
		fs.createSymlink("/workspace/link1", "/workspace/link2")
		fs.createSymlink("/workspace/link2", "/tmp/outside/file.txt")
		fs.createDir("/tmp/outside")
		fs.createFile("/tmp/outside/file.txt", []byte("content"), 0644)

		cfg := config.DefaultConfig()
		editTool := NewEditFileTool(fs, newMockBinaryDetectorForWrite(), checksumManager, cfg, workspaceRoot)

		ops := []Operation{
			{
				Before:               "content",
				After:                "modified",
				ExpectedReplacements: 1,
			},
		}

		// Try to edit through escaping chain - should fail
		editReq, err := NewEditFileRequest(EditFileDTO{Path: "link1", Operations: ops}, cfg, workspaceRoot, fs)

		if err == nil {
			_, err = editTool.Run(context.Background(), editReq)
		}

		type outsideWorkspace interface{ OutsideWorkspace() bool }
		var targetErr outsideWorkspace
		if err == nil || !errors.As(err, &targetErr) || !targetErr.OutsideWorkspace() {
			t.Errorf("expected OutsideWorkspace error for escaping symlink chain, got %v", err)
		}
	})
}
