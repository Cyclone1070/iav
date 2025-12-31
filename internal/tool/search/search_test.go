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
	"github.com/Cyclone1070/iav/internal/tool/service/executor"
	"github.com/Cyclone1070/iav/internal/tool/service/path"
)

// Local mocks for search tests

type mockFileInfoForSearch struct {
	name  string
	isDir bool
}

func (m *mockFileInfoForSearch) Name() string       { return m.name }
func (m *mockFileInfoForSearch) Size() int64        { return 0 }
func (m *mockFileInfoForSearch) Mode() os.FileMode  { return 0o644 }
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

	expected := "file.go:\n  Line 10: func foo()\n  Line 20: func bar()\n"
	if resp.FormattedMatches != expected {
		t.Errorf("expected FormattedMatches:\n%q\ngot:\n%q", expected, resp.FormattedMatches)
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
		return &executor.Result{Stdout: "", ExitCode: 1}, nil
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

	if resp.TotalCount != 1 {
		t.Fatalf("expected 1 match, got %d", resp.TotalCount)
	}

	if !strings.Contains(resp.FormattedMatches, "[truncated]") {
		t.Error("expected truncation marker in formatted matches")
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
		return &executor.Result{Stdout: "", ExitCode: 1}, nil
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
		return &executor.Result{Stdout: "", ExitCode: 1}, nil
	}

	searchTool := NewSearchContentTool(fs, mockRunner, cfg, path.NewResolver(workspaceRoot))

	req := &SearchContentRequest{Query: "nonexistent", SearchPath: "", CaseSensitive: true, IncludeIgnored: false, Offset: 0, Limit: 100}
	resp, err := searchTool.Run(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.FormattedMatches != "No matches found." {
		t.Errorf("expected 'No matches found.', got %q", resp.FormattedMatches)
	}
}

func TestSearchContent_Pagination(t *testing.T) {
	fs := newMockFileSystemForSearch()
	fs.createDir("/workspace")
	workspaceRoot := "/workspace"
	cfg := config.DefaultConfig()

	var rgOutput string
	for i := range 10 {
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

	if resp.TotalCount != 10 {
		t.Errorf("expected TotalCount 10, got %d", resp.TotalCount)
	}

	if !strings.Contains(resp.FormattedMatches, "Line 3") {
		t.Errorf("expected page to contain Line 3, got output: %q", resp.FormattedMatches)
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

	if resp.TotalCount != 3 {
		t.Fatalf("expected 3 matches, got %d", resp.TotalCount)
	}

	// Verify sorting (by file, then line number) and grouping
	expected := "a.txt:\n  Line 5: match\n  Line 10: match\n\nb.txt:\n  Line 5: match\n"
	if resp.FormattedMatches != expected {
		t.Errorf("expected sorted and grouped output:\n%q\ngot:\n%q", expected, resp.FormattedMatches)
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

	if resp.TotalCount != 2 {
		t.Fatalf("expected 2 matches (invalid JSON skipped), got %d", resp.TotalCount)
	}
}

func TestSearchContent_CommandFailure(t *testing.T) {
	fs := newMockFileSystemForSearch()
	fs.createDir("/workspace")
	workspaceRoot := "/workspace"
	cfg := config.DefaultConfig()

	mockRunner := &mockCommandExecutorForSearch{}
	mockRunner.runFunc = func(ctx context.Context, cmd []string, dir string, env []string) (*executor.Result, error) {
		return &executor.Result{Stdout: "", ExitCode: 2}, nil
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

	if resp.TotalCount != 1 {
		t.Fatalf("expected 1 match, got %d", resp.TotalCount)
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

	if resp.TotalCount != 2 {
		t.Fatalf("expected 2 matches, got %d", resp.TotalCount)
	}

	if !strings.Contains(resp.FormattedMatches, "ignored.go") || !strings.Contains(resp.FormattedMatches, "visible.go") {
		t.Error("expected matches in both ignored.go and visible.go")
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

func TestSearchContent_EmptySearchPath(t *testing.T) {
	fs := newMockFileSystemForSearch()
	fs.createDir("/workspace")
	workspaceRoot := "/workspace"
	cfg := config.DefaultConfig()

	var capturedCmd []string
	mockRunner := &mockCommandExecutorForSearch{}
	mockRunner.runFunc = func(ctx context.Context, cmd []string, dir string, env []string) (*executor.Result, error) {
		capturedCmd = cmd
		return &executor.Result{Stdout: "", ExitCode: 0}, nil
	}

	searchTool := NewSearchContentTool(fs, mockRunner, cfg, path.NewResolver(workspaceRoot))

	// SearchPath is empty, should default to "." (workspace root)
	req := &SearchContentRequest{Query: "pattern", SearchPath: ""}
	_, err := searchTool.Run(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The last argument to rg should be the search path
	if len(capturedCmd) < 1 {
		t.Fatal("expected non-empty command")
	}
	lastArg := capturedCmd[len(capturedCmd)-1]
	if lastArg != "/workspace" {
		t.Errorf("expected search path '/workspace', got %q", lastArg)
	}
}

func TestSearchContent_HitMaxResults(t *testing.T) {
	fs := newMockFileSystemForSearch()
	fs.createDir("/workspace")
	workspaceRoot := "/workspace"
	cfg := config.DefaultConfig()
	cfg.Tools.MaxSearchContentResults = 2

	mockRunner := &mockCommandExecutorForSearch{}
	mockRunner.runFunc = func(ctx context.Context, cmd []string, dir string, env []string) (*executor.Result, error) {
		output := `{"type":"match","data":{"path":{"text":"/workspace/file1.go"},"lines":{"text":"match1"},"line_number":1}}
{"type":"match","data":{"path":{"text":"/workspace/file2.go"},"lines":{"text":"match2"},"line_number":1}}
{"type":"match","data":{"path":{"text":"/workspace/file3.go"},"lines":{"text":"match3"},"line_number":1}}`
		return &executor.Result{Stdout: output, ExitCode: 0}, nil
	}

	searchTool := NewSearchContentTool(fs, mockRunner, cfg, path.NewResolver(workspaceRoot))

	req := &SearchContentRequest{Query: "match", SearchPath: "", CaseSensitive: true, IncludeIgnored: false, Offset: 0, Limit: 100}
	resp, err := searchTool.Run(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !resp.HitMaxResults {
		t.Error("expected HitMaxResults to be true")
	}

	if resp.TotalCount != 2 {
		t.Errorf("expected 2 matches (capped), got %d", resp.TotalCount)
	}
}
