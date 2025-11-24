package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Cyclone1070/deployforme/internal/tools/models"
	"github.com/Cyclone1070/deployforme/internal/tools/services"
)

const (
	// maxSearchContentResults is the hard limit for search results to prevent resource exhaustion.
	maxSearchContentResults = 10000
	// maxLineLength is the maximum length of a line before truncation.
	maxLineLength = 10000
)

// SearchContent searches for content matching a regex pattern within files.
// It uses `ripgrep` (preferred) or `grep` (fallback) for efficient searching.
// If includeIgnored is true, searches will include files that match .gitignore patterns.
func SearchContent(ctx *models.WorkspaceContext, query string, searchPath string, caseSensitive bool, includeIgnored bool, offset int, limit int) (*models.SearchContentResponse, error) {
	// 1. Validate pagination
	if offset < 0 {
		return nil, models.ErrInvalidPaginationOffset
	}
	if limit <= 0 || limit > 1000 {
		return nil, models.ErrInvalidPaginationLimit
	}

	// 2. Validate query
	if query == "" {
		return nil, fmt.Errorf("query cannot be empty")
	}

	// 3. Resolve search path
	absSearchPath, _, err := services.Resolve(ctx, searchPath)
	if err != nil {
		return nil, err
	}

	// 4. Verify search path exists
	_, err = ctx.FS.Stat(absSearchPath)
	if err != nil {
		return nil, models.ErrFileMissing
	}

	// 5. Build ripgrep command
	// rg --json "query" searchPath [--no-ignore]
	cmd := []string{"rg", "--json"}
	if !caseSensitive {
		cmd = append(cmd, "-i")
	}
	if includeIgnored {
		cmd = append(cmd, "--no-ignore")
	}
	cmd = append(cmd, query, absSearchPath)

	// 6. Execute command
	output, err := ctx.CommandExecutor.Run(context.Background(), cmd)
	if err != nil {
		exitCode := services.GetExitCode(err)
		if exitCode == 1 {
			// rg returns 1 for no matches (valid case)
			return &models.SearchContentResponse{
				Matches:    []models.SearchContentMatch{},
				Offset:     offset,
				Limit:      limit,
				TotalCount: 0,
				Truncated:  false,
			}, nil
		}
		// Exit code 2+ = real error
		return nil, fmt.Errorf("rg command failed: %w", err)
	}

	// 7. Parse JSON output
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var matches []models.SearchContentMatch

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Parse JSON line
		var rgMatch struct {
			Type string `json:"type"`
			Data struct {
				Path struct {
					Text string `json:"text"`
				} `json:"path"`
				Lines struct {
					Text string `json:"text"`
				} `json:"lines"`
				LineNumber int `json:"line_number"`
			} `json:"data"`
		}

		if err := json.Unmarshal([]byte(line), &rgMatch); err != nil {
			// Skip invalid JSON lines
			continue
		}

		// Only process "match" type entries
		if rgMatch.Type != "match" {
			continue
		}

		// Convert absolute path to relative
		relPath, err := filepath.Rel(ctx.WorkspaceRoot, rgMatch.Data.Path.Text)
		if err != nil {
			// Skip if we can't make it relative
			continue
		}

		// Truncate very long lines
		lineContent := rgMatch.Data.Lines.Text
		if len(lineContent) > maxLineLength {
			lineContent = lineContent[:maxLineLength] + "... [truncated]"
		}

		matches = append(matches, models.SearchContentMatch{
			File:        relPath,
			LineNumber:  rgMatch.Data.LineNumber,
			LineContent: lineContent,
		})

		// Hard limit to prevent resource exhaustion
		if len(matches) >= maxSearchContentResults {
			break
		}
	}

	// 8. Sort results for consistency (by file, then line number)
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].File != matches[j].File {
			return matches[i].File < matches[j].File
		}
		return matches[i].LineNumber < matches[j].LineNumber
	})

	// 9. Apply pagination
	totalCount := len(matches)
	start := min(offset, totalCount)
	end := start + limit
	truncated := end < totalCount
	if end > totalCount {
		end = totalCount
	}

	paginatedMatches := matches[start:end]

	return &models.SearchContentResponse{
		Matches:    paginatedMatches,
		Offset:     offset,
		Limit:      limit,
		TotalCount: totalCount,
		Truncated:  truncated,
	}, nil
}
