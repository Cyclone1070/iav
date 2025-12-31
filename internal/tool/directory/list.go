package directory

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Cyclone1070/iav/internal/config"
	"github.com/Cyclone1070/iav/internal/tool/helper/pagination"
	"github.com/Cyclone1070/iav/internal/tool/service/path"
)

// directoryEntry is a private helper struct for internal processing of directory entries.
type directoryEntry struct {
	RelativePath string
	IsDir        bool
}

// dirLister defines the filesystem operations needed for listing directories.
type dirLister interface {
	Stat(path string) (os.FileInfo, error)
	ListDir(path string) ([]os.FileInfo, error)
}

// ignoreMatcher defines the interface for gitignore pattern matching.
type ignoreMatcher interface {
	ShouldIgnore(relativePath string) bool
}

// ListDirectoryTool handles directory listing operations.
type ListDirectoryTool struct {
	fs            dirLister
	ignoreMatcher ignoreMatcher
	config        *config.Config
	pathResolver  pathResolver
}

// NewListDirectoryTool creates a new ListDirectoryTool with injected dependencies.
func NewListDirectoryTool(
	fs dirLister,
	ignoreMatcher ignoreMatcher,
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
			return nil, fmt.Errorf("path does not exist: %s", abs)
		}
		return nil, fmt.Errorf("failed to stat %s: %w", abs, err)
	}

	if !info.IsDir() {
		return nil, fmt.Errorf("not a directory: %s", abs)
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

	if capHit {
		paginationResult.Truncated = true
	}

	var sb strings.Builder
	for _, entry := range directoryEntries {
		path := entry.RelativePath
		if entry.IsDir && !strings.HasSuffix(path, "/") {
			path += "/"
		}
		sb.WriteString(path + "\n")
	}

	return &ListDirectoryResponse{
		DirectoryPath:    rel,
		FormattedEntries: sb.String(),
		Offset:           req.Offset,
		Limit:            limit,
		TotalCount:       paginationResult.TotalCount,
		HitMaxResults:    capHit,
	}, nil
}

// listRecursive recursively lists all entries up to maxDepth
// Returns entries, boolean (true if cap hit), error
func (t *ListDirectoryTool) listRecursive(ctx context.Context, abs string, currentDepth int, maxDepth int, visited map[string]bool, includeIgnored bool, maxResults int, currentCount *int) ([]directoryEntry, bool, error) {
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
		return []directoryEntry{}, false, nil
	}

	// Detect symlink loops using canonical path
	canonicalPath, err := filepath.EvalSymlinks(abs)
	if err != nil {
		// If we can't resolve symlinks, use the original path
		canonicalPath = abs
	}

	if visited[canonicalPath] {
		// Symlink loop detected, skip
		return []directoryEntry{}, false, nil
	}
	visited[canonicalPath] = true

	allEntries, err := t.fs.ListDir(abs)
	if err != nil {
		// Detect specific error conditions using errors.Is
		if errors.Is(err, path.ErrOutsideWorkspace) {
			return nil, false, err
		}

		// Wrap other errors for context
		return nil, false, fmt.Errorf("failed to list directory %s: %w", abs, err)
	}

	var directoryEntries []directoryEntry
	for _, entry := range allEntries {
		if *currentCount >= maxResults {
			return directoryEntries, true, nil
		}

		// Calculate relative path for this entry
		entryAbs := filepath.Join(abs, entry.Name())
		entryRel, err := t.pathResolver.Rel(entryAbs)
		if err != nil {
			// This indicates a bug in path resolution or directory structure - don't mask it
			return nil, false, fmt.Errorf("relative path for %s: %w", entryAbs, err)
		}

		// Normalize to forward slashes
		entryRel = filepath.ToSlash(entryRel)

		// Apply gitignore filtering (unless IncludeIgnored is true)
		if !includeIgnored && t.ignoreMatcher != nil {
			if t.ignoreMatcher.ShouldIgnore(entryRel) {
				continue // Skip gitignored files
			}
		}

		isDir := entry.IsDir()
		if !isDir && (entry.Mode()&os.ModeSymlink != 0) {
			// Check if symlink points to a directory
			if target, err := t.fs.Stat(entryAbs); err == nil {
				isDir = target.IsDir()
			}
		}

		directoryEntry := directoryEntry{
			RelativePath: entryRel,
			IsDir:        isDir,
		}

		directoryEntries = append(directoryEntries, directoryEntry)
		*currentCount++

		// Recurse into subdirectories
		if isDir {
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
