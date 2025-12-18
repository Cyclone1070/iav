package shell

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Cyclone1070/iav/internal/config"
)

// Local mocks for shell tests

// mockFileInfo implements os.FileInfo for testing
type mockFileInfo struct {
	name  string
	isDir bool
}

func (m *mockFileInfo) Name() string       { return m.name }
func (m *mockFileInfo) Size() int64        { return 0 }
func (m *mockFileInfo) Mode() os.FileMode  { return 0644 }
func (m *mockFileInfo) ModTime() time.Time { return time.Time{} }
func (m *mockFileInfo) IsDir() bool        { return m.isDir }
func (m *mockFileInfo) Sys() interface{}   { return nil }

type mockFileSystemForShell struct {
	files map[string][]byte
	dirs  map[string]bool
}

func newMockFileSystemForShell() *mockFileSystemForShell {
	return &mockFileSystemForShell{
		files: make(map[string][]byte),
		dirs:  make(map[string]bool),
	}
}

func (m *mockFileSystemForShell) createDir(path string) {
	m.dirs[path] = true
}

func (m *mockFileSystemForShell) createFile(path string, content []byte) {
	m.files[path] = content
}

func (m *mockFileSystemForShell) Stat(path string) (os.FileInfo, error) {
	if m.dirs[path] {
		return &mockFileInfo{name: path, isDir: true}, nil
	}
	if _, ok := m.files[path]; ok {
		return &mockFileInfo{name: path, isDir: false}, nil
	}
	return nil, os.ErrNotExist
}

func (m *mockFileSystemForShell) Lstat(path string) (os.FileInfo, error) {
	return m.Stat(path)
}

func (m *mockFileSystemForShell) Readlink(path string) (string, error) {
	return "", os.ErrInvalid
}

func (m *mockFileSystemForShell) UserHomeDir() (string, error) {
	return "/home/user", nil
}

func (m *mockFileSystemForShell) ReadFileRange(path string, offset, limit int64) ([]byte, error) {
	content, ok := m.files[path]
	if !ok {
		return nil, os.ErrNotExist
	}
	return content, nil
}

type mockCommandExecutorForShell struct {
	runFunc   func(ctx context.Context, cmd []string) ([]byte, error)
	startFunc func(ctx context.Context, cmd []string, opts ProcessOptions) (Process, io.Reader, io.Reader, error)
}

func (m *mockCommandExecutorForShell) Run(ctx context.Context, cmd []string) ([]byte, error) {
	if m.runFunc != nil {
		return m.runFunc(ctx, cmd)
	}
	return nil, errors.New("not implemented")
}

func (m *mockCommandExecutorForShell) Start(ctx context.Context, cmd []string, opts ProcessOptions) (Process, io.Reader, io.Reader, error) {
	if m.startFunc != nil {
		return m.startFunc(ctx, cmd, opts)
	}
	return nil, nil, nil, errors.New("not implemented")
}

type mockProcessForShell struct {
	waitFunc   func() error
	signalFunc func(sig os.Signal) error
	killFunc   func() error
}

func (m *mockProcessForShell) Wait() error {
	if m.waitFunc != nil {
		return m.waitFunc()
	}
	return nil
}

func (m *mockProcessForShell) Signal(sig os.Signal) error {
	if m.signalFunc != nil {
		return m.signalFunc(sig)
	}
	return nil
}

func (m *mockProcessForShell) Kill() error {
	if m.killFunc != nil {
		return m.killFunc()
	}
	return nil
}

// mockExitError simulates an exit error with a specific exit code
type mockExitError struct {
	exitCode int
}

func (e *mockExitError) Error() string {
	return "exit status " + string(rune(e.exitCode))
}

func (e *mockExitError) ExitCode() int {
	return e.exitCode
}

func newMockExitError(code int) error {
	return &mockExitError{exitCode: code}
}

// Test functions

