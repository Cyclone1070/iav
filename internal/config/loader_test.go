package config_test

import (
	"errors"
	"os"
	"testing"

	"github.com/Cyclone1070/iav/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockFileSystem is a local mock implementing config.FileSystem for testing
type mockFileSystem struct {
	Files       map[string][]byte
	HomeDirErr  error
	ReadFileErr error
}

func (m *mockFileSystem) UserHomeDir() (string, error) {
	if m.HomeDirErr != nil {
		return "", m.HomeDirErr
	}
	return "/home/user", nil
}

func (m *mockFileSystem) ReadFile(path string) ([]byte, error) {
	if m.ReadFileErr != nil {
		return nil, m.ReadFileErr
	}
	if data, ok := m.Files[path]; ok {
		return data, nil
	}
	return nil, os.ErrNotExist
}

// SetOperationError sets an error for a specific operation
func (m *mockFileSystem) SetOperationError(operation string, err error) {
	if operation == "ReadFile" {
		m.ReadFileErr = err
	}
}

// createMockFS helper to reduce boilerplate
func createMockFS(files map[string][]byte) *mockFileSystem {
	return &mockFileSystem{
		Files: files,
	}
}

// --- HAPPY PATH TESTS ---

func TestLoad_NoConfigFile_ReturnsDefaults(t *testing.T) {
	// Config file doesn't exist - should return all defaults
	fs := createMockFS(nil)
	loader := config.NewLoaderWithFS(fs)

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
		"tools": {"max_file_size": 10485760, "default_shell_timeout": 1800, "initial_scanner_buffer_size": 131072},
		"ui": {"tick_interval_ms": 200, "color_primary": "99"},
		"policy": {"shell_allow": ["docker"], "shell_deny": ["rm"]}
	}`
	fs := createMockFS(map[string][]byte{
		"/home/user/.config/iav/config.json": []byte(configJSON),
	})
	loader := config.NewLoaderWithFS(fs)

	cfg, err := loader.Load()

	require.NoError(t, err)
	assert.Equal(t, 100, cfg.Orchestrator.MaxTurns)
	assert.Equal(t, 16384, cfg.Provider.FallbackMaxOutputTokens)
	assert.Equal(t, int64(10485760), cfg.Tools.MaxFileSize)
	assert.Equal(t, 131072, cfg.Tools.InitialScannerBufferSize)
	assert.Equal(t, 200, cfg.UI.TickIntervalMs)
	assert.Equal(t, []string{"docker"}, cfg.Policy.ShellAllow)
}

func TestLoad_PartialOverride_MergesWithDefaults(t *testing.T) {
	// Config file only overrides max_turns - rest should be defaults
	configJSON := `{"orchestrator": {"max_turns": 200}}`
	fs := createMockFS(map[string][]byte{
		"/home/user/.config/iav/config.json": []byte(configJSON),
	})
	loader := config.NewLoaderWithFS(fs)

	cfg, err := loader.Load()

	require.NoError(t, err)
	assert.Equal(t, 200, cfg.Orchestrator.MaxTurns)             // Overridden
	assert.Equal(t, 8192, cfg.Provider.FallbackMaxOutputTokens) // Default
	assert.Equal(t, int64(5*1024*1024), cfg.Tools.MaxFileSize)  // Default
	assert.Contains(t, cfg.Policy.ShellAllow, "docker")         // Default list
}

func TestLoad_EmptyConfigFile_ReturnsDefaults(t *testing.T) {
	// Empty JSON object - should use all defaults
	fs := createMockFS(map[string][]byte{
		"/home/user/.config/iav/config.json": []byte(`{}`),
	})
	loader := config.NewLoaderWithFS(fs)

	cfg, err := loader.Load()

	require.NoError(t, err)
	assert.Equal(t, 50, cfg.Orchestrator.MaxTurns)
}

func TestLoad_NestedPartialOverride_OnlySpecifiedFieldsChange(t *testing.T) {
	// Override only one field in a nested struct
	configJSON := `{"ui": {"color_primary": "255"}}`
	fs := createMockFS(map[string][]byte{
		"/home/user/.config/iav/config.json": []byte(configJSON),
	})
	loader := config.NewLoaderWithFS(fs)

	cfg, err := loader.Load()

	require.NoError(t, err)
	assert.Equal(t, "255", cfg.UI.ColorPrimary) // Overridden
	assert.Equal(t, "42", cfg.UI.ColorSuccess)  // Default preserved
	assert.Equal(t, 300, cfg.UI.TickIntervalMs) // Default preserved
}

// --- UNHAPPY PATH TESTS ---

func TestLoad_MalformedJSON_ReturnsError(t *testing.T) {
	fs := createMockFS(map[string][]byte{
		"/home/user/.config/iav/config.json": []byte(`{invalid json`),
	})
	loader := config.NewLoaderWithFS(fs)

	cfg, err := loader.Load()

	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "invalid")
}

func TestLoad_PermissionDenied_ReturnsError(t *testing.T) {
	fs := createMockFS(nil)
	fs.SetOperationError("ReadFile", os.ErrPermission)

	loader := config.NewLoaderWithFS(fs)

	cfg, err := loader.Load()

	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.True(t, errors.Is(err, os.ErrPermission))
}

func TestLoad_HomeDirError_ReturnsDefaults(t *testing.T) {
	// Can't determine home dir - gracefully fall back to defaults
	fs := createMockFS(nil)
	fs.HomeDirErr = errors.New("homeless")

	loader := config.NewLoaderWithFS(fs)

	cfg, err := loader.Load()

	require.NoError(t, err)
	assert.Equal(t, 50, cfg.Orchestrator.MaxTurns) // Default
}

func TestLoad_WrongJSONType_ReturnsError(t *testing.T) {
	// JSON is valid but wrong type (array instead of object)
	fs := createMockFS(map[string][]byte{
		"/home/user/.config/iav/config.json": []byte(`["not", "an", "object"]`),
	})
	loader := config.NewLoaderWithFS(fs)

	cfg, err := loader.Load()

	assert.Error(t, err)
	assert.Nil(t, cfg)
}

// --- EDGE CASE TESTS ---

func TestLoad_ZeroValueExplicit_Overrides(t *testing.T) {
	// Setting model_type_priority_pro to 0 SHOULD override default (2)
	// This requires fixing the merge strategy
	configJSON := `{"provider": {"model_type_priority_pro": 0}}`
	fs := createMockFS(map[string][]byte{
		"/home/user/.config/iav/config.json": []byte(configJSON),
	})
	loader := config.NewLoaderWithFS(fs)

	cfg, err := loader.Load()

	require.NoError(t, err)
	assert.Equal(t, 0, cfg.Provider.ModelTypePriorityPro) // Expect 0, not default 2
}

func TestLoad_EmptyStringColor_Overrides(t *testing.T) {
	// Empty string SHOULD override default color if explicitly set
	configJSON := `{"ui": {"color_primary": ""}}`
	fs := createMockFS(map[string][]byte{
		"/home/user/.config/iav/config.json": []byte(configJSON),
	})
	loader := config.NewLoaderWithFS(fs)

	cfg, err := loader.Load()

	require.NoError(t, err)
	assert.Equal(t, "", cfg.UI.ColorPrimary) // Expect empty string, not default
}

func TestLoad_EmptyPolicyArray_ReplacesDefault(t *testing.T) {
	// Empty array SHOULD replace default (user explicitly wants empty)
	configJSON := `{"policy": {"shell_allow": []}}`
	fs := createMockFS(map[string][]byte{
		"/home/user/.config/iav/config.json": []byte(configJSON),
	})
	loader := config.NewLoaderWithFS(fs)

	cfg, err := loader.Load()

	require.NoError(t, err)
	assert.Empty(t, cfg.Policy.ShellAllow) // Empty array, not default
}

func TestLoad_NullPolicyArray_KeepsDefault(t *testing.T) {
	// Null/missing array should keep defaults
	configJSON := `{"policy": {"shell_deny": ["rm"]}}`
	fs := createMockFS(map[string][]byte{
		"/home/user/.config/iav/config.json": []byte(configJSON),
	})
	loader := config.NewLoaderWithFS(fs)

	cfg, err := loader.Load()

	require.NoError(t, err)
	assert.Contains(t, cfg.Policy.ShellAllow, "docker")   // Default kept
	assert.Equal(t, []string{"rm"}, cfg.Policy.ShellDeny) // Overridden
}

func TestLoad_VeryLargeValues_Accepted(t *testing.T) {
	// Extreme values should be accepted (no validation in loader)
	configJSON := `{"orchestrator": {"max_turns": 999999999}}`
	fs := createMockFS(map[string][]byte{
		"/home/user/.config/iav/config.json": []byte(configJSON),
	})
	loader := config.NewLoaderWithFS(fs)

	cfg, err := loader.Load()

	require.NoError(t, err)
	assert.Equal(t, 999999999, cfg.Orchestrator.MaxTurns)
}

func TestLoad_NegativeValues_Rejected(t *testing.T) {
	// Negative values should be rejected by validation
	configJSON := `{"tools": {"default_shell_timeout": -1}}`
	fs := createMockFS(map[string][]byte{
		"/home/user/.config/iav/config.json": []byte(configJSON),
	})
	loader := config.NewLoaderWithFS(fs)

	cfg, err := loader.Load()

	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "validation failed")
}

func TestLoad_UnknownFields_Ignored(t *testing.T) {
	// Unknown fields in JSON should be silently ignored
	configJSON := `{"orchestrator": {"max_turns": 100}, "unknown_field": "ignored"}`
	fs := createMockFS(map[string][]byte{
		"/home/user/.config/iav/config.json": []byte(configJSON),
	})
	loader := config.NewLoaderWithFS(fs)

	cfg, err := loader.Load()

	require.NoError(t, err)
	assert.Equal(t, 100, cfg.Orchestrator.MaxTurns)
}

func TestLoad_UnicodeInStrings_Handled(t *testing.T) {
	// Unicode characters in string fields
	configJSON := `{"policy": {"shell_allow": ["👾"]}}`
	fs := createMockFS(map[string][]byte{
		"/home/user/.config/iav/config.json": []byte(configJSON),
	})
	loader := config.NewLoaderWithFS(fs)

	cfg, err := loader.Load()

	require.NoError(t, err)
	assert.Equal(t, []string{"👾"}, cfg.Policy.ShellAllow)
}
