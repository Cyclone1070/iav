// Package main provides a command-line interface for the iav tool.
// It supports file operations (read, write, edit, list) and shell command execution.
package main

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/Cyclone1070/iav/internal/config"
	"github.com/Cyclone1070/iav/internal/orchestrator"
	orchadapter "github.com/Cyclone1070/iav/internal/orchestrator/adapter"
	"github.com/Cyclone1070/iav/internal/provider/gemini"
	provider "github.com/Cyclone1070/iav/internal/provider/model"
	"github.com/Cyclone1070/iav/internal/tool/contentutil"
	"github.com/Cyclone1070/iav/internal/tool/directory"
	"github.com/Cyclone1070/iav/internal/tool/file"
	"github.com/Cyclone1070/iav/internal/tool/fsutil"
	"github.com/Cyclone1070/iav/internal/tool/gitutil"
	"github.com/Cyclone1070/iav/internal/tool/hashutil"
	"github.com/Cyclone1070/iav/internal/tool/pathutil"
	"github.com/Cyclone1070/iav/internal/tool/search"
	"github.com/Cyclone1070/iav/internal/tool/shell"
	"github.com/Cyclone1070/iav/internal/tool/todo"
	"github.com/Cyclone1070/iav/internal/ui"
	uiservices "github.com/Cyclone1070/iav/internal/ui/service"
	"github.com/charmbracelet/bubbles/spinner"
	"google.golang.org/genai"
)

// Dependencies holds the components required to run the application.
type Dependencies struct {
	Config          *config.Config
	UI              ui.UserInterface
	ProviderFactory func(context.Context) (provider.Provider, error)
	Tools           []orchadapter.Tool
}

func createRealUI(cfg *config.Config) ui.UserInterface {
	channels := ui.NewUIChannels(cfg)
	renderer := uiservices.NewGlamourRenderer()
	spinnerFactory := func() spinner.Model {
		return spinner.New(spinner.WithSpinner(spinner.Dot))
	}
	return ui.NewUI(channels, renderer, spinnerFactory)
}

func createRealProviderFactory(cfg *config.Config) func(context.Context) (provider.Provider, error) {
	return func(ctx context.Context) (provider.Provider, error) {
		apiKey := os.Getenv("GEMINI_API_KEY")
		if apiKey == "" {
			return nil, fmt.Errorf("GEMINI_API_KEY environment variable is required")
		}

		genaiClient, err := genai.NewClient(ctx, &genai.ClientConfig{APIKey: apiKey})
		if err != nil {
			return nil, fmt.Errorf("failed to create Gemini client: %w", err)
		}

		geminiClient := gemini.NewRealGeminiClient(genaiClient)
		return gemini.NewGeminiProviderWithLatest(ctx, cfg, geminiClient)
	}
}

func createTools(cfg *config.Config, workspaceRoot string) ([]orchadapter.Tool, error) {
	// Canonicalize workspace root
	canonicalRoot, err := pathutil.CanonicaliseRoot(workspaceRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to canonicalize workspace root: %w", err)
	}

	// Instantiate concrete dependencies
	osFS := fsutil.NewOSFileSystem()
	binaryDetector := contentutil.NewSystemBinaryDetector(cfg.Tools.BinaryDetectionSampleSize)
	checksumManager := hashutil.NewChecksumManager()
	commandExecutor := shell.NewOSCommandExecutor()
	todoStore := todo.NewInMemoryTodoStore()

	// Initialize gitignore service
	var gitignoreService interface {
		ShouldIgnore(relativePath string) bool
	}
	svc, err := gitutil.NewService(canonicalRoot, osFS)
	if err != nil {
		// Log error but continue with NoOpService
		fmt.Fprintf(os.Stderr, "Warning: failed to initialize gitignore service: %v\n", err)
		gitignoreService = &gitutil.NoOpService{}
	} else {
		gitignoreService = svc
	}

	// Docker configuration
	dockerConfig := shell.DockerConfig{
		CheckCommand: []string{"docker", "info"},
		StartCommand: []string{"open", "-a", "Docker"},
	}

	// Instantiate all tools with their dependencies
	readFileTool := file.NewReadFileTool(osFS, binaryDetector, checksumManager, cfg, canonicalRoot)
	writeFileTool := file.NewWriteFileTool(osFS, binaryDetector, checksumManager, cfg, canonicalRoot)
	editFileTool := file.NewEditFileTool(osFS, binaryDetector, checksumManager, cfg, canonicalRoot)
	listDirectoryTool := directory.NewListDirectoryTool(osFS, gitignoreService, cfg, canonicalRoot)
	findFileTool := directory.NewFindFileTool(osFS, commandExecutor, cfg, canonicalRoot)
	searchContentTool := search.NewSearchContentTool(osFS, commandExecutor, cfg, canonicalRoot)
	shellTool := shell.NewShellTool(osFS, commandExecutor, cfg, dockerConfig, canonicalRoot)
	readTodosTool := todo.NewReadTodosTool(todoStore, cfg)
	writeTodosTool := todo.NewWriteTodosTool(todoStore, cfg)

	// Create adapters
	return []orchadapter.Tool{
		orchadapter.NewReadFileAdapter(readFileTool),
		orchadapter.NewWriteFileAdapter(writeFileTool),
		orchadapter.NewEditFileAdapter(editFileTool),
		orchadapter.NewListDirectoryAdapter(listDirectoryTool),
		orchadapter.NewFindFileAdapter(findFileTool),
		orchadapter.NewSearchContentAdapter(searchContentTool),
		orchadapter.NewShellAdapter(shellTool),
		orchadapter.NewReadTodosAdapter(readTodosTool),
		orchadapter.NewWriteTodosAdapter(writeTodosTool),
	}, nil
}