func TestShellTool_Run_SimpleCommand(t *testing.T) {
	mockFS := newMockFileSystemForShell()
	mockFS.createDir("/workspace")
	workspaceRoot := "/workspace"
	cfg := config.DefaultConfig()

	factory := &mockCommandExecutorForShell{}
	factory.startFunc = func(ctx context.Context, command []string, opts ProcessOptions) (Process, io.Reader, io.Reader, error) {
		if command[0] != "echo" {
			return nil, nil, nil, errors.New("unexpected command")
		}
		stdout := strings.NewReader("hello\n")
		stderr := strings.NewReader("")
		proc := &mockProcessForShell{}
		proc.waitFunc = func() error { return nil }
		return proc, stdout, stderr, nil
	}

	tool := NewShellTool(mockFS, factory, cfg, DockerConfig{}, workspaceRoot)

	dto := ShellDTO{Command: []string{"echo", "hello"}}
	req, err := NewShellRequest(dto, cfg, workspaceRoot, mockFS)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	resp, err := tool.Run(context.Background(), req)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if resp.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", resp.ExitCode)
	}
	if strings.TrimSpace(resp.Stdout) != "hello" {
		t.Errorf("Stdout = %q, want %q", resp.Stdout, "hello")
	}
}

func TestShellTool_Run_WorkingDir(t *testing.T) {
	mockFS := newMockFileSystemForShell()
	mockFS.createDir("/workspace")
	mockFS.createDir("/workspace/subdir")
	workspaceRoot := "/workspace"
	cfg := config.DefaultConfig()

	var capturedDir string
	factory := &mockCommandExecutorForShell{}
	factory.startFunc = func(ctx context.Context, command []string, opts ProcessOptions) (Process, io.Reader, io.Reader, error) {
		capturedDir = opts.Dir
		proc := &mockProcessForShell{}
		proc.waitFunc = func() error { return nil }
		return proc, strings.NewReader(""), strings.NewReader(""), nil
	}

	tool := NewShellTool(mockFS, factory, cfg, DockerConfig{}, workspaceRoot)

	dto := ShellDTO{Command: []string{"pwd"}, WorkingDir: "subdir"}
	req, err := NewShellRequest(dto, cfg, workspaceRoot, mockFS)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	_, err = tool.Run(context.Background(), req)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	expectedDir := "/workspace/subdir"
	if capturedDir != expectedDir {
		t.Errorf("Working directory = %q, want %q", capturedDir, expectedDir)
	}
}

func TestShellTool_Run_Env(t *testing.T) {
	mockFS := newMockFileSystemForShell()
	mockFS.createDir("/workspace")
	workspaceRoot := "/workspace"
	cfg := config.DefaultConfig()

	var capturedEnv []string
	factory := &mockCommandExecutorForShell{}
	factory.startFunc = func(ctx context.Context, command []string, opts ProcessOptions) (Process, io.Reader, io.Reader, error) {
		capturedEnv = opts.Env
		proc := &mockProcessForShell{}
		proc.waitFunc = func() error { return nil }
		return proc, strings.NewReader(""), strings.NewReader(""), nil
	}

	tool := NewShellTool(mockFS, factory, cfg, DockerConfig{}, workspaceRoot)

	dto := ShellDTO{
		Command: []string{"env"},
		Env: map[string]string{
			"CUSTOM_VAR": "custom_value",
			"TEST_MODE":  "true",
		},
	}
	req, err := NewShellRequest(dto, cfg, workspaceRoot, mockFS)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	_, err = tool.Run(context.Background(), req)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	hasCustomVar := false
	hasTestMode := false
	for _, envVar := range capturedEnv {
		if envVar == "CUSTOM_VAR=custom_value" {
			hasCustomVar = true
		}
		if envVar == "TEST_MODE=true" {
			hasTestMode = true
		}
	}

	if !hasCustomVar {
		t.Error("Expected CUSTOM_VAR=custom_value in environment")
	}
	if !hasTestMode {
		t.Error("Expected TEST_MODE=true in environment")
	}
}

