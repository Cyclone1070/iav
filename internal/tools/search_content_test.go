package tools

import (
	"context"
	"fmt"
	"io"
	"slices"
	"strings"
	"testing"

	"github.com/Cyclone1070/iav/internal/config"
	"github.com/Cyclone1070/iav/internal/testing/mocks"
	"github.com/Cyclone1070/iav/internal/tools/models"
	"github.com/Cyclone1070/iav/internal/tools/services"
)

func TestSearchContent_BasicRegex(t *testing.T) {
	workspaceRoot := "/workspace"

	fs := mocks.NewMockFileSystem()
	fs.CreateDir("/workspace")

	// Simulate rg JSON output
	rgOutput := `{"type":"match","data":{"path":{"text":"/workspace/file.go"},"lines":{"text":"func foo()"},"line_number":10}}
{"type":"match","data":{"path":{"text":"/workspace/file.go"},"lines":{"text":"func bar()"},"line_number":20}}`

	mockRunner := &mocks.MockCommandExecutor{
		StartFunc: func(ctx context.Context, cmd []string, opts models.ProcessOptions) (models.Process, io.Reader, io.Reader, error) {
			return &mocks.MockProcess{}, strings.NewReader(rgOutput), strings.NewReader(""), nil
		},
	}

	ctx := &models.WorkspaceContext{
		FS:              fs,
		BinaryDetector:  mocks.NewMockBinaryDetector(),
		ChecksumManager: services.NewChecksumManager(),
		WorkspaceRoot:   workspaceRoot,
		CommandExecutor: mockRunner,
		Config:          *config.DefaultConfig(),
	}

	resp, err := SearchContent(context.Background(), ctx, models.SearchContentRequest{Query: "func .*", SearchPath: "", CaseSensitive: true, IncludeIgnored: false, Offset: 0, Limit: 100})
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

	fs := mocks.NewMockFileSystem()
	fs.CreateDir("/workspace")

	var capturedCmd []string
	mockRunner := &mocks.MockCommandExecutor{
		StartFunc: func(ctx context.Context, cmd []string, opts models.ProcessOptions) (models.Process, io.Reader, io.Reader, error) {
			capturedCmd = cmd
			return &mocks.MockProcess{
				WaitFunc: func() error {
					return &mocks.MockExitError{Code: 1} // No matches
				},
			}, strings.NewReader(""), strings.NewReader(""), nil
		},
	}

	ctx := &models.WorkspaceContext{
		FS:              fs,
		BinaryDetector:  mocks.NewMockBinaryDetector(),
		ChecksumManager: services.NewChecksumManager(),
		WorkspaceRoot:   workspaceRoot,
		CommandExecutor: mockRunner,
		Config:          *config.DefaultConfig(),
	}

	_, _ = SearchContent(context.Background(), ctx, models.SearchContentRequest{Query: "pattern", SearchPath: "", CaseSensitive: false, IncludeIgnored: false, Offset: 0, Limit: 100})

	// Verify -i flag is present
	foundFlag := slices.Contains(capturedCmd, "-i")

	if !foundFlag {
		t.Errorf("expected -i flag for case-insensitive search, got cmd: %v", capturedCmd)
	}
}

func TestSearchContent_PathOutsideWorkspace(t *testing.T) {
	workspaceRoot := "/workspace"

	fs := mocks.NewMockFileSystem()
	fs.CreateDir("/workspace")

	ctx := &models.WorkspaceContext{
		FS:              fs,
		BinaryDetector:  mocks.NewMockBinaryDetector(),
		ChecksumManager: services.NewChecksumManager(),
		WorkspaceRoot:   workspaceRoot,
		CommandExecutor: &mocks.MockCommandExecutor{},
		Config:          *config.DefaultConfig(),
	}

	_, err := SearchContent(context.Background(), ctx, models.SearchContentRequest{Query: "pattern", SearchPath: "../outside", CaseSensitive: true, IncludeIgnored: false, Offset: 0, Limit: 100})
	if err != models.ErrOutsideWorkspace {
		t.Errorf("expected ErrOutsideWorkspace, got %v", err)
	}
}

