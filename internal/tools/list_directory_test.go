package tools

import (
	"os"
	"testing"

	"github.com/Cyclone1070/deployforme/internal/tools/models"
	"github.com/Cyclone1070/deployforme/internal/tools/services"
)

func TestListDirectory(t *testing.T) {
	workspaceRoot := "/workspace"
	maxFileSize := int64(1024 * 1024) // 1MB

	t.Run("list workspace root with mixed files and directories", func(t *testing.T) {
		fs := services.NewMockFileSystem(maxFileSize)
		fs.CreateDir("/workspace")
		fs.CreateFile("/workspace/file1.txt", []byte("content1"), 0o644)
		fs.CreateFile("/workspace/file2.txt", []byte("content2"), 0o644)
		fs.CreateDir("/workspace/subdir1")
		fs.CreateDir("/workspace/subdir2")

		ctx := &models.WorkspaceContext{
			FS:              fs,
			BinaryDetector:  services.NewMockBinaryDetector(),
			ChecksumManager: services.NewChecksumManager(),
			MaxFileSize:     maxFileSize,
			WorkspaceRoot:   workspaceRoot,
		}

		resp, err := ListDirectory(ctx, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if resp.DirectoryPath != "" {
			t.Errorf("expected DirectoryPath to be empty for workspace root, got %q", resp.DirectoryPath)
		}

		if len(resp.Entries) != 4 {
			t.Fatalf("expected 4 entries, got %d", len(resp.Entries))
		}

		// Verify sorting: directories first, then files alphabetically
		expectedOrder := []struct {
			name  string
			isDir bool
		}{
			{"subdir1", true},
			{"subdir2", true},
			{"file1.txt", false},
			{"file2.txt", false},
		}

		for i, expected := range expectedOrder {
			entry := resp.Entries[i]
			if entry.RelativePath != expected.name {
				t.Errorf("entry %d: expected RelativePath %q, got %q", i, expected.name, entry.RelativePath)
			}
			if entry.IsDir != expected.isDir {
				t.Errorf("entry %d: expected IsDir %v, got %v", i, expected.isDir, entry.IsDir)
			}
		}
	})

	t.Run("list nested directory", func(t *testing.T) {
		fs := services.NewMockFileSystem(maxFileSize)
		fs.CreateDir("/workspace")
		fs.CreateDir("/workspace/src")
		fs.CreateFile("/workspace/src/main.go", []byte("package main"), 0o644)
		fs.CreateFile("/workspace/src/utils.go", []byte("package main"), 0o644)
		fs.CreateDir("/workspace/src/internal")

		ctx := &models.WorkspaceContext{
			FS:              fs,
			BinaryDetector:  services.NewMockBinaryDetector(),
			ChecksumManager: services.NewChecksumManager(),
			MaxFileSize:     maxFileSize,
			WorkspaceRoot:   workspaceRoot,
		}

		resp, err := ListDirectory(ctx, "src")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if resp.DirectoryPath != "src" {
			t.Errorf("expected DirectoryPath 'src', got %q", resp.DirectoryPath)
		}

		if len(resp.Entries) != 3 {
			t.Fatalf("expected 3 entries, got %d", len(resp.Entries))
		}

		// Verify relative paths are correct (relative to workspace root)
		// ListDirectory already sorts entries, so we can compare directly
		expectedPaths := []string{"src/internal", "src/main.go", "src/utils.go"}
		for i, expected := range expectedPaths {
			if resp.Entries[i].RelativePath != expected {
				t.Errorf("entry %d: expected path %q, got %q", i, expected, resp.Entries[i].RelativePath)
			}
		}
	})

	t.Run("list empty directory", func(t *testing.T) {
		fs := services.NewMockFileSystem(maxFileSize)
		fs.CreateDir("/workspace")
		fs.CreateDir("/workspace/empty")

		ctx := &models.WorkspaceContext{
			FS:              fs,
			BinaryDetector:  services.NewMockBinaryDetector(),
			ChecksumManager: services.NewChecksumManager(),
			MaxFileSize:     maxFileSize,
			WorkspaceRoot:   workspaceRoot,
		}

		resp, err := ListDirectory(ctx, "empty")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(resp.Entries) != 0 {
			t.Errorf("expected 0 entries, got %d", len(resp.Entries))
		}
	})

	t.Run("path resolves to file not directory", func(t *testing.T) {
		fs := services.NewMockFileSystem(maxFileSize)
		fs.CreateDir("/workspace")
		fs.CreateFile("/workspace/file.txt", []byte("content"), 0o644)

		ctx := &models.WorkspaceContext{
			FS:              fs,
			BinaryDetector:  services.NewMockBinaryDetector(),
			ChecksumManager: services.NewChecksumManager(),
			MaxFileSize:     maxFileSize,
			WorkspaceRoot:   workspaceRoot,
		}

		_, err := ListDirectory(ctx, "file.txt")
		if err == nil {
			t.Fatal("expected error when listing a file, got nil")
		}

		// Should return a descriptive error
		if err.Error() == "" {
			t.Error("expected non-empty error message")
		}
	})

	t.Run("path outside workspace", func(t *testing.T) {
		fs := services.NewMockFileSystem(maxFileSize)
		fs.CreateDir("/workspace")
		fs.CreateDir("/outside")

		ctx := &models.WorkspaceContext{
			FS:              fs,
			BinaryDetector:  services.NewMockBinaryDetector(),
			ChecksumManager: services.NewChecksumManager(),
			MaxFileSize:     maxFileSize,
			WorkspaceRoot:   workspaceRoot,
		}

		_, err := ListDirectory(ctx, "../outside")
		if err != models.ErrOutsideWorkspace {
			t.Errorf("expected ErrOutsideWorkspace, got %v", err)
		}
	})

	t.Run("directory does not exist", func(t *testing.T) {
		fs := services.NewMockFileSystem(maxFileSize)
		fs.CreateDir("/workspace")

		ctx := &models.WorkspaceContext{
			FS:              fs,
			BinaryDetector:  services.NewMockBinaryDetector(),
			ChecksumManager: services.NewChecksumManager(),
			MaxFileSize:     maxFileSize,
			WorkspaceRoot:   workspaceRoot,
		}

		_, err := ListDirectory(ctx, "nonexistent")
		if err == nil {
			t.Fatal("expected error for nonexistent directory, got nil")
		}

		// Should propagate filesystem error
		if err != models.ErrFileMissing && err.Error() == "" {
			t.Errorf("expected meaningful error, got %v", err)
		}
	})

	t.Run("filesystem error propagation", func(t *testing.T) {
		fs := services.NewMockFileSystem(maxFileSize)
		fs.CreateDir("/workspace")
		fs.CreateDir("/workspace/testdir")
		fs.SetOperationError("ListDir", os.ErrPermission)

		ctx := &models.WorkspaceContext{
			FS:              fs,
			BinaryDetector:  services.NewMockBinaryDetector(),
			ChecksumManager: services.NewChecksumManager(),
			MaxFileSize:     maxFileSize,
			WorkspaceRoot:   workspaceRoot,
		}

		_, err := ListDirectory(ctx, "testdir")
		if err == nil {
			t.Fatal("expected error from filesystem, got nil")
		}

		// Error should be wrapped but should contain the original error
		if err.Error() == "" {
			t.Error("expected non-empty error message")
		}
		// Check that the error message contains permission-related text
		if err.Error() == "permission denied" || err.Error() == "failed to list directory: permission denied" {
			// This is acceptable - error is propagated
		} else {
			t.Errorf("expected error related to permission, got %v", err)
		}
	})

	t.Run("relative path input", func(t *testing.T) {
		fs := services.NewMockFileSystem(maxFileSize)
		fs.CreateDir("/workspace")
		fs.CreateDir("/workspace/src")
		fs.CreateFile("/workspace/src/file.txt", []byte("content"), 0o644)

		ctx := &models.WorkspaceContext{
			FS:              fs,
			BinaryDetector:  services.NewMockBinaryDetector(),
			ChecksumManager: services.NewChecksumManager(),
			MaxFileSize:     maxFileSize,
			WorkspaceRoot:   workspaceRoot,
		}

		resp, err := ListDirectory(ctx, "src")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if resp.DirectoryPath != "src" {
			t.Errorf("expected DirectoryPath 'src', got %q", resp.DirectoryPath)
		}
	})

	t.Run("absolute path input", func(t *testing.T) {
		fs := services.NewMockFileSystem(maxFileSize)
		fs.CreateDir("/workspace")
		fs.CreateDir("/workspace/src")
		fs.CreateFile("/workspace/src/file.txt", []byte("content"), 0o644)

		ctx := &models.WorkspaceContext{
			FS:              fs,
			BinaryDetector:  services.NewMockBinaryDetector(),
			ChecksumManager: services.NewChecksumManager(),
			MaxFileSize:     maxFileSize,
			WorkspaceRoot:   workspaceRoot,
		}

		resp, err := ListDirectory(ctx, "/workspace/src")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if resp.DirectoryPath != "src" {
			t.Errorf("expected DirectoryPath 'src', got %q", resp.DirectoryPath)
		}
	})

	t.Run("dot path alias for workspace root", func(t *testing.T) {
		fs := services.NewMockFileSystem(maxFileSize)
		fs.CreateDir("/workspace")
		fs.CreateFile("/workspace/file.txt", []byte("content"), 0o644)

		ctx := &models.WorkspaceContext{
			FS:              fs,
			BinaryDetector:  services.NewMockBinaryDetector(),
			ChecksumManager: services.NewChecksumManager(),
			MaxFileSize:     maxFileSize,
			WorkspaceRoot:   workspaceRoot,
		}

		resp, err := ListDirectory(ctx, ".")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if resp.DirectoryPath != "" {
			t.Errorf("expected empty DirectoryPath for workspace root, got %q", resp.DirectoryPath)
		}
	})

	t.Run("verify entry metadata correctness", func(t *testing.T) {
		fs := services.NewMockFileSystem(maxFileSize)
		fs.CreateDir("/workspace")
		fs.CreateFile("/workspace/file.txt", []byte("hello world"), 0o644)
		fs.CreateDir("/workspace/subdir")

		ctx := &models.WorkspaceContext{
			FS:              fs,
			BinaryDetector:  services.NewMockBinaryDetector(),
			ChecksumManager: services.NewChecksumManager(),
			MaxFileSize:     maxFileSize,
			WorkspaceRoot:   workspaceRoot,
		}

		resp, err := ListDirectory(ctx, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Find file and directory entries
		var fileEntry, dirEntry *models.DirectoryEntry
		for i := range resp.Entries {
			switch resp.Entries[i].RelativePath {
			case "file.txt":
				fileEntry = &resp.Entries[i]
			case "subdir":
				dirEntry = &resp.Entries[i]
			}
		}

		if fileEntry == nil {
			t.Fatal("expected to find file.txt entry")
		}

		if dirEntry == nil {
			t.Fatal("expected to find subdir entry")
		}

		// Verify file entry
		if fileEntry.IsDir {
			t.Error("file.txt should not be marked as directory")
		}

		if fileEntry.Size != 11 {
			t.Errorf("expected file size 11, got %d", fileEntry.Size)
		}

		if fileEntry.RelativePath != "file.txt" {
			t.Errorf("expected RelativePath 'file.txt', got %q", fileEntry.RelativePath)
		}

		// Verify directory entry
		if !dirEntry.IsDir {
			t.Error("subdir should be marked as directory")
		}

		if dirEntry.Size != 0 {
			t.Errorf("expected directory size 0, got %d", dirEntry.Size)
		}

		if dirEntry.RelativePath != "subdir" {
			t.Errorf("expected RelativePath 'subdir', got %q", dirEntry.RelativePath)
		}
	})

	t.Run("sorting: directories before files, alphabetical within each group", func(t *testing.T) {
		fs := services.NewMockFileSystem(maxFileSize)
		fs.CreateDir("/workspace")
		fs.CreateFile("/workspace/zebra.txt", []byte("z"), 0o644)
		fs.CreateFile("/workspace/alpha.txt", []byte("a"), 0o644)
		fs.CreateDir("/workspace/zulu")
		fs.CreateDir("/workspace/alpha")

		ctx := &models.WorkspaceContext{
			FS:              fs,
			BinaryDetector:  services.NewMockBinaryDetector(),
			ChecksumManager: services.NewChecksumManager(),
			MaxFileSize:     maxFileSize,
			WorkspaceRoot:   workspaceRoot,
		}

		resp, err := ListDirectory(ctx, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(resp.Entries) != 4 {
			t.Fatalf("expected 4 entries, got %d", len(resp.Entries))
		}

		// Verify order: alpha (dir), zulu (dir), alpha.txt (file), zebra.txt (file)
		expectedOrder := []string{"alpha", "zulu", "alpha.txt", "zebra.txt"}
		for i, expected := range expectedOrder {
			if resp.Entries[i].RelativePath != expected {
				t.Errorf("entry %d: expected %q, got %q", i, expected, resp.Entries[i].RelativePath)
			}
		}

		// Verify directories come first
		if !resp.Entries[0].IsDir || !resp.Entries[1].IsDir {
			t.Error("directories should come before files")
		}

		if resp.Entries[2].IsDir || resp.Entries[3].IsDir {
			t.Error("files should come after directories")
		}
	})

	t.Run("nested directory with relative path", func(t *testing.T) {
		fs := services.NewMockFileSystem(maxFileSize)
		fs.CreateDir("/workspace")
		fs.CreateDir("/workspace/src")
		fs.CreateDir("/workspace/src/app")
		fs.CreateFile("/workspace/src/app/main.go", []byte("package main"), 0o644)

		ctx := &models.WorkspaceContext{
			FS:              fs,
			BinaryDetector:  services.NewMockBinaryDetector(),
			ChecksumManager: services.NewChecksumManager(),
			MaxFileSize:     maxFileSize,
			WorkspaceRoot:   workspaceRoot,
		}

		resp, err := ListDirectory(ctx, "src/app")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if resp.DirectoryPath != "src/app" {
			t.Errorf("expected DirectoryPath 'src/app', got %q", resp.DirectoryPath)
		}

		if len(resp.Entries) != 1 {
			t.Fatalf("expected 1 entry, got %d", len(resp.Entries))
		}

		if resp.Entries[0].RelativePath != "src/app/main.go" {
			t.Errorf("expected RelativePath 'src/app/main.go', got %q", resp.Entries[0].RelativePath)
		}
	})
}
