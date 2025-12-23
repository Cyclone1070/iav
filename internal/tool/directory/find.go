package directory

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Cyclone1070/iav/internal/config"
	"github.com/Cyclone1070/iav/internal/tool/executil"
	"github.com/Cyclone1070/iav/internal/tool/paginationutil"
	"github.com/Cyclone1070/iav/internal/tool/pathutil"
)

// dirFinder defines the filesystem operations needed for finding files.
// Note: Does NOT include ListDir - this tool uses the fd command instead.
type dirFinder interface {
	Stat(path string) (os.FileInfo, error)
}

// commandExecutor defines the interface for executing shell commands.
type commandExecutor interface {
	Start(ctx context.Context, cmd []string, dir string, env []string) (executil.Process, io.Reader, io.Reader, error)
}

// FindFileTool handles file finding operations.
type FindFileTool struct {
	fs              dirFinder
	commandExecutor commandExecutor
	config          *config.Config
	pathResolver    *pathutil.Resolver
}

// NewFindFileTool creates a new FindFileTool with injected dependencies.
func NewFindFileTool(
	fs dirFinder,
	commandExecutor commandExecutor,
	cfg *config.Config,
	pathResolver *pathutil.Resolver,
) *FindFileTool {
	return &FindFileTool{
		fs:              fs,
		commandExecutor: commandExecutor,
		config:          cfg,
		pathResolver:    pathResolver,
	}
}

// Run searches for files matching a glob pattern within the workspace using the fd command.
// It supports pagination, optional ignoring of .gitignore rules, and workspace path validation.
func (t *FindFileTool) Run(ctx context.Context, req *FindFileRequest) (*FindFileResponse, error) {
	if err := req.Validate(t.config); err != nil {
		return nil, err
	}

	searchPath := req.SearchPath
	if searchPath == "" {
		searchPath = "."
	}

	absSearchPath, _, err := t.pathResolver.Resolve(searchPath)
	if err != nil {
		return nil, err
	}

	// Validate pattern syntax
	if _, err := filepath.Match(req.Pattern, ""); err != nil {
		return nil, fmt.Errorf("%w %s: %v", ErrInvalidPattern, req.Pattern, err)
	}

	// Verify search path exists and is a directory
	info, err := t.fs.Stat(absSearchPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s", ErrFileMissing, absSearchPath)
		}
		return nil, &StatError{Path: absSearchPath, Cause: err}
	}

	if !info.IsDir() {
		return nil, fmt.Errorf("%w: %s", ErrNotADirectory, absSearchPath)
	}

	// Use configured limits
	limit := t.config.Tools.DefaultFindFileLimit
	if req.Limit != 0 {
		limit = req.Limit
	}

	// fd --glob "pattern" searchPath
	cmd := []string{"fd", "--glob", req.Pattern, absSearchPath}

	// Handle ignored files
	if req.IncludeIgnored {
		cmd = append(cmd, "--no-ignore", "--hidden")
	}

	// Max depth
	if req.MaxDepth > 0 {
		cmd = append(cmd, "--max-depth", fmt.Sprintf("%d", req.MaxDepth))
	}

	// Execute command with streaming
	proc, stdout, _, err := t.commandExecutor.Start(ctx, cmd, absSearchPath, nil)
	if err != nil {
		return nil, &executil.CommandError{Cmd: "fd", Cause: err, Stage: "start"}
	}
	// We will wait explicitly to check for errors

	// Capture all output to safe buffer with limit
	// We read all matches then slice, as fd doesn't support offset/limit natively in a way that guarantees consistent sorting without reading all.
	// For massive result sets, this could be optimized, but for now we rely on MaxFindFileResults cap.

	// Max results hard cap for memory safety
	maxResults := t.config.Tools.MaxFindFileResults

	var matches []string
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Convert absolute to relative
		relPath, err := filepath.Rel(t.pathResolver.WorkspaceRoot(), line)
		if err != nil {
			relPath = line // Fallback
		}
		matches = append(matches, filepath.ToSlash(relPath))

		if len(matches) >= maxResults {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		// Try to wait to clean up process even on scan error
		_ = proc.Wait()
		return nil, &executil.CommandError{Cmd: "fd", Cause: err, Stage: "read output"}
	}

	// Check command exit status
	if err := proc.Wait(); err != nil {
		return nil, &executil.CommandError{Cmd: "fd", Cause: err, Stage: "execution"}
	}

	// Sort ensures consistent pagination
	sort.Strings(matches)

	// Apply pagination
	paginatedMatches, paginationResult := paginationutil.ApplyPagination(matches, req.Offset, limit)

	return &FindFileResponse{
		Matches:    paginatedMatches,
		Offset:     req.Offset,
		Limit:      limit,
		TotalCount: paginationResult.TotalCount,
		Truncated:  paginationResult.Truncated,
	}, nil
}
