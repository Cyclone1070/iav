package tools

import (
	"context"
	"fmt"
	"slices"
	"testing"

	"github.com/Cyclone1070/deployforme/internal/tools/models"
	"github.com/Cyclone1070/deployforme/internal/tools/services"
)

func TestFindFile_BasicGlob(t *testing.T) {
	workspaceRoot := "/workspace"
	maxFileSize := int64(1024 * 1024)

	fs := services.NewMockFileSystem(maxFileSize)
	fs.CreateDir("/workspace")

	mockRunner := &services.MockCommandRunner{
		RunFunc: func(ctx context.Context, cmd []string) ([]byte, error) {
			// Simulate fd output
			output := "/workspace/a/b/file.go\n/workspace/a/file.go\n"
			return []byte(output), nil
		},
	}

	ctx := &models.WorkspaceContext{
		FS:              fs,
		BinaryDetector:  services.NewMockBinaryDetector(),
		ChecksumManager: services.NewChecksumManager(),
		MaxFileSize:     maxFileSize,
		WorkspaceRoot:   workspaceRoot,
		CommandRunner:   mockRunner,
	}

	resp, err := FindFile(ctx, "*.go", "", 0, 0, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Matches) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(resp.Matches))
	}

	// Verify sorting (alphabetical)
	expectedMatches := []string{"a/b/file.go", "a/file.go"}
	for i, expected := range expectedMatches {
		if resp.Matches[i] != expected {
			t.Errorf("match %d: expected %q, got %q", i, expected, resp.Matches[i])
		}
	}
}

func TestFindFile_Pagination(t *testing.T) {
	workspaceRoot := "/workspace"
	maxFileSize := int64(1024 * 1024)

	fs := services.NewMockFileSystem(maxFileSize)
	fs.CreateDir("/workspace")

	// Simulate 10 files
	var output string
	for i := range 10 {
		output += fmt.Sprintf("/workspace/file%d.txt\n", i)
	}

	mockRunner := &services.MockCommandRunner{
		RunFunc: func(ctx context.Context, cmd []string) ([]byte, error) {
			return []byte(output), nil
		},
	}

	ctx := &models.WorkspaceContext{
		FS:              fs,
		BinaryDetector:  services.NewMockBinaryDetector(),
		ChecksumManager: services.NewChecksumManager(),
		MaxFileSize:     maxFileSize,
		WorkspaceRoot:   workspaceRoot,
		CommandRunner:   mockRunner,
	}

	// Request offset=2, limit=2
	resp, err := FindFile(ctx, "*.txt", "", 0, 2, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Matches) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(resp.Matches))
	}

	if resp.TotalCount != 10 {
		t.Errorf("expected TotalCount 10, got %d", resp.TotalCount)
	}

	if !resp.Truncated {
		t.Error("expected Truncated=true")
	}

	// Verify correct slice (file2.txt, file3.txt after sorting)
	if resp.Matches[0] != "file2.txt" {
		t.Errorf("expected file2.txt, got %s", resp.Matches[0])
	}
}

func TestFindFile_InvalidGlob(t *testing.T) {
	workspaceRoot := "/workspace"
	maxFileSize := int64(1024 * 1024)

	fs := services.NewMockFileSystem(maxFileSize)
	fs.CreateDir("/workspace")

	mockRunner := &services.MockCommandRunner{
		RunFunc: func(ctx context.Context, cmd []string) ([]byte, error) {
			// Simulate fd error for invalid glob
			return []byte(""), &services.MockExitError{Code: 2}
		},
	}

	ctx := &models.WorkspaceContext{
		FS:              fs,
		BinaryDetector:  services.NewMockBinaryDetector(),
		ChecksumManager: services.NewChecksumManager(),
		MaxFileSize:     maxFileSize,
		WorkspaceRoot:   workspaceRoot,
		CommandRunner:   mockRunner,
	}

	_, err := FindFile(ctx, "[", "", 0, 0, 100)
	if err == nil {
		t.Fatal("expected error for invalid glob, got nil")
	}
}

func TestFindFile_PathOutsideWorkspace(t *testing.T) {
	workspaceRoot := "/workspace"
	maxFileSize := int64(1024 * 1024)

	fs := services.NewMockFileSystem(maxFileSize)
	fs.CreateDir("/workspace")

	ctx := &models.WorkspaceContext{
		FS:              fs,
		BinaryDetector:  services.NewMockBinaryDetector(),
		ChecksumManager: services.NewChecksumManager(),
		MaxFileSize:     maxFileSize,
		WorkspaceRoot:   workspaceRoot,
		CommandRunner:   &services.MockCommandRunner{},
	}

	_, err := FindFile(ctx, "*.go", "../outside", 0, 0, 100)
	if err != models.ErrOutsideWorkspace {
		t.Errorf("expected ErrOutsideWorkspace, got %v", err)
	}
}