func main() {
	// Load configuration (from defaults + ~/.config/iav/config.json)
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to load config: %v\n", err)
		fmt.Fprintf(os.Stderr, "Using default configuration.\n")
		cfg = config.DefaultConfig()
	}

	// Create dependencies
	deps := Dependencies{
		Config:          cfg,
		UI:              createRealUI(cfg),
		ProviderFactory: createRealProviderFactory(cfg),
		Tools:           nil, // Will be created in runInteractive
	}

	// Run interactive mode (blocks until exit)
	// NOTE: We use context.Background() intentionally for TUI mode because:
	// 1. The UI manages its own lifecycle via Ctrl+C / Quit messages
	// 2. We don't need external cancellation in interactive mode
	// TODO: For future headless/CI mode, accept a cancellable context and handle SIGTERM/SIGINT.
	runInteractive(context.Background(), deps)
}

func runInteractive(ctx context.Context, deps Dependencies) {
	userInterface := deps.UI

	// Create cancellable context for goroutines
	orchCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	var wg sync.WaitGroup

	// Provider shared between goroutines (GeminiProvider is thread-safe)
	var providerClient provider.Provider
	providerReady := make(chan struct{})

	// Goroutine #1: Initialize & REPL
	wg.Add(1)
	go func() {
		defer wg.Done()

		<-userInterface.Ready() // Wait for UI to be ready

		// === WORKSPACE INITIALIZATION ===
		userInterface.WriteStatus("thinking", "Initializing workspace...")

		workspaceRoot, err := os.Getwd()
		if err != nil {
			userInterface.WriteStatus("error", "Initialization failed")
			userInterface.WriteMessage(fmt.Sprintf("Error: failed to get working directory: %v", err))
			userInterface.WriteMessage("The application cannot start. Press Ctrl+C to exit.")
			return // DEGRADED MODE: UI runs but app doesn't start
		}

		// Create tools with dependency injection
		toolList, err := createTools(deps.Config, workspaceRoot)
		if err != nil {
			userInterface.WriteStatus("error", "Initialization failed")
			userInterface.WriteMessage(fmt.Sprintf("Error: failed to initialize tools: %v", err))
			userInterface.WriteMessage("The application cannot start. Press Ctrl+C to exit.")
			return
		}

		// === PROVIDER INITIALIZATION ===
		userInterface.WriteStatus("thinking", "Initializing AI...")

		p, err := deps.ProviderFactory(orchCtx)
		if err != nil {
			userInterface.WriteStatus("error", "AI initialization failed")
			userInterface.WriteMessage(fmt.Sprintf("Error initializing provider: %v", err))
			userInterface.WriteMessage("The application cannot start. Press Ctrl+C to exit.")
			return // DEGRADED MODE
		}

		// Share provider with other goroutines
		providerClient = p
		close(providerReady)

		// Set initial model in status bar (use the model from the provider)
		userInterface.SetModel(p.GetModel())

		// === ORCHESTRATOR INITIALIZATION ===
		// Initialize components
		policy := orchestrator.NewPolicy(deps.Config)
		policyService := orchestrator.NewPolicyService(policy, userInterface)
		orch := orchestrator.New(deps.Config, providerClient, policyService, userInterface, toolList)

		userInterface.WriteStatus("ready", "Ready")

		// === REPL LOOP (with cancellation) ===
		for {
			select {
			case <-orchCtx.Done():
				return // Exit on cancellation
			default:
				// Initial prompt
				goal, err := userInterface.ReadInput(orchCtx, "What would you like to do?")
				if err != nil {
					return // UI closed or context cancelled
				}

				if err := orch.Run(orchCtx, goal); err != nil {
					userInterface.WriteMessage(fmt.Sprintf("Error: %v", err))
				}

				userInterface.WriteStatus("ready", "Ready")
			}
		}
	}()

	// Goroutine #2: Command handler (with cancellation)
	wg.Add(1)
	go func() {
		defer wg.Done()

		for {
			select {
			case <-orchCtx.Done():
				return
			case cmd := <-userInterface.Commands():
				switch cmd.Type {
				case "list_models":
					// Wait for provider to be ready
					select {
					case <-providerReady:
						models, err := providerClient.ListModels(orchCtx)
						if err != nil {
							userInterface.WriteMessage(fmt.Sprintf("Error listing models: %v", err))
						} else {
							userInterface.WriteModelList(models)
						}
					case <-orchCtx.Done():
						return
					}
				case "switch_model":
					// Wait for provider to be ready
					select {
					case <-providerReady:
						model := cmd.Args["model"]
						if err := providerClient.SetModel(model); err != nil {
							userInterface.WriteMessage(fmt.Sprintf("Error switching model: %v", err))
						} else {
							userInterface.SetModel(model)
							userInterface.WriteMessage(fmt.Sprintf("Switched to model: %s", model))
						}
					case <-orchCtx.Done():
						return
					}
				}
			}
		}
	}()

	// Run UI in main thread (blocks until exit)
	if err := userInterface.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running UI: %v\n", err)
		os.Exit(1)
	}

	// UI exited, trigger shutdown
	cancel()

	// Wait for goroutines to finish
	wg.Wait()
}
