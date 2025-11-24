package tools

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/Cyclone1070/deployforme/internal/tools/models"
	"github.com/Cyclone1070/deployforme/internal/tools/services"
)

func TestSearchContent_BasicRegex(t *testing.T) {
	workspaceRoot := "/workspace"
	maxFileSize := int64(1024 * 1024)

	fs := services.NewMockFileSystem(maxFileSize)
	fs.CreateDir("/workspace")

	// Simulate rg JSON output
	rgOutput := `{"type":"match","data":{"path":{"text":"/workspace/file.go"},"lines":{"text":"func foo()"},"line_number":10}}
{"type":"match","data":{"path":{"text":"/workspace/file.go"},"lines":{"text":"func bar()"},"line_number":20}}`

	mockRunner := &services.MockCommandExecutor{
		RunFunc: func(ctx context.Context, cmd []string) ([]byte, error) {
			return []byte(rgOutput), nil
		},
	}

	ctx := &models.WorkspaceContext{
		FS:              fs,
		BinaryDetector:  services.NewMockBinaryDetector(),
		ChecksumManager: services.NewChecksumManager(),
		MaxFileSize:     maxFileSize,
		WorkspaceRoot:   workspaceRoot,
		CommandExecutor: mockRunner,
	}

	resp, err := SearchContent(ctx, "func .*", "", true, 0, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Matches) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(resp.Matches))
	}

	// Verify first match
	if resp.Matches[0].File != "file.go" {
		t.Errorf("expected file.go, got %s", resp.Matches[0].File)
	}
	if resp.Matches[0].LineNumber != 10 {
		t.Errorf("expected line 10, got %d", resp.Matches[0].LineNumber)
	}
	if resp.Matches[0].LineContent != "func foo()" {
		t.Errorf("expected 'func foo()', got %q", resp.Matches[0].LineContent)
	}
}

func TestSearchContent_CaseInsensitive(t *testing.T) {
	workspaceRoot := "/workspace"
	maxFileSize := int64(1024 * 1024)

	fs := services.NewMockFileSystem(maxFileSize)
	fs.CreateDir("/workspace")

	var capturedCmd []string
	mockRunner := &services.MockCommandExecutor{
		RunFunc: func(ctx context.Context, cmd []string) ([]byte, error) {
			capturedCmd = cmd
			return []byte(""), &services.MockExitError{Code: 1} // No matches
		},
	}

	ctx := &models.WorkspaceContext{
		FS:              fs,
		BinaryDetector:  services.NewMockBinaryDetector(),
		ChecksumManager: services.NewChecksumManager(),
		MaxFileSize:     maxFileSize,
		WorkspaceRoot:   workspaceRoot,
		CommandExecutor: mockRunner,
	}

	_, _ = SearchContent(ctx, "pattern", "", false, 0, 100)

	// Verify -i flag is present
	foundFlag := slices.Contains(capturedCmd, "-i")

	if !foundFlag {
		t.Errorf("expected -i flag for case-insensitive search, got cmd: %v", capturedCmd)
	}
}

func TestSearchContent_PathOutsideWorkspace(t *testing.T) {
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
		CommandExecutor: &services.MockCommandExecutor{},
	}

	_, err := SearchContent(ctx, "pattern", "../outside", true, 0, 100)
	if err != models.ErrOutsideWorkspace {
		t.Errorf("expected ErrOutsideWorkspace, got %v", err)
	}
}

func TestSearchContent_EmptyQuery(t *testing.T) {
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
		CommandExecutor: &services.MockCommandExecutor{},
	}

	_, err := SearchContent(ctx, "", "", true, 0, 100)
	if err == nil {
		t.Fatal("expected error for empty query, got nil")
	}
}

func TestSearchContent_HugeLimit(t *testing.T) {
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
		CommandExecutor: &services.MockCommandExecutor{},
	}

	_, err := SearchContent(ctx, "pattern", "", true, 0, 1000000)
	if err != models.ErrInvalidPaginationLimit {
		t.Errorf("expected ErrInvalidPaginationLimit, got %v", err)
	}
}

