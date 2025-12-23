package config

import (
	"fmt"
	"regexp"
	"strconv"
)

var hexColorRegex = regexp.MustCompile(`^#([0-9A-Fa-f]{3}|[0-9A-Fa-f]{6})$`)

func validateColor(color, fieldName string) error {
	if color == "" {
		return nil
	}
	if num, err := strconv.Atoi(color); err == nil {
		if num >= 0 && num <= 255 {
			return nil
		}
		return fmt.Errorf("%s: ANSI color must be 0-255, got %d", fieldName, num)
	}
	if hexColorRegex.MatchString(color) {
		return nil
	}
	return fmt.Errorf("%s: invalid color format %q", fieldName, color)
}

// Validate checks config values for logical correctness.
// Returns an error if any values are invalid.
func (c *Config) Validate() error {
	var errs []string

	// Orchestrator validation
	if c.Orchestrator.MaxTurns < 1 {
		errs = append(errs, "orchestrator.max_turns must be >= 1")
	}

	// Provider validation
	if c.Provider.FallbackMaxOutputTokens < 1 {
		errs = append(errs, "provider.fallback_max_output_tokens must be >= 1")
	}
	if c.Provider.FallbackContextWindow < 1 {
		errs = append(errs, "provider.fallback_context_window must be >= 1")
	}

	// Tools validation
	if c.Tools.MaxFileSize < 1 {
		errs = append(errs, "tools.max_file_size must be >= 1")
	}
	if c.Tools.DefaultListDirectoryLimit < 1 {
		errs = append(errs, "tools.default_list_directory_limit must be >= 1")
	}
	if c.Tools.MaxListDirectoryLimit < 1 {
		errs = append(errs, "tools.max_list_directory_limit must be >= 1")
	}
	if c.Tools.MaxListDirectoryResults < 1 {
		errs = append(errs, "tools.max_list_directory_results must be >= 1")
	}
	if c.Tools.DefaultMaxCommandOutputSize < 1 {
		errs = append(errs, "tools.default_max_command_output_size must be >= 1")
	}
	if c.Tools.DefaultShellTimeout < 1 {
		errs = append(errs, "tools.default_shell_timeout must be >= 1")
	}

	// Tools validation - Search & Find
	if c.Tools.MaxLineLength < 1 {
		errs = append(errs, "tools.max_line_length must be >= 1")
	}
	if c.Tools.MaxScanTokenSize < 1 {
		errs = append(errs, "tools.max_scan_token_size must be >= 1")
	}
	if c.Tools.InitialScannerBufferSize < 1 {
		errs = append(errs, "tools.initial_scanner_buffer_size must be >= 1")
	}
	if c.Tools.MaxSearchContentResults < 1 {
		errs = append(errs, "tools.max_search_content_results must be >= 1")
	}
	if c.Tools.DefaultSearchContentLimit < 1 {
		errs = append(errs, "tools.default_search_content_limit must be >= 1")
	}
	if c.Tools.MaxSearchContentLimit < 1 {
		errs = append(errs, "tools.max_search_content_limit must be >= 1")
	}
	if c.Tools.MaxFindFileResults < 1 {
		errs = append(errs, "tools.max_find_file_results must be >= 1")
	}
	if c.Tools.DefaultFindFileLimit < 1 {
		errs = append(errs, "tools.default_find_file_limit must be >= 1")
	}
	if c.Tools.MaxFindFileLimit < 1 {
		errs = append(errs, "tools.max_find_file_limit must be >= 1")
	}

	// Semantic validation: Default <= Max constraints
	if c.Tools.DefaultListDirectoryLimit > c.Tools.MaxListDirectoryLimit {
		errs = append(errs, "tools.default_list_directory_limit must be <= tools.max_list_directory_limit")
	}
	if c.Tools.DefaultSearchContentLimit > c.Tools.MaxSearchContentLimit {
		errs = append(errs, "tools.default_search_content_limit must be <= tools.max_search_content_limit")
	}
	if c.Tools.DefaultFindFileLimit > c.Tools.MaxFindFileLimit {
		errs = append(errs, "tools.default_find_file_limit must be <= tools.max_find_file_limit")
	}

	// UI validation
	if c.UI.StatusChannelBuffer < 1 {
		errs = append(errs, "ui.status_channel_buffer must be >= 1")
	}
	if c.UI.MessageChannelBuffer < 1 {
		errs = append(errs, "ui.message_channel_buffer must be >= 1")
	}
	if c.UI.SetModelChannelBuffer < 1 {
		errs = append(errs, "ui.set_model_channel_buffer must be >= 1")
	}
	if c.UI.CommandChannelBuffer < 1 {
		errs = append(errs, "ui.command_channel_buffer must be >= 1")
	}
	if c.UI.TickIntervalMs < 1 {
		errs = append(errs, "ui.tick_interval_ms must be >= 1")
	}

	// Provider priority validation
	if c.Provider.ModelTypePriorityPro < 0 {
		errs = append(errs, "provider.model_type_priority_pro must be >= 0")
	}
	if c.Provider.ModelTypePriorityFlash < 0 {
		errs = append(errs, "provider.model_type_priority_flash must be >= 0")
	}

	// UI validation - additional fields
	if c.UI.DotAnimationCycle < 1 {
		errs = append(errs, "ui.dot_animation_cycle must be >= 1")
	}
	if c.UI.ViewportHeightReserve < 0 {
		errs = append(errs, "ui.viewport_height_reserve must be >= 0")
	}
	if c.UI.PermissionBoxWidth < 1 {
		errs = append(errs, "ui.permission_box_width must be >= 1")
	}

	// Tools validation - Docker
	if c.Tools.DockerRetryAttempts < 1 {
		errs = append(errs, "tools.docker_retry_attempts must be >= 1")
	}
	if c.Tools.DockerRetryIntervalMs < 1 {
		errs = append(errs, "tools.docker_retry_interval_ms must be >= 1")
	}
	if c.Tools.DockerGracefulShutdownMs < 1 {
		errs = append(errs, "tools.docker_graceful_shutdown_ms must be >= 1")
	}

	// UI Color validation
	if err := validateColor(c.UI.ColorPrimary, "ui.color_primary"); err != nil {
		errs = append(errs, err.Error())
	}
	if err := validateColor(c.UI.ColorSuccess, "ui.color_success"); err != nil {
		errs = append(errs, err.Error())
	}
	if err := validateColor(c.UI.ColorError, "ui.color_error"); err != nil {
		errs = append(errs, err.Error())
	}
	if err := validateColor(c.UI.ColorWarning, "ui.color_warning"); err != nil {
		errs = append(errs, err.Error())
	}
	if err := validateColor(c.UI.ColorMuted, "ui.color_muted"); err != nil {
		errs = append(errs, err.Error())
	}
	if err := validateColor(c.UI.ColorPurple, "ui.color_purple"); err != nil {
		errs = append(errs, err.Error())
	}
	if err := validateColor(c.UI.ColorAssistant, "ui.color_assistant"); err != nil {
		errs = append(errs, err.Error())
	}

	// Policy validation
	for i, entry := range c.Policy.ShellAllow {
		if entry == "" {
			errs = append(errs, fmt.Sprintf("policy.shell_allow[%d]: cannot be empty", i))
		}
	}
	for i, entry := range c.Policy.ShellDeny {
		if entry == "" {
			errs = append(errs, fmt.Sprintf("policy.shell_deny[%d]: cannot be empty", i))
		}
	}
	for i, entry := range c.Policy.ToolsAllow {
		if entry == "" {
			errs = append(errs, fmt.Sprintf("policy.tools_allow[%d]: cannot be empty", i))
		}
	}
	for i, entry := range c.Policy.ToolsDeny {
		if entry == "" {
			errs = append(errs, fmt.Sprintf("policy.tools_deny[%d]: cannot be empty", i))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("config validation failed: %v", errs)
	}

	return nil
}
