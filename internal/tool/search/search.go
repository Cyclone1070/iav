package search

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Cyclone1070/iav/internal/config"
	"github.com/Cyclone1070/iav/internal/tool/helper/pagination"
	"github.com/Cyclone1070/iav/internal/tool/service/executor"
)

// SearchContentTool handles content searching operations.
type SearchContentTool struct {
	fs              fileSystem
	commandExecutor commandExecutor
	config          *config.Config
	pathResolver    pathResolver
}

// NewSearchContentTool creates a new SearchContentTool with injected dependencies.
func NewSearchContentTool(
	fs fileSystem,
	commandExecutor commandExecutor,
	cfg *config.Config,
	pathResolver pathResolver,
) *SearchContentTool {
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
	return &SearchContentTool{
		fs:              fs,
		commandExecutor: commandExecutor,
		config:          cfg,
		pathResolver:    pathResolver,
	}
}

// Run searches for content matching a regex pattern using ripgrep.
// It validates the search path is within workspace boundaries, respects gitignore rules
// (unless includeIgnored is true), and returns matches with pagination support.
func (t *SearchContentTool) Run(ctx context.Context, req *SearchContentRequest) (*SearchContentResponse, error) {
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

	// Check if search path exists
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

	limit := req.Limit

	maxResults := t.config.Tools.MaxSearchContentResults

	// Hard limit on line length to avoid memory issues
	maxLineLength := t.config.Tools.MaxLineLength

	// Build ripgrep command
	// rg --json "query" searchPath [--no-ignore]
	cmd := []string{"rg", "--json"}
	if !req.CaseSensitive {
		cmd = append(cmd, "-i")
	}
	if req.IncludeIgnored {
		cmd = append(cmd, "--no-ignore")
	}
	cmd = append(cmd, req.Query, absSearchPath)

	// Execute command
	res, err := t.commandExecutor.Run(ctx, cmd, absSearchPath, nil)
	if err != nil && (res == nil || res.ExitCode != 1) { // rg returns 1 for no matches
		return nil, &executor.CommandError{Cmd: "rg", Cause: err, Stage: "execution"}
	}

	// Process output
	var matches []SearchContentMatch
	if res == nil {
		return &SearchContentResponse{
			Matches:    nil,
			Offset:     req.Offset,
			Limit:      limit,
			TotalCount: 0,
			Truncated:  false,
		}, nil
	}
	lines := strings.Split(res.Stdout, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Parse JSON line
		var rgMatch struct {
			Type string `json:"type"`
			Data struct {
				Path struct {
					Text string `json:"text"`
				} `json:"path"`
				Lines struct {
					Text string `json:"text"`
				} `json:"lines"`
				LineNumber int `json:"line_number"`
			} `json:"data"`
		}

		if err := json.Unmarshal([]byte(line), &rgMatch); err != nil {
			continue
		}

		if rgMatch.Type == "match" {
			relPath, err := t.pathResolver.Rel(rgMatch.Data.Path.Text)
			if err != nil {
				relPath = rgMatch.Data.Path.Text
			}

			lineContent := strings.TrimSpace(rgMatch.Data.Lines.Text)
			if len(lineContent) > maxLineLength {
				lineContent = lineContent[:maxLineLength] + "...[truncated]"
			}

			matches = append(matches, SearchContentMatch{
				File:        filepath.ToSlash(relPath),
				LineNumber:  rgMatch.Data.LineNumber,
				LineContent: lineContent,
			})

			if len(matches) >= maxResults {
				break
			}
		}
	}

	// Sort results for consistency (by file, then line number)
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].File != matches[j].File {
			return matches[i].File < matches[j].File
		}
		return matches[i].LineNumber < matches[j].LineNumber
	})

	// Apply pagination
	paginatedMatches, paginationResult := pagination.ApplyPagination(matches, req.Offset, limit)

	return &SearchContentResponse{
		Matches:    paginatedMatches,
		Offset:     req.Offset,
		Limit:      limit,
		TotalCount: paginationResult.TotalCount,
		Truncated:  paginationResult.Truncated,
	}, nil
}
