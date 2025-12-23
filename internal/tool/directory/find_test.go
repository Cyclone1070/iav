package directory

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Cyclone1070/iav/internal/config"
	"github.com/Cyclone1070/iav/internal/tool/executil"
	"github.com/Cyclone1070/iav/internal/tool/pathutil"
)

// Local mocks for find tests

type mockFileInfoForFind struct {
	name  string
	isDir bool
}

func (m *mockFileInfoForFind) Name() string       { return m.name }
func (m *mockFileInfoForFind) Size() int64        { return 0 }
func (m *mockFileInfoForFind) Mode() os.FileMode  { return 0644 }
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
	startFunc func(ctx context.Context, cmd []string, dir string, env []string) (executil.Process, io.Reader, io.Reader, error)
}

func (m *mockCommandExecutorForFind) Start(ctx context.Context, cmd []string, dir string, env []string) (executil.Process, io.Reader, io.Reader, error) {
	if m.startFunc != nil {
		return m.startFunc(ctx, cmd, dir, env)
	}
	return nil, nil, nil, fmt.Errorf("not implemented")
}

type mockProcessForFind struct {
	waitFunc func() error
}

func (m *mockProcessForFind) Wait() error {
	if m.waitFunc != nil {
		return m.waitFunc()
	}
	return nil
}

