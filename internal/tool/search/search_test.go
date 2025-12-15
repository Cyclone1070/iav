package search

import (
	"context"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Cyclone1070/iav/internal/config"
	toolserrors "github.com/Cyclone1070/iav/internal/tool/errutil"
	"github.com/Cyclone1070/iav/internal/tool/shell"
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
func (m *mockFileInfoForSearch) Sys() interface{}   { return nil }

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
	startFunc func(ctx context.Context, cmd []string, opts shell.ProcessOptions) (shell.Process, io.Reader, io.Reader, error)
}

func (m *mockCommandExecutorForSearch) Start(ctx context.Context, cmd []string, opts shell.ProcessOptions) (shell.Process, io.Reader, io.Reader, error) {
	if m.startFunc != nil {
		return m.startFunc(ctx, cmd, opts)
	}
	return nil, nil, nil, fmt.Errorf("not implemented")
}

type mockProcessForSearch struct {
	waitFunc func() error
}

func (m *mockProcessForSearch) Wait() error {
	if m.waitFunc != nil {
		return m.waitFunc()
	}
	return nil
}

func (m *mockProcessForSearch) Signal(sig os.Signal) error { return nil }
func (m *mockProcessForSearch) Kill() error                { return nil }

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

	rgOutput := `{"type":"match","data":{"path":{"text":"/workspace/file.go"},"lines":{"text":"func foo()"},"line_number":10}}
{"type":"match","data":{"path":{"text":"/workspace/file.go"},"lines":{"text":"func bar()"},"line_number":20}}`

	mockRunner := &mockCommandExecutorForSearch{}
	mockRunner.startFunc = func(ctx context.Context, cmd []string, opts shell.ProcessOptions) (shell.Process, io.Reader, io.Reader, error) {
		return &mockProcessForSearch{}, strings.NewReader(rgOutput), strings.NewReader(""), nil
	}

	tool := NewSearchContentTool(fs, mockRunner, config.DefaultConfig(), "/workspace")
	resp, err := tool.Run(context.Background(), SearchContentRequest{Query: "func .*", SearchPath: "", CaseSensitive: true, IncludeIgnored: false, Offset: 0, Limit: 100})
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

	var capturedCmd []string
	mockRunner := &mockCommandExecutorForSearch{}
	mockRunner.startFunc = func(ctx context.Context, cmd []string, opts shell.ProcessOptions) (shell.Process, io.Reader, io.Reader, error) {
		capturedCmd = cmd
		proc := &mockProcessForSearch{}
		proc.waitFunc = func() error {
			return newMockExitErrorForSearch(1)
		}
		return proc, strings.NewReader(""), strings.NewReader(""), nil
	}

	tool := NewSearchContentTool(fs, mockRunner, config.DefaultConfig(), "/workspace")
	_, _ = tool.Run(context.Background(), SearchContentRequest{Query: "pattern", SearchPath: "", CaseSensitive: false, IncludeIgnored: false, Offset: 0, Limit: 100})

	foundFlag := slices.Contains(capturedCmd, "-i")
	if !foundFlag {
		t.Errorf("expected -i flag for case-insensitive search, got cmd: %v", capturedCmd)
	}
}

func TestSearchContent_PathOutsideWorkspace(t *testing.T) {
	fs := newMockFileSystemForSearch()
	fs.createDir("/workspace")

	tool := NewSearchContentTool(fs, &mockCommandExecutorForSearch{}, config.DefaultConfig(), "/workspace")
	_, err := tool.Run(context.Background(), SearchContentRequest{Query: "pattern", SearchPath: "../outside", CaseSensitive: true, IncludeIgnored: false, Offset: 0, Limit: 100})
	if err != toolserrors.ErrOutsideWorkspace {
		t.Errorf("expected ErrOutsideWorkspace, got %v", err)
	}
}

