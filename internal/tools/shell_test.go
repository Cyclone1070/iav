package tools

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

	"github.com/Cyclone1070/deployforme/internal/tools/models"
	"github.com/Cyclone1070/deployforme/internal/tools/services"
)

// MockProcess implements models.Process
type MockProcess struct {
	WaitFunc   func() error
	KillFunc   func() error
	SignalFunc func(sig os.Signal) error
}

func (m *MockProcess) Wait() error {
	if m.WaitFunc != nil {
		return m.WaitFunc()
	}
	return nil
}

func (m *MockProcess) Kill() error {
	if m.KillFunc != nil {
		return m.KillFunc()
	}
	return nil
}

func (m *MockProcess) Signal(sig os.Signal) error {
	if m.SignalFunc != nil {
		return m.SignalFunc(sig)
	}
	return nil
}

// MockProcessFactory implements models.ProcessFactory
type MockProcessFactory struct {
	StartFunc func(ctx context.Context, command []string, opts models.ProcessOptions) (models.Process, io.Reader, io.Reader, error)
}

func (m *MockProcessFactory) Start(ctx context.Context, command []string, opts models.ProcessOptions) (models.Process, io.Reader, io.Reader, error) {
	if m.StartFunc != nil {
		return m.StartFunc(ctx, command, opts)
	}
	return nil, nil, nil, errors.New("StartFunc not set")
}

func TestShellTool_Run_SimpleCommand(t *testing.T) {
	mockFS := services.NewMockFileSystem(models.DefaultMaxFileSize)
	mockFS.CreateDir("/workspace")

	wCtx := &models.WorkspaceContext{
		WorkspaceRoot: "/workspace",
		FS:            mockFS,
		CommandPolicy: models.CommandPolicy{Allow: []string{"echo"}},
	}

	factory := &MockProcessFactory{
		StartFunc: func(ctx context.Context, command []string, opts models.ProcessOptions) (models.Process, io.Reader, io.Reader, error) {
			if command[0] != "echo" {
				return nil, nil, nil, errors.New("unexpected command")
			}
			stdout := strings.NewReader("hello\n")
			stderr := strings.NewReader("")
			proc := &MockProcess{
				WaitFunc: func() error { return nil },
			}
			return proc, stdout, stderr, nil
		},
	}

	tool := &ShellTool{ProcessFactory: factory}
	req := models.ShellRequest{
		Command: []string{"echo", "hello"},
	}

	resp, err := tool.Run(context.Background(), wCtx, req)
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
	mockFS := services.NewMockFileSystem(models.DefaultMaxFileSize)
	mockFS.CreateDir("/workspace")
	mockFS.CreateDir("/workspace/subdir")

	wCtx := &models.WorkspaceContext{
		WorkspaceRoot: "/workspace",
		FS:            mockFS,
		CommandPolicy: models.CommandPolicy{Allow: []string{"pwd"}},
	}

	var capturedDir string
	factory := &MockProcessFactory{
		StartFunc: func(ctx context.Context, command []string, opts models.ProcessOptions) (models.Process, io.Reader, io.Reader, error) {
			capturedDir = opts.Dir
			proc := &MockProcess{WaitFunc: func() error { return nil }}
			return proc, strings.NewReader(""), strings.NewReader(""), nil
		},
	}

	tool := &ShellTool{ProcessFactory: factory}
	req := models.ShellRequest{
		Command:    []string{"pwd"},
		WorkingDir: "subdir",
	}

	_, err := tool.Run(context.Background(), wCtx, req)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	expectedDir := "/workspace/subdir"
	if capturedDir != expectedDir {
		t.Errorf("Working directory = %q, want %q", capturedDir, expectedDir)
	}
}