func (m *mockProcessForFind) Signal(sig os.Signal) error { return nil }
func (m *mockProcessForFind) Kill() error                { return nil }

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
	mockRunner.startFunc = func(ctx context.Context, cmd []string, dir string, env []string) (executil.Process, io.Reader, io.Reader, error) {
		output := "/workspace/a/b/file.go\n/workspace/a/file.go\n"
		return &mockProcessForFind{}, strings.NewReader(output), strings.NewReader(""), nil
	}

	findTool := NewFindFileTool(fs, mockRunner, cfg, pathutil.NewResolver(workspaceRoot, fs))

	req := &FindFileRequest{Pattern: "*.go", SearchPath: "", MaxDepth: 0, IncludeIgnored: false, Offset: 0, Limit: 100}
	resp, err := findTool.Run(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Matches) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(resp.Matches))
	}

	expectedMatches := []string{"a/b/file.go", "a/file.go"}
	for i, expected := range expectedMatches {
		if resp.Matches[i] != expected {
			t.Errorf("match %d: expected %q, got %q", i, expected, resp.Matches[i])
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
	mockRunner.startFunc = func(ctx context.Context, cmd []string, dir string, env []string) (executil.Process, io.Reader, io.Reader, error) {
		return &mockProcessForFind{}, strings.NewReader(output), strings.NewReader(""), nil
	}

	findTool := NewFindFileTool(fs, mockRunner, cfg, pathutil.NewResolver(workspaceRoot, fs))

	req := &FindFileRequest{Pattern: "*.txt", SearchPath: "", MaxDepth: 0, IncludeIgnored: false, Offset: 2, Limit: 2}
	resp, err := findTool.Run(context.Background(), req)
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

	if resp.Matches[0] != "file2.txt" {
		t.Errorf("expected file2.txt, got %s", resp.Matches[0])
	}
}

func TestFindFile_InvalidGlob(t *testing.T) {
	fs := newMockFileSystemForFind()
	fs.createDir("/workspace")
	workspaceRoot := "/workspace"
	cfg := config.DefaultConfig()

	mockRunner := &mockCommandExecutorForFind{}
	mockRunner.startFunc = func(ctx context.Context, cmd []string, dir string, env []string) (executil.Process, io.Reader, io.Reader, error) {
		proc := &mockProcessForFind{}
		proc.waitFunc = func() error {
			return newMockExitErrorForFind(2)
		}
		return proc, strings.NewReader(""), strings.NewReader(""), nil
	}

	findTool := NewFindFileTool(fs, mockRunner, cfg, pathutil.NewResolver(workspaceRoot, fs))

	req := &FindFileRequest{Pattern: "[", SearchPath: "", MaxDepth: 0, IncludeIgnored: false, Offset: 0, Limit: 100}
	_, err := findTool.Run(context.Background(), req)
	if err == nil || !errors.Is(err, ErrInvalidPattern) {
		t.Fatalf("expected ErrInvalidPattern for invalid glob, got %v", err)
	}
}

func TestFindFile_PathOutsideWorkspace(t *testing.T) {
	fs := newMockFileSystemForFind()
	fs.createDir("/workspace")
	workspaceRoot := "/workspace"
	cfg := config.DefaultConfig()

	findTool := NewFindFileTool(fs, &mockCommandExecutorForFind{}, cfg, pathutil.NewResolver(workspaceRoot, fs))

	req := &FindFileRequest{Pattern: "*.go", SearchPath: "../outside", MaxDepth: 0, IncludeIgnored: false, Offset: 0, Limit: 0}
	_, err := findTool.Run(context.Background(), req)

	if err == nil || !errors.Is(err, pathutil.ErrOutsideWorkspace) {
		t.Errorf("expected ErrOutsideWorkspace, got %v", err)
	}
}

func TestFindFile_NonExistentPath(t *testing.T) {
	fs := newMockFileSystemForFind()
	fs.createDir("/workspace")
	workspaceRoot := "/workspace"
	cfg := config.DefaultConfig()

	findTool := NewFindFileTool(fs, &mockCommandExecutorForFind{}, cfg, pathutil.NewResolver(workspaceRoot, fs))

	req := &FindFileRequest{Pattern: "*.go", SearchPath: "nonexistent/dir", MaxDepth: 0, IncludeIgnored: false, Offset: 0, Limit: 0}
	_, err := findTool.Run(context.Background(), req)
	if err == nil || !errors.Is(err, ErrFileMissing) {
		t.Fatalf("expected ErrFileMissing for non-existent path, got %v", err)
	}
}

func TestFindFile_CommandFailure(t *testing.T) {
	fs := newMockFileSystemForFind()
	fs.createDir("/workspace")
	workspaceRoot := "/workspace"
	cfg := config.DefaultConfig()

	mockRunner := &mockCommandExecutorForFind{}
	mockRunner.startFunc = func(ctx context.Context, cmd []string, dir string, env []string) (executil.Process, io.Reader, io.Reader, error) {
		proc := &mockProcessForFind{}
		proc.waitFunc = func() error {
			return newMockExitErrorForFind(2)
		}
		return proc, strings.NewReader(""), strings.NewReader(""), nil
	}

	findTool := NewFindFileTool(fs, mockRunner, cfg, pathutil.NewResolver(workspaceRoot, fs))

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
	mockRunner.startFunc = func(ctx context.Context, cmd []string, dir string, env []string) (executil.Process, io.Reader, io.Reader, error) {
		capturedCmd = cmd
		return &mockProcessForFind{}, strings.NewReader(""), strings.NewReader(""), nil
	}

	findTool := NewFindFileTool(fs, mockRunner, cfg, pathutil.NewResolver(workspaceRoot, fs))
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
	mockRunner.startFunc = func(ctx context.Context, cmd []string, dir string, env []string) (executil.Process, io.Reader, io.Reader, error) {
		output := "/workspace/🚀.txt\n/workspace/文件.txt\n"
		return &mockProcessForFind{}, strings.NewReader(output), strings.NewReader(""), nil
	}

	findTool := NewFindFileTool(fs, mockRunner, cfg, pathutil.NewResolver(workspaceRoot, fs))

	req := &FindFileRequest{Pattern: "*.txt", SearchPath: "", MaxDepth: 0, IncludeIgnored: false, Offset: 0, Limit: 100}
	resp, err := findTool.Run(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Matches) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(resp.Matches))
	}

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
	mockRunner.startFunc = func(ctx context.Context, cmd []string, dir string, env []string) (executil.Process, io.Reader, io.Reader, error) {
		return &mockProcessForFind{}, strings.NewReader(deepPath + "\n"), strings.NewReader(""), nil
	}

	findTool := NewFindFileTool(fs, mockRunner, cfg, pathutil.NewResolver(workspaceRoot, fs))

	req := &FindFileRequest{Pattern: "*.txt", SearchPath: "", MaxDepth: 0, IncludeIgnored: false, Offset: 0, Limit: 100}
	resp, err := findTool.Run(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(resp.Matches))
	}
}

