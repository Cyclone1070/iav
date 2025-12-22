package shell

import (
	"fmt"

	"github.com/Cyclone1070/iav/internal/config"
)

// DockerConfig contains configuration for Docker readiness checks.
type DockerConfig struct {
	CheckCommand []string // e.g., ["docker", "info"]
	StartCommand []string // e.g., ["docker", "desktop", "start"]
}

// ShellRequest represents a request to execute a command on the local machine.
type ShellRequest struct {
	Command        []string          `json:"command"`
	WorkingDir     string            `json:"working_dir,omitempty"`
	TimeoutSeconds int               `json:"timeout_seconds,omitempty"`
	Env            map[string]string `json:"env,omitempty"`
	EnvFiles       []string          `json:"env_files,omitempty"` // Paths to .env files to load (relative to workspace root)
}

func (r *ShellRequest) Validate(cfg *config.Config) error {
	if len(r.Command) == 0 {
		return ErrCommandRequired
	}
	if r.TimeoutSeconds < 0 {
		return fmt.Errorf("%w: %d", ErrInvalidTimeout, r.TimeoutSeconds)
	}
	return nil
}

// ShellResponse represents the result of a local command execution.
type ShellResponse struct {
	Stdout         string   `json:"stdout"`
	Stderr         string   `json:"stderr"`
	ExitCode       int      `json:"exit_code"`
	Truncated      bool     `json:"truncated"`
	DurationMs     int64    `json:"duration_ms"`
	WorkingDir     string   `json:"working_dir"`
	Notes          []string `json:"notes,omitempty"`
	BackgroundPIDs []int    `json:"background_pids,omitempty"`
}
