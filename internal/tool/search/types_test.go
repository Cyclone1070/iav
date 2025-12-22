package search

import (
	"errors"
	"testing"

	"github.com/Cyclone1070/iav/internal/config"
)

func TestSearchContentRequest_Validation(t *testing.T) {
	cfg := config.DefaultConfig()

	tests := []struct {
		name    string
		req     SearchContentRequest
		wantErr error
	}{
		{"Valid", SearchContentRequest{Query: "foo"}, nil},
		{"EmptyQuery", SearchContentRequest{Query: ""}, ErrQueryRequired},
		{"NegativeOffset", SearchContentRequest{Query: "foo", Offset: -1}, ErrInvalidOffset},
		{"NegativeLimit", SearchContentRequest{Query: "foo", Limit: -1}, ErrInvalidLimit},
		{"LimitExceedsMax", SearchContentRequest{Query: "foo", Limit: cfg.Tools.MaxSearchContentLimit + 1}, ErrLimitExceeded},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate(cfg)
			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("Validate() error = %v, want nil", err)
				}
			} else {
				if err == nil {
					t.Errorf("Validate() error = nil, want %v", tt.wantErr)
				} else if !errors.Is(err, tt.wantErr) {
					t.Errorf("Validate() error = %v, want %v", err, tt.wantErr)
				}
			}
		})
	}
}
