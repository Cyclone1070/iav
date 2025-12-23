package config

// Config holds all application configuration values.
// Defaults are set in DefaultConfig() and can be overridden via dotfile.
// NOTE: Values in config files override defaults, including explicit zero values.
// Missing keys are left at their default values.
type Config struct {
	Orchestrator OrchestratorConfig `json:"orchestrator"`
	Provider     ProviderConfig     `json:"provider"`
	Tools        ToolsConfig        `json:"tools"`
	UI           UIConfig           `json:"ui"`
	Policy       PolicyConfig       `json:"policy"`
}

type OrchestratorConfig struct {
	MaxTurns int `json:"max_turns"` // Default: 50 - Maximum agent loop iterations
}

type ProviderConfig struct {
	FallbackMaxOutputTokens int `json:"fallback_max_output_tokens"` // Default: 8192
	FallbackContextWindow   int `json:"fallback_context_window"`    // Default: 1_000_000
	ModelTypePriorityPro    int `json:"model_type_priority_pro"`    // Default: 2
	ModelTypePriorityFlash  int `json:"model_type_priority_flash"`  // Default: 1
}

type ToolsConfig struct {
	// File Operations
	MaxFileSize int64 `json:"max_file_size"` // Default: 5 * 1024 * 1024 (5MB)

	// Directory Listing
	DefaultListDirectoryLimit int `json:"default_list_directory_limit"` // Default: 1000
	MaxListDirectoryLimit     int `json:"max_list_directory_limit"`     // Default: 10000
	MaxListDirectoryResults   int `json:"max_list_directory_results"`   // Default: 50000

	// Command Execution
	DefaultMaxCommandOutputSize int64 `json:"default_max_command_output_size"` // Default: 10 * 1024 * 1024 (10MB)
	DefaultShellTimeout         int   `json:"default_shell_timeout"`           // Default: 600 (10 minutes, in seconds)

	// Search
	MaxSearchContentResults   int `json:"max_search_content_results"`   // Default: 10000
	MaxLineLength             int `json:"max_line_length"`              // Default: 10000
	MaxScanTokenSize          int `json:"max_scan_token_size"`          // Default: 10 * 1024 * 1024 (10MB)
	InitialScannerBufferSize  int `json:"initial_scanner_buffer_size"`  // Default: 64 * 1024 (64KB)
	DefaultSearchContentLimit int `json:"default_search_content_limit"` // Default: 100
	MaxSearchContentLimit     int `json:"max_search_content_limit"`     // Default: 1000
	MaxFindFileResults        int `json:"max_find_file_results"`        // Default: 10000
	DefaultFindFileLimit      int `json:"default_find_file_limit"`      // Default: 100
	MaxFindFileLimit          int `json:"max_find_file_limit"`          // Default: 1000

	// Docker
	DockerRetryAttempts      int `json:"docker_retry_attempts"`       // Default: 10
	DockerRetryIntervalMs    int `json:"docker_retry_interval_ms"`    // Default: 1000
	DockerGracefulShutdownMs int `json:"docker_graceful_shutdown_ms"` // Default: 2000
}

type UIConfig struct {
	// Channel Buffer Sizes
	StatusChannelBuffer   int `json:"status_channel_buffer"`    // Default: 10
	MessageChannelBuffer  int `json:"message_channel_buffer"`   // Default: 10
	SetModelChannelBuffer int `json:"set_model_channel_buffer"` // Default: 10
	CommandChannelBuffer  int `json:"command_channel_buffer"`   // Default: 10

	// Animation
	TickIntervalMs    int `json:"tick_interval_ms"`    // Default: 300
	DotAnimationCycle int `json:"dot_animation_cycle"` // Default: 4

	// Layout
	ViewportHeightReserve int `json:"viewport_height_reserve"` // Default: 6
	PermissionBoxWidth    int `json:"permission_box_width"`    // Default: 60

	// Colors (lipgloss color codes)
	ColorPrimary   string `json:"color_primary"`   // Default: "63" (Blue)
	ColorSuccess   string `json:"color_success"`   // Default: "42" (Green)
	ColorError     string `json:"color_error"`     // Default: "196" (Red)
	ColorWarning   string `json:"color_warning"`   // Default: "214" (Orange)
	ColorMuted     string `json:"color_muted"`     // Default: "240" (Gray)
	ColorPurple    string `json:"color_purple"`    // Default: "141" (Purple)
	ColorAssistant string `json:"color_assistant"` // Default: "252"
}

type PolicyConfig struct {
	// Default shell commands - allow list
	ShellAllow []string `json:"shell_allow"`

	// Default shell commands - deny list
	ShellDeny []string `json:"shell_deny"`

	// Default tools - allow list
	ToolsAllow []string `json:"tools_allow"`

	// Default tools - deny list (empty by default)
	ToolsDeny []string `json:"tools_deny"`
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	return &Config{
		Orchestrator: OrchestratorConfig{
			MaxTurns: 50,
		},
		Provider: ProviderConfig{
			FallbackMaxOutputTokens: 8192,
			FallbackContextWindow:   1_000_000,
			ModelTypePriorityPro:    2,
			ModelTypePriorityFlash:  1,
		},
		Tools: ToolsConfig{
			MaxFileSize:                 5 * 1024 * 1024,
			DefaultListDirectoryLimit:   1000,
			MaxListDirectoryLimit:       10000,
			MaxListDirectoryResults:     50000,
			DefaultMaxCommandOutputSize: 10 * 1024 * 1024,
			DefaultShellTimeout:         600,
			MaxSearchContentResults:     10000,
			MaxLineLength:               10000,
			MaxScanTokenSize:            10 * 1024 * 1024,
			InitialScannerBufferSize:    64 * 1024,
			DefaultSearchContentLimit:   100,
			MaxSearchContentLimit:       1000,
			MaxFindFileResults:          10000,
			DefaultFindFileLimit:        100,
			MaxFindFileLimit:            1000,
			DockerRetryAttempts:         10,
			DockerRetryIntervalMs:       1000,
			DockerGracefulShutdownMs:    2000,
		},
		UI: UIConfig{
			StatusChannelBuffer:   10,
			MessageChannelBuffer:  10,
			SetModelChannelBuffer: 10,
			CommandChannelBuffer:  10,
			TickIntervalMs:        300,
			DotAnimationCycle:     4,
			ViewportHeightReserve: 6,
			PermissionBoxWidth:    60,
			ColorPrimary:          "63",
			ColorSuccess:          "42",
			ColorError:            "196",
			ColorWarning:          "214",
			ColorMuted:            "240",
			ColorPurple:           "141",
			ColorAssistant:        "252",
		},
		Policy: PolicyConfig{
			ShellAllow: []string{
				"docker", "docker-compose",
				"terraform", "tofu",
				"ansible", "ansible-playbook", "ansible-galaxy", "ansible-vault",
				"ls", "cat", "grep", "find", "head", "tail", "wc",
				"mkdir", "cp", "mv", "touch",
				"git", "curl", "wget",
				"make", "go", "npm", "yarn", "pip",
			},
			ShellDeny: []string{
				"sudo", "chmod", "chown",
				"shutdown", "reboot", "halt",
				"dd", "mkfs", "fdisk",
			},
			ToolsAllow: []string{
				"read_file", "list_directory", "find_file", "search_content",
			},
			ToolsDeny: []string{},
		},
	}
}
