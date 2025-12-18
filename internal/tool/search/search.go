package search

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Cyclone1070/iav/internal/config"
	"github.com/Cyclone1070/iav/internal/tool/paginationutil"
	"github.com/Cyclone1070/iav/internal/tool/shell"
)

// SearchContentTool handles content searching operations.
type SearchContentTool struct {
	fs              fileSystem
	commandExecutor commandExecutor
	config          *config.Config
	workspaceRoot   string
}

// NewSearchContentTool creates a new SearchContentTool with injected dependencies.
func NewSearchContentTool(
	fs fileSystem,
	commandExecutor commandExecutor,
	cfg *config.Config,
	workspaceRoot string,
) *SearchContentTool {
	return &SearchContentTool{
		fs:              fs,
		commandExecutor: commandExecutor,
		config:          cfg,
		workspaceRoot:   workspaceRoot,
	}
}

// Run searches for content matching a regex pattern using ripgrep.
// It validates the search path is within workspace boundaries, respects gitignore rules
// (unless includeIgnored is true), and returns matches with pagination support.
func (t *SearchContentTool) Run(ctx context.Context, req *SearchContentRequest) (*SearchContentResponse, error) {
	// Runtime Validation
	absSearchPath := req.SearchAbsPath()

	// Check if search path exists
	info, err := t.fs.Stat(absSearchPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, &FileMissingError{Path: absSearchPath}
		}
		return nil, &StatError{Path: absSearchPath, Cause: err}
	}

	if !info.IsDir() {
		return nil, &NotDirectoryError{Path: absSearchPath}
	}

	// Use configured limits - constructor already checked bounds
	limit := t.config.Tools.DefaultSearchContentLimit
	if req.Limit() != 0 {
		limit = req.Limit()
	}

	maxResults := t.config.Tools.MaxSearchContentResults

	// Hard limit on line length to avoid memory issues
	maxLineLength := t.config.Tools.MaxLineLength

	// 10MB default for crazy long lines (minified JS etc)
	maxScanTokenSize := t.config.Tools.MaxScanTokenSize

	// Configure scanner buffer
	initialBufSize := t.config.Tools.InitialScannerBufferSize

	// Build ripgrep command
	// rg --json "query" searchPath [--no-ignore]
	cmd := []string{"rg", "--json"}
	if !req.CaseSensitive() {
		cmd = append(cmd, "-i")
	}
	if req.IncludeIgnored() {
		cmd = append(cmd, "--no-ignore")
	}
	cmd = append(cmd, req.Query(), absSearchPath)

	// Execute command with streaming
	proc, stdout, _, err := t.commandExecutor.Start(ctx, cmd, shell.ProcessOptions{Dir: absSearchPath})
	if err != nil {
		return nil, &CommandStartError{Cmd: "rg", Cause: err}
	}
	// process will be waited on explicitly later

	// Stream and process JSON output line by line
	var matches []SearchContentMatch
	scanner := bufio.NewScanner(stdout)
	// Increase buffer size to handle very long lines (e.g. minified JS)
	buf := make([]byte, 0, initialBufSize)
	scanner.Buffer(buf, maxScanTokenSize)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
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
			// Skip malformed lines (though rg output should be reliable)
			continue
		}

		if rgMatch.Type == "match" {
			// Convert absolute path to workspace-relative
			relPath, err := filepath.Rel(t.workspaceRoot, rgMatch.Data.Path.Text)
			if err != nil {
				// Should work if using absolute paths, but fallback to absolute if fails
				relPath = rgMatch.Data.Path.Text
			}

			lineContent := strings.TrimSpace(rgMatch.Data.Lines.Text)
			// Check if line is too long, truncate if necessary
			// This prevents returning massive lines that could crash the response
			if len(lineContent) > maxLineLength {
				lineContent = lineContent[:maxLineLength] + "...[truncated]"
			}

			matches = append(matches, SearchContentMatch{
				File:        filepath.ToSlash(relPath),
				LineNumber:  rgMatch.Data.LineNumber,
				LineContent: lineContent,
			})

			// Safety check for memory
			if len(matches) >= maxResults {
				break
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, &CommandOutputError{Cmd: "rg", Cause: err}
	}

	// Wait for command to complete
	if err := proc.Wait(); err != nil {
		exitCode := getExitCode(err)
		if exitCode == 1 {
			// rg returns 1 for no matches (valid case)
			// We just continue with empty matches
		} else {
			// Exit code 2+ = real error
			return nil, &CommandFailedError{Cmd: "rg", Cause: err}
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
	paginatedMatches, paginationResult := paginationutil.ApplyPagination(matches, req.Offset(), limit)

	return &SearchContentResponse{
		Matches:    paginatedMatches,
		Offset:     req.Offset(),
		Limit:      limit,
		TotalCount: paginationResult.TotalCount,
		Truncated:  paginationResult.Truncated,
	}, nil
}

// getExitCode extracts the exit code from an error returned by a process.
// Returns 0 if err is nil, the exit code if it's an ExitError, or -1 for unknown error types.
func getExitCode(err error) int {
	if err == nil {
		return 0
	}

	// Check for exec.ExitError (real processes)
	type exitCoder interface {
		ExitCode() int
	}
	if ec, ok := err.(exitCoder); ok {
		return ec.ExitCode()
	}

	// Unknown error type
	return -1
}
