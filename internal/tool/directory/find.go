package directory

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Cyclone1070/iav/internal/config"
	"github.com/Cyclone1070/iav/internal/tool/helper/pagination"
	"github.com/Cyclone1070/iav/internal/tool/service/executor"
)

// dirFinder defines the filesystem operations needed for finding files.
// Note: Does NOT include ListDir - this tool uses the fd command instead.
type dirFinder interface {
	Stat(path string) (os.FileInfo, error)
}

// commandExecutor defines the interface for executing find commands.
// This is a consumer-defined interface per architecture guidelines §3.
type commandExecutor interface {
	Run(ctx context.Context, cmd []string, dir string, env []string) (*executor.Result, error)
}

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
	if fs == nil {
		panic("fs is required")
	}
	if commandExecutor == nil {
		panic("commandExecutor is required")
	}
	if cfg == nil {
		panic("cfg is required")
	}
	if pathResolver == nil {
		panic("pathResolver is required")
	}
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
		return nil, fmt.Errorf("invalid pattern %s: %v", req.Pattern, err)
	}

	// Verify search path exists and is a directory
	info, err := t.fs.Stat(absSearchPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("path does not exist: %s", absSearchPath)
		}
		return nil, fmt.Errorf("failed to stat %s: %w", absSearchPath, err)
	}

	if !info.IsDir() {
		return nil, fmt.Errorf("not a directory: %s", absSearchPath)
	}

	limit := req.Limit

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
		return nil, fmt.Errorf("fd failed to start: %w", err)
	}

	// res is guaranteed non-nil if err is nil.
	// fd returns exit 1 if no files are found (not an error for us).
	if res.ExitCode != 0 && res.ExitCode != 1 {
		return nil, fmt.Errorf("fd failed with exit code %d: %s", res.ExitCode, res.Stderr)
	}

	// Capture all output
	maxResults := t.config.Tools.MaxFindFileResults

	var matches []string
	lines := strings.SplitSeq(res.Stdout, "\n")
	for line := range lines {
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
