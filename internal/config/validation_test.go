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

func TestValidate_SearchAndFindTools(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Config)
	}{
		{"Zero_MaxLineLength_Fails", func(c *Config) { c.Tools.MaxLineLength = 0 }},
		{"Zero_MaxScanTokenSize_Fails", func(c *Config) { c.Tools.MaxScanTokenSize = 0 }},
		{"Zero_InitialScannerBufferSize_Fails", func(c *Config) { c.Tools.InitialScannerBufferSize = 0 }},
		{"Zero_MaxSearchContentResults_Fails", func(c *Config) { c.Tools.MaxSearchContentResults = 0 }},
		{"Zero_DefaultSearchContentLimit_Fails", func(c *Config) { c.Tools.DefaultSearchContentLimit = 0 }},
		{"Zero_MaxSearchContentLimit_Fails", func(c *Config) { c.Tools.MaxSearchContentLimit = 0 }},
		{"Zero_MaxFindFileResults_Fails", func(c *Config) { c.Tools.MaxFindFileResults = 0 }},
		{"Zero_DefaultFindFileLimit_Fails", func(c *Config) { c.Tools.DefaultFindFileLimit = 0 }},
		{"Zero_MaxFindFileLimit_Fails", func(c *Config) { c.Tools.MaxFindFileLimit = 0 }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			tt.mutate(cfg)
			if err := cfg.Validate(); err == nil {
				t.Error("expected validation error")
			}
		})
	}
}

func TestValidate_SemanticConstraints(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Config)
	}{
		{"DefaultListDir_ExceedsMax_Fails", func(c *Config) {
			c.Tools.DefaultListDirectoryLimit = 20000
			c.Tools.MaxListDirectoryLimit = 10000
		}},
		{"DefaultSearch_ExceedsMax_Fails", func(c *Config) {
			c.Tools.DefaultSearchContentLimit = 2000
			c.Tools.MaxSearchContentLimit = 1000
		}},
		{"DefaultFindFile_ExceedsMax_Fails", func(c *Config) {
			c.Tools.DefaultFindFileLimit = 2000
			c.Tools.MaxFindFileLimit = 1000
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			tt.mutate(cfg)
			if err := cfg.Validate(); err == nil {
				t.Error("expected validation error")
			}
		})
	}

	t.Run("Default_Equals_Max_ShouldPass", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Tools.DefaultListDirectoryLimit = 5000
		cfg.Tools.MaxListDirectoryLimit = 5000
		if err := cfg.Validate(); err != nil {
			t.Errorf("Default == Max should pass: %v", err)
		}
	})
}

func TestValidate_Colors(t *testing.T) {
	tests := []struct {
		name      string
		color     string
		wantError bool
	}{
		{"ANSI_0_Valid", "0", false},
		{"ANSI_255_Valid", "255", false},
		{"ANSI_256_Invalid", "256", true},
		{"Hex_Short_Valid", "#FFF", false},
		{"Hex_Long_Valid", "#FF5500", false},
		{"Emoji_Invalid", "🎨", true},
		{"Empty_Valid", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.UI.ColorPrimary = tt.color
			err := cfg.Validate()
			if tt.wantError && err == nil {
				t.Error("expected error")
			}
			if !tt.wantError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidate_PolicyArrays(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Policy.ShellAllow = []string{"valid", ""}
	err := cfg.Validate()
	if err == nil {
		t.Error("expected error for empty policy entry")
	}
}