func TestShellTool_Run_EnvFiles(t *testing.T) {
	mockFS := newMockFileSystemForShell()
	mockFS.createDir("/workspace")
	workspaceRoot := "/workspace"
	cfg := config.DefaultConfig()

	// Create env files
	envFile1Content := `DB_HOST=localhost
DB_PORT=5432
API_KEY=secret123`
	mockFS.createFile("/workspace/.env", []byte(envFile1Content))

	envFile2Content := `DB_PORT=3306
CACHE_URL=redis://localhost`
	mockFS.createFile("/workspace/.env.local", []byte(envFile2Content))

	var capturedEnv []string
	factory := &mockCommandExecutorForShell{}
	factory.startFunc = func(ctx context.Context, command []string, opts ProcessOptions) (Process, io.Reader, io.Reader, error) {
		capturedEnv = opts.Env
		proc := &mockProcessForShell{}
		proc.waitFunc = func() error { return nil }
		return proc, strings.NewReader(""), strings.NewReader(""), nil
	}

	tool := NewShellTool(mockFS, factory, cfg, DockerConfig{}, workspaceRoot)

	t.Run("Single env file", func(t *testing.T) {
		dto := ShellDTO{Command: []string{"env"}, EnvFiles: []string{".env"}}
		req, err := NewShellRequest(dto, cfg, workspaceRoot, mockFS)
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}

		_, err = tool.Run(context.Background(), req)
		if err != nil {
			t.Fatalf("Run failed: %v", err)
		}

		// Check that env vars from file are present
		hasDBHost := false
		hasDBPort := false
		hasAPIKey := false
		for _, envVar := range capturedEnv {
			if envVar == "DB_HOST=localhost" {
				hasDBHost = true
			}
			if envVar == "DB_PORT=5432" {
				hasDBPort = true
			}
			if envVar == "API_KEY=secret123" {
				hasAPIKey = true
			}
		}

		if !hasDBHost {
			t.Error("Expected DB_HOST=localhost in environment")
		}
		if !hasDBPort {
			t.Error("Expected DB_PORT=5432 in environment")
		}
		if !hasAPIKey {
			t.Error("Expected API_KEY=secret123 in environment")
		}
	})

	t.Run("Multiple env files with override - explicit ordering", func(t *testing.T) {
		dto := ShellDTO{Command: []string{"env"}, EnvFiles: []string{".env", ".env.local"}}
		req, err := NewShellRequest(dto, cfg, workspaceRoot, mockFS)
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}

		_, err = tool.Run(context.Background(), req)
		if err != nil {
			t.Fatalf("Run failed: %v", err)
		}

		// Count all DB_PORT occurrences and track the last one
		dbPortCount := 0
		lastDBPort := ""
		for _, envVar := range capturedEnv {
			if strings.HasPrefix(envVar, "DB_PORT=") {
				dbPortCount++
				lastDBPort = envVar
			}
		}

		// Both values should be in the array (env append behavior)
		if dbPortCount < 2 {
			t.Errorf("Expected at least 2 DB_PORT entries (from both files), got %d", dbPortCount)
		}

		// The LAST value wins (OS behavior) - should be from .env.local
		if lastDBPort != "DB_PORT=3306" {
			t.Errorf("Expected last DB_PORT=3306 (from .env.local), got %s", lastDBPort)
		}
	})

	t.Run("Request.Env overrides EnvFiles", func(t *testing.T) {
		dto := ShellDTO{
			Command:  []string{"env"},
			EnvFiles: []string{".env"},
			Env:      map[string]string{"DB_HOST": "production.example.com"},
		}
		req, err := NewShellRequest(dto, cfg, workspaceRoot, mockFS)
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}

		_, err = tool.Run(context.Background(), req)
		if err != nil {
			t.Fatalf("Run failed: %v", err)
		}

		// Request.Env should override EnvFiles
		var dbHostValue string
		for _, envVar := range capturedEnv {
			if strings.HasPrefix(envVar, "DB_HOST=") {
				dbHostValue = envVar
			}
		}

		if !strings.Contains(dbHostValue, "production.example.com") {
			t.Errorf("Expected DB_HOST=production.example.com from Request.Env, got %s", dbHostValue)
		}
	})

	t.Run("Nonexistent env file", func(t *testing.T) {
		dto := ShellDTO{Command: []string{"env"}, EnvFiles: []string{".env.missing"}}
		req, err := NewShellRequest(dto, cfg, workspaceRoot, mockFS)
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}

		tool := NewShellTool(mockFS, &mockCommandExecutorForShell{}, cfg, DockerConfig{}, workspaceRoot)
		_, err = tool.Run(context.Background(), req)
		if err == nil {
			t.Fatal("Expected error for nonexistent env file, got nil")
		}
		if !strings.Contains(err.Error(), ".env.missing") {
			t.Errorf("Expected error to mention .env.missing, got: %v", err)
		}
	})

	t.Run("Env file outside workspace", func(t *testing.T) {
		dto := ShellDTO{Command: []string{"env"}, EnvFiles: []string{"../../etc/passwd"}}
		_, err := NewShellRequest(dto, cfg, workspaceRoot, mockFS)
		if err == nil {
			t.Error("Expected error for env file outside workspace, got nil")
		}
		var ow interface{ OutsideWorkspace() bool }
		if !errors.As(err, &ow) || !ow.OutsideWorkspace() {
			t.Errorf("Expected OutsideWorkspace error from NewShellRequest, got %v", err)
		}
	})
}

