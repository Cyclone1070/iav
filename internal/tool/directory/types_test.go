package directory

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

func TestFindFileRequest_Validation(t *testing.T) {
	cfg := config.DefaultConfig()
	fs := &mockFSForTypes{dirs: map[string]bool{"/workspace": true}}
	workspaceRoot := "/workspace"

	tests := []struct {
		name    string
		dto     FindFileDTO
		wantErr bool
	}{
		{"Valid", FindFileDTO{Pattern: "*.txt"}, false},
		{"EmptyPattern", FindFileDTO{Pattern: ""}, true},
		{"PathTraversalInPattern", FindFileDTO{Pattern: "../outside"}, true},
		{"AbsolutePathInPattern", FindFileDTO{Pattern: "/etc/passwd"}, true},
		{"NegativeOffset", FindFileDTO{Pattern: "*.txt", Offset: -1}, true},
		{"NegativeLimit", FindFileDTO{Pattern: "*.txt", Limit: -1}, true},
		{"LimitExceedsMax", FindFileDTO{Pattern: "*.txt", Limit: cfg.Tools.MaxFindFileLimit + 1}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewFindFileRequest(tt.dto, cfg, workspaceRoot, fs)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewFindFileRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestListDirectoryRequest_Validation(t *testing.T) {
	cfg := config.DefaultConfig()
	fs := &mockFSForTypes{dirs: map[string]bool{"/workspace": true}}
	workspaceRoot := "/workspace"

	tests := []struct {
		name    string
		dto     ListDirectoryDTO
		wantErr bool
	}{
		{"Valid", ListDirectoryDTO{Path: "."}, false},
		{"EmptyPath", ListDirectoryDTO{Path: ""}, false}, // path defaults to .
		{"NegativeOffset", ListDirectoryDTO{Path: ".", Offset: -1}, true},
		{"NegativeLimit", ListDirectoryDTO{Path: ".", Limit: -1}, true},
		{"LimitExceedsMax", ListDirectoryDTO{Path: ".", Limit: cfg.Tools.MaxListDirectoryLimit + 1}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewListDirectoryRequest(tt.dto, cfg, workspaceRoot, fs)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewListDirectoryRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
