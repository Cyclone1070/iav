package search

import (
	"context"
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Cyclone1070/iav/internal/config"
	"github.com/Cyclone1070/iav/internal/tool/executor"
	"github.com/Cyclone1070/iav/internal/tool/service/path"
)

// Local mocks for search tests

type mockFileInfoForSearch struct {
	name  string
	isDir bool
}

func (m *mockFileInfoForSearch) Name() string       { return m.name }
func (m *mockFileInfoForSearch) Size() int64        { return 0 }
func (m *mockFileInfoForSearch) Mode() os.FileMode  { return 0644 }
func (m *mockFileInfoForSearch) ModTime() time.Time { return time.Time{} }
func (m *mockFileInfoForSearch) IsDir() bool        { return m.isDir }
func (m *mockFileInfoForSearch) Sys() any           { return nil }

type mockFileSystemForSearch struct {
	dirs map[string]bool
}

func newMockFileSystemForSearch() *mockFileSystemForSearch {
	return &mockFileSystemForSearch{
		dirs: make(map[string]bool),
	}
}

func (m *mockFileSystemForSearch) createDir(path string) {
	m.dirs[path] = true
}

func (m *mockFileSystemForSearch) Stat(path string) (os.FileInfo, error) {
	if m.dirs[path] {
		return &mockFileInfoForSearch{name: path, isDir: true}, nil
	}
	return nil, os.ErrNotExist
}

func (m *mockFileSystemForSearch) Lstat(path string) (os.FileInfo, error) {
	return m.Stat(path)
}

func (m *mockFileSystemForSearch) Readlink(path string) (string, error) {
	return "", os.ErrInvalid
}

func (m *mockFileSystemForSearch) UserHomeDir() (string, error) {
	return "/home/user", nil
}

type mockCommandExecutorForSearch struct {
	runFunc func(ctx context.Context, cmd []string, dir string, env []string) (*executor.Result, error)
}

func (m *mockCommandExecutorForSearch) Run(ctx context.Context, cmd []string, dir string, env []string) (*executor.Result, error) {
	if m.runFunc != nil {
		return m.runFunc(ctx, cmd, dir, env)
	}
	return nil, fmt.Errorf("not implemented")
}

type mockExitErrorForSearch struct {
	code int
}

func (e *mockExitErrorForSearch) Error() string { return fmt.Sprintf("exit status %d", e.code) }
func (e *mockExitErrorForSearch) ExitCode() int { return e.code }

func newMockExitErrorForSearch(code int) error {
	return &mockExitErrorForSearch{code: code}
}

// Test functions

func TestSearchContent_BasicRegex(t *testing.T) {
	fs := newMockFileSystemForSearch()
	fs.createDir("/workspace")
	workspaceRoot := "/workspace"
	cfg := config.DefaultConfig()

	rgOutput := `{"type":"match","data":{"path":{"text":"/workspace/file.go"},"lines":{"text":"func foo()"},"line_number":10}}
{"type":"match","data":{"path":{"text":"/workspace/file.go"},"lines":{"text":"func bar()"},"line_number":20}}`

	mockRunner := &mockCommandExecutorForSearch{}
	mockRunner.runFunc = func(ctx context.Context, cmd []string, dir string, env []string) (*executor.Result, error) {
		return &executor.Result{Stdout: rgOutput, ExitCode: 0}, nil
	}

	searchTool := NewSearchContentTool(fs, mockRunner, cfg, path.NewResolver(workspaceRoot))

	req := &SearchContentRequest{Query: "func .*", SearchPath: "", CaseSensitive: true, IncludeIgnored: false, Offset: 0, Limit: 100}
	resp, err := searchTool.Run(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Matches) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(resp.Matches))
	}

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
	fs := newMockFileSystemForSearch()
	fs.createDir("/workspace")
	workspaceRoot := "/workspace"
	cfg := config.DefaultConfig()

	var capturedCmd []string
	mockRunner := &mockCommandExecutorForSearch{}
	mockRunner.runFunc = func(ctx context.Context, cmd []string, dir string, env []string) (*executor.Result, error) {
		capturedCmd = cmd
		return &executor.Result{Stdout: "", ExitCode: 1}, newMockExitErrorForSearch(1)
	}

	searchTool := NewSearchContentTool(fs, mockRunner, cfg, path.NewResolver(workspaceRoot))

	req := &SearchContentRequest{Query: "pattern", SearchPath: "", CaseSensitive: false, IncludeIgnored: false, Offset: 0, Limit: 100}
	_, _ = searchTool.Run(context.Background(), req)

	foundFlag := slices.Contains(capturedCmd, "-i")
	if !foundFlag {
		t.Errorf("expected -i flag for case-insensitive search, got cmd: %v", capturedCmd)
	}
}

