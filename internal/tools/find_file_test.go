package tools

import (
	"context"
	"fmt"
	"io"
	"slices"
	"strings"
	"testing"

	"github.com/Cyclone1070/iav/internal/tools/models"
	"github.com/Cyclone1070/iav/internal/tools/services"
)

func TestFindFile_BasicGlob(t *testing.T) {
	workspaceRoot := "/workspace"
	maxFileSize := int64(1024 * 1024)

	fs := services.NewMockFileSystem(maxFileSize)
	fs.CreateDir("/workspace")

	mockRunner := &services.MockCommandExecutor{
		StartFunc: func(ctx context.Context, cmd []string, opts models.ProcessOptions) (models.Process, io.Reader, io.Reader, error) {
			// Simulate fd output
			output := "/workspace/a/b/file.go\n/workspace/a/file.go\n"
			return &services.MockProcess{}, strings.NewReader(output), strings.NewReader(""), nil
		},
	}

	ctx := &models.WorkspaceContext{
		FS:              fs,
		BinaryDetector:  services.NewMockBinaryDetector(),
		ChecksumManager: services.NewChecksumManager(),
		WorkspaceRoot:   workspaceRoot,
		CommandExecutor: mockRunner,
	}

	resp, err := FindFile(context.Background(), ctx, models.FindFileRequest{Pattern: "*.go", SearchPath: "", MaxDepth: 0, IncludeIgnored: false, Offset: 0, Limit: 100})
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

	mockRunner := &services.MockCommandExecutor{
		StartFunc: func(ctx context.Context, cmd []string, opts models.ProcessOptions) (models.Process, io.Reader, io.Reader, error) {
			return &services.MockProcess{}, strings.NewReader(output), strings.NewReader(""), nil
		},
	}

	ctx := &models.WorkspaceContext{
		FS:              fs,
		BinaryDetector:  services.NewMockBinaryDetector(),
		ChecksumManager: services.NewChecksumManager(),
		WorkspaceRoot:   workspaceRoot,
		CommandExecutor: mockRunner,
	}

	// Request offset=2, limit=2
	resp, err := FindFile(context.Background(), ctx, models.FindFileRequest{Pattern: "*.txt", SearchPath: "", MaxDepth: 0, IncludeIgnored: false, Offset: 2, Limit: 2})
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

	mockRunner := &services.MockCommandExecutor{
		StartFunc: func(ctx context.Context, cmd []string, opts models.ProcessOptions) (models.Process, io.Reader, io.Reader, error) {
			// Simulate fd error for invalid glob
			return &services.MockProcess{
				WaitFunc: func() error {
					return &services.MockExitError{Code: 2}
				},
			}, strings.NewReader(""), strings.NewReader(""), nil
		},
	}

	ctx := &models.WorkspaceContext{
		FS:              fs,
		BinaryDetector:  services.NewMockBinaryDetector(),
		ChecksumManager: services.NewChecksumManager(),
		WorkspaceRoot:   workspaceRoot,
		CommandExecutor: mockRunner,
	}

	_, err := FindFile(context.Background(), ctx, models.FindFileRequest{Pattern: "[", SearchPath: "", MaxDepth: 0, IncludeIgnored: false, Offset: 0, Limit: 100})
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
		WorkspaceRoot:   workspaceRoot,
		CommandExecutor: &services.MockCommandExecutor{},
	}

	_, err := FindFile(context.Background(), ctx, models.FindFileRequest{Pattern: "*.go", SearchPath: "../outside", MaxDepth: 0, IncludeIgnored: false, Offset: 0, Limit: 100})
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
		WorkspaceRoot:   workspaceRoot,
		CommandExecutor: &services.MockCommandExecutor{},
	}

	_, err := FindFile(context.Background(), ctx, models.FindFileRequest{Pattern: "*.go", SearchPath: "nonexistent/dir", MaxDepth: 0, IncludeIgnored: false, Offset: 0, Limit: 100})
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
		WorkspaceRoot:   workspaceRoot,
		CommandExecutor: &services.MockCommandExecutor{},
	}

	_, err := FindFile(context.Background(), ctx, models.FindFileRequest{Pattern: "*.go", SearchPath: "", MaxDepth: 0, IncludeIgnored: false, Offset: 0, Limit: -1})
	if err != models.ErrInvalidPaginationLimit {
		t.Errorf("expected ErrInvalidPaginationLimit, got %v", err)
	}
}

