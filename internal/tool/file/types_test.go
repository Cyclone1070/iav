package file

import (
	"testing"

	"github.com/Cyclone1070/iav/internal/config"
)

func TestReadFileRequest_Validation(t *testing.T) {

	tests := []struct {
		name         string
		req          ReadFileRequest
		wantErr      bool
		verifyValues func(t *testing.T, req ReadFileRequest)
	}{
		{"Valid", ReadFileRequest{Path: "file.txt"}, false, nil},
		{"EmptyPath", ReadFileRequest{Path: ""}, true, nil},
		{"NegativeStartLine_Clamps", ReadFileRequest{Path: "file.txt", StartLine: -1}, false, func(t *testing.T, req ReadFileRequest) {
			if req.StartLine != 1 {
				t.Errorf("expected StartLine 1, got %d", req.StartLine)
			}
		}},
		{"ZeroStartLine_Clamps", ReadFileRequest{Path: "file.txt", StartLine: 0}, false, func(t *testing.T, req ReadFileRequest) {
			if req.StartLine != 1 {
				t.Errorf("expected StartLine 1, got %d", req.StartLine)
			}
		}},
		{"NegativeEndLine_Clamps", ReadFileRequest{Path: "file.txt", EndLine: -1}, false, func(t *testing.T, req ReadFileRequest) {
			if req.EndLine != 0 {
				t.Errorf("expected EndLine 0, got %d", req.EndLine)
			}
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && tt.verifyValues != nil {
				tt.verifyValues(t, tt.req)
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

	tests := []struct {
		name         string
		req          EditFileRequest
		wantErr      bool
		verifyValues func(t *testing.T, req EditFileRequest)
	}{
		{"Valid", EditFileRequest{Path: "file.txt", Operations: []EditOperation{{Before: "old", After: "new"}}}, false, nil},
		{"EmptyPath", EditFileRequest{Path: "", Operations: []EditOperation{{Before: "old"}}}, true, nil},
		{"EmptyOperations", EditFileRequest{Path: "file.txt", Operations: []EditOperation{}}, true, nil},
		{"EmptyBeforeIsAppend", EditFileRequest{Path: "file.txt", Operations: []EditOperation{{Before: ""}}}, false, nil},
		{"NegativeReplacements_Clamps", EditFileRequest{Path: "file.txt", Operations: []EditOperation{{Before: "old", ExpectedReplacements: -1}}}, false, func(t *testing.T, req EditFileRequest) {
			if req.Operations[0].ExpectedReplacements != 1 {
				t.Errorf("expected ExpectedReplacements 1, got %d", req.Operations[0].ExpectedReplacements)
			}
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && tt.verifyValues != nil {
				tt.verifyValues(t, tt.req)
			}
		})
	}
}

// Helper
func ptr[T any](v T) *T {
	return &v
}