func TestFindFile_NoMatches(t *testing.T) {
	fs := newMockFileSystemForFind()
	fs.createDir("/workspace")
	workspaceRoot := "/workspace"
	cfg := config.DefaultConfig()

	mockRunner := &mockCommandExecutorForFind{}
	mockRunner.startFunc = func(ctx context.Context, cmd []string, dir string, env []string) (executil.Process, io.Reader, io.Reader, error) {
		return &mockProcessForFind{}, strings.NewReader(""), strings.NewReader(""), nil
	}

	findTool := NewFindFileTool(fs, mockRunner, cfg, pathutil.NewResolver(workspaceRoot, fs))

	req := &FindFileRequest{Pattern: "*.nonexistent", SearchPath: "", MaxDepth: 0, IncludeIgnored: false, Offset: 0, Limit: 100}
	resp, err := findTool.Run(context.Background(), req)
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
	fs := newMockFileSystemForFind()
	fs.createDir("/workspace")
	workspaceRoot := "/workspace"
	cfg := config.DefaultConfig()

	mockRunner := &mockCommandExecutorForFind{}
	mockRunner.startFunc = func(ctx context.Context, cmd []string, dir string, env []string) (executil.Process, io.Reader, io.Reader, error) {
		if slices.Contains(cmd, "--no-ignore") {
			t.Error("expected --no-ignore to NOT be present when includeIgnored=false")
		}
		output := "/workspace/visible.go\n"
		return &mockProcessForFind{}, strings.NewReader(output), strings.NewReader(""), nil
	}

	findTool := NewFindFileTool(fs, mockRunner, cfg, pathutil.NewResolver(workspaceRoot, fs))

	req := &FindFileRequest{Pattern: "*.go", SearchPath: "", MaxDepth: 0, IncludeIgnored: false, Offset: 0, Limit: 100}
	resp, err := findTool.Run(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(resp.Matches))
	}

	// Test with includeIgnored=true
	mockRunner.startFunc = func(ctx context.Context, cmd []string, dir string, env []string) (executil.Process, io.Reader, io.Reader, error) {
		if !slices.Contains(cmd, "--no-ignore") {
			t.Error("expected --no-ignore to be present when includeIgnored=true")
		}
		output := "/workspace/ignored.go\n/workspace/visible.go\n"
		return &mockProcessForFind{}, strings.NewReader(output), strings.NewReader(""), nil
	}

	req = &FindFileRequest{Pattern: "*.go", SearchPath: "", MaxDepth: 0, IncludeIgnored: true, Offset: 0, Limit: 100}
	resp, err = findTool.Run(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error for includeIgnored=true: %v", err)
	}

	if len(resp.Matches) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(resp.Matches))
	}

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

func TestFindFile_LimitValidation(t *testing.T) {
	fs := newMockFileSystemForFind()
	fs.createDir("/workspace")
	workspaceRoot := "/workspace"
	// Config needs limits for validation
	cfg := config.DefaultConfig()
	cfg.Tools.DefaultFindFileLimit = 25
	cfg.Tools.MaxFindFileLimit = 50

	mockRunner := &mockCommandExecutorForFind{}
	mockRunner.startFunc = func(ctx context.Context, cmd []string, dir string, env []string) (executil.Process, io.Reader, io.Reader, error) {
		return &mockProcessForFind{}, strings.NewReader(""), strings.NewReader(""), nil
	}

	t.Run("zero limit uses default", func(t *testing.T) {
		findTool := NewFindFileTool(fs, mockRunner, cfg, pathutil.NewResolver(workspaceRoot, fs))

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
		findTool := NewFindFileTool(fs, mockRunner, cfg, pathutil.NewResolver(workspaceRoot, fs))

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
