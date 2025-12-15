package shell

import (
	"os"
	"testing"
)

// mockFileSystem is a local mock implementing shell.fileSystem for testing
type mockFileSystemForEnv struct {
	files map[string][]byte
}

func newMockFileSystemForEnv() *mockFileSystemForEnv {
	return &mockFileSystemForEnv{
		files: make(map[string][]byte),
	}
}

func (m *mockFileSystemForEnv) createFile(path string, content []byte) {
	m.files[path] = content
}

func (m *mockFileSystemForEnv) Stat(path string) (os.FileInfo, error) {
	if _, ok := m.files[path]; ok {
		return nil, nil
	}
	return nil, os.ErrNotExist
}

func (m *mockFileSystemForEnv) Lstat(path string) (os.FileInfo, error) {
	return m.Stat(path)
}

func (m *mockFileSystemForEnv) Readlink(path string) (string, error) {
	return "", os.ErrInvalid
}

func (m *mockFileSystemForEnv) UserHomeDir() (string, error) {
	return "/home/user", nil
}

func (m *mockFileSystemForEnv) ReadFileRange(path string, offset, limit int64) ([]byte, error) {
	content, ok := m.files[path]
	if !ok {
		return nil, os.ErrNotExist
	}
	return content, nil
}

func TestParseEnvFile(t *testing.T) {
	t.Run("Basic KEY=VALUE parsing", func(t *testing.T) {
		fs := newMockFileSystemForEnv()
		content := `KEY1=value1
KEY2=value2
KEY3=value3`
		fs.createFile("/test1.env", []byte(content))

		env, err := ParseEnvFile(fs, "/test1.env")
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
		fs := newMockFileSystemForEnv()
		content := `# This is a comment
KEY1=value1

# Another comment
KEY2=value2
`
		fs.createFile("/test2.env", []byte(content))

		env, err := ParseEnvFile(fs, "/test2.env")
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
		fs := newMockFileSystemForEnv()
		content := `KEY1="value with spaces"
KEY2='single quoted'
KEY3=unquoted`
		fs.createFile("/test3.env", []byte(content))

		env, err := ParseEnvFile(fs, "/test3.env")
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
		fs := newMockFileSystemForEnv()
		content := `INVALID LINE WITHOUT EQUALS`
		fs.createFile("/test4.env", []byte(content))

		_, err := ParseEnvFile(fs, "/test4.env")
		if err == nil {
			t.Error("Expected error for invalid format, got nil")
		}
	})

	t.Run("File not found", func(t *testing.T) {
		fs := newMockFileSystemForEnv()

		_, err := ParseEnvFile(fs, "/nonexistent/file.env")
		if err == nil {
			t.Error("Expected error for nonexistent file, got nil")
		}
	})

	t.Run("Values with equals sign", func(t *testing.T) {
		fs := newMockFileSystemForEnv()
		content := `DATABASE_URL=postgres://user:pass@localhost:5432/db?sslmode=disable`
		fs.createFile("/test5.env", []byte(content))

		env, err := ParseEnvFile(fs, "/test5.env")
		if err != nil {
			t.Fatalf("ParseEnvFile failed: %v", err)
		}

		expected := "postgres://user:pass@localhost:5432/db?sslmode=disable"
		if env["DATABASE_URL"] != expected {
			t.Errorf("Expected '%s', got '%s'", expected, env["DATABASE_URL"])
		}
	})
}
