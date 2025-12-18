package search

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

func TestSearchContentRequest_Validation(t *testing.T) {
	cfg := config.DefaultConfig()
	fs := &mockFSForTypes{dirs: map[string]bool{"/workspace": true}}
	workspaceRoot := "/workspace"

	tests := []struct {
		name    string
		dto     SearchContentDTO
		wantErr bool
	}{
		{"Valid", SearchContentDTO{Query: "foo"}, false},
		{"EmptyQuery", SearchContentDTO{Query: ""}, true},
		{"NegativeOffset", SearchContentDTO{Query: "foo", Offset: -1}, true},
		{"NegativeLimit", SearchContentDTO{Query: "foo", Limit: -1}, true},
		{"LimitExceedsMax", SearchContentDTO{Query: "foo", Limit: cfg.Tools.MaxSearchContentLimit + 1}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewSearchContentRequest(tt.dto, cfg, workspaceRoot, fs)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewSearchContentRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
