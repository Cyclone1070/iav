package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const (
	// ConfigDir is the directory name under ~/.config
	ConfigDir = "iav"
	// ConfigFile is the config file name
	ConfigFile = "config.json"
)

// FileSystem abstracts file operations for testability
type FileSystem interface {
	UserHomeDir() (string, error)
	ReadFile(path string) ([]byte, error)
}

// OSFileSystem implements FileSystem using the real OS
type OSFileSystem struct{}

func (OSFileSystem) UserHomeDir() (string, error) {
	return os.UserHomeDir()
}

func (OSFileSystem) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// Loader handles configuration loading with injected dependencies
type Loader struct {
	fs FileSystem
}

// NewLoader creates a production Loader using the real filesystem
func NewLoader() *Loader {
	return &Loader{fs: OSFileSystem{}}
}

// NewLoaderWithFS creates a Loader with a custom filesystem (for testing)
func NewLoaderWithFS(fs FileSystem) *Loader {
	return &Loader{fs: fs}
}

// Load reads configuration from ~/.config/iav/config.json
// and merges it with defaults. Dotfile values override defaults.
// Returns default config if dotfile doesn't exist.
// Returns error only for parse errors, permission issues, or validation failures.
//
// LIMITATION: Zero values (0, false, "") in the config file are treated as "not set"
// and will NOT override defaults. To disable a feature, use validation-passing minimum
// values (e.g., 1 instead of 0). This is a known limitation of the merge strategy.
func (l *Loader) Load() (*Config, error) {
	cfg := DefaultConfig()

	homeDir, err := l.fs.UserHomeDir()
	if err != nil {
		return cfg, nil // Use defaults if can't get home dir
	}

	configPath := filepath.Join(homeDir, ".config", ConfigDir, ConfigFile)

	data, err := l.fs.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil // Use defaults if file doesn't exist
		}
		return nil, err // Return error for permission issues
	}

	// Parse JSON
	var fileConfig Config
	if err := json.Unmarshal(data, &fileConfig); err != nil {
		return nil, err // Return error for malformed JSON
	}

	// Merge (fileConfig overrides default cfg)
	mergeConfig(cfg, &fileConfig)

	// Validate the merged configuration
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// mergeConfig merges src into dst, overriding non-zero values
func mergeConfig(dst, src *Config) {
	// Orchestrator
	if src.Orchestrator.MaxTurns != 0 {
		dst.Orchestrator.MaxTurns = src.Orchestrator.MaxTurns
	}

	// Provider
	if src.Provider.FallbackMaxOutputTokens != 0 {
		dst.Provider.FallbackMaxOutputTokens = src.Provider.FallbackMaxOutputTokens
	}
	if src.Provider.FallbackContextWindow != 0 {
		dst.Provider.FallbackContextWindow = src.Provider.FallbackContextWindow
	}
	if src.Provider.ModelTypePriorityPro != 0 {
		dst.Provider.ModelTypePriorityPro = src.Provider.ModelTypePriorityPro
	}
	if src.Provider.ModelTypePriorityFlash != 0 {
		dst.Provider.ModelTypePriorityFlash = src.Provider.ModelTypePriorityFlash
	}

	// Tools
	if src.Tools.MaxFileSize != 0 {
		dst.Tools.MaxFileSize = src.Tools.MaxFileSize
	}
	if src.Tools.BinaryDetectionSampleSize != 0 {
		dst.Tools.BinaryDetectionSampleSize = src.Tools.BinaryDetectionSampleSize
	}
	if src.Tools.DefaultListDirectoryLimit != 0 {
		dst.Tools.DefaultListDirectoryLimit = src.Tools.DefaultListDirectoryLimit
	}
	if src.Tools.MaxListDirectoryLimit != 0 {
		dst.Tools.MaxListDirectoryLimit = src.Tools.MaxListDirectoryLimit
	}
	if src.Tools.DefaultMaxCommandOutputSize != 0 {
		dst.Tools.DefaultMaxCommandOutputSize = src.Tools.DefaultMaxCommandOutputSize
	}
	if src.Tools.DefaultShellTimeout != 0 {
		dst.Tools.DefaultShellTimeout = src.Tools.DefaultShellTimeout
	}
	if src.Tools.MaxSearchContentResults != 0 {
		dst.Tools.MaxSearchContentResults = src.Tools.MaxSearchContentResults
	}
	if src.Tools.MaxLineLength != 0 {
		dst.Tools.MaxLineLength = src.Tools.MaxLineLength
	}
	if src.Tools.MaxScanTokenSize != 0 {
		dst.Tools.MaxScanTokenSize = src.Tools.MaxScanTokenSize
	}
	if src.Tools.MaxFindFileResults != 0 {
		dst.Tools.MaxFindFileResults = src.Tools.MaxFindFileResults
	}
	if src.Tools.DockerRetryAttempts != 0 {
		dst.Tools.DockerRetryAttempts = src.Tools.DockerRetryAttempts
	}
	if src.Tools.DockerRetryIntervalMs != 0 {
		dst.Tools.DockerRetryIntervalMs = src.Tools.DockerRetryIntervalMs
	}
	if src.Tools.DockerGracefulShutdownMs != 0 {
		dst.Tools.DockerGracefulShutdownMs = src.Tools.DockerGracefulShutdownMs
	}

	// UI
	if src.UI.StatusChannelBuffer != 0 {
		dst.UI.StatusChannelBuffer = src.UI.StatusChannelBuffer
	}
	if src.UI.MessageChannelBuffer != 0 {
		dst.UI.MessageChannelBuffer = src.UI.MessageChannelBuffer
	}
	if src.UI.SetModelChannelBuffer != 0 {
		dst.UI.SetModelChannelBuffer = src.UI.SetModelChannelBuffer
	}
	if src.UI.CommandChannelBuffer != 0 {
		dst.UI.CommandChannelBuffer = src.UI.CommandChannelBuffer
	}
	if src.UI.TickIntervalMs != 0 {
		dst.UI.TickIntervalMs = src.UI.TickIntervalMs
	}
	if src.UI.DotAnimationCycle != 0 {
		dst.UI.DotAnimationCycle = src.UI.DotAnimationCycle
	}
	if src.UI.ViewportHeightReserve != 0 {
		dst.UI.ViewportHeightReserve = src.UI.ViewportHeightReserve
	}
	if src.UI.PermissionBoxWidth != 0 {
		dst.UI.PermissionBoxWidth = src.UI.PermissionBoxWidth
	}
	if src.UI.ColorPrimary != "" {
		dst.UI.ColorPrimary = src.UI.ColorPrimary
	}
	if src.UI.ColorSuccess != "" {
		dst.UI.ColorSuccess = src.UI.ColorSuccess
	}
	if src.UI.ColorError != "" {
		dst.UI.ColorError = src.UI.ColorError
	}
	if src.UI.ColorWarning != "" {
		dst.UI.ColorWarning = src.UI.ColorWarning
	}
	if src.UI.ColorMuted != "" {
		dst.UI.ColorMuted = src.UI.ColorMuted
	}
	if src.UI.ColorPurple != "" {
		dst.UI.ColorPurple = src.UI.ColorPurple
	}
	if src.UI.ColorAssistant != "" {
		dst.UI.ColorAssistant = src.UI.ColorAssistant
	}

	// Policy - arrays replace entirely if non-nil
	if src.Policy.ShellAllow != nil {
		dst.Policy.ShellAllow = src.Policy.ShellAllow
	}
	if src.Policy.ShellDeny != nil {
		dst.Policy.ShellDeny = src.Policy.ShellDeny
	}
	if src.Policy.ToolsAllow != nil {
		dst.Policy.ToolsAllow = src.Policy.ToolsAllow
	}
	if src.Policy.ToolsDeny != nil {
		dst.Policy.ToolsDeny = src.Policy.ToolsDeny
	}
}

// Load is a convenience function using the default loader
func Load() (*Config, error) {
	return NewLoader().Load()
}
