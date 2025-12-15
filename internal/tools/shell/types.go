package shell

import (
	"fmt"

	"github.com/Cyclone1070/iav/internal/config"
)

// ShellRequest represents a request to execute a command on the local machine.
type ShellRequest struct {
	Command        []string          `json:"command"`
	WorkingDir     string            `json:"working_dir,omitempty"`
	TimeoutSeconds int               `json:"timeout_seconds,omitempty"`
	Env            map[string]string `json:"env,omitempty"`
	EnvFiles       []string          `json:"env_files,omitempty"` // Paths to .env files to load (relative to workspace root)
}

// Validate validates the ShellRequest
func (r ShellRequest) Validate(cfg *config.Config) error {
	if len(r.Command) == 0 {
		return fmt.Errorf("command cannot be empty")
	}
	if r.TimeoutSeconds < 0 {
		return fmt.Errorf("timeout_seconds cannot be negative")
	}
	return nil
}

// ShellResponse represents the result of a local command execution.
type ShellResponse struct {
	Stdout         string
	Stderr         string
	ExitCode       int
	Truncated      bool
	DurationMs     int64
	WorkingDir     string
	Notes          []string
	BackgroundPIDs []int
}

// DockerConfig contains configuration for Docker readiness checks.
type DockerConfig struct {
	CheckCommand []string // e.g., ["docker", "info"]
	StartCommand []string // e.g., ["docker", "desktop", "start"]
}

// ProcessOptions contains options for starting a process.
type ProcessOptions struct {
	Dir string
	Env []string
}
