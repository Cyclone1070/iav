package directory

import (
	"context"
	"errors"
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

// Local mocks for find tests

type mockFileInfoForFind struct {
	name  string
	isDir bool
}

func (m *mockFileInfoForFind) Name() string       { return m.name }
func (m *mockFileInfoForFind) Size() int64        { return 0 }
func (m *mockFileInfoForFind) Mode() os.FileMode  { return 0o644 }
func (m *mockFileInfoForFind) ModTime() time.Time { return time.Time{} }
func (m *mockFileInfoForFind) IsDir() bool        { return m.isDir }
func (m *mockFileInfoForFind) Sys() any           { return nil }

type mockFileSystemForFind struct {
	dirs map[string]bool
}

func newMockFileSystemForFind() *mockFileSystemForFind {
	return &mockFileSystemForFind{
		dirs: make(map[string]bool),
	}
}

func (m *mockFileSystemForFind) createDir(path string) {
	m.dirs[path] = true
}

func (m *mockFileSystemForFind) Stat(path string) (os.FileInfo, error) {
	if m.dirs[path] {
		return &mockFileInfoForFind{name: path, isDir: true}, nil
	}
	return nil, os.ErrNotExist
}

func (m *mockFileSystemForFind) Lstat(path string) (os.FileInfo, error) {
	return m.Stat(path)
}

func (m *mockFileSystemForFind) Readlink(path string) (string, error) {
	return "", os.ErrInvalid
}

func (m *mockFileSystemForFind) UserHomeDir() (string, error) {
	return "/home/user", nil
}

func (m *mockFileSystemForFind) ListDir(path string) ([]os.FileInfo, error) {
	return nil, nil
}

type mockCommandExecutorForFind struct {
	runFunc func(ctx context.Context, cmd []string, dir string, env []string) (*executor.Result, error)
}

func (m *mockCommandExecutorForFind) Run(ctx context.Context, cmd []string, dir string, env []string) (*executor.Result, error) {
	if m.runFunc != nil {
		return m.runFunc(ctx, cmd, dir, env)
	}
	return nil, fmt.Errorf("not implemented")
}

type mockExitErrorForFind struct {
	code int
}

func (e *mockExitErrorForFind) Error() string { return fmt.Sprintf("exit status %d", e.code) }
func (e *mockExitErrorForFind) ExitCode() int { return e.code }

func newMockExitErrorForFind(code int) error {
	return &mockExitErrorForFind{code: code}
}

// Test functions

func TestFindFile_BasicGlob(t *testing.T) {
	fs := newMockFileSystemForFind()
	fs.createDir("/workspace")
	workspaceRoot := "/workspace"
	cfg := config.DefaultConfig()

	mockRunner := &mockCommandExecutorForFind{}
	mockRunner.runFunc = func(ctx context.Context, cmd []string, dir string, env []string) (*executor.Result, error) {
		output := "/workspace/a/b/file.go\n/workspace/a/file.go\n"
		return &executor.Result{Stdout: output, ExitCode: 0}, nil
	}

	findTool := NewFindFileTool(fs, mockRunner, cfg, path.NewResolver(workspaceRoot))

	req := &FindFileRequest{Pattern: "*.go", SearchPath: "", MaxDepth: 0, IncludeIgnored: false, Offset: 0, Limit: 100}
	resp, err := findTool.Run(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	matches := strings.Split(strings.TrimSpace(resp.FormattedMatches), "\n")
	if len(matches) != 2 {
		t.Fatalf("expected 2 matches, got %d. Output: %v", len(matches), resp.FormattedMatches)
	}

	expectedMatches := []string{"a/b/file.go", "a/file.go"}
	for i, expected := range expectedMatches {
		if matches[i] != expected {
			t.Errorf("match %d: expected %q, got %q", i, expected, matches[i])
		}
	}
}

func TestFindFile_Pagination(t *testing.T) {
	fs := newMockFileSystemForFind()
	fs.createDir("/workspace")
	workspaceRoot := "/workspace"
	cfg := config.DefaultConfig()

	var output string
	for i := range 10 {
		output += fmt.Sprintf("/workspace/file%d.txt\n", i)
	}

	mockRunner := &mockCommandExecutorForFind{}
	mockRunner.runFunc = func(ctx context.Context, cmd []string, dir string, env []string) (*executor.Result, error) {
		return &executor.Result{Stdout: output, ExitCode: 0}, nil
	}

	findTool := NewFindFileTool(fs, mockRunner, cfg, path.NewResolver(workspaceRoot))

	req := &FindFileRequest{Pattern: "*.txt", SearchPath: "", MaxDepth: 0, IncludeIgnored: false, Offset: 2, Limit: 2}
	resp, err := findTool.Run(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	matches := strings.Split(strings.TrimSpace(resp.FormattedMatches), "\n")
	if len(matches) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(matches))
	}

	if resp.TotalCount != 10 {
		t.Errorf("expected TotalCount 10, got %d", resp.TotalCount)
	}

	if matches[0] != "file2.txt" {
		t.Errorf("expected file2.txt, got %s", matches[0])
	}
}

func TestFindFile_InvalidGlob(t *testing.T) {
	fs := newMockFileSystemForFind()
	fs.createDir("/workspace")
	workspaceRoot := "/workspace"
	cfg := config.DefaultConfig()

	mockRunner := &mockCommandExecutorForFind{}
	mockRunner.runFunc = func(ctx context.Context, cmd []string, dir string, env []string) (*executor.Result, error) {
		return &executor.Result{Stdout: "", ExitCode: 2}, newMockExitErrorForFind(2)
	}

	findTool := NewFindFileTool(fs, mockRunner, cfg, path.NewResolver(workspaceRoot))

	req := &FindFileRequest{Pattern: "[", SearchPath: "", MaxDepth: 0, IncludeIgnored: false, Offset: 0, Limit: 100}
	_, err := findTool.Run(context.Background(), req)
	if err == nil {
		t.Fatalf("expected error for invalid glob, got nil")
	}
}

func TestFindFile_PathOutsideWorkspace(t *testing.T) {
	fs := newMockFileSystemForFind()
	fs.createDir("/workspace")
	workspaceRoot := "/workspace"
	cfg := config.DefaultConfig()

	findTool := NewFindFileTool(fs, &mockCommandExecutorForFind{}, cfg, path.NewResolver(workspaceRoot))

	req := &FindFileRequest{Pattern: "*.go", SearchPath: "../outside", MaxDepth: 0, IncludeIgnored: false, Offset: 0, Limit: 0}
	_, err := findTool.Run(context.Background(), req)

	if err == nil || !errors.Is(err, path.ErrOutsideWorkspace) {
		t.Errorf("expected ErrOutsideWorkspace, got %v", err)
	}
}

func TestFindFile_NonExistentPath(t *testing.T) {
	fs := newMockFileSystemForFind()
	fs.createDir("/workspace")
	workspaceRoot := "/workspace"
	cfg := config.DefaultConfig()

	findTool := NewFindFileTool(fs, &mockCommandExecutorForFind{}, cfg, path.NewResolver(workspaceRoot))

	req := &FindFileRequest{Pattern: "*.go", SearchPath: "nonexistent/dir", MaxDepth: 0, IncludeIgnored: false, Offset: 0, Limit: 0}
	_, err := findTool.Run(context.Background(), req)
	if err == nil {
		t.Fatalf("expected error for non-existent path, got nil")
	}
}

func TestFindFile_CommandFailure(t *testing.T) {
	fs := newMockFileSystemForFind()
	fs.createDir("/workspace")
	workspaceRoot := "/workspace"
	cfg := config.DefaultConfig()

	mockRunner := &mockCommandExecutorForFind{}
	mockRunner.runFunc = func(ctx context.Context, cmd []string, dir string, env []string) (*executor.Result, error) {
		return &executor.Result{Stdout: "", ExitCode: 2}, newMockExitErrorForFind(2)
	}

	findTool := NewFindFileTool(fs, mockRunner, cfg, path.NewResolver(workspaceRoot))

	req := &FindFileRequest{Pattern: "*.go", SearchPath: "", MaxDepth: 0, IncludeIgnored: false, Offset: 0, Limit: 100}
	_, err := findTool.Run(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for command failure, got nil")
	}
}

func TestFindFile_ShellInjection(t *testing.T) {
	fs := newMockFileSystemForFind()
	fs.createDir("/workspace")
	workspaceRoot := "/workspace"
	cfg := config.DefaultConfig()

	var capturedCmd []string
	mockRunner := &mockCommandExecutorForFind{}
	mockRunner.runFunc = func(ctx context.Context, cmd []string, dir string, env []string) (*executor.Result, error) {
		capturedCmd = cmd
		return &executor.Result{Stdout: "", ExitCode: 0}, nil
	}

	findTool := NewFindFileTool(fs, mockRunner, cfg, path.NewResolver(workspaceRoot))
	pattern := "*.go; rm -rf /"

	req := &FindFileRequest{Pattern: pattern, SearchPath: "", MaxDepth: 0, IncludeIgnored: false, Offset: 0, Limit: 100}
	_, _ = findTool.Run(context.Background(), req)

	found := slices.Contains(capturedCmd, pattern)
	if !found {
		t.Errorf("expected pattern to be passed as literal argument, got cmd: %v", capturedCmd)
	}
}

func TestFindFile_UnicodeFilenames(t *testing.T) {
	fs := newMockFileSystemForFind()
	fs.createDir("/workspace")
	workspaceRoot := "/workspace"
	cfg := config.DefaultConfig()

	mockRunner := &mockCommandExecutorForFind{}
	mockRunner.runFunc = func(ctx context.Context, cmd []string, dir string, env []string) (*executor.Result, error) {
		output := "/workspace/🚀.txt\n/workspace/文件.txt\n"
		return &executor.Result{Stdout: output, ExitCode: 0}, nil
	}

	findTool := NewFindFileTool(fs, mockRunner, cfg, path.NewResolver(workspaceRoot))

	req := &FindFileRequest{Pattern: "*.txt", SearchPath: "", MaxDepth: 0, IncludeIgnored: false, Offset: 0, Limit: 100}
	resp, err := findTool.Run(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	matches := strings.Split(strings.TrimSpace(resp.FormattedMatches), "\n")
	if len(matches) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(matches))
	}

	foundEmoji := false
	foundChinese := false
	for _, match := range matches {
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
	fs := newMockFileSystemForFind()
	fs.createDir("/workspace")
	workspaceRoot := "/workspace"
	cfg := config.DefaultConfig()

	deepPath := "/workspace"
	for i := range 100 {
		deepPath += fmt.Sprintf("/dir%d", i)
	}
	deepPath += "/file.txt"

	mockRunner := &mockCommandExecutorForFind{}
	mockRunner.runFunc = func(ctx context.Context, cmd []string, dir string, env []string) (*executor.Result, error) {
		return &executor.Result{Stdout: deepPath + "\n", ExitCode: 0}, nil
	}

	findTool := NewFindFileTool(fs, mockRunner, cfg, path.NewResolver(workspaceRoot))

	req := &FindFileRequest{Pattern: "*.txt", SearchPath: "", MaxDepth: 0, IncludeIgnored: false, Offset: 0, Limit: 100}
	resp, err := findTool.Run(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	matches := strings.Split(strings.TrimSpace(resp.FormattedMatches), "\n")
	if len(matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matches))
	}
}