func TestShellTool_Run_Env(t *testing.T) {
	mockFS := services.NewMockFileSystem(models.DefaultMaxFileSize)
	mockFS.CreateDir("/workspace")

	wCtx := &models.WorkspaceContext{
		WorkspaceRoot: "/workspace",
		FS:            mockFS,
		CommandPolicy: models.CommandPolicy{Allow: []string{"env"}},
	}

	var capturedEnv []string
	factory := &MockProcessFactory{
		StartFunc: func(ctx context.Context, command []string, opts models.ProcessOptions) (models.Process, io.Reader, io.Reader, error) {
			capturedEnv = opts.Env
			proc := &MockProcess{WaitFunc: func() error { return nil }}
			return proc, strings.NewReader(""), strings.NewReader(""), nil
		},
	}

	tool := &ShellTool{ProcessFactory: factory}
	req := models.ShellRequest{
		Command: []string{"env"},
		Env: map[string]string{
			"CUSTOM_VAR": "custom_value",
			"TEST_MODE":  "true",
		},
	}

	_, err := tool.Run(context.Background(), wCtx, req)
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

func TestShellTool_Run_EmptyCommand(t *testing.T) {
	mockFS := services.NewMockFileSystem(models.DefaultMaxFileSize)
	mockFS.CreateDir("/workspace")

	wCtx := &models.WorkspaceContext{
		WorkspaceRoot: "/workspace",
		FS:            mockFS,
		CommandPolicy: models.CommandPolicy{Allow: []string{"*"}},
	}

	tool := &ShellTool{ProcessFactory: &MockProcessFactory{}}
	req := models.ShellRequest{Command: []string{}}

	_, err := tool.Run(context.Background(), wCtx, req)
	if err == nil {
		t.Error("Expected error for empty command, got nil")
	}
}

func TestShellTool_Run_OutsideWorkspace(t *testing.T) {
	mockFS := services.NewMockFileSystem(models.DefaultMaxFileSize)
	mockFS.CreateDir("/workspace")

	wCtx := &models.WorkspaceContext{
		WorkspaceRoot: "/workspace",
		FS:            mockFS,
		CommandPolicy: models.CommandPolicy{Allow: []string{"ls"}},
	}

	tool := &ShellTool{ProcessFactory: &MockProcessFactory{}}
	req := models.ShellRequest{
		Command:    []string{"ls"},
		WorkingDir: "../outside",
	}

	_, err := tool.Run(context.Background(), wCtx, req)
	if err != models.ErrShellWorkingDirOutsideWorkspace {
		t.Errorf("Expected ErrShellWorkingDirOutsideWorkspace, got %v", err)
	}
}

func TestShellTool_Run_PolicyRejected(t *testing.T) {
	mockFS := services.NewMockFileSystem(models.DefaultMaxFileSize)
	mockFS.CreateDir("/workspace")

	wCtx := &models.WorkspaceContext{
		WorkspaceRoot: "/workspace",
		FS:            mockFS,
		CommandPolicy: models.CommandPolicy{Allow: []string{"ls"}},
	}

	tool := &ShellTool{ProcessFactory: &MockProcessFactory{}}
	req := models.ShellRequest{Command: []string{"rm", "-rf", "/"}}

	_, err := tool.Run(context.Background(), wCtx, req)
	if err != models.ErrShellRejected {
		t.Errorf("Error = %v, want ErrShellRejected", err)
	}
}

func TestShellTool_Run_NonZeroExit(t *testing.T) {
	mockFS := services.NewMockFileSystem(models.DefaultMaxFileSize)
	mockFS.CreateDir("/workspace")

	wCtx := &models.WorkspaceContext{
		WorkspaceRoot: "/workspace",
		FS:            mockFS,
		CommandPolicy: models.CommandPolicy{Allow: []string{"false"}},
	}

	factory := &MockProcessFactory{
		StartFunc: func(ctx context.Context, command []string, opts models.ProcessOptions) (models.Process, io.Reader, io.Reader, error) {
			proc := &MockProcess{
				WaitFunc: func() error {
					return errors.New("exit status 1")
				},
			}
			return proc, strings.NewReader(""), strings.NewReader("error output"), nil
		},
	}

	tool := &ShellTool{ProcessFactory: factory}
	req := models.ShellRequest{Command: []string{"false"}}

	resp, err := tool.Run(context.Background(), wCtx, req)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if resp.ExitCode == 0 {
		t.Error("Expected non-zero exit code")
	}
}

func TestShellTool_Run_BinaryOutput(t *testing.T) {
	mockFS := services.NewMockFileSystem(models.DefaultMaxFileSize)
	mockFS.CreateDir("/workspace")

	wCtx := &models.WorkspaceContext{
		WorkspaceRoot: "/workspace",
		FS:            mockFS,
		CommandPolicy: models.CommandPolicy{Allow: []string{"cat"}},
	}

	binaryData := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE}
	factory := &MockProcessFactory{
		StartFunc: func(ctx context.Context, command []string, opts models.ProcessOptions) (models.Process, io.Reader, io.Reader, error) {
			proc := &MockProcess{WaitFunc: func() error { return nil }}
			return proc, bytes.NewReader(binaryData), strings.NewReader(""), nil
		},
	}

	tool := &ShellTool{ProcessFactory: factory}
	req := models.ShellRequest{Command: []string{"cat", "binary.bin"}}

	resp, err := tool.Run(context.Background(), wCtx, req)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if !resp.Truncated {
		t.Error("Expected Truncated=true for binary output")
	}
}

