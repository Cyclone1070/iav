package search

import (
	"testing"

	"github.com/Cyclone1070/iav/internal/config"
)

func TestSearchContentRequest_Validation(t *testing.T) {
	cfg := config.DefaultConfig()

	tests := []struct {
		name         string
		req          SearchContentRequest
		wantErr      bool
		verifyValues func(t *testing.T, req SearchContentRequest)
	}{
		{"Valid", SearchContentRequest{Query: "foo"}, false, nil},
		{"EmptyQuery", SearchContentRequest{Query: ""}, true, nil},
		{"NegativeOffset_Clamps", SearchContentRequest{Query: "foo", Offset: -1}, false, func(t *testing.T, req SearchContentRequest) {
			if req.Offset != 0 {
				t.Errorf("expected offset 0, got %d", req.Offset)
			}
		}},
		{"NegativeLimit_Defaults", SearchContentRequest{Query: "foo", Limit: -1}, false, func(t *testing.T, req SearchContentRequest) {
			if req.Limit != cfg.Tools.DefaultSearchContentLimit {
				t.Errorf("expected default limit %d, got %d", cfg.Tools.DefaultSearchContentLimit, req.Limit)
			}
		}},
		{"LimitExceedsMax_Caps", SearchContentRequest{Query: "foo", Limit: cfg.Tools.MaxSearchContentLimit + 1}, false, func(t *testing.T, req SearchContentRequest) {
			if req.Limit != cfg.Tools.MaxSearchContentLimit {
				t.Errorf("expected max limit %d, got %d", cfg.Tools.MaxSearchContentLimit, req.Limit)
			}
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate(cfg)
			if tt.wantErr {
				if err == nil {
					t.Error("Validate() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Validate() unexpected error: %v", err)
				}
			}
			if err == nil && tt.verifyValues != nil {
				tt.verifyValues(t, tt.req)
			}
		})
	}
}
