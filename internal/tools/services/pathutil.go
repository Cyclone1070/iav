package services

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Cyclone1070/deployforme/internal/tools/models"
)

// CanonicaliseRoot canonicalises a workspace root path by making it absolute
// and resolving symlinks. Returns an error if the path doesn't exist or isn't a directory.
func CanonicaliseRoot(root string) (string, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("failed to resolve workspace root: %w", err)
	}

	// Resolve symlinks in the workspace root to get canonical path
	resolved, err := filepath.EvalSymlinks(absRoot)
	if err != nil {
		return "", fmt.Errorf("failed to resolve workspace root symlinks: %w", err)
	}

	info, err := os.Stat(resolved)
	if err != nil {
		return "", fmt.Errorf("workspace root does not exist: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("workspace root is not a directory: %s", resolved)
	}
	return resolved, nil
}

// Resolve normalises a path and ensures it's within the workspace root.
// It handles symlink resolution component-by-component, path traversal prevention,
// and validates that the resolved path stays within the workspace boundary.
// This prevents symlink escape attacks even when the final file doesn't exist.
func Resolve(ctx *models.WorkspaceContext, path string) (abs string, rel string, err error) {
	if ctx.WorkspaceRoot == "" {
		return "", "", fmt.Errorf("workspace root not set")
	}

	// Handle tilde expansion
	if strings.HasPrefix(path, "~/") {
		home, err := ctx.FS.UserHomeDir()
		if err != nil {
			return "", "", fmt.Errorf("failed to expand tilde: %w", err)
		}
		path = filepath.Join(home, path[2:])
	}

	// Clean the path
	cleaned := filepath.Clean(path)

	// If absolute, check it's within workspace
	if filepath.IsAbs(cleaned) {
		abs = cleaned
	} else {
		// Relative path - join with workspace root
		abs = filepath.Join(ctx.WorkspaceRoot, cleaned)
	}

	// Clean the absolute path
	abs = filepath.Clean(abs)

	// Resolve symlinks component-by-component to prevent escape attacks
	resolved, err := resolveSymlink(ctx, abs)
	if err != nil {
		return "", "", err
	}
	abs = resolved

	// WorkspaceRoot is already absolute and symlink-resolved
	workspaceRootAbs := filepath.Clean(ctx.WorkspaceRoot)

	// Calculate relative path
	rel, err = filepath.Rel(workspaceRootAbs, abs)
	if err != nil {
		workspaceRootWithSep := workspaceRootAbs + string(filepath.Separator)
		if abs == workspaceRootAbs {
			rel = "."
		} else if strings.HasPrefix(abs, workspaceRootWithSep) {
			rel = abs[len(workspaceRootWithSep):]
		} else {
			return "", "", models.ErrOutsideWorkspace
		}
	}

	// Segment-by-segment traversal validation
	relSegments := strings.SplitSeq(filepath.ToSlash(rel), "/")
	for segment := range relSegments {
		if segment == ".." {
			return "", "", models.ErrOutsideWorkspace
		}
	}

	// Normalise to use forward slashes for relative path
	rel = filepath.ToSlash(rel)
	if rel == "." {
		rel = ""
	}

	return abs, rel, nil
}

// resolveSymlink resolves symlinks by walking each path component.
// This prevents symlink escape attacks even when the final file doesn't exist.
// It handles missing intermediate directories gracefully to allow directory creation.
// It follows symlink chains and validates that every hop stays within the workspace boundary.
func resolveSymlink(ctx *models.WorkspaceContext, path string) (string, error) {
	workspaceRootAbs := filepath.Clean(ctx.WorkspaceRoot)
	const maxHops = 64

	// Split path into components for component-wise traversal
	parts := strings.Split(filepath.ToSlash(path), "/")
	if len(parts) == 0 {
		return path, nil
	}

	// Handle absolute paths (first component is empty on Unix)
	var currentAbs string
	startIdx := 0
	if filepath.IsAbs(path) {
		if len(parts) > 0 && parts[0] == "" {
			currentAbs = "/"
			startIdx = 1
		} else {
			currentAbs = path
		}
	} else {
		currentAbs = path
	}

	// Walk each component, resolving symlinks as we go
	for i := startIdx; i < len(parts); i++ {
		if parts[i] == "" || parts[i] == "." {
			continue
		}

		// Handle ".." by going up one directory level
		if parts[i] == ".." {
			// Go up one level
			if currentAbs == "" || currentAbs == "/" {
				// Can't go up from root
				return "", models.ErrOutsideWorkspace
			}
			currentAbs = filepath.Dir(currentAbs)
			// Validate we're still within workspace after going up
			if !isWithinWorkspace(currentAbs, workspaceRootAbs) {
				return "", models.ErrOutsideWorkspace
			}
			continue
		}

		// Build the next path component
		next := buildNextPathComponent(currentAbs, parts[i])

		// Follow symlink chain for this component
		resolved, exists, err := followSymlinkChain(ctx, next, workspaceRootAbs, maxHops)
		if err != nil {
			return "", err
		}

		if !exists {
			// Component doesn't exist - handle missing directories
			// Special case: if current equals workspace root, it's okay
			if resolved == workspaceRootAbs {
				currentAbs = resolved
				continue
			}
			// If we're not at the final component, this means a directory is missing
			// Append remaining components and return (caller can create directories)
			if i < len(parts)-1 {
				currentAbs = appendRemainingComponents(currentAbs, parts, i)
				// Validate the complete path is within workspace
				if !isWithinWorkspace(currentAbs, workspaceRootAbs) {
					return "", models.ErrOutsideWorkspace
				}
				return currentAbs, nil
			}
			// For final component, validate parent is within workspace (if we have one)
			if currentAbs != "" && currentAbs != workspaceRootAbs {
				if !isWithinWorkspace(currentAbs, workspaceRootAbs) {
					return "", models.ErrOutsideWorkspace
				}
			}
			currentAbs = resolved
			continue
		}

		currentAbs = resolved

		// Validate current path is within workspace after each step
		if !isWithinWorkspace(currentAbs, workspaceRootAbs) {
			return "", models.ErrOutsideWorkspace
		}
	}

	// Final validation that resolved path is within workspace
	if !isWithinWorkspace(currentAbs, workspaceRootAbs) {
		return "", models.ErrOutsideWorkspace
	}

	return currentAbs, nil
}

