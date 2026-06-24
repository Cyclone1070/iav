package cmd

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Cyclone1070/iav/internal/domain"
)

type mockFS struct {
	files      map[string][]byte
	symlinks   map[string]string
	currentDir string
}

func (m *mockFS) Open(name string) (fs.File, error) {
	name = filepath.Clean(name)
	if data, ok := m.files[name]; ok {
		return &mockFile{name: name, data: data}, nil
	}
	return nil, fs.ErrNotExist
}

func (m *mockFS) Stat(name string) (fs.FileInfo, error) {
	name = filepath.Clean(name)
	if _, ok := m.files[name]; ok {
		return &mockFileInfo{name: filepath.Base(name), size: int64(len(m.files[name])), isDir: false}, nil
	}
	prefix := name
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	for f := range m.files {
		if strings.HasPrefix(f, prefix) {
			return &mockFileInfo{name: filepath.Base(name), size: 0, isDir: true}, nil
		}
	}
	if name == "/" || name == "." || name == "" {
		return &mockFileInfo{name: name, size: 0, isDir: true}, nil
	}
	return nil, os.ErrNotExist
}

func (m *mockFS) WalkDir(root string, fn fs.WalkDirFunc) error {
	root = filepath.Clean(root)
	rootInfo, err := m.Stat(root)
	if err != nil {
		return err
	}
	err = fn(root, &mockDirEntry{info: rootInfo}, nil)
	if err != nil {
		if errors.Is(err, filepath.SkipDir) {
			return nil
		}
		return err
	}

	return m.walkChildren(root, fn)
}

func (m *mockFS) walkChildren(root string, fn fs.WalkDirFunc) error {
	visited := make(map[string]bool)
	for path := range m.files {
		childPath, ok := shouldVisit(root, path, visited)
		if ok {
			if err := m.visitChild(childPath, fn); err != nil {
				return err
			}
		}
	}
	return nil
}

func (m *mockFS) visitChild(childPath string, fn fs.WalkDirFunc) error {
	childInfo, err := m.Stat(childPath)
	if err != nil {
		return nil
	}
	if err := fn(childPath, &mockDirEntry{info: childInfo}, nil); err != nil {
		if errors.Is(err, filepath.SkipDir) {
			return nil
		}
		return err
	}
	return nil
}

func shouldVisit(root, path string, visited map[string]bool) (string, bool) {
	if !strings.HasPrefix(path, root) {
		return "", false
	}
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == "." || strings.HasPrefix(rel, "..") {
		return "", false
	}
	parts := strings.Split(rel, string(filepath.Separator))
	childPath := filepath.Join(root, parts[0])
	if visited[childPath] {
		return "", false
	}
	visited[childPath] = true
	return childPath, true
}

func (m *mockFS) EvalSymlinks(path string) (string, error) {
	path = filepath.Clean(path)
	if target, ok := m.symlinks[path]; ok {
		return target, nil
	}
	if _, ok := m.files[path]; ok {
		return path, nil
	}
	if info, err := m.Stat(path); err == nil && info.IsDir() {
		return path, nil
	}
	return "", os.ErrNotExist
}

func (m *mockFS) Abs(path string) (string, error) {
	if filepath.IsAbs(path) {
		return filepath.Clean(path), nil
	}
	return filepath.Clean(filepath.Join(m.currentDir, path)), nil
}

type mockFile struct {
	name string
	data []byte
	off  int64
}

func (f *mockFile) Stat() (fs.FileInfo, error) {
	return &mockFileInfo{name: filepath.Base(f.name), size: int64(len(f.data)), isDir: false}, nil
}

func (f *mockFile) Read(b []byte) (int, error) {
	if f.off >= int64(len(f.data)) {
		return 0, io.EOF
	}
	n := copy(b, f.data[f.off:])
	f.off += int64(n)
	return n, nil
}

func (f *mockFile) Close() error { return nil }

type mockFileInfo struct {
	name  string
	size  int64
	isDir bool
}

func (m *mockFileInfo) Name() string { return m.name }
func (m *mockFileInfo) Size() int64  { return m.size }
func (m *mockFileInfo) Mode() fs.FileMode {
	if m.isDir {
		return fs.ModeDir | 0755
	}
	return 0644
}
func (m *mockFileInfo) ModTime() time.Time { return time.Time{} }
func (m *mockFileInfo) IsDir() bool        { return m.isDir }
func (m *mockFileInfo) Sys() any           { return nil }

type mockDirEntry struct {
	info fs.FileInfo
}

func (d *mockDirEntry) Name() string               { return d.info.Name() }
func (d *mockDirEntry) IsDir() bool                { return d.info.IsDir() }
func (d *mockDirEntry) Type() fs.FileMode          { return d.info.Mode().Type() }
func (d *mockDirEntry) Info() (fs.FileInfo, error) { return d.info, nil }