func TestShellTool_Run_OutsideWorkspace(t *testing.T) {
	mockFS := newMockFileSystemForShell()
	mockFS.createDir("/workspace")
	workspaceRoot := "/workspace"
	cfg := config.DefaultConfig()

	dto := ShellDTO{Command: []string{"ls"}, WorkingDir: "../outside"}
	_, err := NewShellRequest(dto, cfg, workspaceRoot, mockFS)
	if err == nil {
		t.Fatal("Expected error from NewShellRequest")
	}
	var ow interface{ OutsideWorkspace() bool }
	if !errors.As(err, &ow) || !ow.OutsideWorkspace() {
		t.Errorf("Expected OutsideWorkspace error from NewShellRequest, got %v", err)
	}
}

func TestShellTool_Run_NonZeroExit(t *testing.T) {
	mockFS := newMockFileSystemForShell()
	mockFS.createDir("/workspace")
	workspaceRoot := "/workspace"
	cfg := config.DefaultConfig()

	factory := &mockCommandExecutorForShell{}
	factory.startFunc = func(ctx context.Context, command []string, opts ProcessOptions) (Process, io.Reader, io.Reader, error) {
		proc := &mockProcessForShell{}
		proc.waitFunc = func() error {
			return errors.New("exit status 1")
		}
		return proc, strings.NewReader(""), strings.NewReader("error output"), nil
	}

	tool := NewShellTool(mockFS, factory, cfg, DockerConfig{}, workspaceRoot)

	dto := ShellDTO{Command: []string{"false"}}
	req, err := NewShellRequest(dto, cfg, workspaceRoot, mockFS)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	resp, err := tool.Run(context.Background(), req)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if resp.ExitCode == 0 {
		t.Error("Expected non-zero exit code")
	}
}

func TestShellTool_Run_BinaryOutput(t *testing.T) {
	mockFS := newMockFileSystemForShell()
	mockFS.createDir("/workspace")
	workspaceRoot := "/workspace"
	cfg := config.DefaultConfig()

	binaryData := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE}
	factory := &mockCommandExecutorForShell{}
	factory.startFunc = func(ctx context.Context, command []string, opts ProcessOptions) (Process, io.Reader, io.Reader, error) {
		proc := &mockProcessForShell{}
		proc.waitFunc = func() error { return nil }
		return proc, bytes.NewReader(binaryData), strings.NewReader(""), nil
	}

	tool := NewShellTool(mockFS, factory, cfg, DockerConfig{}, workspaceRoot)

	dto := ShellDTO{Command: []string{"cat", "binary.bin"}}
	req, err := NewShellRequest(dto, cfg, workspaceRoot, mockFS)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	resp, err := tool.Run(context.Background(), req)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if !resp.Truncated {
		t.Error("Expected Truncated=true for binary output")
	}
}

