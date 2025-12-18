package shell

import (
	"os"
	"testing"

	"github.com/Cyclone1070/iav/internal/config"
)

// Minimal mock for validation tests
type mockFSForTypes struct {
	dirs map[string]bool
}

func (m *mockFSForTypes) Lstat(path string) (os.FileInfo, error) {
	if m.dirs[path] {
		return &mockFileInfoForTypes{isDir: true}, nil
	}
	return nil, os.ErrNotExist
}

func (m *mockFSForTypes) Readlink(path string) (string, error) {
	return "", os.ErrInvalid
}

func (m *mockFSForTypes) UserHomeDir() (string, error) {
	return "/home/user", nil
}

type mockFileInfoForTypes struct {
	os.FileInfo
	isDir bool
}

func (m *mockFileInfoForTypes) IsDir() bool { return m.isDir }

func TestShellRequest_Validation(t *testing.T) {
	cfg := config.DefaultConfig()
	fs := &mockFSForTypes{dirs: map[string]bool{"/workspace": true}}
	workspaceRoot := "/workspace"

	tests := []struct {
		name    string
		dto     ShellDTO
		wantErr bool
	}{
		{"Valid", ShellDTO{Command: []string{"echo", "hello"}}, false},
		{"EmptyCommand", ShellDTO{Command: []string{}}, true},
		{"NegativeTimeout", ShellDTO{Command: []string{"echo"}, TimeoutSeconds: -1}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewShellRequest(tt.dto, cfg, workspaceRoot, fs)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewShellRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
