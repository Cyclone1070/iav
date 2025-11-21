package services

import (
	"testing"

	"github.com/Cyclone1070/deployforme/internal/tools/models"
)

func TestEvaluatePolicy(t *testing.T) {
	tests := []struct {
		name    string
		policy  models.CommandPolicy
		command []string
		wantErr error
	}{
		{
			name:    "Allowed command",
			policy:  models.CommandPolicy{Allow: []string{"echo"}},
			command: []string{"echo", "hello"},
			wantErr: nil,
		},
		{
			name:    "Allowed via SessionAllow",
			policy:  models.CommandPolicy{SessionAllow: map[string]bool{"rm": true}},
			command: []string{"rm", "-rf", "/"},
			wantErr: nil,
		},
		{
			name:    "Ask list - Approval Required",
			policy:  models.CommandPolicy{Ask: []string{"deploy"}},
			command: []string{"deploy", "prod"},
			wantErr: models.ErrShellApprovalRequired,
		},
		{
			name:    "Ask list - Session Allowed",
			policy:  models.CommandPolicy{Ask: []string{"deploy"}, SessionAllow: map[string]bool{"deploy": true}},
			command: []string{"deploy", "prod"},
			wantErr: nil,
		},
		{
			name:    "Default Deny",
			policy:  models.CommandPolicy{},
			command: []string{"unknown"},
			wantErr: models.ErrShellRejected,
		},
		{
			name:    "Empty Command",
			policy:  models.CommandPolicy{},
			command: []string{},
			wantErr: models.ErrShellRejected, // Or specific error if parser fails first
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := EvaluatePolicy(tt.policy, tt.command)
			if err != tt.wantErr {
				t.Errorf("EvaluatePolicy() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Command Parser Tests (merged from command_parser_test.go)

func TestGetCommandRoot(t *testing.T) {
	tests := []struct {
		cmd  []string
		want string
	}{
		{[]string{"docker", "run"}, "docker"},
		{[]string{"/usr/bin/ls", "-la"}, "ls"},
		{[]string{"./script.sh"}, "script.sh"},
		{[]string{}, ""},
	}

	for _, tt := range tests {
		if got := GetCommandRoot(tt.cmd); got != tt.want {
			t.Errorf("GetCommandRoot(%v) = %q, want %q", tt.cmd, got, tt.want)
		}
	}
}

func TestIsDockerCommand(t *testing.T) {
	tests := []struct {
		cmd  []string
		want bool
	}{
		{[]string{"docker", "ps"}, true},
		{[]string{"/usr/bin/docker", "run"}, true},
		{[]string{"dockerd"}, false},
		{[]string{"echo", "docker"}, false},
	}

	for _, tt := range tests {
		if got := IsDockerCommand(tt.cmd); got != tt.want {
			t.Errorf("IsDockerCommand(%v) = %v, want %v", tt.cmd, got, tt.want)
		}
	}
}

func TestIsDockerComposeUpDetached(t *testing.T) {
	tests := []struct {
		cmd  []string
		want bool
	}{
		{[]string{"docker", "compose", "up", "-d"}, true},
		{[]string{"docker", "compose", "up", "--detach"}, true},
		{[]string{"docker", "compose", "up"}, false},
		{[]string{"docker", "run", "-d"}, false},
	}

	for _, tt := range tests {
		if got := IsDockerComposeUpDetached(tt.cmd); got != tt.want {
			t.Errorf("IsDockerComposeUpDetached(%v) = %v, want %v", tt.cmd, got, tt.want)
		}
	}
}

func TestParse_ComplexCases(t *testing.T) {
	// TestParse_MixedFlags: Root is still "docker"
	cmd := []string{"docker", "-H", "tcp://1.2.3.4:2375", "run"}
	if got := GetCommandRoot(cmd); got != "docker" {
		t.Errorf("MixedFlags root = %q, want %q", got, "docker")
	}

	// TestParse_QuotedArgs: Root is "sh", NOT "docker"
	// Note: The parser receives the slice already split by the shell or caller.
	// If the caller passes ["sh", "-c", "docker run"], root is "sh".
	cmd2 := []string{"sh", "-c", "docker run"}
	if got := GetCommandRoot(cmd2); got != "sh" {
		t.Errorf("QuotedArgs root = %q, want %q", got, "sh")
	}
	if IsDockerCommand(cmd2) {
		t.Error("QuotedArgs IsDockerCommand = true, want false")
	}
}