func TestSearchContent_VeryLongLine(t *testing.T) {
	workspaceRoot := "/workspace"

	fs := mocks.NewMockFileSystem()
	fs.CreateDir("/workspace")

	// Create a very long line (1MB)
	longLine := strings.Repeat("a", 1024*1024)
	rgOutput := fmt.Sprintf(`{"type":"match","data":{"path":{"text":"/workspace/file.txt"},"lines":{"text":"%s"},"line_number":1}}`, longLine)

	mockRunner := &mocks.MockCommandExecutor{
		StartFunc: func(ctx context.Context, cmd []string, opts models.ProcessOptions) (models.Process, io.Reader, io.Reader, error) {
			return &mocks.MockProcess{}, strings.NewReader(rgOutput), strings.NewReader(""), nil
		},
	}

	ctx := &models.WorkspaceContext{
		FS:              fs,
		BinaryDetector:  mocks.NewMockBinaryDetector(),
		ChecksumManager: services.NewChecksumManager(),
		WorkspaceRoot:   workspaceRoot,
		CommandExecutor: mockRunner,
		Config:          *config.DefaultConfig(),
	}

	resp, err := SearchContent(context.Background(), ctx, models.SearchContentRequest{Query: "pattern", SearchPath: "", CaseSensitive: true, IncludeIgnored: false, Offset: 0, Limit: 100})
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

	fs := mocks.NewMockFileSystem()
	fs.CreateDir("/workspace")

	var capturedCmd []string
	mockRunner := &mocks.MockCommandExecutor{
		StartFunc: func(ctx context.Context, cmd []string, opts models.ProcessOptions) (models.Process, io.Reader, io.Reader, error) {
			capturedCmd = cmd
			return &mocks.MockProcess{
				WaitFunc: func() error {
					return &mocks.MockExitError{Code: 1}
				},
			}, strings.NewReader(""), strings.NewReader(""), nil
		},
	}

	ctx := &models.WorkspaceContext{
		FS:              fs,
		BinaryDetector:  mocks.NewMockBinaryDetector(),
		ChecksumManager: services.NewChecksumManager(),
		WorkspaceRoot:   workspaceRoot,
		CommandExecutor: mockRunner,
		Config:          *config.DefaultConfig(),
	}

	query := "foo; rm -rf /"
	_, _ = SearchContent(context.Background(), ctx, models.SearchContentRequest{Query: query, SearchPath: "", CaseSensitive: true, IncludeIgnored: false, Offset: 0, Limit: 100})

	// Verify query is passed as literal argument
	found := slices.Contains(capturedCmd, query)

	if !found {
		t.Errorf("expected query to be passed as literal argument, got cmd: %v", capturedCmd)
	}
}

