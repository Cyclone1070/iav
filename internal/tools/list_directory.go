package tools

import (
	"errors"
	"fmt"
	"path/filepath"
	"sort"

	"github.com/Cyclone1070/deployforme/internal/tools/models"
	"github.com/Cyclone1070/deployforme/internal/tools/services"
)

// ListDirectory lists the contents of a directory within the workspace.
// If path is empty, lists the workspace root.
// Returns a sorted list of entries (directories first, then files alphabetically).
func ListDirectory(ctx *models.WorkspaceContext, path string) (*models.ListDirectoryResponse, error) {
	// Default to workspace root if path is empty
	targetPath := path
	if targetPath == "" {
		targetPath = "."
	}

	// Resolve path
	abs, rel, err := services.Resolve(ctx, targetPath)
	if err != nil {
		return nil, err
	}

	// Ensure it's a directory
	isDir, err := services.IsDirectory(ctx, targetPath)
	if err != nil {
		return nil, fmt.Errorf("failed to check if path is directory: %w", err)
	}

	if !isDir {
		return nil, fmt.Errorf("path is not a directory: %s", rel)
	}

	// List directory contents
	entries, err := ctx.FS.ListDir(abs)
	if err != nil {
		// Propagate sentinel errors directly
		if errors.Is(err, models.ErrOutsideWorkspace) || errors.Is(err, models.ErrFileMissing) {
			return nil, err
		}
		// Wrap other errors for context
		return nil, fmt.Errorf("failed to list directory: %w", err)
	}

	// Convert to DirectoryEntry slice
	directoryEntries := make([]models.DirectoryEntry, 0, len(entries))
	for _, entry := range entries {
		// Calculate relative path for this entry
		entryAbs := filepath.Join(abs, entry.Name())
		entryRel, err := filepath.Rel(ctx.WorkspaceRoot, entryAbs)
		if err != nil {
			// Fallback: construct relative path manually
			if rel == "" {
				entryRel = entry.Name()
			} else {
				entryRel = filepath.Join(rel, entry.Name())
			}
		}

		// Normalise to forward slashes
		entryRel = filepath.ToSlash(entryRel)

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

	return &models.ListDirectoryResponse{
		DirectoryPath: rel,
		Entries:       directoryEntries,
	}, nil
}