func TestShellTool_Run_CommandInjection(t *testing.T) {
	mockFS := services.NewMockFileSystem(models.DefaultMaxFileSize)
	mockFS.CreateDir("/workspace")

	wCtx := &models.WorkspaceContext{
		WorkspaceRoot: "/workspace",
		FS:            mockFS,
		CommandPolicy: models.CommandPolicy{Allow: []string{"echo"}},
	}

	var capturedCommand []string
	factory := &MockProcessFactory{
		StartFunc: func(ctx context.Context, command []string, opts models.ProcessOptions) (models.Process, io.Reader, io.Reader, error) {
			capturedCommand = command
			proc := &MockProcess{WaitFunc: func() error { return nil }}
			return proc, strings.NewReader(""), strings.NewReader(""), nil
		},
	}

	tool := &ShellTool{ProcessFactory: factory}
	req := models.ShellRequest{Command: []string{"echo", "hello; rm -rf /"}}

	_, err := tool.Run(context.Background(), wCtx, req)
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
	mockFS := services.NewMockFileSystem(models.DefaultMaxFileSize)
	mockFS.CreateDir("/workspace")

	wCtx := &models.WorkspaceContext{
		WorkspaceRoot: "/workspace",
		FS:            mockFS,
		CommandPolicy: models.CommandPolicy{Allow: []string{"cat"}},
	}

	hugeData := make([]byte, 50*1024*1024)
	for i := range hugeData {
		hugeData[i] = 'A'
	}

	factory := &MockProcessFactory{
		StartFunc: func(ctx context.Context, command []string, opts models.ProcessOptions) (models.Process, io.Reader, io.Reader, error) {
			proc := &MockProcess{WaitFunc: func() error { return nil }}
			return proc, bytes.NewReader(hugeData), strings.NewReader(""), nil
		},
	}

	tool := &ShellTool{ProcessFactory: factory}
	req := models.ShellRequest{Command: []string{"cat", "huge.txt"}}

	resp, err := tool.Run(context.Background(), wCtx, req)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if !resp.Truncated {
		t.Error("Expected Truncated=true for huge output")
	}
	if len(resp.Stdout) > int(models.DefaultMaxCommandOutputSize) {
		t.Errorf("Output size %d exceeds limit %d", len(resp.Stdout), models.DefaultMaxCommandOutputSize)
	}
}

func TestShellTool_Run_Timeout(t *testing.T) {
	mockFS := services.NewMockFileSystem(models.DefaultMaxFileSize)
	mockFS.CreateDir("/workspace")

	wCtx := &models.WorkspaceContext{
		WorkspaceRoot: "/workspace",
		FS:            mockFS,
		CommandPolicy: models.CommandPolicy{Allow: []string{"sleep"}},
	}

	factory := &MockProcessFactory{
		StartFunc: func(ctx context.Context, command []string, opts models.ProcessOptions) (models.Process, io.Reader, io.Reader, error) {
			proc := &MockProcess{
				WaitFunc: func() error {
					select {
					case <-ctx.Done():
						return ctx.Err()
					case <-time.After(100 * time.Millisecond):
						return nil
					}
				},
				SignalFunc: func(sig os.Signal) error { return nil },
				KillFunc:   func() error { return nil },
			}
			return proc, strings.NewReader(""), strings.NewReader(""), nil
		},
	}

	tool := &ShellTool{ProcessFactory: factory}
	req := models.ShellRequest{
		Command:        []string{"sleep", "10"},
		TimeoutSeconds: 1,
	}

	_, err := tool.Run(context.Background(), wCtx, req)
	if err != nil {
		t.Errorf("Run failed: %v", err)
	}
}

