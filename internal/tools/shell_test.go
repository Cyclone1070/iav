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

	"github.com/Cyclone1070/iav/internal/config"
	"github.com/Cyclone1070/iav/internal/tools/models"
	"github.com/Cyclone1070/iav/internal/tools/services"
	"github.com/Cyclone1070/iav/internal/testing/mocks"
)

func TestShellTool_Run_SimpleCommand(t *testing.T) {
	mockFS := mocks.NewMockFileSystem(config.DefaultConfig().Tools.MaxFileSize)
	mockFS.CreateDir("/workspace")

	wCtx := &models.WorkspaceContext{
		WorkspaceRoot:   "/workspace",
		FS:              mockFS,
		BinaryDetector:  mocks.NewMockBinaryDetector(),
		CommandExecutor: &mocks.MockCommandExecutor{},
		Config:          *config.DefaultConfig(),
	}

	factory := &mocks.MockCommandExecutor{
		StartFunc: func(ctx context.Context, command []string, opts models.ProcessOptions) (models.Process, io.Reader, io.Reader, error) {
			if command[0] != "echo" {
				return nil, nil, nil, errors.New("unexpected command")
			}
			stdout := strings.NewReader("hello\n")
			stderr := strings.NewReader("")
			proc := &mocks.MockProcess{
				WaitFunc: func() error { return nil },
			}
			return proc, stdout, stderr, nil
		},
	}

	tool := &ShellTool{CommandExecutor: factory}
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
	mockFS := mocks.NewMockFileSystem(config.DefaultConfig().Tools.MaxFileSize)
	mockFS.CreateDir("/workspace")
	mockFS.CreateDir("/workspace/subdir")

	wCtx := &models.WorkspaceContext{
		WorkspaceRoot:   "/workspace",
		FS:              mockFS,
		BinaryDetector:  mocks.NewMockBinaryDetector(),
		CommandExecutor: &mocks.MockCommandExecutor{},
		Config:          *config.DefaultConfig(),
	}

	var capturedDir string
	factory := &mocks.MockCommandExecutor{
		StartFunc: func(ctx context.Context, command []string, opts models.ProcessOptions) (models.Process, io.Reader, io.Reader, error) {
			capturedDir = opts.Dir
			proc := &mocks.MockProcess{WaitFunc: func() error { return nil }}
			return proc, strings.NewReader(""), strings.NewReader(""), nil
		},
	}

	tool := &ShellTool{CommandExecutor: factory}
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
	mockFS := mocks.NewMockFileSystem(config.DefaultConfig().Tools.MaxFileSize)
	mockFS.CreateDir("/workspace")

	wCtx := &models.WorkspaceContext{
		WorkspaceRoot:   "/workspace",
		FS:              mockFS,
		BinaryDetector:  mocks.NewMockBinaryDetector(),
		CommandExecutor: &mocks.MockCommandExecutor{},
		Config:          *config.DefaultConfig(),
	}

	var capturedEnv []string
	factory := &mocks.MockCommandExecutor{
		StartFunc: func(ctx context.Context, command []string, opts models.ProcessOptions) (models.Process, io.Reader, io.Reader, error) {
			capturedEnv = opts.Env
			proc := &mocks.MockProcess{WaitFunc: func() error { return nil }}
			return proc, strings.NewReader(""), strings.NewReader(""), nil
		},
	}

	tool := &ShellTool{CommandExecutor: factory}
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

func TestShellTool_Run_EnvFiles(t *testing.T) {
	mockFS := mocks.NewMockFileSystem(config.DefaultConfig().Tools.MaxFileSize)
	mockFS.CreateDir("/workspace")

	// Create env files
	envFile1Content := `DB_HOST=localhost
DB_PORT=5432
API_KEY=secret123`
	mockFS.CreateFile("/workspace/.env", []byte(envFile1Content), 0644)

	envFile2Content := `DB_PORT=3306
CACHE_URL=redis://localhost`
	mockFS.CreateFile("/workspace/.env.local", []byte(envFile2Content), 0644)

	wCtx := &models.WorkspaceContext{
		WorkspaceRoot:   "/workspace",
		FS:              mockFS,
		BinaryDetector:  mocks.NewMockBinaryDetector(),
		CommandExecutor: &mocks.MockCommandExecutor{},
		Config:          *config.DefaultConfig(),
	}

	var capturedEnv []string
	factory := &mocks.MockCommandExecutor{
		StartFunc: func(ctx context.Context, command []string, opts models.ProcessOptions) (models.Process, io.Reader, io.Reader, error) {
			capturedEnv = opts.Env
			proc := &mocks.MockProcess{WaitFunc: func() error { return nil }}
			return proc, strings.NewReader(""), strings.NewReader(""), nil
		},
	}

	tool := &ShellTool{CommandExecutor: factory}

	t.Run("Single env file", func(t *testing.T) {
		req := models.ShellRequest{
			Command:  []string{"env"},
			EnvFiles: []string{".env"},
		}

		_, err := tool.Run(context.Background(), wCtx, req)
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
		req := models.ShellRequest{
			Command:  []string{"env"},
			EnvFiles: []string{".env", ".env.local"},
		}

		_, err := tool.Run(context.Background(), wCtx, req)
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
		req := models.ShellRequest{
			Command:  []string{"env"},
			EnvFiles: []string{".env"},
			Env: map[string]string{
				"DB_HOST": "production.example.com",
			},
		}

		_, err := tool.Run(context.Background(), wCtx, req)
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
		req := models.ShellRequest{
			Command:  []string{"env"},
			EnvFiles: []string{".env.missing"},
		}

		_, err := tool.Run(context.Background(), wCtx, req)
		if err == nil {
			t.Error("Expected error for nonexistent env file, got nil")
		}
		if !strings.Contains(err.Error(), ".env.missing") {
			t.Errorf("Expected error to mention .env.missing, got: %v", err)
		}
	})

	t.Run("Env file outside workspace", func(t *testing.T) {
		req := models.ShellRequest{
			Command:  []string{"env"},
			EnvFiles: []string{"../../etc/passwd"},
		}

		_, err := tool.Run(context.Background(), wCtx, req)
		if err == nil {
			t.Error("Expected error for env file outside workspace, got nil")
		}
	})
}

func TestShellTool_Run_OutsideWorkspace(t *testing.T) {
	mockFS := mocks.NewMockFileSystem(config.DefaultConfig().Tools.MaxFileSize)
	mockFS.CreateDir("/workspace")

	wCtx := &models.WorkspaceContext{
		WorkspaceRoot:   "/workspace",
		FS:              mockFS,
		BinaryDetector:  mocks.NewMockBinaryDetector(),
		CommandExecutor: &mocks.MockCommandExecutor{},
		Config:          *config.DefaultConfig(),
	}

	tool := &ShellTool{CommandExecutor: &mocks.MockCommandExecutor{}}
	req := models.ShellRequest{
		Command:    []string{"ls"},
		WorkingDir: "../outside",
	}

	_, err := tool.Run(context.Background(), wCtx, req)
	if err != models.ErrShellWorkingDirOutsideWorkspace {
		t.Errorf("Expected ErrShellWorkingDirOutsideWorkspace, got %v", err)
	}
}

func TestShellTool_Run_NonZeroExit(t *testing.T) {
	mockFS := mocks.NewMockFileSystem(config.DefaultConfig().Tools.MaxFileSize)
	mockFS.CreateDir("/workspace")

	wCtx := &models.WorkspaceContext{
		WorkspaceRoot:   "/workspace",
		FS:              mockFS,
		BinaryDetector:  mocks.NewMockBinaryDetector(),
		CommandExecutor: &mocks.MockCommandExecutor{},
		Config:          *config.DefaultConfig(),
	}

	factory := &mocks.MockCommandExecutor{
		StartFunc: func(ctx context.Context, command []string, opts models.ProcessOptions) (models.Process, io.Reader, io.Reader, error) {
			proc := &mocks.MockProcess{
				WaitFunc: func() error {
					return errors.New("exit status 1")
				},
			}
			return proc, strings.NewReader(""), strings.NewReader("error output"), nil
		},
	}

	tool := &ShellTool{CommandExecutor: factory}
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
	mockFS := mocks.NewMockFileSystem(config.DefaultConfig().Tools.MaxFileSize)
	mockFS.CreateDir("/workspace")

	wCtx := &models.WorkspaceContext{
		WorkspaceRoot:   "/workspace",
		FS:              mockFS,
		BinaryDetector:  mocks.NewMockBinaryDetector(),
		CommandExecutor: &mocks.MockCommandExecutor{},
		Config:          *config.DefaultConfig(),
	}

	binaryData := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE}
	factory := &mocks.MockCommandExecutor{
		StartFunc: func(ctx context.Context, command []string, opts models.ProcessOptions) (models.Process, io.Reader, io.Reader, error) {
			proc := &mocks.MockProcess{WaitFunc: func() error { return nil }}
			return proc, bytes.NewReader(binaryData), strings.NewReader(""), nil
		},
	}

	tool := &ShellTool{CommandExecutor: factory}
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
	mockFS := mocks.NewMockFileSystem(config.DefaultConfig().Tools.MaxFileSize)
	mockFS.CreateDir("/workspace")

	wCtx := &models.WorkspaceContext{
		WorkspaceRoot:   "/workspace",
		FS:              mockFS,
		BinaryDetector:  mocks.NewMockBinaryDetector(),
		CommandExecutor: &mocks.MockCommandExecutor{},
		Config:          *config.DefaultConfig(),
	}

	var capturedCommand []string
	factory := &mocks.MockCommandExecutor{
		StartFunc: func(ctx context.Context, command []string, opts models.ProcessOptions) (models.Process, io.Reader, io.Reader, error) {
			capturedCommand = command
			proc := &mocks.MockProcess{WaitFunc: func() error { return nil }}
			return proc, strings.NewReader(""), strings.NewReader(""), nil
		},
	}

	tool := &ShellTool{CommandExecutor: factory}
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
	mockFS := mocks.NewMockFileSystem(config.DefaultConfig().Tools.MaxFileSize)
	mockFS.CreateDir("/workspace")

	wCtx := &models.WorkspaceContext{
		WorkspaceRoot:   "/workspace",
		FS:              mockFS,
		BinaryDetector:  mocks.NewMockBinaryDetector(),
		CommandExecutor: &mocks.MockCommandExecutor{},
		Config:          *config.DefaultConfig(),
	}

	hugeData := make([]byte, 50*1024*1024)
	for i := range hugeData {
		hugeData[i] = 'A'
	}

	factory := &mocks.MockCommandExecutor{
		StartFunc: func(ctx context.Context, command []string, opts models.ProcessOptions) (models.Process, io.Reader, io.Reader, error) {
			proc := &mocks.MockProcess{WaitFunc: func() error { return nil }}
			return proc, bytes.NewReader(hugeData), strings.NewReader(""), nil
		},
	}

	tool := &ShellTool{CommandExecutor: factory}
	req := models.ShellRequest{Command: []string{"cat", "huge.txt"}}

	resp, err := tool.Run(context.Background(), wCtx, req)
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
	mockFS := mocks.NewMockFileSystem(config.DefaultConfig().Tools.MaxFileSize)
	mockFS.CreateDir("/workspace")

	wCtx := &models.WorkspaceContext{
		WorkspaceRoot:   "/workspace",
		FS:              mockFS,
		BinaryDetector:  mocks.NewMockBinaryDetector(),
		CommandExecutor: &mocks.MockCommandExecutor{},
		Config:          *config.DefaultConfig(),
	}

	factory := &mocks.MockCommandExecutor{
		StartFunc: func(ctx context.Context, command []string, opts models.ProcessOptions) (models.Process, io.Reader, io.Reader, error) {
			proc := &mocks.MockProcess{
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

	tool := &ShellTool{CommandExecutor: factory}
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
	mockFS := mocks.NewMockFileSystem(config.DefaultConfig().Tools.MaxFileSize)
	mockFS.CreateDir("/workspace")

	workspaceRoot := "/workspace"
	cfg := config.DefaultConfig()
	checksumManager := services.NewChecksumManager()

	wCtx := &models.WorkspaceContext{
		WorkspaceRoot:   workspaceRoot,
		FS:              mockFS,
		BinaryDetector:  mocks.NewMockBinaryDetector(),
		ChecksumManager: checksumManager,
		CommandExecutor: &mocks.MockCommandExecutor{},
		Config:          *cfg,
		DockerConfig: models.DockerConfig{
			CheckCommand: []string{"docker", "info"},
		},
	}

	factory := &mocks.MockCommandExecutor{
		RunFunc: func(ctx context.Context, command []string) ([]byte, error) {
			// Handle Docker check command
			if len(command) >= 2 && command[0] == "docker" && command[1] == "info" {
				return []byte(""), nil
			}
			return nil, errors.New("unexpected command in RunFunc")
		},
		StartFunc: func(ctx context.Context, command []string, opts models.ProcessOptions) (models.Process, io.Reader, io.Reader, error) {
			if command[0] == "docker" && command[1] == "run" {
				return &mocks.MockProcess{}, strings.NewReader("container running"), strings.NewReader(""), nil
			}
			return nil, nil, nil, errors.New("unexpected command")
		},
	}

	tool := &ShellTool{CommandExecutor: factory}
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
	mockFS := mocks.NewMockFileSystem(config.DefaultConfig().Tools.MaxFileSize)
	mockFS.CreateDir("/workspace")

	wCtx := &models.WorkspaceContext{
		WorkspaceRoot:   "/workspace",
		FS:              mockFS,
		BinaryDetector:  mocks.NewMockBinaryDetector(),
		CommandExecutor: &mocks.MockCommandExecutor{},
		Config:          *config.DefaultConfig(),
	}

	var capturedEnv []string
	factory := &mocks.MockCommandExecutor{
		StartFunc: func(ctx context.Context, command []string, opts models.ProcessOptions) (models.Process, io.Reader, io.Reader, error) {
			capturedEnv = opts.Env
			proc := &mocks.MockProcess{WaitFunc: func() error { return nil }}
			return proc, strings.NewReader(""), strings.NewReader(""), nil
		},
	}

	tool := &ShellTool{CommandExecutor: factory}
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
	mockFS := mocks.NewMockFileSystem(config.DefaultConfig().Tools.MaxFileSize)
	mockFS.CreateDir("/workspace")

	wCtx := &models.WorkspaceContext{
		WorkspaceRoot:   "/workspace",
		FS:              mockFS,
		BinaryDetector:  mocks.NewMockBinaryDetector(),
		CommandExecutor: &mocks.MockCommandExecutor{},
		Config:          *config.DefaultConfig(),
	}

	factory := &mocks.MockCommandExecutor{
		StartFunc: func(ctx context.Context, command []string, opts models.ProcessOptions) (models.Process, io.Reader, io.Reader, error) {
			proc := &mocks.MockProcess{
				WaitFunc: func() error {
					<-ctx.Done()
					return ctx.Err()
				},
			}
			return proc, strings.NewReader(""), strings.NewReader(""), nil
		},
	}

	tool := &ShellTool{CommandExecutor: factory}
	req := models.ShellRequest{
		Command:        []string{"sleep", "100"},
		TimeoutSeconds: 10,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// Test that Run handles context cancellation gracefully and returns the error
	resp, err := tool.Run(ctx, wCtx, req)
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
	mockFS := mocks.NewMockFileSystem(config.DefaultConfig().Tools.MaxFileSize)
	mockFS.CreateDir("/workspace")

	wCtx := &models.WorkspaceContext{
		WorkspaceRoot:   "/workspace",
		FS:              mockFS,
		BinaryDetector:  mocks.NewMockBinaryDetector(),
		CommandExecutor: &mocks.MockCommandExecutor{},
		Config:          *config.DefaultConfig(),
	}

	factory := &mocks.MockCommandExecutor{
		StartFunc: func(ctx context.Context, command []string, opts models.ProcessOptions) (models.Process, io.Reader, io.Reader, error) {
			proc := &mocks.MockProcess{
				WaitFunc: func() error {
					// Return a specific exit code using the mock error type
					return &mocks.MockExitError{Code: 42}
				},
			}
			return proc, strings.NewReader(""), strings.NewReader(""), nil
		},
	}

	tool := &ShellTool{CommandExecutor: factory}
	req := models.ShellRequest{Command: []string{"exit42"}}

	resp, err := tool.Run(context.Background(), wCtx, req)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if resp.ExitCode != 42 {
		t.Errorf("ExitCode = %d, want 42", resp.ExitCode)
	}
}