func TestShellTool_Run_CommandInjection(t *testing.T) {
	mockFS := newMockFileSystemForShell()
	mockFS.createDir("/workspace")
	workspaceRoot := "/workspace"
	cfg := config.DefaultConfig()

	var capturedCommand []string
	factory := &mockCommandExecutorForShell{}
	factory.startFunc = func(ctx context.Context, command []string, opts ProcessOptions) (Process, io.Reader, io.Reader, error) {
		capturedCommand = command
		proc := &mockProcessForShell{}
		proc.waitFunc = func() error { return nil }
		return proc, strings.NewReader(""), strings.NewReader(""), nil
	}

	tool := NewShellTool(mockFS, factory, cfg, DockerConfig{}, workspaceRoot)

	dto := ShellDTO{Command: []string{"echo", "hello; rm -rf /"}}
	req, err := NewShellRequest(dto, cfg, workspaceRoot, mockFS)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	_, err = tool.Run(context.Background(), req)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if len(capturedCommand) != 2 {
		t.Errorf("Expected 2 command parts, got %d", len(capturedCommand))
	}
	if capturedCommand[1] != "hello; rm -rf /" {
		t.Errorf("Expected literal argument, got %q", capturedCommand[1])
	}
}

func TestShellTool_Run_HugeOutput(t *testing.T) {
	mockFS := newMockFileSystemForShell()
	mockFS.createDir("/workspace")
	workspaceRoot := "/workspace"
	cfg := config.DefaultConfig()

	hugeData := make([]byte, 50*1024*1024)
	for i := range hugeData {
		hugeData[i] = 'A'
	}

	factory := &mockCommandExecutorForShell{}
	factory.startFunc = func(ctx context.Context, command []string, opts ProcessOptions) (Process, io.Reader, io.Reader, error) {
		proc := &mockProcessForShell{}
		proc.waitFunc = func() error { return nil }
		return proc, bytes.NewReader(hugeData), strings.NewReader(""), nil
	}

	tool := NewShellTool(mockFS, factory, cfg, DockerConfig{}, workspaceRoot)

	dto := ShellDTO{Command: []string{"cat", "huge.txt"}}
	req, err := NewShellRequest(dto, cfg, workspaceRoot, mockFS)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	resp, err := tool.Run(context.Background(), req)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if !resp.Truncated {
		t.Error("Expected Truncated=true for huge output")
	}
	if len(resp.Stdout) > int(config.DefaultConfig().Tools.DefaultMaxCommandOutputSize) {
		t.Errorf("Output size %d exceeds limit %d", len(resp.Stdout), config.DefaultConfig().Tools.DefaultMaxCommandOutputSize)
	}
}

func TestShellTool_Run_Timeout(t *testing.T) {
	mockFS := newMockFileSystemForShell()
	mockFS.createDir("/workspace")
	workspaceRoot := "/workspace"
	cfg := config.DefaultConfig()

	factory := &mockCommandExecutorForShell{}
	factory.startFunc = func(ctx context.Context, command []string, opts ProcessOptions) (Process, io.Reader, io.Reader, error) {
		proc := &mockProcessForShell{}
		proc.waitFunc = func() error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(100 * time.Millisecond):
				return nil
			}
		}
		proc.signalFunc = func(sig os.Signal) error { return nil }
		proc.killFunc = func() error { return nil }
		return proc, strings.NewReader(""), strings.NewReader(""), nil
	}

	tool := NewShellTool(mockFS, factory, cfg, DockerConfig{}, workspaceRoot)

	dto := ShellDTO{Command: []string{"sleep", "10"}, TimeoutSeconds: 1}
	req, err := NewShellRequest(dto, cfg, workspaceRoot, mockFS)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	_, err = tool.Run(context.Background(), req)
	if err != nil {
		t.Errorf("Run failed: %v", err)
	}
}

func TestShellTool_Run_DockerCheck(t *testing.T) {
	mockFS := newMockFileSystemForShell()
	mockFS.createDir("/workspace")

	factory := &mockCommandExecutorForShell{}
	factory.runFunc = func(ctx context.Context, command []string) ([]byte, error) {
		// Handle Docker check command
		if len(command) >= 2 && command[0] == "docker" && command[1] == "info" {
			return []byte(""), nil
		}
		return nil, errors.New("unexpected command in RunFunc")
	}
	factory.startFunc = func(ctx context.Context, command []string, opts ProcessOptions) (Process, io.Reader, io.Reader, error) {
		if command[0] == "docker" && command[1] == "run" {
			proc := &mockProcessForShell{}
			proc.waitFunc = func() error { return nil }
			return proc, strings.NewReader("container running"), strings.NewReader(""), nil
		}
		return nil, nil, nil, errors.New("unexpected command")
	}

	dockerConfig := DockerConfig{
		CheckCommand: []string{"docker", "info"},
	}

	tool := NewShellTool(mockFS, factory, config.DefaultConfig(), dockerConfig, "/workspace")

	dto := ShellDTO{Command: []string{"docker", "run", "hello"}}
	req, err := NewShellRequest(dto, config.DefaultConfig(), "/workspace", mockFS)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	resp, err := tool.Run(context.Background(), req)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if !strings.Contains(resp.Stdout, "container running") {
		t.Errorf("Stdout = %q, want 'container running'", resp.Stdout)
	}
}