func TestFindFile_CommandFailure(t *testing.T) {
	workspaceRoot := "/workspace"
	maxFileSize := int64(1024 * 1024)

	fs := services.NewMockFileSystem(maxFileSize)
	fs.CreateDir("/workspace")

	mockRunner := &services.MockCommandExecutor{
		StartFunc: func(ctx context.Context, cmd []string, opts models.ProcessOptions) (models.Process, io.Reader, io.Reader, error) {
			return &services.MockProcess{
				WaitFunc: func() error {
					return &services.MockExitError{Code: 2}
				},
			}, strings.NewReader(""), strings.NewReader(""), nil
		},
	}

	ctx := &models.WorkspaceContext{
		FS:              fs,
		BinaryDetector:  services.NewMockBinaryDetector(),
		ChecksumManager: services.NewChecksumManager(),
		WorkspaceRoot:   workspaceRoot,
		CommandExecutor: mockRunner,
	}

	_, err := FindFile(context.Background(), ctx, models.FindFileRequest{Pattern: "*.go", SearchPath: "", MaxDepth: 0, IncludeIgnored: false, Offset: 0, Limit: 100})
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
	mockRunner := &services.MockCommandExecutor{
		StartFunc: func(ctx context.Context, cmd []string, opts models.ProcessOptions) (models.Process, io.Reader, io.Reader, error) {
			capturedCmd = cmd
			return &services.MockProcess{}, strings.NewReader(""), strings.NewReader(""), nil
		},
	}

	ctx := &models.WorkspaceContext{
		FS:              fs,
		BinaryDetector:  services.NewMockBinaryDetector(),
		ChecksumManager: services.NewChecksumManager(),
		WorkspaceRoot:   workspaceRoot,
		CommandExecutor: mockRunner,
	}

	pattern := "*.go; rm -rf /"
	_, _ = FindFile(context.Background(), ctx, models.FindFileRequest{Pattern: pattern, SearchPath: "", MaxDepth: 0, IncludeIgnored: false, Offset: 0, Limit: 100})

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

	mockRunner := &services.MockCommandExecutor{
		StartFunc: func(ctx context.Context, cmd []string, opts models.ProcessOptions) (models.Process, io.Reader, io.Reader, error) {
			output := "/workspace/🚀.txt\n/workspace/文件.txt\n"
			return &services.MockProcess{}, strings.NewReader(output), strings.NewReader(""), nil
		},
	}

	ctx := &models.WorkspaceContext{
		FS:              fs,
		BinaryDetector:  services.NewMockBinaryDetector(),
		ChecksumManager: services.NewChecksumManager(),
		WorkspaceRoot:   workspaceRoot,
		CommandExecutor: mockRunner,
	}

	resp, err := FindFile(context.Background(), ctx, models.FindFileRequest{Pattern: "*.txt", SearchPath: "", MaxDepth: 0, IncludeIgnored: false, Offset: 0, Limit: 100})
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

	mockRunner := &services.MockCommandExecutor{
		StartFunc: func(ctx context.Context, cmd []string, opts models.ProcessOptions) (models.Process, io.Reader, io.Reader, error) {
			return &services.MockProcess{}, strings.NewReader(deepPath + "\n"), strings.NewReader(""), nil
		},
	}

	ctx := &models.WorkspaceContext{
		FS:              fs,
		BinaryDetector:  services.NewMockBinaryDetector(),
		ChecksumManager: services.NewChecksumManager(),
		WorkspaceRoot:   workspaceRoot,
		CommandExecutor: mockRunner,
	}

	resp, err := FindFile(context.Background(), ctx, models.FindFileRequest{Pattern: "*.txt", SearchPath: "", MaxDepth: 0, IncludeIgnored: false, Offset: 0, Limit: 100})
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
		WorkspaceRoot:   workspaceRoot,
		CommandExecutor: &services.MockCommandExecutor{},
	}

	_, err := FindFile(context.Background(), ctx, models.FindFileRequest{Pattern: "../*.go", SearchPath: "", MaxDepth: 0, IncludeIgnored: false, Offset: 0, Limit: 100})
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
		WorkspaceRoot:   workspaceRoot,
		CommandExecutor: &services.MockCommandExecutor{},
	}

	_, err := FindFile(context.Background(), ctx, models.FindFileRequest{Pattern: "/etc/*.conf", SearchPath: "", MaxDepth: 0, IncludeIgnored: false, Offset: 0, Limit: 100})
	if err == nil {
		t.Fatal("expected error for absolute pattern, got nil")
	}
}

