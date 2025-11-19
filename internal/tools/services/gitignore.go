package services

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Cyclone1070/deployforme/internal/tools/models"
	"github.com/go-git/go-git/v5/plumbing/format/gitignore"
)

// gitignoreService implements models.GitignoreService

type gitignoreService struct {
	matcher gitignore.Matcher
}

// NewGitignoreService creates a new gitignore service by loading .gitignore from workspace root
// Returns a service that never ignores if .gitignore doesn't exist (no error)
func NewGitignoreService(workspaceRoot string, fs models.FileSystem) (models.GitignoreService, error) {
	gitignorePath := filepath.Join(workspaceRoot, ".gitignore")

	// Check if .gitignore exists
	_, err := fs.Stat(gitignorePath)
	if err != nil {
		// .gitignore doesn't exist - return a service that never ignores
		return &gitignoreService{matcher: nil}, nil
	}

	// Read .gitignore file
	content, err := fs.ReadFileRange(gitignorePath, 0, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to read .gitignore: %w", err)
	}

	// Parse gitignore patterns line by line
	var patterns []gitignore.Pattern
	lines := splitLines(string(content))
	for _, line := range lines {
		if line == "" {
			continue // Skip blank lines
		}
		pattern := gitignore.ParsePattern(line, nil)
		if pattern != nil {
			patterns = append(patterns, pattern)
		}
	}
	matcher := gitignore.NewMatcher(patterns)

	return &gitignoreService{matcher: matcher}, nil
}

// ShouldIgnore checks if a relative path matches gitignore patterns
func (g *gitignoreService) ShouldIgnore(relativePath string) bool {
	if g.matcher == nil {
		return false
	}

	// Convert to gitignore format (split path into segments)
	segments := splitPath(relativePath)
	return g.matcher.Match(segments, false)
}

// splitPath splits a path into segments for gitignore matching
func splitPath(path string) []string {
	if path == "" {
		return []string{}
	}

	// Normalize path separators
	normalized := filepath.ToSlash(path)

	// Split by forward slash
	parts := strings.Split(normalized, "/")
	var segments []string
	for _, part := range parts {
		if part != "" && part != "." {
			segments = append(segments, part)
		}
	}

	return segments
}

// splitLines splits content into lines, handling both \n and \r\n
func splitLines(content string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(content); i++ {
		if content[i] == '\n' {
			lines = append(lines, content[start:i])
			start = i + 1
		} else if content[i] == '\r' && i+1 < len(content) && content[i+1] == '\n' {
			lines = append(lines, content[start:i])
			start = i + 2
			i++ // Skip the \n
		}
	}
	if start < len(content) {
		lines = append(lines, content[start:])
	}
	return lines
}

