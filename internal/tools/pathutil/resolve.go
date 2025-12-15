package pathutil

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	toolserrors "github.com/Cyclone1070/iav/internal/tools/errors"
)

// FileSystem defines the minimal filesystem interface needed for path resolution.
// This is a consumer-defined interface per architecture guidelines §2.
type FileSystem interface {
	Lstat(path string) (os.FileInfo, error)
	Readlink(path string) (string, error)
	UserHomeDir() (string, error)
}

// CanonicaliseRoot canonicalises a workspace root path by making it absolute and resolving symlinks.
// Returns an error if the path doesn't exist or isn't a directory.
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
// It handles symlink resolution component-by-component, prevents path traversal attacks,
// and validates that the resolved path stays within the workspace boundary.
// Returns the absolute path, relative path, and any error.
func Resolve(workspaceRoot string, fs FileSystem, path string) (abs string, rel string, err error) {
	if workspaceRoot == "" {
		return "", "", fmt.Errorf("workspace root not set")
	}

	// Handle tilde expansion
	if strings.HasPrefix(path, "~/") {
		home, err := fs.UserHomeDir()
		if err != nil {
			return "", "", fmt.Errorf("failed to expand tilde: %w", err)
		}
		path = filepath.Join(home, path[2:])
	}

	// Get absolute path of the input to ensure we can calculate relation to root
	var absInput string
	if filepath.IsAbs(path) {
		absInput = filepath.Clean(path)
	} else {
		absInput = filepath.Join(workspaceRoot, path)
	}

	workspaceRootAbs := filepath.Clean(workspaceRoot)

	// Calculate initial relative path to see if it's lexically within root
	// We use filepath.Rel which handles cleaning
	relPath, err := filepath.Rel(workspaceRootAbs, absInput)
	if err != nil {
		return "", "", toolserrors.ErrOutsideWorkspace
	}

	// Check for path traversal attempts in the relative path
	if strings.HasPrefix(relPath, "..") {
		return "", "", toolserrors.ErrOutsideWorkspace
	}

	// If the path is just the root, we're done
	if relPath == "." {
		return workspaceRootAbs, "", nil
	}

	// Resolve symlinks component-by-component using the relative path
	// This ensures we only validate components *inside* the workspace
	resolvedAbs, err := resolveRelativePath(workspaceRoot, fs, relPath)
	if err != nil {
		return "", "", err
	}

	// Calculate final relative path from the resolved absolute path
	finalRel, err := filepath.Rel(workspaceRootAbs, resolvedAbs)
	if err != nil {
		return "", "", toolserrors.ErrOutsideWorkspace
	}

	// Normalise to use forward slashes for relative path
	finalRel = filepath.ToSlash(finalRel)
	if finalRel == "." {
		finalRel = ""
	}

	return resolvedAbs, finalRel, nil
}

// resolveRelativePath resolves a relative path component-by-component with symlink resolution.
// It assumes the input relPath is lexically within the workspace (does not start with ..).
// Returns the resolved absolute path or an error if the path escapes the workspace.
func resolveRelativePath(workspaceRoot string, fs FileSystem, relPath string) (string, error) {
	workspaceRootAbs := filepath.Clean(workspaceRoot)
	const maxHops = 64

	// Split path into components for component-wise traversal
	parts := strings.Split(filepath.ToSlash(relPath), "/")
	if len(parts) == 0 {
		return workspaceRootAbs, nil
	}

	currentAbs := workspaceRootAbs

	// Walk each component, resolving symlinks as we go
	for i := range parts {
		if parts[i] == "" || parts[i] == "." {
			continue
		}

		// Handle ".." by going up one directory level
		if parts[i] == ".." {
			// Go up one level
			if currentAbs == workspaceRootAbs {
				// Can't go up from root
				return "", toolserrors.ErrOutsideWorkspace
			}
			currentAbs = filepath.Dir(currentAbs)
			// Validate we're still within workspace after going up
			if !isWithinWorkspace(currentAbs, workspaceRootAbs) {
				return "", toolserrors.ErrOutsideWorkspace
			}
			continue
		}

		// Build the next path component
		next := filepath.Join(currentAbs, parts[i])

		// Follow symlink chain for this component
		resolved, exists, err := followSymlinkChain(fs, next, workspaceRootAbs, maxHops)
		if err != nil {
			return "", err
		}

		if !exists {
			// Component doesn't exist - handle missing directories
			// If we're not at the final component, this means a directory is missing
			// Append remaining components and return (caller can create directories)
			if i < len(parts)-1 {
				currentAbs = appendRemainingComponents(currentAbs, parts, i)
				// Validate the complete path is within workspace
				if !isWithinWorkspace(currentAbs, workspaceRootAbs) {
					return "", toolserrors.ErrOutsideWorkspace
				}
				return currentAbs, nil
			}
			// For final component, validate parent is within workspace
			// (currentAbs is the parent here)
			if !isWithinWorkspace(currentAbs, workspaceRootAbs) {
				return "", toolserrors.ErrOutsideWorkspace
			}
			currentAbs = resolved
			continue
		}

		currentAbs = resolved

		// Validate current path is within workspace after each step
		if !isWithinWorkspace(currentAbs, workspaceRootAbs) {
			return "", toolserrors.ErrOutsideWorkspace
		}
	}

	return currentAbs, nil
}

// followSymlinkChain follows a symlink chain until it reaches a non-symlink or detects a loop.
// Returns the resolved path, whether the path exists, and any error.
// Returns an error if the chain exceeds maxHops or escapes the workspace.
func followSymlinkChain(fs FileSystem, path string, workspaceRoot string, maxHops int) (resolved string, exists bool, err error) {
	visited := make(map[string]struct{})
	current := path

	for hopCount := 0; hopCount <= maxHops; hopCount++ {
		// Check for loops
		if _, seen := visited[current]; seen {
			return "", false, fmt.Errorf("symlink loop detected: %s", current)
		}
		visited[current] = struct{}{}

		// Check if current path is a symlink
		info, err := fs.Lstat(current)
		if err != nil {
			if os.IsNotExist(err) {
				return current, false, nil
			}
			return "", false, fmt.Errorf("failed to lstat path: %w", err)
		}

		// If not a symlink, we're done
		if info.Mode()&os.ModeSymlink == 0 {
			// Validate path is within workspace
			if !isWithinWorkspace(current, workspaceRoot) {
				return "", false, toolserrors.ErrOutsideWorkspace
			}
			return current, true, nil
		}

		// Read the symlink target
		linkTarget, err := fs.Readlink(current)
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
			return "", false, toolserrors.ErrOutsideWorkspace
		}

		// Continue following the chain
		current = targetAbs
	}

	return "", false, fmt.Errorf("symlink chain too long (max %d hops)", maxHops)
}

// buildNextPathComponent joins a path component to the current path, handling edge cases.
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

// isWithinWorkspace checks if a path is within the workspace root boundary.
// Returns true if the path is the workspace root or a subdirectory/file within it.
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