func TestSearchContent_VeryLongLine(t *testing.T) {
	workspaceRoot := "/workspace"
	maxFileSize := int64(1024 * 1024)

	fs := services.NewMockFileSystem(maxFileSize)
	fs.CreateDir("/workspace")

	// Create a very long line (1MB)
	longLine := strings.Repeat("a", 1024*1024)
	rgOutput := fmt.Sprintf(`{"type":"match","data":{"path":{"text":"/workspace/file.txt"},"lines":{"text":"%s"},"line_number":1}}`, longLine)

	mockRunner := &services.MockCommandExecutor{
		RunFunc: func(ctx context.Context, cmd []string) ([]byte, error) {
			return []byte(rgOutput), nil
		},
	}

	ctx := &models.WorkspaceContext{
		FS:              fs,
		BinaryDetector:  services.NewMockBinaryDetector(),
		ChecksumManager: services.NewChecksumManager(),
		MaxFileSize:     maxFileSize,
		WorkspaceRoot:   workspaceRoot,
		CommandExecutor: mockRunner,
	}

	resp, err := SearchContent(ctx, "pattern", "", true, 0, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(resp.Matches))
	}

	// Verify line was truncated
	if len(resp.Matches[0].LineContent) > 10100 {
		t.Errorf("expected line to be truncated to ~10000 chars, got %d", len(resp.Matches[0].LineContent))
	}

	if !strings.Contains(resp.Matches[0].LineContent, "[truncated]") {
		t.Error("expected truncation marker in line content")
	}
}

func TestSearchContent_CommandInjection(t *testing.T) {
	workspaceRoot := "/workspace"
	maxFileSize := int64(1024 * 1024)

	fs := services.NewMockFileSystem(maxFileSize)
	fs.CreateDir("/workspace")

	var capturedCmd []string
	mockRunner := &services.MockCommandExecutor{
		RunFunc: func(ctx context.Context, cmd []string) ([]byte, error) {
			capturedCmd = cmd
			return []byte(""), &services.MockExitError{Code: 1}
		},
	}

	ctx := &models.WorkspaceContext{
		FS:              fs,
		BinaryDetector:  services.NewMockBinaryDetector(),
		ChecksumManager: services.NewChecksumManager(),
		MaxFileSize:     maxFileSize,
		WorkspaceRoot:   workspaceRoot,
		CommandExecutor: mockRunner,
	}

	query := "foo; rm -rf /"
	_, _ = SearchContent(ctx, query, "", true, 0, 100)

	// Verify query is passed as literal argument
	found := slices.Contains(capturedCmd, query)

	if !found {
		t.Errorf("expected query to be passed as literal argument, got cmd: %v", capturedCmd)
	}
}

