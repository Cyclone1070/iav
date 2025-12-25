package shell

import (
	"fmt"
	"strings"
)

// ParseEnvFile parses a .env file and returns a map of environment variables.
// It supports:
// - KEY=VALUE format
// - Comments starting with #
// - Empty lines
// - Basic quoted values (single and double quotes)
//
// It does NOT support:
// - Multi-line values
// - Variable expansion
// - Complex shell escaping
func ParseEnvFile(fs envFileReader, path string) (map[string]string, error) {
	content, err := fs.ReadFileRange(path, 0, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to read env file %s: %w", path, err)
	}

	env := make(map[string]string)
	lines := strings.Split(string(content), "\n")

	for i, rawLine := range lines {
		lineNum := i + 1
		line := strings.TrimSpace(rawLine)

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Split on first =
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("%w: %s:%d: %s", ErrEnvFileParse, path, lineNum, line)
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Remove quotes if present
		if len(value) >= 2 {
			if (value[0] == '"' && value[len(value)-1] == '"') ||
				(value[0] == '\'' && value[len(value)-1] == '\'') {
				value = value[1 : len(value)-1]
			}
		}

		env[key] = value
	}

	return env, nil
}