func TestFindFile_NonExistentPath(t *testing.T) {
	workspaceRoot := "/workspace"
	maxFileSize := int64(1024 * 1024)

	fs := services.NewMockFileSystem(maxFileSize)
	fs.CreateDir("/workspace")

	ctx := &models.WorkspaceContext{
		FS:              fs,
		BinaryDetector:  services.NewMockBinaryDetector(),
		ChecksumManager: services.NewChecksumManager(),
		MaxFileSize:     maxFileSize,
		WorkspaceRoot:   workspaceRoot,
		CommandRunner:   &services.MockCommandRunner{},
	}

	_, err := FindFile(ctx, "*.go", "nonexistent/dir", 0, 0, 100)
	if err != models.ErrFileMissing {
		t.Errorf("expected ErrFileMissing, got %v", err)
	}
}

func TestFindFile_NegativeLimit(t *testing.T) {
	workspaceRoot := "/workspace"
	maxFileSize := int64(1024 * 1024)

	fs := services.NewMockFileSystem(maxFileSize)
	fs.CreateDir("/workspace")

	ctx := &models.WorkspaceContext{
		FS:              fs,
		BinaryDetector:  services.NewMockBinaryDetector(),
		ChecksumManager: services.NewChecksumManager(),
		MaxFileSize:     maxFileSize,
		WorkspaceRoot:   workspaceRoot,
		CommandRunner:   &services.MockCommandRunner{},
	}

	_, err := FindFile(ctx, "*.go", "", 0, 0, -1)
	if err != models.ErrInvalidPaginationLimit {
		t.Errorf("expected ErrInvalidPaginationLimit, got %v", err)
	}
}

func TestFindFile_CommandFailure(t *testing.T) {
	workspaceRoot := "/workspace"
	maxFileSize := int64(1024 * 1024)

	fs := services.NewMockFileSystem(maxFileSize)
	fs.CreateDir("/workspace")

	mockRunner := &services.MockCommandRunner{
		RunFunc: func(ctx context.Context, cmd []string) ([]byte, error) {
			return []byte(""), &services.MockExitError{Code: 2}
		},
	}

	ctx := &models.WorkspaceContext{
		FS:              fs,
		BinaryDetector:  services.NewMockBinaryDetector(),
		ChecksumManager: services.NewChecksumManager(),
		MaxFileSize:     maxFileSize,
		WorkspaceRoot:   workspaceRoot,
		CommandRunner:   mockRunner,
	}

	_, err := FindFile(ctx, "*.go", "", 0, 0, 100)
	if err == nil {
		t.Fatal("expected error for command failure, got nil")
	}
}

func TestFindFile_ShellInjection(t *testing.T) {
	workspaceRoot := "/workspace"
	maxFileSize := int64(1024 * 1024)

	fs := services.NewMockFileSystem(maxFileSize)
	fs.CreateDir("/workspace")

	var capturedCmd []string
	mockRunner := &services.MockCommandRunner{
		RunFunc: func(ctx context.Context, cmd []string) ([]byte, error) {
			capturedCmd = cmd
			return []byte(""), nil
		},
	}

	ctx := &models.WorkspaceContext{
		FS:              fs,
		BinaryDetector:  services.NewMockBinaryDetector(),
		ChecksumManager: services.NewChecksumManager(),
		MaxFileSize:     maxFileSize,
		WorkspaceRoot:   workspaceRoot,
		CommandRunner:   mockRunner,
	}

	pattern := "*.go; rm -rf /"
	_, _ = FindFile(ctx, pattern, "", 0, 0, 100)

	// Verify pattern is passed as literal argument, not shell-interpreted
	found := slices.Contains(capturedCmd, pattern)

	if !found {
		t.Errorf("expected pattern to be passed as literal argument, got cmd: %v", capturedCmd)
	}
}

