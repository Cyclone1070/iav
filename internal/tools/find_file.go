package tools

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Cyclone1070/deployforme/internal/tools/models"
	"github.com/Cyclone1070/deployforme/internal/tools/services"
)

const (
	// maxFindFileResults is the hard limit for find results to prevent resource exhaustion.
	maxFindFileResults = 10000
)

// FindFile searches for files matching a glob pattern within the workspace.
// It uses `fd` (preferred) or `find` (fallback) for efficient searching.
// If includeIgnored is true, searches will include files that match .gitignore patterns.
// FindFile searches for files matching a glob pattern using the fd command.
// It validates the search path is within workspace boundaries, respects gitignore rules
// (unless includeIgnored is true), and returns matches with pagination support.
func FindFile(ctx *models.WorkspaceContext, req models.FindFileRequest) (*models.FindFileResponse, error) {
	// Validate pattern (reject path traversal attempts)
	if strings.Contains(req.Pattern, "..") || strings.HasPrefix(req.Pattern, "/") {
		return nil, fmt.Errorf("invalid pattern: path traversal not allowed")
	}

	// Resolve search path
	absSearchPath, _, err := services.Resolve(ctx, req.SearchPath)
	if err != nil {
		return nil, err
	}

	// Check if search path exists and is a directory
	info, err := ctx.FS.Stat(absSearchPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, models.ErrFileMissing
		}
		return nil, fmt.Errorf("failed to stat search path: %w", err)
	}

	if !info.IsDir() {
		return nil, fmt.Errorf("search path is not a directory")
	}

	// Validate pattern
	if req.Pattern == "" {
		return nil, fmt.Errorf("pattern cannot be empty")
	}

	// Validate and set defaults for pagination
	limit := req.Limit
	if limit == 0 {
		limit = models.DefaultListDirectoryLimit
	}
	if limit > models.MaxListDirectoryLimit {
		limit = models.MaxListDirectoryLimit
	}
	if limit < 0 {
		return nil, models.ErrInvalidPaginationLimit
	}

	offset := max(req.Offset, 0)

	// Build fd command
	// fd -g "pattern" searchPath --max-depth N [--no-ignore]
	cmd := []string{"fd", "-g", req.Pattern}
	if req.MaxDepth > 0 {
		cmd = append(cmd, "--max-depth", fmt.Sprintf("%d", req.MaxDepth))
	}
	if req.IncludeIgnored {
		cmd = append(cmd, "--no-ignore")
	}
	cmd = append(cmd, absSearchPath)

	// Execute command with streaming
	proc, stdout, _, err := ctx.CommandExecutor.Start(context.Background(), cmd, models.ProcessOptions{Dir: absSearchPath})
	if err != nil {
		return nil, fmt.Errorf("failed to start fd command: %w", err)
	}
	defer proc.Wait()

	// Stream and process output line by line
	var allMatches []string
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Convert to relative path
		relPath, err := filepath.Rel(ctx.WorkspaceRoot, line)
		if err != nil {
			// Skip paths that can't be made relative (shouldn't happen with fd)
			continue
		}

		// Normalize to forward slashes
		relPath = filepath.ToSlash(relPath)

		allMatches = append(allMatches, relPath)

		// Safety limit to prevent unbounded growth
		if len(allMatches) >= maxFindFileResults {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading fd output: %w", err)
	}

	// Wait for command to complete
	if err := proc.Wait(); err != nil {
		// fd returns 0 even if no matches are found.
		// Non-zero exit code indicates an actual error
		return nil, fmt.Errorf("fd command failed: %w", err)
	}

	// Sort matches alphabetically
	sort.Strings(allMatches)

	// Apply pagination
	totalCount := len(allMatches)
	truncated := false

	if offset >= totalCount {
		allMatches = []string{}
	} else {
		allMatches = allMatches[offset:]
		if len(allMatches) > limit {
			allMatches = allMatches[:limit]
			truncated = true
		}
	}

	return &models.FindFileResponse{
		Matches:    allMatches,
		Offset:     offset,
		Limit:      limit,
		TotalCount: totalCount,
		Truncated:  truncated,
	}, nil
}