func TestSearchContent_PathOutsideWorkspace(t *testing.T) {
	fs := newMockFileSystemForSearch()
	fs.createDir("/workspace")
	workspaceRoot := "/workspace"
	cfg := config.DefaultConfig()

	searchTool := NewSearchContentTool(fs, &mockCommandExecutorForSearch{}, cfg, path.NewResolver(workspaceRoot))

	req := &SearchContentRequest{Query: "pattern", SearchPath: "../outside", CaseSensitive: true, IncludeIgnored: false, Offset: 0, Limit: 100}
	_, err := searchTool.Run(context.Background(), req)

	if err == nil || !strings.Contains(err.Error(), "outside workspace") {
		t.Errorf("expected error for path outside workspace, got %v", err)
	}
}

func TestSearchContent_VeryLongLine(t *testing.T) {
	fs := newMockFileSystemForSearch()
	fs.createDir("/workspace")
	workspaceRoot := "/workspace"
	cfg := config.DefaultConfig()

	longLine := strings.Repeat("a", 1024*1024)
	rgOutput := fmt.Sprintf(`{"type":"match","data":{"path":{"text":"/workspace/file.txt"},"lines":{"text":"%s"},"line_number":1}}`, longLine)

	mockRunner := &mockCommandExecutorForSearch{}
	mockRunner.runFunc = func(ctx context.Context, cmd []string, dir string, env []string) (*executor.Result, error) {
		return &executor.Result{Stdout: rgOutput, ExitCode: 0}, nil
	}

	searchTool := NewSearchContentTool(fs, mockRunner, cfg, path.NewResolver(workspaceRoot))

	req := &SearchContentRequest{Query: "pattern", SearchPath: "", CaseSensitive: true, IncludeIgnored: false, Offset: 0, Limit: 100}
	resp, err := searchTool.Run(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(resp.Matches))
	}

	if len(resp.Matches[0].LineContent) > 10100 {
		t.Errorf("expected line to be truncated to ~10000 chars, got %d", len(resp.Matches[0].LineContent))
	}

	if !strings.Contains(resp.Matches[0].LineContent, "[truncated]") {
		t.Error("expected truncation marker in line content")
	}
}

func TestSearchContent_CommandInjection(t *testing.T) {
	fs := newMockFileSystemForSearch()
	fs.createDir("/workspace")
	workspaceRoot := "/workspace"
	cfg := config.DefaultConfig()

	var capturedCmd []string
	mockRunner := &mockCommandExecutorForSearch{}
	mockRunner.runFunc = func(ctx context.Context, cmd []string, dir string, env []string) (*executor.Result, error) {
		capturedCmd = cmd
		return &executor.Result{Stdout: "", ExitCode: 1}, newMockExitErrorForSearch(1)
	}

	searchTool := NewSearchContentTool(fs, mockRunner, cfg, path.NewResolver(workspaceRoot))
	query := "foo; rm -rf /"

	req := &SearchContentRequest{Query: query, SearchPath: "", CaseSensitive: true, IncludeIgnored: false, Offset: 0, Limit: 100}
	_, _ = searchTool.Run(context.Background(), req)

	found := slices.Contains(capturedCmd, query)
	if !found {
		t.Errorf("expected query to be passed as literal argument, got cmd: %v", capturedCmd)
	}
}

func TestSearchContent_NoMatches(t *testing.T) {
	fs := newMockFileSystemForSearch()
	fs.createDir("/workspace")
	workspaceRoot := "/workspace"
	cfg := config.DefaultConfig()

	mockRunner := &mockCommandExecutorForSearch{}
	mockRunner.runFunc = func(ctx context.Context, cmd []string, dir string, env []string) (*executor.Result, error) {
		return &executor.Result{Stdout: "", ExitCode: 1}, newMockExitErrorForSearch(1)
	}

	searchTool := NewSearchContentTool(fs, mockRunner, cfg, path.NewResolver(workspaceRoot))

	req := &SearchContentRequest{Query: "nonexistent", SearchPath: "", CaseSensitive: true, IncludeIgnored: false, Offset: 0, Limit: 100}
	resp, err := searchTool.Run(context.Background(), req)
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
	fs := newMockFileSystemForSearch()
	fs.createDir("/workspace")
	workspaceRoot := "/workspace"
	cfg := config.DefaultConfig()

	var rgOutput string
	for i := 0; i < 10; i++ {
		rgOutput += fmt.Sprintf(`{"type":"match","data":{"path":{"text":"/workspace/file.txt"},"lines":{"text":"line %d"},"line_number":%d}}
`, i, i+1)
	}

	mockRunner := &mockCommandExecutorForSearch{}
	mockRunner.runFunc = func(ctx context.Context, cmd []string, dir string, env []string) (*executor.Result, error) {
		return &executor.Result{Stdout: rgOutput, ExitCode: 0}, nil
	}

	searchTool := NewSearchContentTool(fs, mockRunner, cfg, path.NewResolver(workspaceRoot))

	req := &SearchContentRequest{Query: "pattern", SearchPath: "", CaseSensitive: true, IncludeIgnored: false, Offset: 2, Limit: 2}
	resp, err := searchTool.Run(context.Background(), req)
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

	if resp.Matches[0].LineNumber != 3 {
		t.Errorf("expected line 3, got %d", resp.Matches[0].LineNumber)
	}
}

