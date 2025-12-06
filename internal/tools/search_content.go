package tools

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Cyclone1070/iav/internal/tools/models"
	"github.com/Cyclone1070/iav/internal/tools/services"
)

const (
	// maxSearchContentResults is the hard limit for search results to prevent resource exhaustion.
	maxSearchContentResults = 10000
	// maxLineLength is the maximum length of a line before truncation.
	maxLineLength = 10000
)

// SearchContent searches for content matching a regex pattern using ripgrep.
// It validates the search path is within workspace boundaries, respects gitignore rules
// (unless includeIgnored is true), and returns matches with pagination support.
func SearchContent(ctx context.Context, wCtx *models.WorkspaceContext, req models.SearchContentRequest) (*models.SearchContentResponse, error) {
	// Resolve search path
	absSearchPath, _, err := services.Resolve(wCtx, req.SearchPath)
	if err != nil {
		return nil, err
	}

	// Check if search path exists
	info, err := wCtx.FS.Stat(absSearchPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, models.ErrFileMissing
		}
		return nil, fmt.Errorf("failed to stat search path: %w", err)
	}

	if !info.IsDir() {
		return nil, fmt.Errorf("search path is not a directory")
	}

	// Validate query
	if req.Query == "" {
		return nil, fmt.Errorf("query cannot be empty")
	}

	// Validate and set defaults for pagination
	limit := req.Limit
	if limit == 0 {
		limit = models.DefaultListDirectoryLimit
	}
	if limit < 1 || limit > models.MaxListDirectoryLimit {
		return nil, models.ErrInvalidPaginationLimit
	}

	offset := max(req.Offset, 0)

	// Build ripgrep command
	// rg --json "query" searchPath [--no-ignore]
	cmd := []string{"rg", "--json"}
	if !req.CaseSensitive {
		cmd = append(cmd, "-i")
	}
	if req.IncludeIgnored {
		cmd = append(cmd, "--no-ignore")
	}
	cmd = append(cmd, req.Query, absSearchPath)

	// Execute command with streaming
	proc, stdout, _, err := wCtx.CommandExecutor.Start(ctx, cmd, models.ProcessOptions{Dir: absSearchPath})
	if err != nil {
		return nil, fmt.Errorf("failed to start rg command: %w", err)
	}
	defer proc.Wait()

	// Stream and process JSON output line by line
	var matches []models.SearchContentMatch
	scanner := bufio.NewScanner(stdout)
	// Increase buffer size to handle very long lines (e.g. minified JS)
	// Increase buffer size to handle very long lines (e.g. minified JS)
	maxScanTokenSize := 10 * 1024 * 1024 // Default fallback (10MB)
	if wCtx.Config != nil {
		maxScanTokenSize = wCtx.Config.Tools.MaxScanTokenSize
	}
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, maxScanTokenSize)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
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
		relPath, err := filepath.Rel(wCtx.WorkspaceRoot, rgMatch.Data.Path.Text)
		if err != nil {
			// Skip if we can't make it relative
			continue
		}

		// Normalize to forward slashes
		relPath = filepath.ToSlash(relPath)

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

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading rg output: %w", err)
	}

	// Wait for command to complete
	if err := proc.Wait(); err != nil {
		exitCode := services.GetExitCode(err)
		if exitCode == 1 {
			// rg returns 1 for no matches (valid case)
			// We just continue with empty matches
		} else {
			// Exit code 2+ = real error
			return nil, fmt.Errorf("rg command failed: %w", err)
		}
	}

	// Sort results for consistency (by file, then line number)
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].File != matches[j].File {
			return matches[i].File < matches[j].File
		}
		return matches[i].LineNumber < matches[j].LineNumber
	})

	// Apply pagination
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
