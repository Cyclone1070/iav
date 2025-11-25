// Package main provides a command-line interface for the deployagent tool.
// It supports file operations (read, write, edit, list) and shell command execution.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/Cyclone1070/deployforme/internal/tools"
	"github.com/Cyclone1070/deployforme/internal/tools/models"
	"github.com/Cyclone1070/deployforme/internal/tools/services"
)

func main() {
	// Get current working directory as workspace root
	workspaceRoot, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to get working directory: %v\n", err)
		os.Exit(1)
	}

	// Initialize filesystem service
	fileSystem := services.NewOSFileSystem(models.DefaultMaxFileSize)

	// Initialize binary detector
	binaryDetector := &services.SystemBinaryDetector{}

	// Initialize checksum manager
	checksumMgr := services.NewChecksumManager()

	// Initialize gitignore service
	gitignoreSvc, err := services.NewGitignoreService(workspaceRoot, fileSystem)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to initialize gitignore service: %v\n", err)
	}

	// Create workspace context
	ctx := &models.WorkspaceContext{
		FS:               fileSystem,
		BinaryDetector:   binaryDetector,
		ChecksumManager:  checksumMgr,
		MaxFileSize:      models.DefaultMaxFileSize,
		WorkspaceRoot:    workspaceRoot,
		GitignoreService: gitignoreSvc,
		CommandExecutor:  &services.OSCommandExecutor{},
		DockerConfig: models.DockerConfig{
			CheckCommand: []string{"docker", "info"},
			StartCommand: []string{"docker", "desktop", "start"},
		},
	}

	// Parse command-line arguments
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "read":
		handleRead(ctx, os.Args[2:])
	case "write":
		handleWrite(ctx, os.Args[2:])
	case "edit":
		handleEdit(ctx, os.Args[2:])
	case "list":
		handleList(ctx, os.Args[2:])
	case "shell":
		handleShell(ctx, os.Args[2:])
	case "search":
		handleSearch(ctx, os.Args[2:])
	case "find":
		handleFindFile(ctx, os.Args[2:])
	case "todoread":
		handleTodoRead(ctx, os.Args[2:])
	case "todowrite":
		handleTodoWrite(ctx, os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

// printUsage prints the command-line usage information to stderr.
func printUsage() {
	fmt.Fprintf(os.Stderr, `Usage: deployagent <command> [args...]

Commands:
  read <path> [offset] [limit]              Read a file
  write <path> <content> [perm]             Write a file
  edit <path> <ops_json>                    Edit a file with operations
  list <path> [offset] [limit]              List directory contents
  shell <cmd> [args...]                     Execute a shell command
  search <query> <path> [caseSensitive] [offset] [limit]    Search file contents
  find <pattern> <path> [maxDepth] [offset] [limit]        Find files by pattern
  todoread                                   Read all todos
  todowrite <todos_json>                    Write todos from JSON array

Examples:
  deployagent read README.md
  deployagent read README.md 0 100
  deployagent write test.txt "hello world"
  deployagent list .
  deployagent list . 0 10
  deployagent shell echo hello
  deployagent search "func main" . false 0 10
  deployagent find "*.go" . 2 0 50
  deployagent todoread
  deployagent todowrite '[{"description":"Task 1","status":"pending"}]'
`)
}

// handleRead handles the 'read' command to read files.
func handleRead(ctx *models.WorkspaceContext, args []string) {
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "usage: read <path> [offset] [limit]\n")
		os.Exit(1)
	}

	req := models.ReadFileRequest{
		Path: args[0],
	}

	if len(args) > 1 {
		o, err := strconv.ParseInt(args[1], 10, 64)
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid offset: %v\n", err)
			os.Exit(1)
		}
		req.Offset = &o
	}

	if len(args) > 2 {
		l, err := strconv.ParseInt(args[2], 10, 64)
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid limit: %v\n", err)
			os.Exit(1)
		}
		req.Limit = &l
	}

	resp, err := tools.ReadFile(ctx, req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	output, _ := json.MarshalIndent(resp, "", "  ")
	fmt.Println(string(output))
}