func TestFindFile_UnicodeFilenames(t *testing.T) {
	workspaceRoot := "/workspace"
	maxFileSize := int64(1024 * 1024)

	fs := services.NewMockFileSystem(maxFileSize)
	fs.CreateDir("/workspace")

	mockRunner := &services.MockCommandRunner{
		RunFunc: func(ctx context.Context, cmd []string) ([]byte, error) {
			output := "/workspace/🚀.txt\n/workspace/文件.txt\n"
			return []byte(output), nil
		},
	}

	ctx := &models.WorkspaceContext{
		FS:              fs,
		BinaryDetector:  services.NewMockBinaryDetector(),
		ChecksumManager: services.NewChecksumManager(),
		MaxFileSize:     maxFileSize,
		WorkspaceRoot:   workspaceRoot,
		CommandRunner:   mockRunner,
	}

	resp, err := FindFile(ctx, "*.txt", "", 0, 0, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Matches) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(resp.Matches))
	}

	// Verify unicode handling
	foundEmoji := false
	foundChinese := false
	for _, match := range resp.Matches {
		if match == "🚀.txt" {
			foundEmoji = true
		}
		if match == "文件.txt" {
			foundChinese = true
		}
	}

	if !foundEmoji {
		t.Error("expected to find emoji filename")
	}
	if !foundChinese {
		t.Error("expected to find Chinese filename")
	}
}

func TestFindFile_DeeplyNested(t *testing.T) {
	workspaceRoot := "/workspace"
	maxFileSize := int64(1024 * 1024)

	fs := services.NewMockFileSystem(maxFileSize)
	fs.CreateDir("/workspace")

	// Simulate path with 100 segments
	deepPath := "/workspace"
	for i := range 100 {
		deepPath += fmt.Sprintf("/dir%d", i)
	}
	deepPath += "/file.txt"

	mockRunner := &services.MockCommandRunner{
		RunFunc: func(ctx context.Context, cmd []string) ([]byte, error) {
			return []byte(deepPath + "\n"), nil
		},
	}

	ctx := &models.WorkspaceContext{
		FS:              fs,
		BinaryDetector:  services.NewMockBinaryDetector(),
		ChecksumManager: services.NewChecksumManager(),
		MaxFileSize:     maxFileSize,
		WorkspaceRoot:   workspaceRoot,
		CommandRunner:   mockRunner,
	}

	resp, err := FindFile(ctx, "*.txt", "", 0, 0, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(resp.Matches))
	}
}

func TestFindFile_PatternTraversal(t *testing.T) {
	workspaceRoot := "/workspace"
	maxFileSize := int64(1024 * 1024)

	fs := services.NewMockFileSystem(maxFileSize)
	fs.CreateDir("/workspace")

	ctx := &models.WorkspaceContext{
		FS:              fs,
		BinaryDetector:  services.NewMockBinaryDetector(),
		ChecksumManager: services.NewChecksumManager(),
		MaxFileSize:     maxFileSize,
		WorkspaceRoot:   workspaceRoot,
		CommandRunner:   &services.MockCommandRunner{},
	}

	_, err := FindFile(ctx, "../*.go", "", 0, 0, 100)
	if err == nil {
		t.Fatal("expected error for pattern with path traversal, got nil")
	}
}

func TestFindFile_AbsolutePattern(t *testing.T) {
	workspaceRoot := "/workspace"
	maxFileSize := int64(1024 * 1024)

	fs := services.NewMockFileSystem(maxFileSize)
	fs.CreateDir("/workspace")

	ctx := &models.WorkspaceContext{
		FS:              fs,
		BinaryDetector:  services.NewMockBinaryDetector(),
		ChecksumManager: services.NewChecksumManager(),
		MaxFileSize:     maxFileSize,
		WorkspaceRoot:   workspaceRoot,
		CommandRunner:   &services.MockCommandRunner{},
	}

	_, err := FindFile(ctx, "/etc/*.conf", "", 0, 0, 100)
	if err == nil {
		t.Fatal("expected error for absolute pattern, got nil")
	}
}

func TestFindFile_NoMatches(t *testing.T) {
	workspaceRoot := "/workspace"
	maxFileSize := int64(1024 * 1024)

	fs := services.NewMockFileSystem(maxFileSize)
	fs.CreateDir("/workspace")

	mockRunner := &services.MockCommandRunner{
		RunFunc: func(ctx context.Context, cmd []string) ([]byte, error) {
			// Simulate fd returning exit code 1 (no matches)
			return []byte(""), &services.MockExitError{Code: 1}
		},
	}

	ctx := &models.WorkspaceContext{
		FS:              fs,
		BinaryDetector:  services.NewMockBinaryDetector(),
		ChecksumManager: services.NewChecksumManager(),
		MaxFileSize:     maxFileSize,
		WorkspaceRoot:   workspaceRoot,
		CommandRunner:   mockRunner,
	}

	resp, err := FindFile(ctx, "*.nonexistent", "", 0, 0, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Matches) != 0 {
		t.Errorf("expected 0 matches, got %d", len(resp.Matches))
	}

	if resp.Truncated {
		t.Error("expected Truncated=false for no matches")
	}
}