func TestSearchContent_NoMatches(t *testing.T) {
	workspaceRoot := "/workspace"

	fs := mocks.NewMockFileSystem()
	fs.CreateDir("/workspace")

	mockRunner := &mocks.MockCommandExecutor{
		StartFunc: func(ctx context.Context, cmd []string, opts models.ProcessOptions) (models.Process, io.Reader, io.Reader, error) {
			// Simulate rg returning exit code 1 (no matches)
			return &mocks.MockProcess{
				WaitFunc: func() error {
					return &mocks.MockExitError{Code: 1}
				},
			}, strings.NewReader(""), strings.NewReader(""), nil
		},
	}

	ctx := &models.WorkspaceContext{
		FS:              fs,
		BinaryDetector:  mocks.NewMockBinaryDetector(),
		ChecksumManager: services.NewChecksumManager(),
		WorkspaceRoot:   workspaceRoot,
		CommandExecutor: mockRunner,
		Config:          *config.DefaultConfig(),
	}

	resp, err := SearchContent(context.Background(), ctx, models.SearchContentRequest{Query: "nonexistent", SearchPath: "", CaseSensitive: true, IncludeIgnored: false, Offset: 0, Limit: 100})
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

	fs := mocks.NewMockFileSystem()
	fs.CreateDir("/workspace")

	// Simulate 10 matches
	var rgOutput string
	for i := range 10 {
		rgOutput += fmt.Sprintf(`{"type":"match","data":{"path":{"text":"/workspace/file.txt"},"lines":{"text":"line %d"},"line_number":%d}}
`, i, i+1)
	}

	mockRunner := &mocks.MockCommandExecutor{
		StartFunc: func(ctx context.Context, cmd []string, opts models.ProcessOptions) (models.Process, io.Reader, io.Reader, error) {
			return &mocks.MockProcess{}, strings.NewReader(rgOutput), strings.NewReader(""), nil
		},
	}

	ctx := &models.WorkspaceContext{
		FS:              fs,
		BinaryDetector:  mocks.NewMockBinaryDetector(),
		ChecksumManager: services.NewChecksumManager(),
		WorkspaceRoot:   workspaceRoot,
		CommandExecutor: mockRunner,
		Config:          *config.DefaultConfig(),
	}

	// Request offset=2, limit=2
	resp, err := SearchContent(context.Background(), ctx, models.SearchContentRequest{Query: "pattern", SearchPath: "", CaseSensitive: true, IncludeIgnored: false, Offset: 2, Limit: 2})
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

	fs := mocks.NewMockFileSystem()
	fs.CreateDir("/workspace")

	// Simulate matches from multiple files
	rgOutput := `{"type":"match","data":{"path":{"text":"/workspace/b.txt"},"lines":{"text":"match"},"line_number":5}}
{"type":"match","data":{"path":{"text":"/workspace/a.txt"},"lines":{"text":"match"},"line_number":10}}
{"type":"match","data":{"path":{"text":"/workspace/a.txt"},"lines":{"text":"match"},"line_number":5}}`

	mockRunner := &mocks.MockCommandExecutor{
		StartFunc: func(ctx context.Context, cmd []string, opts models.ProcessOptions) (models.Process, io.Reader, io.Reader, error) {
			return &mocks.MockProcess{}, strings.NewReader(rgOutput), strings.NewReader(""), nil
		},
	}

	ctx := &models.WorkspaceContext{
		FS:              fs,
		BinaryDetector:  mocks.NewMockBinaryDetector(),
		ChecksumManager: services.NewChecksumManager(),
		WorkspaceRoot:   workspaceRoot,
		CommandExecutor: mockRunner,
		Config:          *config.DefaultConfig(),
	}

	resp, err := SearchContent(context.Background(), ctx, models.SearchContentRequest{Query: "pattern", SearchPath: "", CaseSensitive: true, IncludeIgnored: false, Offset: 0, Limit: 100})
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

	fs := mocks.NewMockFileSystem()
	fs.CreateDir("/workspace")

	// Mix valid and invalid JSON
	rgOutput := `{"type":"match","data":{"path":{"text":"/workspace/file.txt"},"lines":{"text":"valid"},"line_number":1}}
invalid json line
{"type":"match","data":{"path":{"text":"/workspace/file.txt"},"lines":{"text":"also valid"},"line_number":2}}`

	mockRunner := &mocks.MockCommandExecutor{
		StartFunc: func(ctx context.Context, cmd []string, opts models.ProcessOptions) (models.Process, io.Reader, io.Reader, error) {
			return &mocks.MockProcess{}, strings.NewReader(rgOutput), strings.NewReader(""), nil
		},
	}

	ctx := &models.WorkspaceContext{
		FS:              fs,
		BinaryDetector:  mocks.NewMockBinaryDetector(),
		ChecksumManager: services.NewChecksumManager(),
		WorkspaceRoot:   workspaceRoot,
		CommandExecutor: mockRunner,
		Config:          *config.DefaultConfig(),
	}

	resp, err := SearchContent(context.Background(), ctx, models.SearchContentRequest{Query: "pattern", SearchPath: "", CaseSensitive: true, IncludeIgnored: false, Offset: 0, Limit: 100})
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

	fs := mocks.NewMockFileSystem()
	fs.CreateDir("/workspace")

	mockRunner := &mocks.MockCommandExecutor{
		StartFunc: func(ctx context.Context, cmd []string, opts models.ProcessOptions) (models.Process, io.Reader, io.Reader, error) {
			return &mocks.MockProcess{
				WaitFunc: func() error {
					return &mocks.MockExitError{Code: 2}
				},
			}, strings.NewReader(""), strings.NewReader(""), nil
		},
	}

	ctx := &models.WorkspaceContext{
		FS:              fs,
		BinaryDetector:  mocks.NewMockBinaryDetector(),
		ChecksumManager: services.NewChecksumManager(),
		WorkspaceRoot:   workspaceRoot,
		CommandExecutor: mockRunner,
		Config:          *config.DefaultConfig(),
	}

	_, err := SearchContent(context.Background(), ctx, models.SearchContentRequest{Query: "pattern", SearchPath: "", CaseSensitive: true, IncludeIgnored: false, Offset: 0, Limit: 100})
	if err == nil {
		t.Fatal("expected error for command failure, got nil")
	}
}
func TestSearchContent_IncludeIgnored(t *testing.T) {
	workspaceRoot := "/workspace"

	fs := mocks.NewMockFileSystem()
	fs.CreateDir("/workspace")

	// Test with includeIgnored=false (default behavior, should respect gitignore)
	mockRunner := &mocks.MockCommandExecutor{
		StartFunc: func(ctx context.Context, cmd []string, opts models.ProcessOptions) (models.Process, io.Reader, io.Reader, error) {
			// Verify --no-ignore is NOT present
			if slices.Contains(cmd, "--no-ignore") {
				t.Error("expected --no-ignore to NOT be present when includeIgnored=false")
			}
			// Simulate rg output without ignored files
			output := `{"type":"match","data":{"path":{"text":"/workspace/visible.go"},"lines":{"text":"func main()"},"line_number":1}}`
			return &mocks.MockProcess{}, strings.NewReader(output), strings.NewReader(""), nil
		},
	}

	ctx := &models.WorkspaceContext{
		FS:              fs,
		BinaryDetector:  mocks.NewMockBinaryDetector(),
		ChecksumManager: services.NewChecksumManager(),
		WorkspaceRoot:   workspaceRoot,
		CommandExecutor: mockRunner,
		Config:          *config.DefaultConfig(),
	}

	resp, err := SearchContent(context.Background(), ctx, models.SearchContentRequest{Query: "func main", SearchPath: "", CaseSensitive: true, IncludeIgnored: false, Offset: 0, Limit: 100})
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
		// Simulate rg output with ignored files
		output := `{"type":"match","data":{"path":{"text":"/workspace/ignored.go"},"lines":{"text":"func main()"},"line_number":1}}
{"type":"match","data":{"path":{"text":"/workspace/visible.go"},"lines":{"text":"func main()"},"line_number":1}}`
		return &mocks.MockProcess{}, strings.NewReader(output), strings.NewReader(""), nil
	}

	resp, err = SearchContent(context.Background(), ctx, models.SearchContentRequest{Query: "func main", SearchPath: "", CaseSensitive: true, IncludeIgnored: true, Offset: 0, Limit: 100})
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
		if match.File == "ignored.go" {
			foundIgnored = true
		}
		if match.File == "visible.go" {
			foundVisible = true
		}
	}

	if !foundIgnored {
		t.Error("expected to find match in ignored.go when includeIgnored=true")
	}
	if !foundVisible {
		t.Error("expected to find match in visible.go when includeIgnored=true")
	}
}