// handleWrite handles the 'write' command to create new files.
// Supports optional file permission// handleWrite handles the 'write' command to create new files.
func handleWrite(ctx *models.WorkspaceContext, args []string) {
	if len(args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: write <path> <content> [perm]\n")
		os.Exit(1)
	}

	req := models.WriteFileRequest{
		Path:    args[0],
		Content: args[1],
	}

	if len(args) > 2 {
		p, err := strconv.ParseUint(args[2], 8, 32)
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid permission: %v\n", err)
			os.Exit(1)
		}
		mode := os.FileMode(p)
		req.Perm = &mode
	}

	resp, err := tools.WriteFile(ctx, req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	output, _ := json.MarshalIndent(resp, "", "  ")
	fmt.Println(string(output))
}

// handleEdit handles the 'edit' command to apply edit operations to existing files.
// Operations must be provided as a JSON array.
func handleEdit(ctx *models.WorkspaceContext, args []string) {
	if len(args) < 2 {
		fmt.Fprintf(os.Stderr, "edit: path and operations JSON required\n")
		os.Exit(1)
	}

	path := args[0]
	opsJSON := args[1]

	var ops []models.Operation
	if err := json.Unmarshal([]byte(opsJSON), &ops); err != nil {
		fmt.Fprintf(os.Stderr, "invalid operations JSON: %v\n", err)
		os.Exit(1)
	}

	resp, err := tools.EditFile(ctx, models.EditFileRequest{Path: path, Operations: ops})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	output, _ := json.MarshalIndent(resp, "", "  ")
	fmt.Println(string(output))
}

// handleList handles the 'list' command to list directory contents.
// Supports optional offset and limit parameters for pagination.
func handleList(ctx *models.WorkspaceContext, args []string) {
	path := "."
	offset := 0
	limit := models.DefaultListDirectoryLimit

	if len(args) > 0 {
		path = args[0]
	}

	if len(args) > 1 {
		o, err := strconv.Atoi(args[1])
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid offset: %v\n", err)
			os.Exit(1)
		}
		offset = o
	}

	if len(args) > 2 {
		l, err := strconv.Atoi(args[2])
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid limit: %v\n", err)
			os.Exit(1)
		}
		limit = l
	}

	resp, err := tools.ListDirectory(ctx, models.ListDirectoryRequest{Path: path, MaxDepth: -1, Offset: offset, Limit: limit})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	output, _ := json.MarshalIndent(resp, "", "  ")
	fmt.Println(string(output))
}

// handleShell handles the 'shell' command to execute shell commands.
// Supports flags for working directory, timeout, and environment variables.
func handleShell(ctx *models.WorkspaceContext, args []string) {
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "shell: command required\n")
		os.Exit(1)
	}

	// Parse flags
	cmdFlagSet := flag.NewFlagSet("shell", flag.ContinueOnError)
	workingDir := cmdFlagSet.String("dir", "", "working directory")
	timeout := cmdFlagSet.Int("timeout", 0, "timeout in seconds (0 = default 1 hour)")
	envStr := cmdFlagSet.String("env", "", "environment variables as KEY=VALUE,KEY=VALUE")

	// Find where flags end
	var cmdArgs []string
	flagEnd := 0
	for i, arg := range args {
		if !strings.HasPrefix(arg, "-") {
			flagEnd = i
			break
		}
	}

	if flagEnd > 0 {
		if err := cmdFlagSet.Parse(args[:flagEnd]); err != nil {
			fmt.Fprintf(os.Stderr, "error parsing flags: %v\n", err)
			os.Exit(1)
		}
		cmdArgs = args[flagEnd:]
	} else {
		cmdArgs = args
	}

	if len(cmdArgs) == 0 {
		fmt.Fprintf(os.Stderr, "shell: command required\n")
		os.Exit(1)
	}

	// Parse environment variables
	env := make(map[string]string)
	if *envStr != "" {
		for pair := range strings.SplitSeq(*envStr, ",") {
			parts := strings.SplitN(pair, "=", 2)
			if len(parts) == 2 {
				env[parts[0]] = parts[1]
			}
		}
	}

	req := models.ShellRequest{
		Command:        cmdArgs,
		WorkingDir:     *workingDir,
		TimeoutSeconds: *timeout,
		Env:            env,
	}

	shellTool := &tools.ShellTool{
		CommandExecutor: &services.OSCommandExecutor{},
	}

	resp, err := shellTool.Run(context.Background(), ctx, req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		if resp != nil {
			fmt.Printf("stdout: %s\n", resp.Stdout)
			fmt.Printf("stderr: %s\n", resp.Stderr)
		}
		os.Exit(1)
	}

	output, _ := json.MarshalIndent(resp, "", "  ")
	fmt.Println(string(output))
}

