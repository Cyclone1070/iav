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

	tests := []struct {
		name    string
		req     ReadFileRequest
		wantErr bool
	}{
		{"Valid", ReadFileRequest{Path: "file.txt"}, false},
		{"EmptyPath", ReadFileRequest{Path: ""}, true},
		{"NegativeOffset", ReadFileRequest{Path: "file.txt", Offset: ptr(int64(-1))}, true},
		{"NegativeLimit", ReadFileRequest{Path: "file.txt", Limit: ptr(int64(-1))}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate(cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestWriteFileRequest_Validation(t *testing.T) {
	cfg := config.DefaultConfig()

	tests := []struct {
		name    string
		req     WriteFileRequest
		wantErr bool
	}{
		{"Valid", WriteFileRequest{Path: "file.txt", Content: "content"}, false},
		{"EmptyPath", WriteFileRequest{Path: "", Content: "content"}, true},
		{"EmptyContent", WriteFileRequest{Path: "file.txt", Content: ""}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate(cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEditFileRequest_Validation(t *testing.T) {
	cfg := config.DefaultConfig()

	tests := []struct {
		name    string
		req     EditFileRequest
		wantErr bool
	}{
		{"Valid", EditFileRequest{Path: "file.txt", Operations: []EditOperation{{Before: "old", After: "new"}}}, false},
		{"EmptyPath", EditFileRequest{Path: "", Operations: []EditOperation{{Before: "old"}}}, true},
		{"EmptyOperations", EditFileRequest{Path: "file.txt", Operations: []EditOperation{}}, true},
		{"EmptyBeforeIsAppend", EditFileRequest{Path: "file.txt", Operations: []EditOperation{{Before: ""}}}, false},
		{"NegativeReplacements", EditFileRequest{Path: "file.txt", Operations: []EditOperation{{Before: "old", ExpectedReplacements: -1}}}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate(cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Helper
func ptr[T any](v T) *T {
	return &v
}