func TestSearchContent_NoMatches(t *testing.T) {
	workspaceRoot := "/workspace"
	maxFileSize := int64(1024 * 1024)

	fs := services.NewMockFileSystem(maxFileSize)
	fs.CreateDir("/workspace")

	mockRunner := &services.MockCommandExecutor{
		RunFunc: func(ctx context.Context, cmd []string) ([]byte, error) {
			// Simulate rg returning exit code 1 (no matches)
			return []byte(""), &services.MockExitError{Code: 1}
		},
	}

	ctx := &models.WorkspaceContext{
		FS:              fs,
		BinaryDetector:  services.NewMockBinaryDetector(),
		ChecksumManager: services.NewChecksumManager(),
		MaxFileSize:     maxFileSize,
		WorkspaceRoot:   workspaceRoot,
		CommandExecutor: mockRunner,
	}

	resp, err := SearchContent(ctx, "nonexistent", "", true, 0, 100)
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

func TestSearchContent_Pagination(t *testing.T) {
	workspaceRoot := "/workspace"
	maxFileSize := int64(1024 * 1024)

	fs := services.NewMockFileSystem(maxFileSize)
	fs.CreateDir("/workspace")

	// Simulate 10 matches
	var rgOutput string
	for i := range 10 {
		rgOutput += fmt.Sprintf(`{"type":"match","data":{"path":{"text":"/workspace/file.txt"},"lines":{"text":"line %d"},"line_number":%d}}
`, i, i+1)
	}

	mockRunner := &services.MockCommandExecutor{
		RunFunc: func(ctx context.Context, cmd []string) ([]byte, error) {
			return []byte(rgOutput), nil
		},
	}

	ctx := &models.WorkspaceContext{
		FS:              fs,
		BinaryDetector:  services.NewMockBinaryDetector(),
		ChecksumManager: services.NewChecksumManager(),
		MaxFileSize:     maxFileSize,
		WorkspaceRoot:   workspaceRoot,
		CommandExecutor: mockRunner,
	}

	// Request offset=2, limit=2
	resp, err := SearchContent(ctx, "pattern", "", true, 2, 2)
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

	// Verify correct slice (lines 3 and 4)
	if resp.Matches[0].LineNumber != 3 {
		t.Errorf("expected line 3, got %d", resp.Matches[0].LineNumber)
	}
}

func TestSearchContent_MultipleFiles(t *testing.T) {
	workspaceRoot := "/workspace"
	maxFileSize := int64(1024 * 1024)

	fs := services.NewMockFileSystem(maxFileSize)
	fs.CreateDir("/workspace")

	// Simulate matches from multiple files
	rgOutput := `{"type":"match","data":{"path":{"text":"/workspace/b.txt"},"lines":{"text":"match"},"line_number":5}}
{"type":"match","data":{"path":{"text":"/workspace/a.txt"},"lines":{"text":"match"},"line_number":10}}
{"type":"match","data":{"path":{"text":"/workspace/a.txt"},"lines":{"text":"match"},"line_number":5}}`

	mockRunner := &services.MockCommandExecutor{
		RunFunc: func(ctx context.Context, cmd []string) ([]byte, error) {
			return []byte(rgOutput), nil
		},
	}

	ctx := &models.WorkspaceContext{
		FS:              fs,
		BinaryDetector:  services.NewMockBinaryDetector(),
		ChecksumManager: services.NewChecksumManager(),
		MaxFileSize:     maxFileSize,
		WorkspaceRoot:   workspaceRoot,
		CommandExecutor: mockRunner,
	}

	resp, err := SearchContent(ctx, "pattern", "", true, 0, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Matches) != 3 {
		t.Fatalf("expected 3 matches, got %d", len(resp.Matches))
	}

	// Verify sorting (by file, then line number)
	// Expected order: a.txt:5, a.txt:10, b.txt:5
	if resp.Matches[0].File != "a.txt" || resp.Matches[0].LineNumber != 5 {
		t.Errorf("expected a.txt:5, got %s:%d", resp.Matches[0].File, resp.Matches[0].LineNumber)
	}
	if resp.Matches[1].File != "a.txt" || resp.Matches[1].LineNumber != 10 {
		t.Errorf("expected a.txt:10, got %s:%d", resp.Matches[1].File, resp.Matches[1].LineNumber)
	}
	if resp.Matches[2].File != "b.txt" || resp.Matches[2].LineNumber != 5 {
		t.Errorf("expected b.txt:5, got %s:%d", resp.Matches[2].File, resp.Matches[2].LineNumber)
	}
}

func TestSearchContent_InvalidJSON(t *testing.T) {
	workspaceRoot := "/workspace"
	maxFileSize := int64(1024 * 1024)

	fs := services.NewMockFileSystem(maxFileSize)
	fs.CreateDir("/workspace")

	// Mix valid and invalid JSON
	rgOutput := `{"type":"match","data":{"path":{"text":"/workspace/file.txt"},"lines":{"text":"valid"},"line_number":1}}
invalid json line
{"type":"match","data":{"path":{"text":"/workspace/file.txt"},"lines":{"text":"also valid"},"line_number":2}}`

	mockRunner := &services.MockCommandExecutor{
		RunFunc: func(ctx context.Context, cmd []string) ([]byte, error) {
			return []byte(rgOutput), nil
		},
	}

	ctx := &models.WorkspaceContext{
		FS:              fs,
		BinaryDetector:  services.NewMockBinaryDetector(),
		ChecksumManager: services.NewChecksumManager(),
		MaxFileSize:     maxFileSize,
		WorkspaceRoot:   workspaceRoot,
		CommandExecutor: mockRunner,
	}

	resp, err := SearchContent(ctx, "pattern", "", true, 0, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should skip invalid JSON and return only 2 valid matches
	if len(resp.Matches) != 2 {
		t.Fatalf("expected 2 matches (invalid JSON skipped), got %d", len(resp.Matches))
	}
}

func TestSearchContent_CommandFailure(t *testing.T) {
	workspaceRoot := "/workspace"
	maxFileSize := int64(1024 * 1024)

	fs := services.NewMockFileSystem(maxFileSize)
	fs.CreateDir("/workspace")

	mockRunner := &services.MockCommandExecutor{
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
		CommandExecutor: mockRunner,
	}

	_, err := SearchContent(ctx, "pattern", "", true, 0, 100)
	if err == nil {
		t.Fatal("expected error for command failure, got nil")
	}
}
