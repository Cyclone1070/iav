package config

import (
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockFileSystem implements FileSystem for testing.
// NOTE: This is a minimal mock for config loading tests.
// For comprehensive filesystem mocking, see internal/tools/services/test_mocks.go.
type MockFileSystem struct {
	HomeDir     string
	HomeDirErr  error
	Files       map[string][]byte
	ReadFileErr error
}

func (m *MockFileSystem) UserHomeDir() (string, error) {
	return m.HomeDir, m.HomeDirErr
}

func (m *MockFileSystem) ReadFile(path string) ([]byte, error) {
	if m.ReadFileErr != nil {
		return nil, m.ReadFileErr
	}
	data, ok := m.Files[path]
	if !ok {
		return nil, os.ErrNotExist
	}
	return data, nil
}

// --- HAPPY PATH TESTS ---

func TestLoad_NoConfigFile_ReturnsDefaults(t *testing.T) {
	// Config file doesn't exist - should return all defaults
	fs := &MockFileSystem{
		HomeDir: "/home/user",
		Files:   map[string][]byte{}, // Empty - no config file
	}
	loader := NewLoaderWithFS(fs)

	cfg, err := loader.Load()

	require.NoError(t, err)
	assert.Equal(t, 50, cfg.Orchestrator.MaxTurns)
	assert.Equal(t, 8192, cfg.Provider.FallbackMaxOutputTokens)
	assert.Equal(t, int64(5*1024*1024), cfg.Tools.MaxFileSize)
}

func TestLoad_FullOverride_AllValuesReplaced(t *testing.T) {
	// Config file overrides every single field
	configJSON := `{
		"orchestrator": {"max_turns": 100},
		"provider": {"fallback_max_output_tokens": 16384, "fallback_context_window": 2000000},
		"tools": {"max_file_size": 10485760, "default_shell_timeout": 1800},
		"ui": {"tick_interval_ms": 200, "color_primary": "99"},
		"policy": {"shell_allow": ["docker"], "shell_deny": ["rm"]}
	}`
	fs := &MockFileSystem{
		HomeDir: "/home/user",
		Files: map[string][]byte{
			"/home/user/.config/iav/config.json": []byte(configJSON),
		},
	}
	loader := NewLoaderWithFS(fs)

	cfg, err := loader.Load()

	require.NoError(t, err)
	assert.Equal(t, 100, cfg.Orchestrator.MaxTurns)
	assert.Equal(t, 16384, cfg.Provider.FallbackMaxOutputTokens)
	assert.Equal(t, int64(10485760), cfg.Tools.MaxFileSize)
	assert.Equal(t, 200, cfg.UI.TickIntervalMs)
	assert.Equal(t, []string{"docker"}, cfg.Policy.ShellAllow)
}

func TestLoad_PartialOverride_MergesWithDefaults(t *testing.T) {
	// Config file only overrides max_turns - rest should be defaults
	configJSON := `{"orchestrator": {"max_turns": 200}}`
	fs := &MockFileSystem{
		HomeDir: "/home/user",
		Files: map[string][]byte{
			"/home/user/.config/iav/config.json": []byte(configJSON),
		},
	}
	loader := NewLoaderWithFS(fs)

	cfg, err := loader.Load()

	require.NoError(t, err)
	assert.Equal(t, 200, cfg.Orchestrator.MaxTurns)             // Overridden
	assert.Equal(t, 8192, cfg.Provider.FallbackMaxOutputTokens) // Default
	assert.Equal(t, int64(5*1024*1024), cfg.Tools.MaxFileSize)  // Default
	assert.Contains(t, cfg.Policy.ShellAllow, "docker")         // Default list
}

func TestLoad_EmptyConfigFile_ReturnsDefaults(t *testing.T) {
	// Empty JSON object - should use all defaults
	fs := &MockFileSystem{
		HomeDir: "/home/user",
		Files: map[string][]byte{
			"/home/user/.config/iav/config.json": []byte(`{}`),
		},
	}
	loader := NewLoaderWithFS(fs)

	cfg, err := loader.Load()

	require.NoError(t, err)
	assert.Equal(t, 50, cfg.Orchestrator.MaxTurns)
}

func TestLoad_NestedPartialOverride_OnlySpecifiedFieldsChange(t *testing.T) {
	// Override only one field in a nested struct
	configJSON := `{"ui": {"color_primary": "255"}}`
	fs := &MockFileSystem{
		HomeDir: "/home/user",
		Files: map[string][]byte{
			"/home/user/.config/iav/config.json": []byte(configJSON),
		},
	}
	loader := NewLoaderWithFS(fs)

	cfg, err := loader.Load()

	require.NoError(t, err)
	assert.Equal(t, "255", cfg.UI.ColorPrimary) // Overridden
	assert.Equal(t, "42", cfg.UI.ColorSuccess)  // Default preserved
	assert.Equal(t, 300, cfg.UI.TickIntervalMs) // Default preserved
}

// --- UNHAPPY PATH TESTS ---

func TestLoad_MalformedJSON_ReturnsError(t *testing.T) {
	fs := &MockFileSystem{
		HomeDir: "/home/user",
		Files: map[string][]byte{
			"/home/user/.config/iav/config.json": []byte(`{invalid json`),
		},
	}
	loader := NewLoaderWithFS(fs)

	cfg, err := loader.Load()

	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "invalid")
}

