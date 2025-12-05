package tools

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/Cyclone1070/iav/internal/tools/models"
	"github.com/Cyclone1070/iav/internal/tools/services"
)

// ListDirectory lists the contents of a directory within the workspace.
// ListDirectory lists the contents of a directory with optional recursion and pagination.
// It validates the path is within workspace boundaries, respects gitignore rules,
// and returns entries sorted by path with pagination support.
func ListDirectory(ctx context.Context, wCtx *models.WorkspaceContext, req models.ListDirectoryRequest) (*models.ListDirectoryResponse, error) {
	// Validate pagination parameters
	if req.Offset < 0 {
		return nil, models.ErrInvalidPaginationOffset
	}
	if req.Limit < 1 || req.Limit > models.MaxListDirectoryLimit {
		return nil, models.ErrInvalidPaginationLimit
	}

	// Default to workspace root if path is empty
	if req.Path == "" {
		req.Path = "."
	}

	// Resolve path
	abs, rel, err := services.Resolve(wCtx, req.Path)
	if err != nil {
		return nil, err
	}

	// Check if path exists and is a directory
	info, err := wCtx.FS.Stat(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, models.ErrFileMissing
		}
		return nil, fmt.Errorf("failed to stat path: %w", err)
	}

	if !info.IsDir() {
		return nil, fmt.Errorf("path is not a directory")
	}

	// Set maxDepth: 0 = non-recursive (only immediate children), -1 or negative = unlimited
	maxDepth := req.MaxDepth
	if maxDepth < 0 {
		maxDepth = -1 // unlimited
	}

	// Collect entries recursively
	visited := make(map[string]bool)
	directoryEntries, err := listRecursive(ctx, wCtx, abs, 0, maxDepth, visited, req.IncludeIgnored)
	if err != nil {
		return nil, err
	}

	// Sort: directories first, then files, both alphabetically by RelativePath
	sort.Slice(directoryEntries, func(i, j int) bool {
		// Directories come before files
		if directoryEntries[i].IsDir && !directoryEntries[j].IsDir {
			return true
		}
		if !directoryEntries[i].IsDir && directoryEntries[j].IsDir {
			return false
		}
		// Within same type, sort alphabetically
		return directoryEntries[i].RelativePath < directoryEntries[j].RelativePath
	})

	// Apply pagination
	totalCount := len(directoryEntries)
	truncated := false

	// Handle offset
	if req.Offset >= totalCount {
		directoryEntries = []models.DirectoryEntry{}
	} else {
		directoryEntries = directoryEntries[req.Offset:]

		// Handle limit
		if len(directoryEntries) > req.Limit {
			directoryEntries = directoryEntries[:req.Limit]
			truncated = true
		}
	}

	return &models.ListDirectoryResponse{
		DirectoryPath: rel,
		Entries:       directoryEntries,
		Offset:        req.Offset,
		Limit:         req.Limit,
		TotalCount:    totalCount,
		Truncated:     truncated,
	}, nil
}

// listRecursive recursively lists all entries up to maxDepth (-1 = unlimited, 0 = current level only)
func listRecursive(ctx context.Context, wCtx *models.WorkspaceContext, abs string, currentDepth int, maxDepth int, visited map[string]bool, includeIgnored bool) ([]models.DirectoryEntry, error) {
	// Check cancellation
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	// Check depth limit (-1 = unlimited, 0 = current level only, 1 = current + 1 level, etc.)
	if maxDepth >= 0 && currentDepth > maxDepth {
		return []models.DirectoryEntry{}, nil
	}

	// Detect symlink loops using canonical path
	canonicalPath, err := filepath.EvalSymlinks(abs)
	if err != nil {
		// If we can't resolve symlinks, use the original path
		canonicalPath = abs
	}

	if visited[canonicalPath] {
		// Symlink loop detected, skip
		return []models.DirectoryEntry{}, nil
	}
	visited[canonicalPath] = true

	allEntries, err := wCtx.FS.ListDir(abs)
	if err != nil {
		// Propagate sentinel errors directly
		if errors.Is(err, models.ErrOutsideWorkspace) || errors.Is(err, models.ErrFileMissing) {
			return nil, err
		}
		// Wrap other errors for context
		return nil, fmt.Errorf("failed to list directory: %w", err)
	}

	var directoryEntries []models.DirectoryEntry
	for _, entry := range allEntries {
		// Calculate relative path for this entry
		entryAbs := filepath.Join(abs, entry.Name())
		entryRel, err := filepath.Rel(wCtx.WorkspaceRoot, entryAbs)
		if err != nil {
			// This indicates a bug in path resolution - don't mask it
			return nil, fmt.Errorf("failed to calculate relative path for entry %s: %w", entry.Name(), err)
		}

		// Normalize to forward slashes
		entryRel = filepath.ToSlash(entryRel)

		// Apply gitignore filtering (unless IncludeIgnored is true)
		if !includeIgnored && wCtx.GitignoreService != nil {
			if wCtx.GitignoreService.ShouldIgnore(entryRel) {
				continue // Skip gitignored files
			}
		}

		directoryEntry := models.DirectoryEntry{
			RelativePath: entryRel,
			IsDir:        entry.IsDir(),
		}

		directoryEntries = append(directoryEntries, directoryEntry)

		// Recurse into subdirectories
		if entry.IsDir() {
			subEntries, err := listRecursive(ctx, wCtx, entryAbs, currentDepth+1, maxDepth, visited, includeIgnored)
			if err != nil {
				return nil, err
			}
			directoryEntries = append(directoryEntries, subEntries...)
		}
	}

	return directoryEntries, nil
}