func TestSearchContent_MultipleFiles(t *testing.T) {
	fs := newMockFileSystemForSearch()
	fs.createDir("/workspace")
	workspaceRoot := "/workspace"
	cfg := config.DefaultConfig()

	rgOutput := `{"type":"match","data":{"path":{"text":"/workspace/b.txt"},"lines":{"text":"match"},"line_number":5}}
{"type":"match","data":{"path":{"text":"/workspace/a.txt"},"lines":{"text":"match"},"line_number":10}}
{"type":"match","data":{"path":{"text":"/workspace/a.txt"},"lines":{"text":"match"},"line_number":5}}`

	mockRunner := &mockCommandExecutorForSearch{}
	mockRunner.runFunc = func(ctx context.Context, cmd []string, dir string, env []string) (*executor.Result, error) {
		return &executor.Result{Stdout: rgOutput, ExitCode: 0}, nil
	}

	searchTool := NewSearchContentTool(fs, mockRunner, cfg, path.NewResolver(workspaceRoot))

	req := &SearchContentRequest{Query: "pattern", SearchPath: "", CaseSensitive: true, IncludeIgnored: false, Offset: 0, Limit: 100}
	resp, err := searchTool.Run(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Matches) != 3 {
		t.Fatalf("expected 3 matches, got %d", len(resp.Matches))
	}

	// Verify sorting (by file, then line number)
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
	fs := newMockFileSystemForSearch()
	fs.createDir("/workspace")
	workspaceRoot := "/workspace"
	cfg := config.DefaultConfig()

	rgOutput := `{"type":"match","data":{"path":{"text":"/workspace/file.txt"},"lines":{"text":"valid"},"line_number":1}}
invalid json line
{"type":"match","data":{"path":{"text":"/workspace/file.txt"},"lines":{"text":"also valid"},"line_number":2}}`

	mockRunner := &mockCommandExecutorForSearch{}
	mockRunner.runFunc = func(ctx context.Context, cmd []string, dir string, env []string) (*executor.Result, error) {
		return &executor.Result{Stdout: rgOutput, ExitCode: 0}, nil
	}

	searchTool := NewSearchContentTool(fs, mockRunner, cfg, path.NewResolver(workspaceRoot))

	req := &SearchContentRequest{Query: "pattern", SearchPath: "", CaseSensitive: true, IncludeIgnored: false, Offset: 0, Limit: 100}
	resp, err := searchTool.Run(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Matches) != 2 {
		t.Fatalf("expected 2 matches (invalid JSON skipped), got %d", len(resp.Matches))
	}
}

func TestSearchContent_CommandFailure(t *testing.T) {
	fs := newMockFileSystemForSearch()
	fs.createDir("/workspace")
	workspaceRoot := "/workspace"
	cfg := config.DefaultConfig()

	mockRunner := &mockCommandExecutorForSearch{}
	mockRunner.runFunc = func(ctx context.Context, cmd []string, dir string, env []string) (*executor.Result, error) {
		return &executor.Result{Stdout: "", ExitCode: 2}, newMockExitErrorForSearch(2)
	}

	searchTool := NewSearchContentTool(fs, mockRunner, cfg, path.NewResolver(workspaceRoot))

	req := &SearchContentRequest{Query: "pattern", SearchPath: "", CaseSensitive: true, IncludeIgnored: false, Offset: 0, Limit: 100}
	_, err := searchTool.Run(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for command failure, got nil")
	}
}

