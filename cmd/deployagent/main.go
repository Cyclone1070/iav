// Package main provides a command-line interface for the deployagent tool.
// It supports file operations (read, write, edit, list) and shell command execution.
package main

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/Cyclone1070/deployforme/internal/orchestrator"
	orchadapter "github.com/Cyclone1070/deployforme/internal/orchestrator/adapter"
	orchmodels "github.com/Cyclone1070/deployforme/internal/orchestrator/models"
	"github.com/Cyclone1070/deployforme/internal/provider/gemini"
	provider "github.com/Cyclone1070/deployforme/internal/provider/models"
	"github.com/Cyclone1070/deployforme/internal/tools"
	"github.com/Cyclone1070/deployforme/internal/tools/models"
	"github.com/Cyclone1070/deployforme/internal/ui"
	uiservices "github.com/Cyclone1070/deployforme/internal/ui/services"
	"github.com/charmbracelet/bubbles/spinner"
	"google.golang.org/genai"
)

// Dependencies holds the components required to run the application.
type Dependencies struct {
	UI              ui.UserInterface
	ProviderFactory func(context.Context) (provider.Provider, error)
	Tools           []orchadapter.Tool
}

func createRealUI() ui.UserInterface {
	channels := ui.NewUIChannels()
	renderer := uiservices.NewGlamourRenderer()
	spinnerFactory := func() spinner.Model {
		return spinner.New(spinner.WithSpinner(spinner.Dot))
	}
	return ui.NewUI(channels, renderer, spinnerFactory)
}

func createRealProviderFactory() func(context.Context) (provider.Provider, error) {
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
		return gemini.NewGeminiProviderWithLatest(ctx, geminiClient)
	}
}

func createTools(ctx *models.WorkspaceContext) []orchadapter.Tool {
	return []orchadapter.Tool{
		orchadapter.NewReadFile(ctx),
		orchadapter.NewWriteFile(ctx),
		orchadapter.NewEditFile(ctx),
		orchadapter.NewListDirectory(ctx),
		orchadapter.NewShell(&tools.ShellTool{CommandExecutor: ctx.CommandExecutor}, ctx),
		orchadapter.NewSearchContent(ctx),
		orchadapter.NewFindFile(ctx),
		orchadapter.NewReadTodos(ctx),
		orchadapter.NewWriteTodos(ctx),
	}
}

func main() {
	// Create dependencies
	deps := Dependencies{
		UI:              createRealUI(),
		ProviderFactory: createRealProviderFactory(),
		Tools:           nil, // Will be created in runInteractive
	}

	// Run interactive mode (blocks until exit)
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

		workspaceCtx, err := tools.NewWorkspaceContext(workspaceRoot)
		if err != nil {
			userInterface.WriteStatus("error", "Initialization failed")
			userInterface.WriteMessage(fmt.Sprintf("Error: failed to initialize workspace: %v", err))
			userInterface.WriteMessage("The application cannot start. Press Ctrl+C to exit.")
			return
		}

		// Create tools
		toolList := createTools(workspaceCtx)

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
		policy := orchmodels.NewPolicy()
		policyService := orchestrator.NewPolicyService(policy, userInterface)
		orch := orchestrator.New(providerClient, policyService, userInterface, toolList)

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