func TestShellTool_Run_EnvInjection(t *testing.T) {
	mockFS := newMockFileSystemForShell()
	mockFS.createDir("/workspace")
	workspaceRoot := "/workspace"
	cfg := config.DefaultConfig()

	var capturedEnv []string
	factory := &mockCommandExecutorForShell{}
	factory.startFunc = func(ctx context.Context, command []string, opts ProcessOptions) (Process, io.Reader, io.Reader, error) {
		capturedEnv = opts.Env
		proc := &mockProcessForShell{}
		proc.waitFunc = func() error { return nil }
		return proc, strings.NewReader(""), strings.NewReader(""), nil
	}

	tool := NewShellTool(mockFS, factory, cfg, DockerConfig{}, workspaceRoot)

	dto := ShellDTO{Command: []string{"env"}, Env: map[string]string{"PATH": ""}}
	req, err := NewShellRequest(dto, cfg, workspaceRoot, mockFS)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	_, err = tool.Run(context.Background(), req)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	hasEmptyPath := slices.Contains(capturedEnv, "PATH=")
	if !hasEmptyPath {
		t.Error("Expected PATH= in environment (empty PATH)")
	}
}

func TestShellTool_Run_ContextCancellation(t *testing.T) {
	mockFS := newMockFileSystemForShell()
	mockFS.createDir("/workspace")
	workspaceRoot := "/workspace"
	cfg := config.DefaultConfig()

	factory := &mockCommandExecutorForShell{}
	factory.startFunc = func(ctx context.Context, command []string, opts ProcessOptions) (Process, io.Reader, io.Reader, error) {
		proc := &mockProcessForShell{}
		proc.waitFunc = func() error {
			<-ctx.Done()
			return ctx.Err()
		}
		return proc, strings.NewReader(""), strings.NewReader(""), nil
	}

	tool := NewShellTool(mockFS, factory, cfg, DockerConfig{}, workspaceRoot)

	dto := ShellDTO{Command: []string{"sleep", "100"}, TimeoutSeconds: 10}
	req, err := NewShellRequest(dto, cfg, workspaceRoot, mockFS)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// Test that Run handles context cancellation gracefully and returns the error
	resp, err := tool.Run(ctx, req)
	if err == nil {
		t.Error("Expected context cancellation error, got nil")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Expected context.DeadlineExceeded, got %v", err)
	}
	if resp == nil {
		t.Error("Expected response")
	}
	if resp.ExitCode != -1 {
		t.Errorf("Expected ExitCode=-1 for cancelled context, got %d", resp.ExitCode)
	}
}

func TestShellTool_Run_SpecificExitCode(t *testing.T) {
	mockFS := newMockFileSystemForShell()
	mockFS.createDir("/workspace")
	workspaceRoot := "/workspace"
	cfg := config.DefaultConfig()

	factory := &mockCommandExecutorForShell{}
	factory.startFunc = func(ctx context.Context, command []string, opts ProcessOptions) (Process, io.Reader, io.Reader, error) {
		proc := &mockProcessForShell{}
		proc.waitFunc = func() error {
			// Return a specific exit code using the mock error type
			return newMockExitError(42)
		}
		return proc, strings.NewReader(""), strings.NewReader(""), nil
	}

	tool := NewShellTool(mockFS, factory, cfg, DockerConfig{}, workspaceRoot)

	dto := ShellDTO{Command: []string{"exit42"}}
	req, err := NewShellRequest(dto, cfg, workspaceRoot, mockFS)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	resp, err := tool.Run(context.Background(), req)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if resp.ExitCode != 42 {
		t.Errorf("ExitCode = %d, want 42", resp.ExitCode)
	}
}
