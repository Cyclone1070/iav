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
		CommandPolicy: models.CommandPolicy{
			Allow:        []string{"git", "docker", "npm", "go", "python", "bash", "sh", "ls"},
			Ask:          []string{},
			SessionAllow: make(map[string]bool),
		},
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
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `Usage: deployagent <command> [args...]

Commands:
  read <path> [offset] [limit]              Read a file
  write <path> <content> [perm]             Write a file
  edit <path> <ops_json>                    Edit a file with operations
  list <path> [offset] [limit]              List directory contents
  shell <cmd> [args...]                     Execute a shell command

Examples:
  deployagent read README.md
  deployagent read README.md 0 100
  deployagent write test.txt "hello world"
  deployagent list .
  deployagent list . 0 10
  deployagent shell echo hello
`)
}

func handleRead(ctx *models.WorkspaceContext, args []string) {
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "read: path required\n")
		os.Exit(1)
	}

	path := args[0]
	var offset, limit *int64

	if len(args) > 1 {
		o, err := strconv.ParseInt(args[1], 10, 64)
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid offset: %v\n", err)
			os.Exit(1)
		}
		offset = &o
	}

	if len(args) > 2 {
		l, err := strconv.ParseInt(args[2], 10, 64)
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid limit: %v\n", err)
			os.Exit(1)
		}
		limit = &l
	}

	resp, err := tools.ReadFile(ctx, path, offset, limit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	output, _ := json.MarshalIndent(resp, "", "  ")
	fmt.Println(string(output))
}

func handleWrite(ctx *models.WorkspaceContext, args []string) {
	if len(args) < 2 {
		fmt.Fprintf(os.Stderr, "write: path and content required\n")
		os.Exit(1)
	}

	path := args[0]
	content := args[1]
	var perm *os.FileMode

	if len(args) > 2 {
		p, err := strconv.ParseUint(args[2], 8, 32)
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid permission: %v\n", err)
			os.Exit(1)
		}
		mode := os.FileMode(p)
		perm = &mode
	}

	resp, err := tools.WriteFile(ctx, path, content, perm)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	output, _ := json.MarshalIndent(resp, "", "  ")
	fmt.Println(string(output))
}

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

	resp, err := tools.EditFile(ctx, path, ops)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	output, _ := json.MarshalIndent(resp, "", "  ")
	fmt.Println(string(output))
}

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

	resp, err := tools.ListDirectory(ctx, path, offset, limit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	output, _ := json.MarshalIndent(resp, "", "  ")
	fmt.Println(string(output))
}

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
		for _, pair := range strings.Split(*envStr, ",") {
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
		ProcessFactory: &services.OSProcessFactory{},
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
