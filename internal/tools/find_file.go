package tools

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Cyclone1070/deployforme/internal/tools/models"
	"github.com/Cyclone1070/deployforme/internal/tools/services"
)

const (
	maxFindFileResults = 10000 // Hard limit to prevent resource exhaustion
)

// FindFile searches for files matching a glob pattern within the workspace.
// It uses `fd` (preferred) or `find` (fallback) for efficient searching.
func FindFile(ctx *models.WorkspaceContext, pattern string, searchPath string, maxDepth int, offset int, limit int) (*models.FindFileResponse, error) {
	// 1. Validate pagination
	if offset < 0 {
		return nil, models.ErrInvalidPaginationOffset
	}
	if limit <= 0 || limit > 1000 {
		return nil, models.ErrInvalidPaginationLimit
	}

	// 2. Validate pattern (reject path traversal attempts)
	if strings.Contains(pattern, "..") || strings.HasPrefix(pattern, "/") {
		return nil, fmt.Errorf("invalid pattern: path traversal not allowed")
	}

	// 3. Resolve search path
	absSearchPath, _, err := services.Resolve(ctx, searchPath)
	if err != nil {
		return nil, err
	}

	// 4. Verify search path exists and is a directory
	info, err := ctx.FS.Stat(absSearchPath)
	if err != nil {
		return nil, models.ErrFileMissing
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("search path must be a directory")
	}

	// 5. Build fd command
	// fd -p -g "pattern" searchPath --max-depth N
	cmd := []string{"fd", "-p", "-g", pattern}
	if maxDepth > 0 {
		cmd = append(cmd, "--max-depth", fmt.Sprintf("%d", maxDepth))
	}
	cmd = append(cmd, absSearchPath)

	// 6. Execute command
	output, err := ctx.CommandExecutor.Run(context.Background(), cmd)
	if err != nil {
		exitCode := services.GetExitCode(err)
		if exitCode == 1 {
			// fd returns 1 for no matches (valid case)
			return &models.FindFileResponse{
				Matches:    []string{},
				Offset:     offset,
				Limit:      limit,
				TotalCount: 0,
				Truncated:  false,
			}, nil
		}
		// Exit code 2+ = real error
		return nil, fmt.Errorf("fd command failed: %w", err)
	}

	// 7. Parse output (newline-separated paths)
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var matches []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Convert absolute path to relative (from workspace root)
		relPath, err := filepath.Rel(ctx.WorkspaceRoot, line)
		if err != nil {
			// Skip if we can't make it relative
			continue
		}

		matches = append(matches, relPath)

		// Hard limit to prevent resource exhaustion
		if len(matches) >= maxFindFileResults {
			break
		}
	}

	// 8. Sort results for consistency
	sort.Strings(matches)

	// 9. Apply pagination
	totalCount := len(matches)
	start := min(offset, totalCount)
	end := start + limit
	truncated := end < totalCount
	if end > totalCount {
		end = totalCount
	}

	paginatedMatches := matches[start:end]

	return &models.FindFileResponse{
		Matches:    paginatedMatches,
		Offset:     offset,
		Limit:      limit,
		TotalCount: totalCount,
		Truncated:  truncated,
	}, nil
}
