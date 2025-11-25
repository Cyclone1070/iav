package services

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/Cyclone1070/deployforme/internal/tools/models"
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
func ParseEnvFile(fs models.FileSystem, path string) (map[string]string, error) {
	content, err := fs.ReadFileRange(path, 0, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to read env file: %w", err)
	}

	env := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(string(content)))
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Split on first =
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid line %d: %s", lineNum, line)
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

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading env file: %w", err)
	}

	return env, nil
}
