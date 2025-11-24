package services

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseEnvFile(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()

	t.Run("Basic KEY=VALUE parsing", func(t *testing.T) {
		envFile := filepath.Join(tmpDir, "test1.env")
		content := `KEY1=value1
KEY2=value2
KEY3=value3`
		if err := os.WriteFile(envFile, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		env, err := ParseEnvFile(envFile)
		if err != nil {
			t.Fatalf("ParseEnvFile failed: %v", err)
		}

		expected := map[string]string{
			"KEY1": "value1",
			"KEY2": "value2",
			"KEY3": "value3",
		}

		for k, v := range expected {
			if env[k] != v {
				t.Errorf("Expected %s=%s, got %s=%s", k, v, k, env[k])
			}
		}
	})

	t.Run("Comments and empty lines", func(t *testing.T) {
		envFile := filepath.Join(tmpDir, "test2.env")
		content := `# This is a comment
KEY1=value1

# Another comment
KEY2=value2
`
		if err := os.WriteFile(envFile, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		env, err := ParseEnvFile(envFile)
		if err != nil {
			t.Fatalf("ParseEnvFile failed: %v", err)
		}

		if len(env) != 2 {
			t.Errorf("Expected 2 keys, got %d", len(env))
		}

		if env["KEY1"] != "value1" || env["KEY2"] != "value2" {
			t.Errorf("Unexpected values: %v", env)
		}
	})

	t.Run("Quoted values", func(t *testing.T) {
		envFile := filepath.Join(tmpDir, "test3.env")
		content := `KEY1="value with spaces"
KEY2='single quoted'
KEY3=unquoted`
		if err := os.WriteFile(envFile, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		env, err := ParseEnvFile(envFile)
		if err != nil {
			t.Fatalf("ParseEnvFile failed: %v", err)
		}

		if env["KEY1"] != "value with spaces" {
			t.Errorf("Expected 'value with spaces', got '%s'", env["KEY1"])
		}
		if env["KEY2"] != "single quoted" {
			t.Errorf("Expected 'single quoted', got '%s'", env["KEY2"])
		}
		if env["KEY3"] != "unquoted" {
			t.Errorf("Expected 'unquoted', got '%s'", env["KEY3"])
		}
	})

	t.Run("Invalid format", func(t *testing.T) {
		envFile := filepath.Join(tmpDir, "test4.env")
		content := `INVALID LINE WITHOUT EQUALS`
		if err := os.WriteFile(envFile, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		_, err := ParseEnvFile(envFile)
		if err == nil {
			t.Error("Expected error for invalid format, got nil")
		}
	})

	t.Run("File not found", func(t *testing.T) {
		_, err := ParseEnvFile("/nonexistent/file.env")
		if err == nil {
			t.Error("Expected error for nonexistent file, got nil")
		}
	})

	t.Run("Values with equals sign", func(t *testing.T) {
		envFile := filepath.Join(tmpDir, "test5.env")
		content := `DATABASE_URL=postgres://user:pass@localhost:5432/db?sslmode=disable`
		if err := os.WriteFile(envFile, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		env, err := ParseEnvFile(envFile)
		if err != nil {
			t.Fatalf("ParseEnvFile failed: %v", err)
		}

		expected := "postgres://user:pass@localhost:5432/db?sslmode=disable"
		if env["DATABASE_URL"] != expected {
			t.Errorf("Expected '%s', got '%s'", expected, env["DATABASE_URL"])
		}
	})
}
