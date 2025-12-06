package config

import "fmt"

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
	if c.Tools.BinaryDetectionSampleSize < 1 {
		errs = append(errs, "tools.binary_detection_sample_size must be >= 1")
	}
	if c.Tools.DefaultListDirectoryLimit < 1 {
		errs = append(errs, "tools.default_list_directory_limit must be >= 1")
	}
	if c.Tools.MaxListDirectoryLimit < 1 {
		errs = append(errs, "tools.max_list_directory_limit must be >= 1")
	}
	if c.Tools.DefaultMaxCommandOutputSize < 1 {
		errs = append(errs, "tools.default_max_command_output_size must be >= 1")
	}
	if c.Tools.DefaultShellTimeout < 1 {
		errs = append(errs, "tools.default_shell_timeout must be >= 1")
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

	if len(errs) > 0 {
		return fmt.Errorf("config validation failed: %v", errs)
	}

	return nil
}