func TestLoad_PermissionDenied_ReturnsError(t *testing.T) {
	fs := &MockFileSystem{
		HomeDir:     "/home/user",
		ReadFileErr: os.ErrPermission,
	}
	loader := NewLoaderWithFS(fs)

	cfg, err := loader.Load()

	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.True(t, errors.Is(err, os.ErrPermission))
}

func TestLoad_HomeDirError_ReturnsDefaults(t *testing.T) {
	// Can't determine home dir - gracefully fall back to defaults
	fs := &MockFileSystem{
		HomeDirErr: errors.New("homeless"),
	}
	loader := NewLoaderWithFS(fs)

	cfg, err := loader.Load()

	require.NoError(t, err)
	assert.Equal(t, 50, cfg.Orchestrator.MaxTurns) // Default
}

func TestLoad_WrongJSONType_ReturnsError(t *testing.T) {
	// JSON is valid but wrong type (array instead of object)
	fs := &MockFileSystem{
		HomeDir: "/home/user",
		Files: map[string][]byte{
			"/home/user/.config/iav/config.json": []byte(`["not", "an", "object"]`),
		},
	}
	loader := NewLoaderWithFS(fs)

	cfg, err := loader.Load()

	assert.Error(t, err)
	assert.Nil(t, cfg)
}

// --- EDGE CASE TESTS ---

func TestLoad_ZeroValueExplicit_DoesNotOverride(t *testing.T) {
	// Setting max_turns to 0 should NOT override (0 is zero-value)
	// This is a known limitation of the merge strategy
	configJSON := `{"orchestrator": {"max_turns": 0}}`
	fs := &MockFileSystem{
		HomeDir: "/home/user",
		Files: map[string][]byte{
			"/home/user/.config/iav/config.json": []byte(configJSON),
		},
	}
	loader := NewLoaderWithFS(fs)

	cfg, err := loader.Load()

	require.NoError(t, err)
	assert.Equal(t, 50, cfg.Orchestrator.MaxTurns) // Default kept (0 ignored)
}

func TestLoad_EmptyStringColor_DoesNotOverride(t *testing.T) {
	// Empty string should not override default color
	configJSON := `{"ui": {"color_primary": ""}}`
	fs := &MockFileSystem{
		HomeDir: "/home/user",
		Files: map[string][]byte{
			"/home/user/.config/iav/config.json": []byte(configJSON),
		},
	}
	loader := NewLoaderWithFS(fs)

	cfg, err := loader.Load()

	require.NoError(t, err)
	assert.Equal(t, "63", cfg.UI.ColorPrimary) // Default kept
}