func TestFindFile_NoMatches(t *testing.T) {
	fs := newMockFileSystemForFind()
	fs.createDir("/workspace")
	workspaceRoot := "/workspace"
	cfg := config.DefaultConfig()

	mockRunner := &mockCommandExecutorForFind{}
	mockRunner.runFunc = func(ctx context.Context, cmd []string, dir string, env []string) (*executor.Result, error) {
		return &executor.Result{Stdout: "", ExitCode: 0}, nil
	}

	findTool := NewFindFileTool(fs, mockRunner, cfg, path.NewResolver(workspaceRoot))

	req := &FindFileRequest{Pattern: "*.nonexistent", SearchPath: "", MaxDepth: 0, IncludeIgnored: false, Offset: 0, Limit: 100}
	resp, err := findTool.Run(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.FormattedMatches != "No matches found." {
		t.Errorf("expected 'No matches found.', got output: %q", resp.FormattedMatches)
	}
}

func TestFindFile_IncludeIgnored(t *testing.T) {
	fs := newMockFileSystemForFind()
	fs.createDir("/workspace")
	workspaceRoot := "/workspace"
	cfg := config.DefaultConfig()

	mockRunner := &mockCommandExecutorForFind{}
	mockRunner.runFunc = func(ctx context.Context, cmd []string, dir string, env []string) (*executor.Result, error) {
		if slices.Contains(cmd, "--no-ignore") {
			t.Error("expected --no-ignore to NOT be present when includeIgnored=false")
		}
		output := "/workspace/visible.go\n"
		return &executor.Result{Stdout: output, ExitCode: 0}, nil
	}

	findTool := NewFindFileTool(fs, mockRunner, cfg, path.NewResolver(workspaceRoot))

	req := &FindFileRequest{Pattern: "*.go", SearchPath: "", MaxDepth: 0, IncludeIgnored: false, Offset: 0, Limit: 100}
	resp, err := findTool.Run(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.FormattedMatches == "" {
		t.Fatal("expected 1 match, got empty string")
	}

	// Test with includeIgnored=true
	mockRunner.runFunc = func(ctx context.Context, cmd []string, dir string, env []string) (*executor.Result, error) {
		if !slices.Contains(cmd, "--no-ignore") {
			t.Error("expected --no-ignore to be present when includeIgnored=true")
		}
		output := "/workspace/ignored.go\n/workspace/visible.go\n"
		return &executor.Result{Stdout: output, ExitCode: 0}, nil
	}

	req = &FindFileRequest{Pattern: "*.go", SearchPath: "", MaxDepth: 0, IncludeIgnored: true, Offset: 0, Limit: 100}
	resp, err = findTool.Run(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error for includeIgnored=true: %v", err)
	}

	matches2 := strings.Split(strings.TrimSpace(resp.FormattedMatches), "\n")
	if len(matches2) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(matches2))
	}

	foundIgnored := false
	foundVisible := false
	for _, match := range matches2 {
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

func TestFindFile_LimitValidation(t *testing.T) {
	fs := newMockFileSystemForFind()
	fs.createDir("/workspace")
	workspaceRoot := "/workspace"
	// Config needs limits for validation
	cfg := config.DefaultConfig()
	cfg.Tools.DefaultFindFileLimit = 25
	cfg.Tools.MaxFindFileLimit = 50

	mockRunner := &mockCommandExecutorForFind{}
	mockRunner.runFunc = func(ctx context.Context, cmd []string, dir string, env []string) (*executor.Result, error) {
		return &executor.Result{Stdout: "", ExitCode: 0}, nil
	}

	t.Run("zero limit uses default", func(t *testing.T) {
		findTool := NewFindFileTool(fs, mockRunner, cfg, path.NewResolver(workspaceRoot))

		req := &FindFileRequest{Pattern: "*.go", Limit: 0}
		resp, err := findTool.Run(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if resp.Limit != cfg.Tools.DefaultFindFileLimit {
			t.Errorf("expected default limit %d, got %d", cfg.Tools.DefaultFindFileLimit, resp.Limit)
		}
	})

	t.Run("custom config limits are respected", func(t *testing.T) {
		findTool := NewFindFileTool(fs, mockRunner, cfg, path.NewResolver(workspaceRoot))

		req := &FindFileRequest{Pattern: "*.go", Limit: 30}
		resp, err := findTool.Run(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Limit != 30 {
			t.Errorf("expected limit 30, got %d", resp.Limit)
		}
	})
}

func TestFindFile_EmptySearchPath(t *testing.T) {
	fs := newMockFileSystemForFind()
	fs.createDir("/workspace")
	workspaceRoot := "/workspace"
	cfg := config.DefaultConfig()

	var capturedCmd []string
	mockRunner := &mockCommandExecutorForFind{}
	mockRunner.runFunc = func(ctx context.Context, cmd []string, dir string, env []string) (*executor.Result, error) {
		capturedCmd = cmd
		return &executor.Result{Stdout: "/workspace/file.go\n", ExitCode: 0}, nil
	}

	findTool := NewFindFileTool(fs, mockRunner, cfg, path.NewResolver(workspaceRoot))

	// SearchPath is empty, should default to "." (workspace root)
	req := &FindFileRequest{Pattern: "*.go", SearchPath: ""}
	_, err := findTool.Run(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The last argument to fd should be the search path
	if len(capturedCmd) < 1 {
		t.Fatal("expected non-empty command")
	}
	lastArg := capturedCmd[len(capturedCmd)-1]
	if lastArg != "/workspace" {
		t.Errorf("expected search path '/workspace', got %q", lastArg)
	}
}

func TestFindFile_HitMaxResults(t *testing.T) {
	fs := newMockFileSystemForFind()
	fs.createDir("/workspace")
	workspaceRoot := "/workspace"
	cfg := config.DefaultConfig()
	cfg.Tools.MaxFindFileResults = 2

	mockRunner := &mockCommandExecutorForFind{}
	mockRunner.runFunc = func(ctx context.Context, cmd []string, dir string, env []string) (*executor.Result, error) {
		output := "file1.go\nfile2.go\nfile3.go\n"
		return &executor.Result{Stdout: output, ExitCode: 0}, nil
	}

	findTool := NewFindFileTool(fs, mockRunner, cfg, path.NewResolver(workspaceRoot))

	req := &FindFileRequest{Pattern: "*.go", SearchPath: "", MaxDepth: 0, IncludeIgnored: false, Offset: 0, Limit: 100}
	resp, err := findTool.Run(context.Background(), req)
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