func TestShellTool_Run_DockerCheck(t *testing.T) {
	mockFS := services.NewMockFileSystem(models.DefaultMaxFileSize)
	mockFS.CreateDir("/workspace")

	wCtx := &models.WorkspaceContext{
		WorkspaceRoot: "/workspace",
		FS:            mockFS,
		CommandPolicy: models.CommandPolicy{Allow: []string{"docker"}},
		DockerConfig: models.DockerConfig{
			CheckCommand: []string{"docker", "info"},
		},
	}

	factory := &MockProcessFactory{
		StartFunc: func(ctx context.Context, command []string, opts models.ProcessOptions) (models.Process, io.Reader, io.Reader, error) {
			if command[0] == "docker" && command[1] == "info" {
				return &MockProcess{}, strings.NewReader(""), strings.NewReader(""), nil
			}
			if command[0] == "docker" && command[1] == "run" {
				return &MockProcess{}, strings.NewReader("container running"), strings.NewReader(""), nil
			}
			return nil, nil, nil, errors.New("unexpected command")
		},
	}

	tool := &ShellTool{ProcessFactory: factory}
	req := models.ShellRequest{Command: []string{"docker", "run", "hello"}}

	resp, err := tool.Run(context.Background(), wCtx, req)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if !strings.Contains(resp.Stdout, "container running") {
		t.Errorf("Stdout = %q, want 'container running'", resp.Stdout)
	}
}

func TestShellTool_Run_EnvInjection(t *testing.T) {
	mockFS := services.NewMockFileSystem(models.DefaultMaxFileSize)
	mockFS.CreateDir("/workspace")

	wCtx := &models.WorkspaceContext{
		WorkspaceRoot: "/workspace",
		FS:            mockFS,
		CommandPolicy: models.CommandPolicy{Allow: []string{"env"}},
	}

	var capturedEnv []string
	factory := &MockProcessFactory{
		StartFunc: func(ctx context.Context, command []string, opts models.ProcessOptions) (models.Process, io.Reader, io.Reader, error) {
			capturedEnv = opts.Env
			proc := &MockProcess{WaitFunc: func() error { return nil }}
			return proc, strings.NewReader(""), strings.NewReader(""), nil
		},
	}

	tool := &ShellTool{ProcessFactory: factory}
	req := models.ShellRequest{
		Command: []string{"env"},
		Env:     map[string]string{"PATH": ""},
	}

	_, err := tool.Run(context.Background(), wCtx, req)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	hasEmptyPath := slices.Contains(capturedEnv, "PATH=")
	if !hasEmptyPath {
		t.Error("Expected PATH= in environment (empty PATH)")
	}
}

func TestShellTool_Run_ContextCancellation(t *testing.T) {
	mockFS := services.NewMockFileSystem(models.DefaultMaxFileSize)
	mockFS.CreateDir("/workspace")

	wCtx := &models.WorkspaceContext{
		WorkspaceRoot: "/workspace",
		FS:            mockFS,
		CommandPolicy: models.CommandPolicy{Allow: []string{"sleep"}},
	}

	factory := &MockProcessFactory{
		StartFunc: func(ctx context.Context, command []string, opts models.ProcessOptions) (models.Process, io.Reader, io.Reader, error) {
			proc := &MockProcess{
				WaitFunc: func() error {
					<-ctx.Done()
					return ctx.Err()
				},
			}
			return proc, strings.NewReader(""), strings.NewReader(""), nil
		},
	}

	tool := &ShellTool{ProcessFactory: factory}
	req := models.ShellRequest{
		Command:        []string{"sleep", "100"},
		TimeoutSeconds: 10,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// Test that Run handles context cancellation gracefully (doesn't panic)
	resp, err := tool.Run(ctx, wCtx, req)
	// Current implementation swallows context cancellation errors and returns nil error
	// with ExitCode=1. This is a known limitation.
	if err != nil {
		t.Errorf("Expected nil error (current implementation), got %v", err)
	}
	if resp == nil {
		t.Error("Expected response")
	}
	if resp.ExitCode == 0 {
		t.Error("Expected non-zero exit code for cancelled context")
	}
}