func TestSearchContent_IncludeIgnored(t *testing.T) {
	fs := newMockFileSystemForSearch()
	fs.createDir("/workspace")
	workspaceRoot := "/workspace"
	cfg := config.DefaultConfig()

	mockRunner := &mockCommandExecutorForSearch{}
	mockRunner.runFunc = func(ctx context.Context, cmd []string, dir string, env []string) (*executor.Result, error) {
		if slices.Contains(cmd, "--no-ignore") {
			t.Error("expected --no-ignore to NOT be present when includeIgnored=false")
		}
		output := `{"type":"match","data":{"path":{"text":"/workspace/visible.go"},"lines":{"text":"func main()"},"line_number":1}}`
		return &executor.Result{Stdout: output, ExitCode: 0}, nil
	}

	searchTool := NewSearchContentTool(fs, mockRunner, cfg, path.NewResolver(workspaceRoot))

	req := &SearchContentRequest{Query: "func main", SearchPath: "", CaseSensitive: true, IncludeIgnored: false, Offset: 0, Limit: 100}
	resp, err := searchTool.Run(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(resp.Matches))
	}

	// Test with includeIgnored=true
	mockRunner.runFunc = func(ctx context.Context, cmd []string, dir string, env []string) (*executor.Result, error) {
		if !slices.Contains(cmd, "--no-ignore") {
			t.Error("expected --no-ignore to be present when includeIgnored=true")
		}
		output := `{"type":"match","data":{"path":{"text":"/workspace/ignored.go"},"lines":{"text":"func main()"},"line_number":1}}
{"type":"match","data":{"path":{"text":"/workspace/visible.go"},"lines":{"text":"func main()"},"line_number":1}}`
		return &executor.Result{Stdout: output, ExitCode: 0}, nil
	}

	req = &SearchContentRequest{Query: "func main", SearchPath: "", CaseSensitive: true, IncludeIgnored: true, Offset: 0, Limit: 100}
	resp, err = searchTool.Run(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Matches) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(resp.Matches))
	}

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
	fs := newMockFileSystemForSearch()
	fs.createDir("/workspace")

	mockRunner := &mockCommandExecutorForSearch{}
	mockRunner.runFunc = func(ctx context.Context, cmd []string, dir string, env []string) (*executor.Result, error) {
		return &executor.Result{Stdout: "", ExitCode: 0}, nil
	}

	t.Run("zero limit uses default", func(t *testing.T) {
		searchTool := NewSearchContentTool(fs, mockRunner, config.DefaultConfig(), path.NewResolver("/workspace"))

		req := &SearchContentRequest{Query: "test", Limit: 0}
		resp, err := searchTool.Run(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Limit != config.DefaultConfig().Tools.DefaultSearchContentLimit {
			t.Errorf("expected default limit %d, got %d", config.DefaultConfig().Tools.DefaultSearchContentLimit, resp.Limit)
		}
	})

	t.Run("custom config limits are respected", func(t *testing.T) {
		cfg := config.DefaultConfig()
		cfg.Tools.DefaultSearchContentLimit = 25
		cfg.Tools.MaxSearchContentLimit = 50

		searchTool := NewSearchContentTool(fs, mockRunner, cfg, path.NewResolver("/workspace"))

		req := &SearchContentRequest{Query: "test", Limit: 30}
		resp, err := searchTool.Run(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Limit != 30 {
			t.Errorf("expected limit 30, got %d", resp.Limit)
		}
	})
}

func TestSearchContent_OffsetValidation(t *testing.T) {
	fs := newMockFileSystemForSearch()
	fs.createDir("/workspace")

	mockRunner := &mockCommandExecutorForSearch{}
	mockRunner.runFunc = func(ctx context.Context, cmd []string, dir string, env []string) (*executor.Result, error) {
		return &executor.Result{Stdout: "", ExitCode: 0}, nil
	}

	searchTool := NewSearchContentTool(fs, mockRunner, config.DefaultConfig(), path.NewResolver("/workspace"))

	t.Run("zero offset is valid", func(t *testing.T) {
		req := &SearchContentRequest{Query: "test", Offset: 0, Limit: 10}
		_, err := searchTool.Run(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error for offset 0: %v", err)
		}
	})

	t.Run("positive offset is valid", func(t *testing.T) {
		req := &SearchContentRequest{Query: "test", Offset: 100, Limit: 10}
		_, err := searchTool.Run(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error for positive offset: %v", err)
		}
	})
}

func TestSearchContent_SearchPathValidation(t *testing.T) {
	workspaceRoot := "/workspace"
	cfg := config.DefaultConfig()

	t.Run("invalid search path leads to error", func(t *testing.T) {
		fs := newMockFileSystemForSearch()
		// No dir created
		runner := &mockCommandExecutorForSearch{}
		searchTool := NewSearchContentTool(fs, runner, cfg, path.NewResolver(workspaceRoot))

		req := &SearchContentRequest{Query: "test", SearchPath: "nonexistent"}
		_, err := searchTool.Run(context.Background(), req)
		if err == nil {
			t.Error("expected error for nonexistent search path, got nil")
		}
	})

	t.Run("search path must be directory", func(t *testing.T) {
		fs := newMockFileSystemForSearch()
		fs.dirs["/workspace/file.txt"] = false // explicitly marked as NOT a dir
		runner := &mockCommandExecutorForSearch{}
		searchTool := NewSearchContentTool(fs, runner, cfg, path.NewResolver(workspaceRoot))

		req := &SearchContentRequest{Query: "test", SearchPath: "file.txt"}
		_, err := searchTool.Run(context.Background(), req)
		if err == nil {
			t.Error("expected error for search path that is a file, got nil")
		}
	})
}
