package directory

import (
	"testing"

	"github.com/Cyclone1070/iav/internal/config"
)

func TestFindFileRequest_Validation(t *testing.T) {
	cfg := config.DefaultConfig()

	tests := []struct {
		name    string
		req     FindFileRequest
		wantErr bool
	}{
		{"Valid", FindFileRequest{Pattern: "*.txt"}, false},
		{"EmptyPattern", FindFileRequest{Pattern: ""}, true},
		{"NegativeOffset", FindFileRequest{Pattern: "*.txt", Offset: -1}, true},
		{"NegativeLimit", FindFileRequest{Pattern: "*.txt", Limit: -1}, true},
		{"LimitExceedsMax", FindFileRequest{Pattern: "*.txt", Limit: cfg.Tools.MaxFindFileLimit + 1}, true},
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

func TestListDirectoryRequest_Validation(t *testing.T) {
	cfg := config.DefaultConfig()

	tests := []struct {
		name    string
		req     ListDirectoryRequest
		wantErr bool
	}{
		{"Valid", ListDirectoryRequest{Path: "."}, false},
		{"EmptyPath", ListDirectoryRequest{Path: ""}, true},
		{"NegativeOffset", ListDirectoryRequest{Path: ".", Offset: -1}, true},
		{"NegativeLimit", ListDirectoryRequest{Path: ".", Limit: -1}, true},
		{"LimitExceedsMax", ListDirectoryRequest{Path: ".", Limit: cfg.Tools.MaxListDirectoryLimit + 1}, true},
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
