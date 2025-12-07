package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidate_AllDefaults_Pass(t *testing.T) {
	cfg := DefaultConfig()
	err := cfg.Validate()
	assert.NoError(t, err)
}

func TestValidate_Orchestrator(t *testing.T) {
	t.Run("Zero MaxTurns Fails", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Orchestrator.MaxTurns = 0
		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "max_turns")
	})
}

func TestValidate_Provider(t *testing.T) {
	t.Run("Negative Context Window Fails", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Provider.FallbackContextWindow = -1
		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "fallback_context_window")
	})

	t.Run("Negative Priority Fails", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Provider.ModelTypePriorityPro = -1
		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "priority_pro")
	})
}

func TestValidate_Tools(t *testing.T) {
	t.Run("Zero File Size Fails", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Tools.MaxFileSize = 0
		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "max_file_size")
	})

	t.Run("Zero Binary Detection Sample Fails", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Tools.BinaryDetectionSampleSize = 0
		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "binary_detection_sample_size")
	})

	t.Run("Zero Docker Retry Fails", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Tools.DockerRetryAttempts = 0
		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "docker_retry_attempts")
	})
}

func TestValidate_UI(t *testing.T) {
	t.Run("Zero Tick Interval Fails", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.UI.TickIntervalMs = 0
		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "tick_interval_ms")
	})

	t.Run("Negative Viewport Reserve Fails", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.UI.ViewportHeightReserve = -1
		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "viewport_height_reserve")
	})
	
	// Zero ViewportHeightReserve is allowed (>= 0), so let's test that specifically
	t.Run("Zero Viewport Reserve Pass", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.UI.ViewportHeightReserve = 0
		err := cfg.Validate()
		assert.NoError(t, err)
	})
}

func TestValidate_MultipleErrors_ReportsAll(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Orchestrator.MaxTurns = 0
	cfg.Tools.MaxFileSize = 0
	cfg.UI.TickIntervalMs = 0
	
	err := cfg.Validate()
	assert.Error(t, err)
	
	msg := err.Error()
	assert.Contains(t, msg, "max_turns")
	assert.Contains(t, msg, "max_file_size")
	assert.Contains(t, msg, "tick_interval_ms")
}
