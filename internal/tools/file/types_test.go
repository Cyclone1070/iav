package file

import (
	"os"
	"testing"

	"github.com/Cyclone1070/iav/internal/config"
)

func TestReadFileRequest_Validate(t *testing.T) {
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
			if err := tt.req.Validate(cfg); (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestWriteFileRequest_Validate(t *testing.T) {
	cfg := config.DefaultConfig()

	tests := []struct {
		name    string
		req     WriteFileRequest
		wantErr bool
	}{
		{"Valid", WriteFileRequest{Path: "file.txt", Content: "content"}, false},
		{"EmptyPath", WriteFileRequest{Path: "", Content: "content"}, true},
		{"EmptyContent", WriteFileRequest{Path: "file.txt", Content: ""}, true},
		{"InvalidPerm", WriteFileRequest{Path: "file.txt", Content: "content", Perm: ptr(os.FileMode(07777))}, true}, // > 0777
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.req.Validate(cfg); (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEditFileRequest_Validate(t *testing.T) {
	cfg := config.DefaultConfig()

	tests := []struct {
		name    string
		req     EditFileRequest
		wantErr bool
	}{
		{"Valid", EditFileRequest{Path: "file.txt", Operations: []Operation{{Before: "old", After: "new"}}}, false},
		{"EmptyPath", EditFileRequest{Path: "", Operations: []Operation{{Before: "old"}}}, true},
		{"EmptyOperations", EditFileRequest{Path: "file.txt", Operations: []Operation{}}, true},
		{"EmptyBefore", EditFileRequest{Path: "file.txt", Operations: []Operation{{Before: ""}}}, true},
		{"NegativeReplacements", EditFileRequest{Path: "file.txt", Operations: []Operation{{Before: "old", ExpectedReplacements: -1}}}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.req.Validate(cfg); (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Helper
func ptr[T any](v T) *T {
	return &v
}