func TestSearchContent_VeryLongLine(t *testing.T) {
	fs := newMockFileSystemForSearch()
	fs.createDir("/workspace")

	longLine := strings.Repeat("a", 1024*1024)
	rgOutput := fmt.Sprintf(`{"type":"match","data":{"path":{"text":"/workspace/file.txt"},"lines":{"text":"%s"},"line_number":1}}`, longLine)

	mockRunner := &mockCommandExecutorForSearch{}
	mockRunner.startFunc = func(ctx context.Context, cmd []string, opts shell.ProcessOptions) (shell.Process, io.Reader, io.Reader, error) {
		return &mockProcessForSearch{}, strings.NewReader(rgOutput), strings.NewReader(""), nil
	}

	tool := NewSearchContentTool(fs, mockRunner, config.DefaultConfig(), "/workspace")
	resp, err := tool.Run(context.Background(), SearchContentRequest{Query: "pattern", SearchPath: "", CaseSensitive: true, IncludeIgnored: false, Offset: 0, Limit: 100})
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

	var capturedCmd []string
	mockRunner := &mockCommandExecutorForSearch{}
	mockRunner.startFunc = func(ctx context.Context, cmd []string, opts shell.ProcessOptions) (shell.Process, io.Reader, io.Reader, error) {
		capturedCmd = cmd
		proc := &mockProcessForSearch{}
		proc.waitFunc = func() error {
			return newMockExitErrorForSearch(1)
		}
		return proc, strings.NewReader(""), strings.NewReader(""), nil
	}

	tool := NewSearchContentTool(fs, mockRunner, config.DefaultConfig(), "/workspace")
	query := "foo; rm -rf /"
	_, _ = tool.Run(context.Background(), SearchContentRequest{Query: query, SearchPath: "", CaseSensitive: true, IncludeIgnored: false, Offset: 0, Limit: 100})

	found := slices.Contains(capturedCmd, query)
	if !found {
		t.Errorf("expected query to be passed as literal argument, got cmd: %v", capturedCmd)
	}
}

func TestSearchContent_NoMatches(t *testing.T) {
	fs := newMockFileSystemForSearch()
	fs.createDir("/workspace")

	mockRunner := &mockCommandExecutorForSearch{}
	mockRunner.startFunc = func(ctx context.Context, cmd []string, opts shell.ProcessOptions) (shell.Process, io.Reader, io.Reader, error) {
		proc := &mockProcessForSearch{}
		proc.waitFunc = func() error {
			return newMockExitErrorForSearch(1)
		}
		return proc, strings.NewReader(""), strings.NewReader(""), nil
	}

	tool := NewSearchContentTool(fs, mockRunner, config.DefaultConfig(), "/workspace")
	resp, err := tool.Run(context.Background(), SearchContentRequest{Query: "nonexistent", SearchPath: "", CaseSensitive: true, IncludeIgnored: false, Offset: 0, Limit: 100})
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

	var rgOutput string
	for i := range 10 {
		rgOutput += fmt.Sprintf(`{"type":"match","data":{"path":{"text":"/workspace/file.txt"},"lines":{"text":"line %d"},"line_number":%d}}
`, i, i+1)
	}

	mockRunner := &mockCommandExecutorForSearch{}
	mockRunner.startFunc = func(ctx context.Context, cmd []string, opts shell.ProcessOptions) (shell.Process, io.Reader, io.Reader, error) {
		return &mockProcessForSearch{}, strings.NewReader(rgOutput), strings.NewReader(""), nil
	}

	tool := NewSearchContentTool(fs, mockRunner, config.DefaultConfig(), "/workspace")
	resp, err := tool.Run(context.Background(), SearchContentRequest{Query: "pattern", SearchPath: "", CaseSensitive: true, IncludeIgnored: false, Offset: 2, Limit: 2})
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

	rgOutput := `{"type":"match","data":{"path":{"text":"/workspace/b.txt"},"lines":{"text":"match"},"line_number":5}}
{"type":"match","data":{"path":{"text":"/workspace/a.txt"},"lines":{"text":"match"},"line_number":10}}
{"type":"match","data":{"path":{"text":"/workspace/a.txt"},"lines":{"text":"match"},"line_number":5}}`

	mockRunner := &mockCommandExecutorForSearch{}
	mockRunner.startFunc = func(ctx context.Context, cmd []string, opts shell.ProcessOptions) (shell.Process, io.Reader, io.Reader, error) {
		return &mockProcessForSearch{}, strings.NewReader(rgOutput), strings.NewReader(""), nil
	}

	tool := NewSearchContentTool(fs, mockRunner, config.DefaultConfig(), "/workspace")
	resp, err := tool.Run(context.Background(), SearchContentRequest{Query: "pattern", SearchPath: "", CaseSensitive: true, IncludeIgnored: false, Offset: 0, Limit: 100})
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

	rgOutput := `{"type":"match","data":{"path":{"text":"/workspace/file.txt"},"lines":{"text":"valid"},"line_number":1}}
invalid json line
{"type":"match","data":{"path":{"text":"/workspace/file.txt"},"lines":{"text":"also valid"},"line_number":2}}`

	mockRunner := &mockCommandExecutorForSearch{}
	mockRunner.startFunc = func(ctx context.Context, cmd []string, opts shell.ProcessOptions) (shell.Process, io.Reader, io.Reader, error) {
		return &mockProcessForSearch{}, strings.NewReader(rgOutput), strings.NewReader(""), nil
	}

	tool := NewSearchContentTool(fs, mockRunner, config.DefaultConfig(), "/workspace")
	resp, err := tool.Run(context.Background(), SearchContentRequest{Query: "pattern", SearchPath: "", CaseSensitive: true, IncludeIgnored: false, Offset: 0, Limit: 100})
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

	mockRunner := &mockCommandExecutorForSearch{}
	mockRunner.startFunc = func(ctx context.Context, cmd []string, opts shell.ProcessOptions) (shell.Process, io.Reader, io.Reader, error) {
		proc := &mockProcessForSearch{}
		proc.waitFunc = func() error {
			return newMockExitErrorForSearch(2)
		}
		return proc, strings.NewReader(""), strings.NewReader(""), nil
	}

	tool := NewSearchContentTool(fs, mockRunner, config.DefaultConfig(), "/workspace")
	_, err := tool.Run(context.Background(), SearchContentRequest{Query: "pattern", SearchPath: "", CaseSensitive: true, IncludeIgnored: false, Offset: 0, Limit: 100})
	if err == nil {
		t.Fatal("expected error for command failure, got nil")
	}
}

