package directory

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Cyclone1070/iav/internal/config"
	"github.com/Cyclone1070/iav/internal/tool/executor"
	"github.com/Cyclone1070/iav/internal/tool/helper/pagination"
)

// FindFileTool handles file finding operations.
type FindFileTool struct {
	fs              dirFinder
	commandExecutor commandExecutor
	config          *config.Config
	pathResolver    pathResolver
}

// NewFindFileTool creates a new FindFileTool with injected dependencies.
func NewFindFileTool(
	fs dirFinder,
	commandExecutor commandExecutor,
	cfg *config.Config,
	pathResolver pathResolver,
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

	absSearchPath, err := t.pathResolver.Abs(searchPath)
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

	// Execute command
	res, err := t.commandExecutor.Run(ctx, cmd, absSearchPath, nil)
	if err != nil {
		return nil, &executor.CommandError{Cmd: "fd", Cause: err, Stage: "execution"}
	}
	if res == nil {
		return nil, &executor.CommandError{Cmd: "fd", Cause: fmt.Errorf("no result"), Stage: "execution"}
	}

	// Capture all output
	maxResults := t.config.Tools.MaxFindFileResults

	var matches []string
	lines := strings.Split(res.Stdout, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		relPath, err := t.pathResolver.Rel(line)
		if err != nil {
			relPath = line
		}
		matches = append(matches, filepath.ToSlash(relPath))

		if len(matches) >= maxResults {
			break
		}
	}

	// Sort ensures consistent pagination
	sort.Strings(matches)

	// Apply pagination
	paginatedMatches, paginationResult := pagination.ApplyPagination(matches, req.Offset, limit)

	return &FindFileResponse{
		Matches:    paginatedMatches,
		Offset:     req.Offset,
		Limit:      limit,
		TotalCount: paginationResult.TotalCount,
		Truncated:  paginationResult.Truncated,
	}, nil
}
