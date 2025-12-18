package file

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

func TestReadFileRequest_Validation(t *testing.T) {
	cfg := config.DefaultConfig()
	fs := &mockFSForTypes{dirs: map[string]bool{"/workspace": true}}
	workspaceRoot := "/workspace"

	tests := []struct {
		name    string
		dto     ReadFileDTO
		wantErr bool
	}{
		{"Valid", ReadFileDTO{Path: "file.txt"}, false},
		{"EmptyPath", ReadFileDTO{Path: ""}, true},
		{"NegativeOffset", ReadFileDTO{Path: "file.txt", Offset: ptr(int64(-1))}, true},
		{"NegativeLimit", ReadFileDTO{Path: "file.txt", Limit: ptr(int64(-1))}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewReadFileRequest(tt.dto, cfg, workspaceRoot, fs)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewReadFileRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestWriteFileRequest_Validation(t *testing.T) {
	cfg := config.DefaultConfig()
	fs := &mockFSForTypes{dirs: map[string]bool{"/workspace": true}}
	workspaceRoot := "/workspace"

	tests := []struct {
		name    string
		dto     WriteFileDTO
		wantErr bool
	}{
		{"Valid", WriteFileDTO{Path: "file.txt", Content: "content"}, false},
		{"EmptyPath", WriteFileDTO{Path: "", Content: "content"}, true},
		{"EmptyContent", WriteFileDTO{Path: "file.txt", Content: ""}, true},
		{"InvalidPerm", WriteFileDTO{Path: "file.txt", Content: "content", Perm: ptr(os.FileMode(07777))}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewWriteFileRequest(tt.dto, cfg, workspaceRoot, fs)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewWriteFileRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEditFileRequest_Validation(t *testing.T) {
	cfg := config.DefaultConfig()
	fs := &mockFSForTypes{dirs: map[string]bool{"/workspace": true}}
	workspaceRoot := "/workspace"

	tests := []struct {
		name    string
		dto     EditFileDTO
		wantErr bool
	}{
		{"Valid", EditFileDTO{Path: "file.txt", Operations: []Operation{{Before: "old", After: "new"}}}, false},
		{"EmptyPath", EditFileDTO{Path: "", Operations: []Operation{{Before: "old"}}}, true},
		{"EmptyOperations", EditFileDTO{Path: "file.txt", Operations: []Operation{}}, true},
		{"EmptyBefore", EditFileDTO{Path: "file.txt", Operations: []Operation{{Before: ""}}}, true},
		{"NegativeReplacements", EditFileDTO{Path: "file.txt", Operations: []Operation{{Before: "old", ExpectedReplacements: -1}}}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewEditFileRequest(tt.dto, cfg, workspaceRoot, fs)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewEditFileRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Helper
func ptr[T any](v T) *T {
	return &v
}