type mockDockerClient struct {
	closeFunc                 func() error
	pruneExpiredResourcesFunc func(ctx context.Context) error
	composeUpFunc             func(ctx context.Context, composeFiles []string, runID string) error
	composeDownFunc           func(ctx context.Context, composeFiles []string, runID string) error
	runContainerInfoFunc      func(ctx context.Context, opts domain.RunOptions) (string, string, int, error)
}

func (m *mockDockerClient) Close() error {
	if m.closeFunc != nil {
		return m.closeFunc()
	}
	return nil
}

func (m *mockDockerClient) PruneExpiredResources(ctx context.Context) error {
	if m.pruneExpiredResourcesFunc != nil {
		return m.pruneExpiredResourcesFunc(ctx)
	}
	return nil
}

func (m *mockDockerClient) ComposeUp(ctx context.Context, composeFiles []string, runID string) error {
	if m.composeUpFunc != nil {
		return m.composeUpFunc(ctx, composeFiles, runID)
	}
	return nil
}

func (m *mockDockerClient) ComposeDown(ctx context.Context, composeFiles []string, runID string) error {
	if m.composeDownFunc != nil {
		return m.composeDownFunc(ctx, composeFiles, runID)
	}
	return nil
}

func (m *mockDockerClient) RunContainerInfo(ctx context.Context, opts domain.RunOptions) (string, string, int, error) {
	if m.runContainerInfoFunc != nil {
		return m.runContainerInfoFunc(ctx, opts)
	}
	return "ok", "", 0, nil
}

func TestRootCommand_BothLintAndTest(t *testing.T) {
	m := &mockFS{
		files: map[string][]byte{
			"/workspace/Dockerfile":              []byte("FROM alpine"),
			"/workspace/docker-compose.yml":      []byte("version: '3'"),
			"/workspace/docker-compose.test.yml": []byte("version: '3'"),
		},
		currentDir: "/workspace",
	}

	oldFS := defaultFS
	defaultFS = m
	defer func() { defaultFS = oldFS }()

	mockClient := &mockDockerClient{}
	var runContainerCalled, composeUpCalled bool
	mockClient.runContainerInfoFunc = func(ctx context.Context, opts domain.RunOptions) (string, string, int, error) {
		runContainerCalled = true
		return "ok", "", 0, nil
	}
	mockClient.composeUpFunc = func(ctx context.Context, composeFiles []string, runID string) error {
		composeUpCalled = true
		return nil
	}

	oldFactory := dockerClientFactory
	dockerClientFactory = func() (dockerClient, error) {
		return mockClient, nil
	}
	defer func() { dockerClientFactory = oldFactory }()

	root := NewRootCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)

	root.SetArgs([]string{"/workspace"})
	err := root.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v, output: %s", err, buf.String())
	}

	if !runContainerCalled {
		t.Errorf("expected Hadolint container run to be called")
	}
	if !composeUpCalled {
		t.Errorf("expected ComposeUp to be called")
	}
}

func TestRootCommand_OnlyLint(t *testing.T) {
	m := &mockFS{
		files: map[string][]byte{
			"/workspace/Dockerfile": []byte("FROM alpine"),
		},
		currentDir: "/workspace",
	}

	oldFS := defaultFS
	defaultFS = m
	defer func() { defaultFS = oldFS }()

	mockClient := &mockDockerClient{}
	var runContainerCalled, composeUpCalled bool
	mockClient.runContainerInfoFunc = func(ctx context.Context, opts domain.RunOptions) (string, string, int, error) {
		runContainerCalled = true
		return "ok", "", 0, nil
	}
	mockClient.composeUpFunc = func(ctx context.Context, composeFiles []string, runID string) error {
		composeUpCalled = true
		return nil
	}

	oldFactory := dockerClientFactory
	dockerClientFactory = func() (dockerClient, error) {
		return mockClient, nil
	}
	defer func() { dockerClientFactory = oldFactory }()

	root := NewRootCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)

	root.SetArgs([]string{"/workspace"})
	err := root.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v, output: %s", err, buf.String())
	}

	if !runContainerCalled {
		t.Errorf("expected Hadolint container run to be called")
	}
	if composeUpCalled {
		t.Errorf("expected ComposeUp NOT to be called")
	}
}

func TestRootCommand_OnlyTest(t *testing.T) {
	m := &mockFS{
		files: map[string][]byte{
			"/workspace/docker-compose.yml":      []byte("version: '3'"),
			"/workspace/docker-compose.test.yml": []byte("version: '3'"),
		},
		currentDir: "/workspace",
	}

	oldFS := defaultFS
	defaultFS = m
	defer func() { defaultFS = oldFS }()

	mockClient := &mockDockerClient{}
	var runContainerCalled, composeUpCalled bool
	mockClient.runContainerInfoFunc = func(ctx context.Context, opts domain.RunOptions) (string, string, int, error) {
		runContainerCalled = true
		return "ok", "", 0, nil
	}
	mockClient.composeUpFunc = func(ctx context.Context, composeFiles []string, runID string) error {
		composeUpCalled = true
		return nil
	}

	oldFactory := dockerClientFactory
	dockerClientFactory = func() (dockerClient, error) {
		return mockClient, nil
	}
	defer func() { dockerClientFactory = oldFactory }()

	root := NewRootCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)

	root.SetArgs([]string{"/workspace"})
	err := root.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v, output: %s", err, buf.String())
	}

	if runContainerCalled {
		t.Errorf("expected Hadolint container run NOT to be called")
	}
	if !composeUpCalled {
		t.Errorf("expected ComposeUp to be called")
	}
}

