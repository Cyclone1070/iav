package directory

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/Cyclone1070/iav/internal/config"
	toolserrors "github.com/Cyclone1070/iav/internal/tool/errutil"
	"github.com/Cyclone1070/iav/internal/tool/paginationutil"
	"github.com/Cyclone1070/iav/internal/tool/pathutil"
)

// ListDirectoryTool handles directory listing operations.
type ListDirectoryTool struct {
	fs               fileSystem
	gitignoreService gitignoreService
	config           *config.Config
	workspaceRoot    string
}

// NewListDirectoryTool creates a new ListDirectoryTool with injected dependencies.
func NewListDirectoryTool(
	fs fileSystem,
	gitignoreService gitignoreService,
	cfg *config.Config,
	workspaceRoot string,
) *ListDirectoryTool {
	return &ListDirectoryTool{
		fs:               fs,
		gitignoreService: gitignoreService,
		config:           cfg,
		workspaceRoot:    workspaceRoot,
	}
}

// Run lists the contents of a directory within the workspace.
// It supports optional recursion and pagination, validating that the path is within
// workspace boundaries, respecting gitignore rules, and returning entries sorted by path.
func (t *ListDirectoryTool) Run(ctx context.Context, req ListDirectoryRequest) (*ListDirectoryResponse, error) {
	// Use configured limits - Validate() already checked bounds
	limit := t.config.Tools.DefaultListDirectoryLimit
	if req.Limit != 0 {
		limit = req.Limit
	}

	// Default to workspace root if path is empty
	if req.Path == "" {
		req.Path = "."
	}

	// Resolve path
	abs, rel, err := pathutil.Resolve(t.workspaceRoot, t.fs, req.Path)
	if err != nil {
		return nil, err
	}

	// Check if path exists and is a directory
	info, err := t.fs.Stat(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, toolserrors.ErrFileMissing
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
	maxResults := t.config.Tools.MaxListDirectoryResults
	var currentCount int

	directoryEntries, capHit, err := t.listRecursive(ctx, abs, 0, maxDepth, visited, req.IncludeIgnored, maxResults, &currentCount)
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
	directoryEntries, paginationResult := paginationutil.ApplyPagination(directoryEntries, req.Offset, limit)

	var truncationReason string
	if capHit {
		paginationResult.Truncated = true
		truncationReason = fmt.Sprintf("Results capped at %d entries.", maxResults)
	} else if paginationResult.Truncated {
		truncationReason = fmt.Sprintf("Page limit reached. More results at offset %d.", req.Offset+limit)
	}

	return &ListDirectoryResponse{
		DirectoryPath:    rel,
		Entries:          directoryEntries,
		Offset:           req.Offset,
		Limit:            limit,
		TotalCount:       paginationResult.TotalCount,
		Truncated:        paginationResult.Truncated,
		TruncationReason: truncationReason,
	}, nil
}

// listRecursive recursively lists all entries up to maxDepth
// Returns entries, boolean (true if cap hit), error
func (t *ListDirectoryTool) listRecursive(ctx context.Context, abs string, currentDepth int, maxDepth int, visited map[string]bool, includeIgnored bool, maxResults int, currentCount *int) ([]DirectoryEntry, bool, error) {
	// Check cap
	if *currentCount >= maxResults {
		return nil, true, nil
	}

	// Check cancellation
	if ctx.Err() != nil {
		return nil, false, ctx.Err()
	}
	// Check depth limit (-1 = unlimited, 0 = current level only, 1 = current + 1 level, etc.)
	if maxDepth >= 0 && currentDepth > maxDepth {
		return []DirectoryEntry{}, false, nil
	}

	// Detect symlink loops using canonical path
	canonicalPath, err := filepath.EvalSymlinks(abs)
	if err != nil {
		// If we can't resolve symlinks, use the original path
		canonicalPath = abs
	}

	if visited[canonicalPath] {
		// Symlink loop detected, skip
		return []DirectoryEntry{}, false, nil
	}
	visited[canonicalPath] = true

	allEntries, err := t.fs.ListDir(abs)
	if err != nil {
		// Propagate sentinel errors directly
		if errors.Is(err, toolserrors.ErrOutsideWorkspace) || errors.Is(err, toolserrors.ErrFileMissing) {
			return nil, false, err
		}
		// Wrap other errors for context
		return nil, false, fmt.Errorf("failed to list directory: %w", err)
	}

	var directoryEntries []DirectoryEntry
	for _, entry := range allEntries {
		if *currentCount >= maxResults {
			return directoryEntries, true, nil
		}

		// Calculate relative path for this entry
		entryAbs := filepath.Join(abs, entry.Name())
		entryRel, err := filepath.Rel(t.workspaceRoot, entryAbs)
		if err != nil {
			// This indicates a bug in path resolution - don't mask it
			return nil, false, fmt.Errorf("failed to calculate relative path for entry %s: %w", entry.Name(), err)
		}

		// Normalize to forward slashes
		entryRel = filepath.ToSlash(entryRel)

		// Apply gitignore filtering (unless IncludeIgnored is true)
		if !includeIgnored && t.gitignoreService != nil {
			if t.gitignoreService.ShouldIgnore(entryRel) {
				continue // Skip gitignored files
			}
		}

		directoryEntry := DirectoryEntry{
			RelativePath: entryRel,
			IsDir:        entry.IsDir(),
		}

		directoryEntries = append(directoryEntries, directoryEntry)
		*currentCount++

		// Recurse into subdirectories
		if entry.IsDir() {
			subEntries, capHit, err := t.listRecursive(ctx, entryAbs, currentDepth+1, maxDepth, visited, includeIgnored, maxResults, currentCount)
			if err != nil {
				return nil, false, err
			}
			directoryEntries = append(directoryEntries, subEntries...)
			if capHit {
				return directoryEntries, true, nil
			}
		}
	}

	return directoryEntries, false, nil
}