func TestSearchContent_LimitValidation(t *testing.T) {
	workspaceRoot := "/workspace"

	fs := mocks.NewMockFileSystem()
	fs.CreateDir("/workspace")

	mockRunner := &mocks.MockCommandExecutor{
		StartFunc: func(ctx context.Context, cmd []string, opts models.ProcessOptions) (models.Process, io.Reader, io.Reader, error) {
			return &mocks.MockProcess{}, strings.NewReader(""), strings.NewReader(""), nil
		},
	}

	t.Run("zero limit uses default", func(t *testing.T) {
		ctx := &models.WorkspaceContext{
			FS:              fs,
			WorkspaceRoot:   workspaceRoot,
			CommandExecutor: mockRunner,
			Config:          *config.DefaultConfig(),
		}

		resp, err := SearchContent(context.Background(), ctx, models.SearchContentRequest{
			Query: "test",
			Limit: 0, // Should use DefaultSearchContentLimit
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Limit != ctx.Config.Tools.DefaultSearchContentLimit {
			t.Errorf("expected default limit %d, got %d", ctx.Config.Tools.DefaultSearchContentLimit, resp.Limit)
		}
	})

	t.Run("custom config limits are respected", func(t *testing.T) {
		cfg := config.DefaultConfig()
		cfg.Tools.DefaultSearchContentLimit = 25
		cfg.Tools.MaxSearchContentLimit = 50

		ctx := &models.WorkspaceContext{
			FS:              fs,
			WorkspaceRoot:   workspaceRoot,
			CommandExecutor: mockRunner,
			Config:          *cfg,
		}

		resp, err := SearchContent(context.Background(), ctx, models.SearchContentRequest{
			Query: "test",
			Limit: 30,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Limit != 30 {
			t.Errorf("expected limit 30, got %d", resp.Limit)
		}
	})
}

func TestSearchContent_OffsetValidation(t *testing.T) {
	workspaceRoot := "/workspace"
	fs := mocks.NewMockFileSystem()
	fs.CreateDir("/workspace")

	mockRunner := &mocks.MockCommandExecutor{
		StartFunc: func(ctx context.Context, cmd []string, opts models.ProcessOptions) (models.Process, io.Reader, io.Reader, error) {
			return &mocks.MockProcess{}, strings.NewReader(""), strings.NewReader(""), nil
		},
	}

	ctx := &models.WorkspaceContext{
		FS:              fs,
		WorkspaceRoot:   workspaceRoot,
		CommandExecutor: mockRunner,
		Config:          *config.DefaultConfig(),
	}

	t.Run("zero offset is valid", func(t *testing.T) {
		_, err := SearchContent(context.Background(), ctx, models.SearchContentRequest{
			Query:  "test",
			Offset: 0,
			Limit:  10,
		})
		if err != nil {
			t.Fatalf("unexpected error for offset 0: %v", err)
		}
	})

	t.Run("positive offset is valid", func(t *testing.T) {
		_, err := SearchContent(context.Background(), ctx, models.SearchContentRequest{
			Query:  "test",
			Offset: 100,
			Limit:  10,
		})
		if err != nil {
			t.Fatalf("unexpected error for positive offset: %v", err)
		}
	})
}