// followSymlinkChain follows a symlink chain until it reaches a non-symlink.
// Returns an error if the chain exceeds maxHops or contains a loop.
// Returns the resolved path and whether the path exists (or error if lstat fails).
func followSymlinkChain(ctx *models.WorkspaceContext, path string, workspaceRoot string, maxHops int) (resolved string, exists bool, err error) {
	visited := make(map[string]struct{})
	current := path

	for hopCount := 0; hopCount <= maxHops; hopCount++ {
		// Check for loops
		if _, seen := visited[current]; seen {
			return "", false, fmt.Errorf("symlink loop detected: %s", current)
		}
		visited[current] = struct{}{}

		// Check if current path is a symlink
		info, err := ctx.FS.Lstat(current)
		if err != nil {
			if err == os.ErrNotExist {
				return current, false, nil
			}
			return "", false, fmt.Errorf("failed to lstat path: %w", err)
		}

		// If not a symlink, we're done
		if info.Mode()&os.ModeSymlink == 0 {
			// Validate path is within workspace
			if !isWithinWorkspace(current, workspaceRoot) {
				return "", false, models.ErrOutsideWorkspace
			}
			return current, true, nil
		}

		// Read the symlink target
		linkTarget, err := ctx.FS.Readlink(current)
		if err != nil {
			return "", false, fmt.Errorf("failed to read symlink: %w", err)
		}

		// Resolve symlink target to absolute path
		var targetAbs string
		if filepath.IsAbs(linkTarget) {
			targetAbs = filepath.Clean(linkTarget)
		} else {
			// Relative symlink - resolve relative to symlink's directory
			targetAbs = filepath.Clean(filepath.Join(filepath.Dir(current), linkTarget))
		}

		// Validate symlink target is within workspace (reject immediately if outside)
		if !isWithinWorkspace(targetAbs, workspaceRoot) {
			return "", false, models.ErrOutsideWorkspace
		}

		// Continue following the chain
		current = targetAbs
	}

	return "", false, fmt.Errorf("symlink chain too long (max %d hops)", maxHops)
}

// buildNextPathComponent builds the next path component by joining current path with a part.
func buildNextPathComponent(current, part string) string {
	switch current {
	case "":
		return part
	case "/":
		return "/" + part
	default:
		return filepath.Join(current, part)
	}
}

// appendRemainingComponents appends remaining path components to the current path.
func appendRemainingComponents(current string, parts []string, start int) string {
	remaining := parts[start:]
	for j := range remaining {
		if remaining[j] == "" || remaining[j] == "." {
			continue
		}
		current = buildNextPathComponent(current, remaining[j])
	}
	return current
}

// isWithinWorkspace checks if a path is within the workspace root.
func isWithinWorkspace(path, workspaceRoot string) bool {
	workspaceRootAbs := filepath.Clean(workspaceRoot)
	pathAbs := filepath.Clean(path)

	// Check if path equals workspace root
	if pathAbs == workspaceRootAbs {
		return true
	}

	// Check if path is a subdirectory/file of workspace root
	rel, err := filepath.Rel(workspaceRootAbs, pathAbs)
	if err != nil {
		return false
	}

	// Check for path traversal attempts
	if strings.HasPrefix(rel, "..") {
		return false
	}

	// Ensure it's actually within (not just a sibling)
	workspaceRootWithSep := workspaceRootAbs + string(filepath.Separator)
	return strings.HasPrefix(pathAbs, workspaceRootWithSep)
}