func TestSearchContent_IncludeIgnored(t *testing.T) {
	fs := newMockFileSystemForSearch()
	fs.createDir("/workspace")

	mockRunner := &mockCommandExecutorForSearch{}
	mockRunner.startFunc = func(ctx context.Context, cmd []string, opts shell.ProcessOptions) (shell.Process, io.Reader, io.Reader, error) {
		if slices.Contains(cmd, "--no-ignore") {
			t.Error("expected --no-ignore to NOT be present when includeIgnored=false")
		}
		output := `{"type":"match","data":{"path":{"text":"/workspace/visible.go"},"lines":{"text":"func main()"},"line_number":1}}`
		return &mockProcessForSearch{}, strings.NewReader(output), strings.NewReader(""), nil
	}

	tool := NewSearchContentTool(fs, mockRunner, config.DefaultConfig(), "/workspace")
	resp, err := tool.Run(context.Background(), SearchContentRequest{Query: "func main", SearchPath: "", CaseSensitive: true, IncludeIgnored: false, Offset: 0, Limit: 100})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(resp.Matches))
	}

	// Test with includeIgnored=true
	mockRunner.startFunc = func(ctx context.Context, cmd []string, opts shell.ProcessOptions) (shell.Process, io.Reader, io.Reader, error) {
		if !slices.Contains(cmd, "--no-ignore") {
			t.Error("expected --no-ignore to be present when includeIgnored=true")
		}
		output := `{"type":"match","data":{"path":{"text":"/workspace/ignored.go"},"lines":{"text":"func main()"},"line_number":1}}
{"type":"match","data":{"path":{"text":"/workspace/visible.go"},"lines":{"text":"func main()"},"line_number":1}}`
		return &mockProcessForSearch{}, strings.NewReader(output), strings.NewReader(""), nil
	}

	resp, err = tool.Run(context.Background(), SearchContentRequest{Query: "func main", SearchPath: "", CaseSensitive: true, IncludeIgnored: true, Offset: 0, Limit: 100})
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
	mockRunner.startFunc = func(ctx context.Context, cmd []string, opts shell.ProcessOptions) (shell.Process, io.Reader, io.Reader, error) {
		return &mockProcessForSearch{}, strings.NewReader(""), strings.NewReader(""), nil
	}

	t.Run("zero limit uses default", func(t *testing.T) {
		tool := NewSearchContentTool(fs, mockRunner, config.DefaultConfig(), "/workspace")
		resp, err := tool.Run(context.Background(), SearchContentRequest{
			Query: "test",
			Limit: 0,
		})
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

		tool := NewSearchContentTool(fs, mockRunner, cfg, "/workspace")
		resp, err := tool.Run(context.Background(), SearchContentRequest{
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
	fs := newMockFileSystemForSearch()
	fs.createDir("/workspace")

	mockRunner := &mockCommandExecutorForSearch{}
	mockRunner.startFunc = func(ctx context.Context, cmd []string, opts shell.ProcessOptions) (shell.Process, io.Reader, io.Reader, error) {
		return &mockProcessForSearch{}, strings.NewReader(""), strings.NewReader(""), nil
	}

	tool := NewSearchContentTool(fs, mockRunner, config.DefaultConfig(), "/workspace")

	t.Run("zero offset is valid", func(t *testing.T) {
		_, err := tool.Run(context.Background(), SearchContentRequest{
			Query:  "test",
			Offset: 0,
			Limit:  10,
		})
		if err != nil {
			t.Fatalf("unexpected error for offset 0: %v", err)
		}
	})

	t.Run("positive offset is valid", func(t *testing.T) {
		_, err := tool.Run(context.Background(), SearchContentRequest{
			Query:  "test",
			Offset: 100,
			Limit:  10,
		})
		if err != nil {
			t.Fatalf("unexpected error for positive offset: %v", err)
		}
	})
}
