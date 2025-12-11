package tools

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/Cyclone1070/iav/internal/config"
	"github.com/Cyclone1070/iav/internal/testing/mocks"
	"github.com/Cyclone1070/iav/internal/tools/models"
	"github.com/Cyclone1070/iav/internal/tools/services"
)

func TestListDirectory(t *testing.T) {
	workspaceRoot := "/workspace"

	t.Run("list workspace root with mixed files and directories", func(t *testing.T) {
		fs := mocks.NewMockFileSystem()
		fs.CreateDir("/workspace")
		fs.CreateFile("/workspace/file1.txt", []byte("content1"), 0o644)
		fs.CreateFile("/workspace/file2.txt", []byte("content2"), 0o644)
		fs.CreateDir("/workspace/subdir1")
		fs.CreateDir("/workspace/subdir2")

		ctx := &models.WorkspaceContext{
			FS:              fs,
			BinaryDetector:  mocks.NewMockBinaryDetector(),
			ChecksumManager: services.NewChecksumManager(),
			WorkspaceRoot:   workspaceRoot,
			Config:          *config.DefaultConfig(),
		}

		resp, err := ListDirectory(context.Background(), ctx, models.ListDirectoryRequest{Path: "", MaxDepth: -1, Offset: 0, Limit: 1000})
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
		fs := mocks.NewMockFileSystem()
		fs.CreateDir("/workspace")
		fs.CreateDir("/workspace/src")
		fs.CreateFile("/workspace/src/main.go", []byte("package main"), 0o644)
		fs.CreateFile("/workspace/src/utils.go", []byte("package main"), 0o644)
		fs.CreateDir("/workspace/src/internal")

		ctx := &models.WorkspaceContext{
			FS:              fs,
			BinaryDetector:  mocks.NewMockBinaryDetector(),
			ChecksumManager: services.NewChecksumManager(),
			WorkspaceRoot:   workspaceRoot,
			Config:          *config.DefaultConfig(),
		}

		resp, err := ListDirectory(context.Background(), ctx, models.ListDirectoryRequest{Path: "src", MaxDepth: -1, Offset: 0, Limit: 1000})
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
		fs := mocks.NewMockFileSystem()
		fs.CreateDir("/workspace")
		fs.CreateDir("/workspace/empty")

		ctx := &models.WorkspaceContext{
			FS:              fs,
			BinaryDetector:  mocks.NewMockBinaryDetector(),
			ChecksumManager: services.NewChecksumManager(),
			WorkspaceRoot:   workspaceRoot,
			Config:          *config.DefaultConfig(),
		}

		resp, err := ListDirectory(context.Background(), ctx, models.ListDirectoryRequest{Path: "empty", MaxDepth: -1, Offset: 0, Limit: 1000})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(resp.Entries) != 0 {
			t.Errorf("expected 0 entries, got %d", len(resp.Entries))
		}
	})

	t.Run("path resolves to file not directory", func(t *testing.T) {
		fs := mocks.NewMockFileSystem()
		fs.CreateDir("/workspace")
		fs.CreateFile("/workspace/file.txt", []byte("content"), 0o644)

		ctx := &models.WorkspaceContext{
			FS:              fs,
			BinaryDetector:  mocks.NewMockBinaryDetector(),
			ChecksumManager: services.NewChecksumManager(),
			WorkspaceRoot:   workspaceRoot,
			Config:          *config.DefaultConfig(),
		}

		_, err := ListDirectory(context.Background(), ctx, models.ListDirectoryRequest{Path: "file.txt", MaxDepth: -1, Offset: 0, Limit: 1000})
		if err == nil {
			t.Fatal("expected error when listing a file, got nil")
		}

		// Should return a descriptive error
		if err.Error() == "" {
			t.Error("expected non-empty error message")
		}
	})

	t.Run("path outside workspace", func(t *testing.T) {
		fs := mocks.NewMockFileSystem()
		fs.CreateDir("/workspace")
		fs.CreateDir("/outside")

		ctx := &models.WorkspaceContext{
			FS:              fs,
			BinaryDetector:  mocks.NewMockBinaryDetector(),
			ChecksumManager: services.NewChecksumManager(),
			WorkspaceRoot:   workspaceRoot,
			Config:          *config.DefaultConfig(),
		}

		_, err := ListDirectory(context.Background(), ctx, models.ListDirectoryRequest{Path: "../outside", MaxDepth: -1, Offset: 0, Limit: 1000})
		if err != models.ErrOutsideWorkspace {
			t.Errorf("expected ErrOutsideWorkspace, got %v", err)
		}
	})

	t.Run("directory does not exist", func(t *testing.T) {
		fs := mocks.NewMockFileSystem()
		fs.CreateDir("/workspace")

		ctx := &models.WorkspaceContext{
			FS:              fs,
			BinaryDetector:  mocks.NewMockBinaryDetector(),
			ChecksumManager: services.NewChecksumManager(),
			WorkspaceRoot:   workspaceRoot,
			Config:          *config.DefaultConfig(),
		}

		_, err := ListDirectory(context.Background(), ctx, models.ListDirectoryRequest{Path: "nonexistent", MaxDepth: -1, Offset: 0, Limit: 1000})
		if err == nil {
			t.Fatal("expected error for nonexistent directory, got nil")
		}

		// Should propagate filesystem error
		if err != models.ErrFileMissing && err.Error() == "" {
			t.Errorf("expected meaningful error, got %v", err)
		}
	})

	t.Run("filesystem error propagation", func(t *testing.T) {
		fs := mocks.NewMockFileSystem()
		fs.CreateDir("/workspace")
		fs.CreateDir("/workspace/testdir")
		fs.SetOperationError("ListDir", os.ErrPermission)

		ctx := &models.WorkspaceContext{
			FS:              fs,
			BinaryDetector:  mocks.NewMockBinaryDetector(),
			ChecksumManager: services.NewChecksumManager(),
			WorkspaceRoot:   workspaceRoot,
			Config:          *config.DefaultConfig(),
		}

		_, err := ListDirectory(context.Background(), ctx, models.ListDirectoryRequest{Path: "testdir", MaxDepth: -1, Offset: 0, Limit: 1000})
		if err == nil {
			t.Fatal("expected error from filesystem, got nil")
		}
		if !strings.Contains(err.Error(), "permission") {
			t.Errorf("expected permission-related error, got: %v", err)
		}
	})

	t.Run("relative path input", func(t *testing.T) {
		fs := mocks.NewMockFileSystem()
		fs.CreateDir("/workspace")
		fs.CreateDir("/workspace/src")
		fs.CreateFile("/workspace/src/file.txt", []byte("content"), 0o644)

		ctx := &models.WorkspaceContext{
			FS:              fs,
			BinaryDetector:  mocks.NewMockBinaryDetector(),
			ChecksumManager: services.NewChecksumManager(),
			WorkspaceRoot:   workspaceRoot,
			Config:          *config.DefaultConfig(),
		}

		resp, err := ListDirectory(context.Background(), ctx, models.ListDirectoryRequest{Path: "src", MaxDepth: -1, Offset: 0, Limit: 1000})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if resp.DirectoryPath != "src" {
			t.Errorf("expected DirectoryPath 'src', got %q", resp.DirectoryPath)
		}
	})

	t.Run("absolute path input", func(t *testing.T) {
		fs := mocks.NewMockFileSystem()
		fs.CreateDir("/workspace")
		fs.CreateDir("/workspace/src")
		fs.CreateFile("/workspace/src/file.txt", []byte("content"), 0o644)

		ctx := &models.WorkspaceContext{
			FS:              fs,
			BinaryDetector:  mocks.NewMockBinaryDetector(),
			ChecksumManager: services.NewChecksumManager(),
			WorkspaceRoot:   workspaceRoot,
			Config:          *config.DefaultConfig(),
		}

		resp, err := ListDirectory(context.Background(), ctx, models.ListDirectoryRequest{Path: "/workspace/src", MaxDepth: -1, Offset: 0, Limit: 1000})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if resp.DirectoryPath != "src" {
			t.Errorf("expected DirectoryPath 'src', got %q", resp.DirectoryPath)
		}
	})

	t.Run("dot path alias for workspace root", func(t *testing.T) {
		fs := mocks.NewMockFileSystem()
		fs.CreateDir("/workspace")
		fs.CreateFile("/workspace/file.txt", []byte("content"), 0o644)

		ctx := &models.WorkspaceContext{
			FS:              fs,
			BinaryDetector:  mocks.NewMockBinaryDetector(),
			ChecksumManager: services.NewChecksumManager(),
			WorkspaceRoot:   workspaceRoot,
			Config:          *config.DefaultConfig(),
		}

		resp, err := ListDirectory(context.Background(), ctx, models.ListDirectoryRequest{Path: ".", MaxDepth: -1, Offset: 0, Limit: 1000})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if resp.DirectoryPath != "" {
			t.Errorf("expected empty DirectoryPath for workspace root, got %q", resp.DirectoryPath)
		}
	})

	t.Run("verify entry metadata correctness", func(t *testing.T) {
		fs := mocks.NewMockFileSystem()
		fs.CreateDir("/workspace")
		fs.CreateFile("/workspace/file.txt", []byte("hello world"), 0o644)
		fs.CreateDir("/workspace/subdir")

		ctx := &models.WorkspaceContext{
			FS:              fs,
			BinaryDetector:  mocks.NewMockBinaryDetector(),
			ChecksumManager: services.NewChecksumManager(),
			WorkspaceRoot:   workspaceRoot,
			Config:          *config.DefaultConfig(),
		}

		resp, err := ListDirectory(context.Background(), ctx, models.ListDirectoryRequest{Path: "", MaxDepth: -1, Offset: 0, Limit: 1000})
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

		if fileEntry.RelativePath != "file.txt" {
			t.Errorf("expected RelativePath 'file.txt', got %q", fileEntry.RelativePath)
		}

		// Verify directory entry
		if !dirEntry.IsDir {
			t.Error("subdir should be marked as directory")
		}

		if dirEntry.RelativePath != "subdir" {
			t.Errorf("expected RelativePath 'subdir', got %q", dirEntry.RelativePath)
		}
	})

	t.Run("sorting: directories before files, alphabetical within each group", func(t *testing.T) {
		fs := mocks.NewMockFileSystem()
		fs.CreateDir("/workspace")
		fs.CreateFile("/workspace/zebra.txt", []byte("z"), 0o644)
		fs.CreateFile("/workspace/alpha.txt", []byte("a"), 0o644)
		fs.CreateDir("/workspace/zulu")
		fs.CreateDir("/workspace/alpha")

		ctx := &models.WorkspaceContext{
			FS:              fs,
			BinaryDetector:  mocks.NewMockBinaryDetector(),
			ChecksumManager: services.NewChecksumManager(),
			WorkspaceRoot:   workspaceRoot,
			Config:          *config.DefaultConfig(),
		}

		resp, err := ListDirectory(context.Background(), ctx, models.ListDirectoryRequest{Path: "", MaxDepth: -1, Offset: 0, Limit: 1000})
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
		fs := mocks.NewMockFileSystem()
		fs.CreateDir("/workspace")
		fs.CreateDir("/workspace/src")
		fs.CreateDir("/workspace/src/app")
		fs.CreateFile("/workspace/src/app/main.go", []byte("package main"), 0o644)

		ctx := &models.WorkspaceContext{
			FS:              fs,
			BinaryDetector:  mocks.NewMockBinaryDetector(),
			ChecksumManager: services.NewChecksumManager(),
			WorkspaceRoot:   workspaceRoot,
			Config:          *config.DefaultConfig(),
		}

		resp, err := ListDirectory(context.Background(), ctx, models.ListDirectoryRequest{Path: "src/app", MaxDepth: -1, Offset: 0, Limit: 1000})
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

func TestListDirectory_Pagination(t *testing.T) {
	workspaceRoot := "/workspace"

	t.Run("pagination with offset and limit", func(t *testing.T) {
		fs := mocks.NewMockFileSystem()
		fs.CreateDir("/workspace")
		// Create 150 files
		for i := range 150 {
			fs.CreateFile(fmt.Sprintf("/workspace/file%d.txt", i), []byte("content"), 0o644)
		}

		ctx := &models.WorkspaceContext{
			FS:              fs,
			BinaryDetector:  mocks.NewMockBinaryDetector(),
			ChecksumManager: services.NewChecksumManager(),
			WorkspaceRoot:   workspaceRoot,
			Config:          *config.DefaultConfig(),
		}

		// First page: offset=0, limit=50
		resp1, err := ListDirectory(context.Background(), ctx, models.ListDirectoryRequest{Path: "", MaxDepth: -1, Offset: 0, Limit: 50})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(resp1.Entries) != 50 {
			t.Errorf("expected 50 entries, got %d", len(resp1.Entries))
		}
		if resp1.TotalCount != 150 {
			t.Errorf("expected TotalCount 150, got %d", resp1.TotalCount)
		}
		if !resp1.Truncated {
			t.Error("expected Truncated=true for first page")
		}
		if resp1.Offset != 0 || resp1.Limit != 50 {
			t.Errorf("expected Offset=0 Limit=50, got Offset=%d Limit=%d", resp1.Offset, resp1.Limit)
		}

		// Second page: offset=50, limit=50
		resp2, err := ListDirectory(context.Background(), ctx, models.ListDirectoryRequest{Path: "", MaxDepth: -1, Offset: 50, Limit: 50})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(resp2.Entries) != 50 {
			t.Errorf("expected 50 entries, got %d", len(resp2.Entries))
		}
		if resp2.TotalCount != 150 {
			t.Errorf("expected TotalCount 150, got %d", resp2.TotalCount)
		}
		if !resp2.Truncated {
			t.Error("expected Truncated=true for second page")
		}

		// Third page: offset=100, limit=50 (should have 50 entries, no truncation)
		resp3, err := ListDirectory(context.Background(), ctx, models.ListDirectoryRequest{Path: "", MaxDepth: -1, Offset: 100, Limit: 50})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(resp3.Entries) != 50 {
			t.Errorf("expected 50 entries, got %d", len(resp3.Entries))
		}
		if resp3.TotalCount != 150 {
			t.Errorf("expected TotalCount 150, got %d", resp3.TotalCount)
		}
		if resp3.Truncated {
			t.Error("expected Truncated=false for last page")
		}

		// Verify no overlap between pages
		firstPagePaths := make(map[string]bool)
		for _, entry := range resp1.Entries {
			firstPagePaths[entry.RelativePath] = true
		}
		for _, entry := range resp2.Entries {
			if firstPagePaths[entry.RelativePath] {
				t.Errorf("found duplicate entry %s between pages", entry.RelativePath)
			}
		}
	})
}

func TestListDirectory_InvalidPagination(t *testing.T) {
	workspaceRoot := "/workspace"

	fs := mocks.NewMockFileSystem()
	fs.CreateDir("/workspace")

	ctx := &models.WorkspaceContext{
		FS:              fs,
		BinaryDetector:  mocks.NewMockBinaryDetector(),
		ChecksumManager: services.NewChecksumManager(),
		WorkspaceRoot:   workspaceRoot,
		Config:          *config.DefaultConfig(),
	}

	t.Run("zero limit uses default", func(t *testing.T) {
		// Limit=0 now uses default, should succeed
		resp, err := ListDirectory(context.Background(), ctx, models.ListDirectoryRequest{Path: "", MaxDepth: -1, Offset: 0, Limit: 0})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Limit != ctx.Config.Tools.DefaultListDirectoryLimit {
			t.Errorf("expected default limit %d, got %d", ctx.Config.Tools.DefaultListDirectoryLimit, resp.Limit)
		}
	})
}

func TestListDirectory_WithSymlinks(t *testing.T) {
	workspaceRoot := "/workspace"

	t.Run("directory with symlinks", func(t *testing.T) {
		fs := mocks.NewMockFileSystem()
		fs.CreateDir("/workspace")
		fs.CreateFile("/workspace/file.txt", []byte("content"), 0o644)
		fs.CreateDir("/workspace/target")
		fs.CreateSymlink("/workspace/link", "/workspace/target")

		ctx := &models.WorkspaceContext{
			FS:              fs,
			BinaryDetector:  mocks.NewMockBinaryDetector(),
			ChecksumManager: services.NewChecksumManager(),
			WorkspaceRoot:   workspaceRoot,
			Config:          *config.DefaultConfig(),
		}

		resp, err := ListDirectory(context.Background(), ctx, models.ListDirectoryRequest{Path: "", MaxDepth: -1, Offset: 0, Limit: 1000})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Should list symlink as an entry
		foundLink := false
		for _, entry := range resp.Entries {
			if entry.RelativePath == "link" {
				foundLink = true
				// Symlink entry itself is not a directory (it's a symlink)
				// The IsDir flag reflects the symlink itself, not what it points to
				if entry.IsDir {
					t.Error("symlink entry should not be marked as directory (it's a symlink, not a directory)")
				}
				break
			}
		}
		if !foundLink {
			t.Error("expected to find symlink in directory listing")
		}
	})
}

func TestListDirectory_UnicodeFilenames(t *testing.T) {
	workspaceRoot := "/workspace"

	t.Run("unicode and special characters", func(t *testing.T) {
		fs := mocks.NewMockFileSystem()
		fs.CreateDir("/workspace")
		fs.CreateFile("/workspace/文件.txt", []byte("content"), 0o644)
		fs.CreateFile("/workspace/🚀.txt", []byte("content"), 0o644)
		fs.CreateFile("/workspace/zebra.txt", []byte("content"), 0o644)
		fs.CreateFile("/workspace/alpha.txt", []byte("content"), 0o644)

		ctx := &models.WorkspaceContext{
			FS:              fs,
			BinaryDetector:  mocks.NewMockBinaryDetector(),
			ChecksumManager: services.NewChecksumManager(),
			WorkspaceRoot:   workspaceRoot,
			Config:          *config.DefaultConfig(),
		}

		resp, err := ListDirectory(context.Background(), ctx, models.ListDirectoryRequest{Path: "", MaxDepth: -1, Offset: 0, Limit: 1000})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify unicode files are listed
		foundUnicode := false
		foundEmoji := false
		for _, entry := range resp.Entries {
			if entry.RelativePath == "文件.txt" {
				foundUnicode = true
			}
			if entry.RelativePath == "🚀.txt" {
				foundEmoji = true
			}
		}
		if !foundUnicode {
			t.Error("expected to find unicode filename")
		}
		if !foundEmoji {
			t.Error("expected to find emoji filename")
		}

		// Verify sorting works (should be alphabetical)
		if len(resp.Entries) < 4 {
			t.Fatalf("expected at least 4 entries, got %d", len(resp.Entries))
		}
		// Sorting should still work with unicode
		for i := 0; i < len(resp.Entries)-1; i++ {
			if resp.Entries[i].RelativePath > resp.Entries[i+1].RelativePath {
				t.Errorf("entries not sorted: %s > %s", resp.Entries[i].RelativePath, resp.Entries[i+1].RelativePath)
			}
		}
	})
}

func TestListDirectory_DotfilesWithGitignore(t *testing.T) {
	workspaceRoot := "/workspace"

	t.Run("dotfiles filtered by gitignore", func(t *testing.T) {
		fs := mocks.NewMockFileSystem()
		fs.CreateDir("/workspace")
		fs.CreateFile("/workspace/.gitignore", []byte("*.log\n"), 0o644)
		fs.CreateFile("/workspace/.hidden", []byte("content"), 0o644)
		fs.CreateFile("/workspace/.test.log", []byte("content"), 0o644)
		fs.CreateFile("/workspace/.keep", []byte("content"), 0o644)

		gitignoreService, err := services.NewGitignoreService(workspaceRoot, fs)
		if err != nil {
			t.Fatalf("unexpected error creating gitignore service: %v", err)
		}

		ctx := &models.WorkspaceContext{
			FS:               fs,
			BinaryDetector:   mocks.NewMockBinaryDetector(),
			ChecksumManager:  services.NewChecksumManager(),
			WorkspaceRoot:    workspaceRoot,
			GitignoreService: gitignoreService,
			Config:           *config.DefaultConfig(),
		}

		resp, err := ListDirectory(context.Background(), ctx, models.ListDirectoryRequest{Path: "", MaxDepth: -1, Offset: 0, Limit: 1000})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// .test.log should be filtered (matches *.log)
		foundTestLog := false
		// .hidden and .keep should be included
		foundHidden := false
		foundKeep := false

		for _, entry := range resp.Entries {
			if entry.RelativePath == ".test.log" {
				foundTestLog = true
			}
			if entry.RelativePath == ".hidden" {
				foundHidden = true
			}
			if entry.RelativePath == ".keep" {
				foundKeep = true
			}
		}

		if foundTestLog {
			t.Error("expected .test.log to be filtered by gitignore")
		}
		if !foundHidden {
			t.Error("expected .hidden to be included")
		}
		if !foundKeep {
			t.Error("expected .keep to be included")
		}
	})
}

func TestListDirectory_DotfilesWithoutGitignore(t *testing.T) {
	workspaceRoot := "/workspace"

	t.Run("all dotfiles included when gitignore service is nil", func(t *testing.T) {
		fs := mocks.NewMockFileSystem()
		fs.CreateDir("/workspace")
		fs.CreateFile("/workspace/.hidden", []byte("content"), 0o644)
		fs.CreateFile("/workspace/.test.log", []byte("content"), 0o644)

		ctx := &models.WorkspaceContext{
			FS:               fs,
			BinaryDetector:   mocks.NewMockBinaryDetector(),
			ChecksumManager:  services.NewChecksumManager(),
			WorkspaceRoot:    workspaceRoot,
			GitignoreService: nil, // No gitignore service
			Config:           *config.DefaultConfig(),
		}

		resp, err := ListDirectory(context.Background(), ctx, models.ListDirectoryRequest{Path: "", MaxDepth: -1, Offset: 0, Limit: 1000})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// All dotfiles should be included
		foundHidden := false
		foundTestLog := false

		for _, entry := range resp.Entries {
			if entry.RelativePath == ".hidden" {
				foundHidden = true
			}
			if entry.RelativePath == ".test.log" {
				foundTestLog = true
			}
		}

		if !foundHidden {
			t.Error("expected .hidden to be included")
		}
		if !foundTestLog {
			t.Error("expected .test.log to be included")
		}
	})
}

func TestListDirectory_LargeDirectory(t *testing.T) {
	workspaceRoot := "/workspace"

	t.Run("large directory pagination", func(t *testing.T) {
		fs := mocks.NewMockFileSystem()
		fs.CreateDir("/workspace")
		// Create 5000 files
		for i := range 5000 {
			fs.CreateFile(fmt.Sprintf("/workspace/file%d.txt", i), []byte("content"), 0o644)
		}

		ctx := &models.WorkspaceContext{
			FS:              fs,
			BinaryDetector:  mocks.NewMockBinaryDetector(),
			ChecksumManager: services.NewChecksumManager(),
			WorkspaceRoot:   workspaceRoot,
			Config:          *config.DefaultConfig(),
		}

		// Paginate through all files
		allPaths := make(map[string]bool)
		offset := 0
		limit := 1000
		pageCount := 0

		for {
			resp, err := ListDirectory(context.Background(), ctx, models.ListDirectoryRequest{Path: "", MaxDepth: -1, Offset: offset, Limit: limit})
			if err != nil {
				t.Fatalf("unexpected error at offset %d: %v", offset, err)
			}

			if resp.TotalCount != 5000 {
				t.Errorf("expected TotalCount 5000, got %d", resp.TotalCount)
			}

			// Collect all paths from this page
			for _, entry := range resp.Entries {
				if allPaths[entry.RelativePath] {
					t.Errorf("found duplicate entry: %s", entry.RelativePath)
				}
				allPaths[entry.RelativePath] = true
			}

			pageCount++
			if !resp.Truncated {
				break
			}
			offset += limit
		}

		// Verify we got all 5000 files
		if len(allPaths) != 5000 {
			t.Errorf("expected 5000 unique entries, got %d", len(allPaths))
		}

		// Should have taken 5 pages (1000 each)
		if pageCount != 5 {
			t.Errorf("expected 5 pages, got %d", pageCount)
		}
	})
}

func TestListDirectory_OffsetBeyondEnd(t *testing.T) {
	workspaceRoot := "/workspace"

	t.Run("offset beyond end returns empty", func(t *testing.T) {
		fs := mocks.NewMockFileSystem()
		fs.CreateDir("/workspace")
		// Create 10 files
		for i := range 10 {
			fs.CreateFile(fmt.Sprintf("/workspace/file%d.txt", i), []byte("content"), 0o644)
		}

		ctx := &models.WorkspaceContext{
			FS:              fs,
			BinaryDetector:  mocks.NewMockBinaryDetector(),
			ChecksumManager: services.NewChecksumManager(),
			WorkspaceRoot:   workspaceRoot,
			Config:          *config.DefaultConfig(),
		}

		resp, err := ListDirectory(context.Background(), ctx, models.ListDirectoryRequest{Path: "", MaxDepth: -1, Offset: 100, Limit: 10})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(resp.Entries) != 0 {
			t.Errorf("expected 0 entries, got %d", len(resp.Entries))
		}
		if resp.TotalCount != 10 {
			t.Errorf("expected TotalCount 10, got %d", resp.TotalCount)
		}
		if resp.Truncated {
			t.Error("expected Truncated=false when offset is beyond end")
		}
	})
}

func TestListDirectory_Recursive(t *testing.T) {
	workspaceRoot := "/workspace"

	fs := mocks.NewMockFileSystem()
	fs.CreateDir("/workspace")
	fs.CreateDir("/workspace/dir1")
	fs.CreateDir("/workspace/dir1/subdir1")
	fs.CreateFile("/workspace/dir1/subdir1/file1.txt", []byte("content"), 0o644)
	fs.CreateFile("/workspace/dir1/file2.txt", []byte("content"), 0o644)
	fs.CreateFile("/workspace/file3.txt", []byte("content"), 0o644)

	ctx := &models.WorkspaceContext{
		FS:              fs,
		BinaryDetector:  mocks.NewMockBinaryDetector(),
		ChecksumManager: services.NewChecksumManager(),
		WorkspaceRoot:   workspaceRoot,
		Config:          *config.DefaultConfig(),
	}

	resp, err := ListDirectory(context.Background(), ctx, models.ListDirectoryRequest{Path: "", MaxDepth: -1, Offset: 0, Limit: 1000})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should include all files and directories recursively
	if len(resp.Entries) != 5 {
		t.Fatalf("expected 5 entries (2 dirs + 3 files), got %d", len(resp.Entries))
	}

	// Verify all paths are present
	expectedPaths := map[string]bool{
		"dir1":                   true,
		"dir1/subdir1":           true,
		"dir1/subdir1/file1.txt": true,
		"dir1/file2.txt":         true,
		"file3.txt":              true,
	}

	for _, entry := range resp.Entries {
		if !expectedPaths[entry.RelativePath] {
			t.Errorf("unexpected entry: %s", entry.RelativePath)
		}
		delete(expectedPaths, entry.RelativePath)
	}

	if len(expectedPaths) > 0 {
		t.Errorf("missing entries: %v", expectedPaths)
	}
}

func TestListDirectory_RecursiveWithDepthLimit(t *testing.T) {
	workspaceRoot := "/workspace"

	fs := mocks.NewMockFileSystem()
	fs.CreateDir("/workspace")
	fs.CreateDir("/workspace/level1")
	fs.CreateDir("/workspace/level1/level2")
	fs.CreateDir("/workspace/level1/level2/level3")
	fs.CreateFile("/workspace/level1/file1.txt", []byte("content"), 0o644)
	fs.CreateFile("/workspace/level1/level2/file2.txt", []byte("content"), 0o644)
	fs.CreateFile("/workspace/level1/level2/level3/file3.txt", []byte("content"), 0o644)

	ctx := &models.WorkspaceContext{
		FS:              fs,
		BinaryDetector:  mocks.NewMockBinaryDetector(),
		ChecksumManager: services.NewChecksumManager(),
		WorkspaceRoot:   workspaceRoot,
		Config:          *config.DefaultConfig(),
	}

	// Depth limit of 2 should go 2 levels deep from root
	resp, err := ListDirectory(context.Background(), ctx, models.ListDirectoryRequest{Path: "", MaxDepth: 2, Offset: 0, Limit: 1000})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should include: level1, level1/level2, level1/file1.txt, level1/level2/file2.txt, level1/level2/level3
	// Should NOT include: level1/level2/level3/file3.txt (that's at depth 3)
	if len(resp.Entries) != 5 {
		t.Fatalf("expected 5 entries (depth 2), got %d", len(resp.Entries))
	}

	// Verify level3 file is not included
	for _, entry := range resp.Entries {
		if entry.RelativePath == "level1/level2/level3/file3.txt" {
			t.Errorf("depth limit violated: found %s", entry.RelativePath)
		}
	}
}

func TestListDirectory_SymlinkLoop(t *testing.T) {
	workspaceRoot := "/workspace"

	fs := mocks.NewMockFileSystem()
	fs.CreateDir("/workspace")
	fs.CreateDir("/workspace/dir1")
	fs.CreateDir("/workspace/dir2")

	// Create symlink loop: dir1/link -> dir2, dir2/link -> dir1
	fs.CreateSymlink("/workspace/dir1/link", "/workspace/dir2")
	fs.CreateSymlink("/workspace/dir2/link", "/workspace/dir1")

	ctx := &models.WorkspaceContext{
		FS:              fs,
		BinaryDetector:  mocks.NewMockBinaryDetector(),
		ChecksumManager: services.NewChecksumManager(),
		WorkspaceRoot:   workspaceRoot,
		Config:          *config.DefaultConfig(),
	}

	// Should detect loop and not hang
	resp, err := ListDirectory(context.Background(), ctx, models.ListDirectoryRequest{Path: "", MaxDepth: -1, Offset: 0, Limit: 1000})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should include dir1, dir2, and the symlinks, but not recurse infinitely
	if len(resp.Entries) > 10 {
		t.Errorf("possible infinite recursion: got %d entries", len(resp.Entries))
	}
}

func TestListDirectory_RecursivePagination(t *testing.T) {
	workspaceRoot := "/workspace"

	fs := mocks.NewMockFileSystem()
	fs.CreateDir("/workspace")

	// Create 20 files across multiple directories
	for i := range 10 {
		fs.CreateFile(fmt.Sprintf("/workspace/file%d.txt", i), []byte("content"), 0o644)
	}
	fs.CreateDir("/workspace/subdir")
	for i := range 10 {
		fs.CreateFile(fmt.Sprintf("/workspace/subdir/file%d.txt", i), []byte("content"), 0o644)
	}

	ctx := &models.WorkspaceContext{
		FS:              fs,
		BinaryDetector:  mocks.NewMockBinaryDetector(),
		ChecksumManager: services.NewChecksumManager(),
		WorkspaceRoot:   workspaceRoot,
		Config:          *config.DefaultConfig(),
	}

	// Request offset=5, limit=5
	resp, err := ListDirectory(context.Background(), ctx, models.ListDirectoryRequest{Path: "", MaxDepth: -1, Offset: 5, Limit: 5})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Entries) != 5 {
		t.Fatalf("expected 5 entries, got %d", len(resp.Entries))
	}

	if resp.TotalCount != 21 {
		t.Errorf("expected TotalCount 21 (1 dir + 20 files), got %d", resp.TotalCount)
	}

	if !resp.Truncated {
		t.Error("expected Truncated=true")
	}
}

func TestListDirectory_NonRecursive(t *testing.T) {
	workspaceRoot := "/workspace"

	fs := mocks.NewMockFileSystem()
	fs.CreateDir("/workspace")
	fs.CreateDir("/workspace/dir1")
	fs.CreateDir("/workspace/dir1/subdir1")
	fs.CreateFile("/workspace/dir1/subdir1/file1.txt", []byte("content"), 0o644)
	fs.CreateFile("/workspace/file2.txt", []byte("content"), 0o644)

	ctx := &models.WorkspaceContext{
		FS:              fs,
		BinaryDetector:  mocks.NewMockBinaryDetector(),
		ChecksumManager: services.NewChecksumManager(),
		WorkspaceRoot:   workspaceRoot,
		Config:          *config.DefaultConfig(),
	}

	// Depth 0 means only list immediate children (non-recursive)
	resp, err := ListDirectory(context.Background(), ctx, models.ListDirectoryRequest{Path: "", MaxDepth: 0, Offset: 0, Limit: 1000})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should only include dir1 and file2.txt (not subdirectories)
	if len(resp.Entries) != 2 {
		t.Fatalf("expected 2 entries (non-recursive), got %d", len(resp.Entries))
	}

	for _, entry := range resp.Entries {
		if entry.RelativePath == "dir1/subdir1" || entry.RelativePath == "dir1/subdir1/file1.txt" {
			t.Errorf("non-recursive mode should not include nested entries: %s", entry.RelativePath)
		}
	}
}
