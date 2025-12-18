package shell

import (
	"os"

	"github.com/Cyclone1070/iav/internal/config"
	"github.com/Cyclone1070/iav/internal/tool/pathutil"
)

// ShellDTO is the wire format for shell command execution
type ShellDTO struct {
	Command        []string          `json:"command"`
	WorkingDir     string            `json:"working_dir,omitempty"`
	TimeoutSeconds int               `json:"timeout_seconds,omitempty"`
	Env            map[string]string `json:"env,omitempty"`
	EnvFiles       []string          `json:"env_files,omitempty"` // Paths to .env files to load (relative to workspace root)
}

// ShellRequest represents a validated request to execute a command on the local machine.
type ShellRequest struct {
	command          []string
	workingDirAbs    string
	workingDirRel    string
	timeoutSeconds   int
	env              map[string]string
	envFilesAbsPaths []string // Resolved absolute paths
}

// NewShellRequest creates a validated ShellRequest from a DTO
func NewShellRequest(
	dto ShellDTO,
	cfg *config.Config,
	workspaceRoot string,
	fs interface {
		Lstat(path string) (os.FileInfo, error)
		Readlink(path string) (string, error)
		UserHomeDir() (string, error)
	},
) (*ShellRequest, error) {
	// Constructor validation
	if len(dto.Command) == 0 {
		return nil, &CommandRequiredError{}
	}
	if dto.TimeoutSeconds < 0 {
		return nil, &NegativeTimeoutError{Value: dto.TimeoutSeconds}
	}

	// WorkingDir defaults to "." if empty
	workingDir := dto.WorkingDir
	if workingDir == "" {
		workingDir = "."
	}

	// Path resolution for working directory
	wdAbs, wdRel, err := resolvePathWithFS(workspaceRoot, fs, workingDir)
	if err != nil {
		return nil, err
	}

	// Path resolution for env files
	var envFilesAbs []string
	for _, envFile := range dto.EnvFiles {
		envFileAbs, _, err := resolvePathWithFS(workspaceRoot, fs, envFile)
		if err != nil {
			return nil, err
		}
		envFilesAbs = append(envFilesAbs, envFileAbs)
	}

	return &ShellRequest{
		command:          dto.Command,
		workingDirAbs:    wdAbs,
		workingDirRel:    wdRel,
		timeoutSeconds:   dto.TimeoutSeconds,
		env:              dto.Env,
		envFilesAbsPaths: envFilesAbs,
	}, nil
}

// Command returns the command to execute
func (r *ShellRequest) Command() []string {
	return r.command
}

// WorkingDirAbs returns the absolute working directory
func (r *ShellRequest) WorkingDirAbs() string {
	return r.workingDirAbs
}

// WorkingDirRel returns the relative working directory
func (r *ShellRequest) WorkingDirRel() string {
	return r.workingDirRel
}

// TimeoutSeconds returns the timeout in seconds
func (r *ShellRequest) TimeoutSeconds() int {
	return r.timeoutSeconds
}

// Env returns the environment variables
func (r *ShellRequest) Env() map[string]string {
	return r.env
}

// EnvFilesAbsPaths returns the resolved absolute paths to env files
func (r *ShellRequest) EnvFilesAbsPaths() []string {
	return r.envFilesAbsPaths
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

// resolvePathWithFS is a helper that calls pathutil.Resolve with the given filesystem
func resolvePathWithFS(
	workspaceRoot string,
	fs interface {
		Lstat(path string) (os.FileInfo, error)
		Readlink(path string) (string, error)
		UserHomeDir() (string, error)
	},
	path string,
) (string, string, error) {
	// Cast to pathutil.FileSystem (the interface is identical)
	fsImpl := fs.(pathutil.FileSystem)
	return pathutil.Resolve(workspaceRoot, fsImpl, path)
}
