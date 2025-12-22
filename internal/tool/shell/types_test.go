package shell

import (
	"testing"

	"github.com/Cyclone1070/iav/internal/config"
)

func TestShellRequest_Validation(t *testing.T) {
	cfg := config.DefaultConfig()

	tests := []struct {
		name    string
		req     ShellRequest
		wantErr bool
	}{
		{"Valid", ShellRequest{Command: []string{"echo", "hello"}}, false},
		{"EmptyCommand", ShellRequest{Command: []string{}}, true},
		{"NegativeTimeout", ShellRequest{Command: []string{"echo"}, TimeoutSeconds: -1}, true},
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