func TestFindFile_NoMatches(t *testing.T) {
	workspaceRoot := "/workspace"
	maxFileSize := int64(1024 * 1024)

	fs := services.NewMockFileSystem(maxFileSize)
	fs.CreateDir("/workspace")

	mockRunner := &services.MockCommandExecutor{
		StartFunc: func(ctx context.Context, cmd []string, opts models.ProcessOptions) (models.Process, io.Reader, io.Reader, error) {
			// Simulate fd returning exit code 0 (no matches, empty output)
			return &services.MockProcess{}, strings.NewReader(""), strings.NewReader(""), nil
		},
	}

	ctx := &models.WorkspaceContext{
		FS:              fs,
		BinaryDetector:  services.NewMockBinaryDetector(),
		ChecksumManager: services.NewChecksumManager(),
		WorkspaceRoot:   workspaceRoot,
		CommandExecutor: mockRunner,
	}

	resp, err := FindFile(context.Background(), ctx, models.FindFileRequest{Pattern: "*.nonexistent", SearchPath: "", MaxDepth: 0, IncludeIgnored: false, Offset: 0, Limit: 100})
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
func TestFindFile_IncludeIgnored(t *testing.T) {
	workspaceRoot := "/workspace"
	maxFileSize := int64(1024 * 1024)

	fs := services.NewMockFileSystem(maxFileSize)
	fs.CreateDir("/workspace")

	// Test with includeIgnored=false (default behavior, should respect gitignore)
	mockRunner := &services.MockCommandExecutor{
		StartFunc: func(ctx context.Context, cmd []string, opts models.ProcessOptions) (models.Process, io.Reader, io.Reader, error) {
			// Verify --no-ignore is NOT present
			if slices.Contains(cmd, "--no-ignore") {
				t.Error("expected --no-ignore to NOT be present when includeIgnored=false")
			}
			// Simulate fd output without ignored files
			output := "/workspace/visible.go\n"
			return &services.MockProcess{}, strings.NewReader(output), strings.NewReader(""), nil
		},
	}

	ctx := &models.WorkspaceContext{
		FS:              fs,
		BinaryDetector:  services.NewMockBinaryDetector(),
		ChecksumManager: services.NewChecksumManager(),
		WorkspaceRoot:   workspaceRoot,
		CommandExecutor: mockRunner,
	}

	resp, err := FindFile(context.Background(), ctx, models.FindFileRequest{Pattern: "*.go", SearchPath: "", MaxDepth: 0, IncludeIgnored: false, Offset: 0, Limit: 100})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(resp.Matches))
	}

	// Test with includeIgnored=true (should include ignored files)
	mockRunner.StartFunc = func(ctx context.Context, cmd []string, opts models.ProcessOptions) (models.Process, io.Reader, io.Reader, error) {
		// Verify --no-ignore IS present
		if !slices.Contains(cmd, "--no-ignore") {
			t.Error("expected --no-ignore to be present when includeIgnored=true")
		}
		// Simulate fd output with ignored files
		output := "/workspace/ignored.go\n/workspace/visible.go\n"
		return &services.MockProcess{}, strings.NewReader(output), strings.NewReader(""), nil
	}

	resp, err = FindFile(context.Background(), ctx, models.FindFileRequest{Pattern: "*.go", SearchPath: "", MaxDepth: 0, IncludeIgnored: true, Offset: 0, Limit: 100})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Matches) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(resp.Matches))
	}

	// Verify both files are present
	foundIgnored := false
	foundVisible := false
	for _, match := range resp.Matches {
		if match == "ignored.go" {
			foundIgnored = true
		}
		if match == "visible.go" {
			foundVisible = true
		}
	}

	if !foundIgnored {
		t.Error("expected to find ignored.go when includeIgnored=true")
	}
	if !foundVisible {
		t.Error("expected to find visible.go when includeIgnored=true")
	}
}