// handleSearch handles the 'search' command to search file contents.
// Supports optional caseSensitive, offset, and limit parameters.
func handleSearch(ctx *models.WorkspaceContext, args []string) {
	if len(args) < 2 {
		fmt.Fprintf(os.Stderr, "search: query and path required\n")
		os.Exit(1)
	}

	query := args[0]
	searchPath := args[1]
	caseSensitive := false
	offset := 0
	limit := 100

	if len(args) > 2 {
		cs, err := strconv.ParseBool(args[2])
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid caseSensitive: %v\n", err)
			os.Exit(1)
		}
		caseSensitive = cs
	}

	if len(args) > 3 {
		o, err := strconv.Atoi(args[3])
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid offset: %v\n", err)
			os.Exit(1)
		}
		offset = o
	}

	if len(args) > 4 {
		l, err := strconv.Atoi(args[4])
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid limit: %v\n", err)
			os.Exit(1)
		}
		limit = l
	}

	resp, err := tools.SearchContent(ctx, models.SearchContentRequest{Query: query, SearchPath: searchPath, CaseSensitive: caseSensitive, IncludeIgnored: false, Offset: offset, Limit: limit})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	output, _ := json.MarshalIndent(resp, "", "  ")
	fmt.Println(string(output))
}

// handleFindFile handles the 'find' command to search for files by pattern.
// Supports optional maxDepth, offset, and limit parameters.
func handleFindFile(ctx *models.WorkspaceContext, args []string) {
	if len(args) < 2 {
		fmt.Fprintf(os.Stderr, "find: pattern and path required\n")
		os.Exit(1)
	}

	pattern := args[0]
	searchPath := args[1]
	maxDepth := -1
	offset := 0
	limit := 100

	if len(args) > 2 {
		md, err := strconv.Atoi(args[2])
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid maxDepth: %v\n", err)
			os.Exit(1)
		}
		maxDepth = md
	}

	if len(args) > 3 {
		o, err := strconv.Atoi(args[3])
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid offset: %v\n", err)
			os.Exit(1)
		}
		offset = o
	}

	if len(args) > 4 {
		l, err := strconv.Atoi(args[4])
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid limit: %v\n", err)
			os.Exit(1)
		}
		limit = l
	}

	resp, err := tools.FindFile(ctx, models.FindFileRequest{Pattern: pattern, SearchPath: searchPath, MaxDepth: maxDepth, IncludeIgnored: false, Offset: offset, Limit: limit})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	output, _ := json.MarshalIndent(resp, "", "  ")
	fmt.Println(string(output))
}

// handleTodoRead handles the 'todoread' command to read all todos.
func handleTodoRead(ctx *models.WorkspaceContext, args []string) {
	resp, err := tools.ReadTodos(ctx, models.ReadTodosRequest{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	output, _ := json.MarshalIndent(resp, "", "  ")
	fmt.Println(string(output))
}

// handleTodoWrite handles the 'todowrite' command to write todos from JSON.
func handleTodoWrite(ctx *models.WorkspaceContext, args []string) {
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "todowrite: todos JSON required\n")
		os.Exit(1)
	}

	todosJSON := args[0]

	var todos []models.Todo
	if err := json.Unmarshal([]byte(todosJSON), &todos); err != nil {
		fmt.Fprintf(os.Stderr, "invalid todos JSON: %v\n", err)
		os.Exit(1)
	}

	resp, err := tools.WriteTodos(ctx, models.WriteTodosRequest{Todos: todos})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	output, _ := json.MarshalIndent(resp, "", "  ")
	fmt.Println(string(output))
}
