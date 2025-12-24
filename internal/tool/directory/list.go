package directory

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/Cyclone1070/iav/internal/config"
	"github.com/Cyclone1070/iav/internal/tool/helper/pagination"
	"github.com/Cyclone1070/iav/internal/tool/service/path"
)

// dirLister defines the filesystem operations needed for listing directories.
type dirLister interface {
	Stat(path string) (os.FileInfo, error)
	ListDir(path string) ([]os.FileInfo, error)
}

// IgnoreMatcher defines the interface for gitignore pattern matching.
type IgnoreMatcher interface {
	ShouldIgnore(relativePath string) bool
}

// ListDirectoryTool handles directory listing operations.
type ListDirectoryTool struct {
	fs            dirLister
	ignoreMatcher IgnoreMatcher
	config        *config.Config
	pathResolver  pathResolver
}

// NewListDirectoryTool creates a new ListDirectoryTool with injected dependencies.
func NewListDirectoryTool(
	fs dirLister,
	ignoreMatcher IgnoreMatcher,
	cfg *config.Config,
	pathResolver pathResolver,
) *ListDirectoryTool {
	if fs == nil {
		panic("fs is required")
	}
	if cfg == nil {
		panic("cfg is required")
	}
	if pathResolver == nil {
		panic("pathResolver is required")
	}
	return &ListDirectoryTool{
		fs:            fs,
		ignoreMatcher: ignoreMatcher,
		config:        cfg,
		pathResolver:  pathResolver,
	}
}

// Run lists the contents of a directory within the workspace.
// It supports optional recursion and pagination, validating that the path is within
// workspace boundaries, respecting gitignore rules, and returning entries sorted by path.
func (t *ListDirectoryTool) Run(ctx context.Context, req *ListDirectoryRequest) (*ListDirectoryResponse, error) {
	if err := req.Validate(t.config); err != nil {
		return nil, err
	}
	path := req.Path
	if path == "" {
		path = "."
	}
	abs, err := t.pathResolver.Abs(path)
	if err != nil {
		return nil, err
	}
	rel, err := t.pathResolver.Rel(abs)
	if err != nil {
		return nil, err
	}

	limit := req.Limit

	// Check if path exists and is a directory
	info, err := t.fs.Stat(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s", ErrFileMissing, abs)
		}
		return nil, &StatError{Path: abs, Cause: err}
	}

	if !info.IsDir() {
		return nil, fmt.Errorf("%w: %s", ErrNotADirectory, abs)
	}

	maxDepth := req.MaxDepth

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
	directoryEntries, paginationResult := pagination.ApplyPagination(directoryEntries, req.Offset, limit)

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
		// Detect specific error conditions using errors.Is
		if errors.Is(err, path.ErrOutsideWorkspace) {
			return nil, false, err
		}

		// Wrap other errors for context
		return nil, false, &ListDirError{Path: abs, Cause: err}
	}

	var directoryEntries []DirectoryEntry
	for _, entry := range allEntries {
		if *currentCount >= maxResults {
			return directoryEntries, true, nil
		}

		// Calculate relative path for this entry
		entryAbs := filepath.Join(abs, entry.Name())
		entryRel, err := t.pathResolver.Rel(entryAbs)
		if err != nil {
			// This indicates a bug in path resolution or directory structure - don't mask it
			return nil, false, &RelPathError{Path: entryAbs, Cause: err}
		}

		// Normalize to forward slashes
		entryRel = filepath.ToSlash(entryRel)

		// Apply gitignore filtering (unless IncludeIgnored is true)
		if !includeIgnored && t.ignoreMatcher != nil {
			if t.ignoreMatcher.ShouldIgnore(entryRel) {
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
