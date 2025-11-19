package tools

import (
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Cyclone1070/deployforme/internal/tools/models"
	"github.com/Cyclone1070/deployforme/internal/tools/services"
)

// ListDirectory lists the contents of a directory within the workspace.
// If path is empty, lists the workspace root.
// Returns a sorted list of entries (directories first, then files alphabetically).
// Pagination is supported via offset and limit parameters.
// Dotfiles are included but respect .gitignore if GitignoreService is configured.
func ListDirectory(ctx *models.WorkspaceContext, path string, offset int, limit int) (*models.ListDirectoryResponse, error) {
	// Validate pagination parameters
	if offset < 0 {
		return nil, models.ErrInvalidPaginationOffset
	}
	if limit < 1 || limit > models.MaxListDirectoryLimit {
		return nil, models.ErrInvalidPaginationLimit
	}

	// Default to workspace root if path is empty
	if path == "" {
		path = "."
	}

	// Resolve path once
	abs, rel, err := services.Resolve(ctx, path)
	if err != nil {
		return nil, err
	}

	// Ensure it's a directory using already-resolved path
	isDir, err := ctx.FS.IsDir(abs)
	if err != nil {
		return nil, fmt.Errorf("failed to check if path is directory: %w", err)
	}

	if !isDir {
		return nil, fmt.Errorf("path is not a directory: %s", rel)
	}

	// List directory contents
	allEntries, err := ctx.FS.ListDir(abs)
	if err != nil {
		// Propagate sentinel errors directly
		if errors.Is(err, models.ErrOutsideWorkspace) || errors.Is(err, models.ErrFileMissing) {
			return nil, err
		}
		// Wrap other errors for context
		return nil, fmt.Errorf("failed to list directory: %w", err)
	}

	// Build entries list with gitignore filtering
	directoryEntries := make([]models.DirectoryEntry, 0, len(allEntries))
	for _, entry := range allEntries {
		// Calculate relative path for this entry
		entryAbs := filepath.Join(abs, entry.Name())
		entryRel, err := filepath.Rel(ctx.WorkspaceRoot, entryAbs)
		if err != nil {
			// This indicates a bug in path resolution - don't mask it
			return nil, fmt.Errorf("failed to calculate relative path for entry %s: %w", entry.Name(), err)
		}

		// Normalize to forward slashes
		entryRel = filepath.ToSlash(entryRel)

		// Filter dotfiles through gitignore
		if strings.HasPrefix(entry.Name(), ".") && ctx.GitignoreService != nil {
			if ctx.GitignoreService.ShouldIgnore(entryRel) {
				continue // Skip gitignored dotfiles
			}
		}

		directoryEntry := models.DirectoryEntry{
			RelativePath: entryRel,
			IsDir:        entry.IsDir(),
			Size:         entry.Size(),
		}

		directoryEntries = append(directoryEntries, directoryEntry)
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
	if offset >= totalCount {
		directoryEntries = []models.DirectoryEntry{}
	} else {
		directoryEntries = directoryEntries[offset:]

		// Handle limit
		if len(directoryEntries) > limit {
			directoryEntries = directoryEntries[:limit]
			truncated = true
		}
	}

	return &models.ListDirectoryResponse{
		DirectoryPath: rel,
		Entries:       directoryEntries,
		Offset:        offset,
		Limit:         limit,
		TotalCount:    totalCount,
		Truncated:     truncated,
	}, nil
}