func TestSubcommand_TestOnly(t *testing.T) {
	m := &mockFS{
		files: map[string][]byte{
			"/workspace/Dockerfile":              []byte("FROM alpine"),
			"/workspace/docker-compose.yml":      []byte("version: '3'"),
			"/workspace/docker-compose.test.yml": []byte("version: '3'"),
		},
		currentDir: "/workspace",
	}

	oldFS := defaultFS
	defaultFS = m
	defer func() { defaultFS = oldFS }()

	mockClient := &mockDockerClient{}
	var runContainerCalled, composeUpCalled bool
	mockClient.runContainerInfoFunc = func(ctx context.Context, opts domain.RunOptions) (string, string, int, error) {
		runContainerCalled = true
		return "ok", "", 0, nil
	}
	mockClient.composeUpFunc = func(ctx context.Context, composeFiles []string, runID string) error {
		composeUpCalled = true
		return nil
	}

	oldFactory := dockerClientFactory
	dockerClientFactory = func() (dockerClient, error) {
		return mockClient, nil
	}
	defer func() { dockerClientFactory = oldFactory }()

	root := NewRootCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)

	root.SetArgs([]string{"test", "/workspace"})
	err := root.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v, output: %s", err, buf.String())
	}

	if runContainerCalled {
		t.Errorf("expected Hadolint container run NOT to be called")
	}
	if !composeUpCalled {
		t.Errorf("expected ComposeUp to be called")
	}
}

func TestSubcommand_LintOnly(t *testing.T) {
	m := &mockFS{
		files: map[string][]byte{
			"/workspace/Dockerfile":              []byte("FROM alpine"),
			"/workspace/docker-compose.yml":      []byte("version: '3'"),
			"/workspace/docker-compose.test.yml": []byte("version: '3'"),
		},
		currentDir: "/workspace",
	}

	oldFS := defaultFS
	defaultFS = m
	defer func() { defaultFS = oldFS }()

	mockClient := &mockDockerClient{}
	var runContainerCalled, composeUpCalled bool
	mockClient.runContainerInfoFunc = func(ctx context.Context, opts domain.RunOptions) (string, string, int, error) {
		runContainerCalled = true
		return "ok", "", 0, nil
	}
	mockClient.composeUpFunc = func(ctx context.Context, composeFiles []string, runID string) error {
		composeUpCalled = true
		return nil
	}

	oldFactory := dockerClientFactory
	dockerClientFactory = func() (dockerClient, error) {
		return mockClient, nil
	}
	defer func() { dockerClientFactory = oldFactory }()

	root := NewRootCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)

	root.SetArgs([]string{"lint", "/workspace"})
	err := root.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v, output: %s", err, buf.String())
	}

	if !runContainerCalled {
		t.Errorf("expected Hadolint container run to be called")
	}
	if composeUpCalled {
		t.Errorf("expected ComposeUp NOT to be called")
	}
}

func TestRootCommand_MultipleTestFilesRunsLintOnce(t *testing.T) {
	m := &mockFS{
		files: map[string][]byte{
			"/workspace/Dockerfile":              []byte("FROM alpine"),
			"/workspace/docker-compose.yml":      []byte("version: '3'"),
			"/workspace/docker-compose.test.yml": []byte("version: '3'"),
			"/workspace/another.test.yml":        []byte("version: '3'"),
		},
		currentDir: "/workspace",
	}

	oldFS := defaultFS
	defaultFS = m
	defer func() { defaultFS = oldFS }()

	mockClient := &mockDockerClient{}
	var runContainerCallCount, composeUpCallCount int
	mockClient.runContainerInfoFunc = func(ctx context.Context, opts domain.RunOptions) (string, string, int, error) {
		runContainerCallCount++
		return "ok", "", 0, nil
	}
	mockClient.composeUpFunc = func(ctx context.Context, composeFiles []string, runID string) error {
		composeUpCallCount++
		return nil
	}

	oldFactory := dockerClientFactory
	dockerClientFactory = func() (dockerClient, error) {
		return mockClient, nil
	}
	defer func() { dockerClientFactory = oldFactory }()

	root := NewRootCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)

	root.SetArgs([]string{"/workspace"})
	err := root.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v, output: %s", err, buf.String())
	}

	if runContainerCallCount != 1 {
		t.Errorf("expected Hadolint container run to be called exactly ONCE, but was called %d times", runContainerCallCount)
	}
	if composeUpCallCount != 2 {
		t.Errorf("expected ComposeUp to be called exactly TWICE (once per test file), but was called %d times", composeUpCallCount)
	}
}

