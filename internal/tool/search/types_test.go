package search

import (
	"testing"

	"github.com/Cyclone1070/iav/internal/config"
)

func TestSearchContentRequest_Validate(t *testing.T) {
	cfg := config.DefaultConfig()

	tests := []struct {
		name    string
		req     SearchContentRequest
		wantErr bool
	}{
		{"Valid", SearchContentRequest{Query: "foo"}, false},
		{"EmptyQuery", SearchContentRequest{Query: ""}, true},
		{"NegativeOffset", SearchContentRequest{Query: "foo", Offset: -1}, true},
		{"NegativeLimit", SearchContentRequest{Query: "foo", Limit: -1}, true},
		{"LimitExceedsMax", SearchContentRequest{Query: "foo", Limit: cfg.Tools.MaxSearchContentLimit + 1}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.req.Validate(cfg); (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