func TestLoad_EmptyPolicyArray_ReplacesDefault(t *testing.T) {
	// Empty array SHOULD replace default (user explicitly wants empty)
	configJSON := `{"policy": {"shell_allow": []}}`
	fs := &MockFileSystem{
		HomeDir: "/home/user",
		Files: map[string][]byte{
			"/home/user/.config/iav/config.json": []byte(configJSON),
		},
	}
	loader := NewLoaderWithFS(fs)

	cfg, err := loader.Load()

	require.NoError(t, err)
	assert.Empty(t, cfg.Policy.ShellAllow) // Empty array, not default
}

func TestLoad_NullPolicyArray_KeepsDefault(t *testing.T) {
	// Null/missing array should keep defaults
	configJSON := `{"policy": {"shell_deny": ["rm"]}}`
	fs := &MockFileSystem{
		HomeDir: "/home/user",
		Files: map[string][]byte{
			"/home/user/.config/iav/config.json": []byte(configJSON),
		},
	}
	loader := NewLoaderWithFS(fs)

	cfg, err := loader.Load()

	require.NoError(t, err)
	assert.Contains(t, cfg.Policy.ShellAllow, "docker")   // Default kept
	assert.Equal(t, []string{"rm"}, cfg.Policy.ShellDeny) // Overridden
}

func TestLoad_VeryLargeValues_Accepted(t *testing.T) {
	// Extreme values should be accepted (no validation in loader)
	configJSON := `{"orchestrator": {"max_turns": 999999999}}`
	fs := &MockFileSystem{
		HomeDir: "/home/user",
		Files: map[string][]byte{
			"/home/user/.config/iav/config.json": []byte(configJSON),
		},
	}
	loader := NewLoaderWithFS(fs)

	cfg, err := loader.Load()

	require.NoError(t, err)
	assert.Equal(t, 999999999, cfg.Orchestrator.MaxTurns)
}

func TestLoad_NegativeValues_Rejected(t *testing.T) {
	// Negative values should be rejected by validation
	configJSON := `{"tools": {"default_shell_timeout": -1}}`
	fs := &MockFileSystem{
		HomeDir: "/home/user",
		Files: map[string][]byte{
			"/home/user/.config/iav/config.json": []byte(configJSON),
		},
	}
	loader := NewLoaderWithFS(fs)

	cfg, err := loader.Load()

	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "validation failed")
}

func TestLoad_UnknownFields_Ignored(t *testing.T) {
	// Unknown fields in JSON should be silently ignored
	configJSON := `{"orchestrator": {"max_turns": 100}, "unknown_field": "ignored"}`
	fs := &MockFileSystem{
		HomeDir: "/home/user",
		Files: map[string][]byte{
			"/home/user/.config/iav/config.json": []byte(configJSON),
		},
	}
	loader := NewLoaderWithFS(fs)

	cfg, err := loader.Load()

	require.NoError(t, err)
	assert.Equal(t, 100, cfg.Orchestrator.MaxTurns)
}

func TestLoad_UnicodeInStrings_Handled(t *testing.T) {
	// Unicode characters in string fields
	configJSON := `{"ui": {"color_primary": "🎨"}}`
	fs := &MockFileSystem{
		HomeDir: "/home/user",
		Files: map[string][]byte{
			"/home/user/.config/iav/config.json": []byte(configJSON),
		},
	}
	loader := NewLoaderWithFS(fs)

	cfg, err := loader.Load()

	require.NoError(t, err)
	assert.Equal(t, "🎨", cfg.UI.ColorPrimary)
}

// --- DEFAULT CONFIG TESTS ---

func TestDefaultConfig_AllFieldsInitialized(t *testing.T) {
	cfg := DefaultConfig()

	// Verify no nil slices
	assert.NotNil(t, cfg.Policy.ShellAllow)
	assert.NotNil(t, cfg.Policy.ShellDeny)
	assert.NotNil(t, cfg.Policy.ToolsAllow)
	assert.NotNil(t, cfg.Policy.ToolsDeny)

	// Verify critical defaults
	assert.Greater(t, cfg.Orchestrator.MaxTurns, 0)
	assert.Greater(t, cfg.Provider.FallbackMaxOutputTokens, 0)
	assert.Greater(t, cfg.Tools.MaxFileSize, int64(0))
}
