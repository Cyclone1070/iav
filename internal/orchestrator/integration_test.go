//go:build integration

package orchestrator

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/Cyclone1070/iav/internal/config"
	orchadapter "github.com/Cyclone1070/iav/internal/orchestrator/adapter"
	orchmodel "github.com/Cyclone1070/iav/internal/orchestrator/model"
	pmodel "github.com/Cyclone1070/iav/internal/provider/model"
	"github.com/Cyclone1070/iav/internal/testing/mock"
	"github.com/Cyclone1070/iav/internal/tool/directory"
	"github.com/Cyclone1070/iav/internal/tool/file"
	"github.com/Cyclone1070/iav/internal/tool/search"
	"github.com/Cyclone1070/iav/internal/tool/service/executor"
	"github.com/Cyclone1070/iav/internal/tool/service/fs"
	"github.com/Cyclone1070/iav/internal/tool/service/git"
	"github.com/Cyclone1070/iav/internal/tool/service/hash"
	"github.com/Cyclone1070/iav/internal/tool/service/path"
	"github.com/Cyclone1070/iav/internal/tool/shell"
	"github.com/Cyclone1070/iav/internal/tool/todo"
	"github.com/Cyclone1070/iav/internal/ui"
	uiservices "github.com/Cyclone1070/iav/internal/ui/service"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/stretchr/testify/assert"
)

func TestOrchestratorProvider_ToolCallResponse(t *testing.T) {
	t.Parallel()

	// Create workspace context dependencies
	workspaceRoot := t.TempDir()
	fileSystem := fs.NewOSFileSystem()
	checksumMgr := hash.NewChecksumManager()
	gitignoreSvc, _ := git.NewService(workspaceRoot, fileSystem)
	cfg := config.DefaultConfig()
	pathResolver := path.NewResolver(workspaceRoot)

	// Create UI
	channels := ui.NewUIChannels(nil)
	renderer := uiservices.NewGlamourRenderer()
	spinnerFactory := func() spinner.Model {
		return spinner.New(spinner.WithSpinner(spinner.Dot))
	}
	userInterface := ui.NewUI(channels, renderer, spinnerFactory)

	// Create cancellable context
	runCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Service UI input requests in background
	go func() {
		count := 0
		for {
			select {
			case req := <-channels.InputReq:
				count++
				t.Logf("UI received InputReq %d", count)
				if count > 1 {
					t.Log("UI cancelling context and returning exit")
					cancel()
					select {
					case channels.InputResp <- "exit":
					case <-runCtx.Done():
					}
				} else {
					t.Log("UI returning continue")
					channels.InputResp <- "continue"
				}
				_ = req
			case <-runCtx.Done():
				t.Log("UI goroutine exiting due to context cancel")
				return
			}
		}
	}()

	// Track UI messages
	messageDone := make(chan []string)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		var msgs []string
		for msg := range channels.MessageChan {
			msgs = append(msgs, msg)
		}
		messageDone <- msgs
	}()

	// Initialize tools and adapters
	listTool := directory.NewListDirectoryTool(fileSystem, gitignoreSvc, cfg, pathResolver)
	listAdapter := orchadapter.NewListDirectoryAdapter(listTool)

	writeTool := file.NewWriteFileTool(fileSystem, checksumMgr, cfg, pathResolver)
	writeAdapter := orchadapter.NewWriteFileAdapter(writeTool)

	toolList := []orchadapter.Tool{
		listAdapter,
		writeAdapter,
	}

	// Mock Provider
	mockProvider := mock.NewMockProvider()
	callCount := 0
	mockProvider.GenerateFunc = func(ctx context.Context, req *pmodel.GenerateRequest) (*pmodel.GenerateResponse, error) {
		callCount++
		t.Logf("Provider call %d", callCount)
		if callCount == 1 {
			return &pmodel.GenerateResponse{
				Content: pmodel.ResponseContent{
					Type: pmodel.ResponseTypeToolCall,
					ToolCalls: []orchmodel.ToolCall{
						{
							ID:   "call_1",
							Name: "list_directory",
							Args: map[string]any{"path": "."},
						},
					},
				},
			}, nil
		}
		if callCount == 2 {
			return &pmodel.GenerateResponse{
				Content: pmodel.ResponseContent{
					Type: pmodel.ResponseTypeText,
					Text: "I found the files and I'm done.",
				},
			}, nil
		}
		return nil, errors.New("unexpected call")
	}

	// Policy
	policy := &orchmodel.Policy{
		Shell: orchmodel.ShellPolicy{},
		Tools: orchmodel.ToolPolicy{},
	}
	policySvc := NewPolicyService(policy, userInterface)

	// Create Orchestrator
	orchestrator := New(cfg, mockProvider, policySvc, userInterface, toolList)

	// Run
	t.Log("Starting Run")
	err := orchestrator.Run(runCtx, "test goal")
	t.Logf("Run finished with err: %v", err)
	close(channels.MessageChan)
	t.Log("Waiting for message goroutine")
	wg.Wait()
	t.Log("Waiting for messages")
	msgs := <-messageDone
	t.Logf("Received %d messages", len(msgs))

	// Assertions
	assert.True(t, err == nil || err == context.Canceled, "expected nil or context.Canceled, got %v", err)
	assert.True(t, len(msgs) > 0)
	foundDone := false
	for _, m := range msgs {
		if strings.Contains(m, "Done") || strings.Contains(m, "done") {
			foundDone = true
			break
		}
	}
	assert.True(t, foundDone)
}

func TestFullToolIntegration(t *testing.T) {
	t.Parallel()

	workspaceRoot := t.TempDir()
	fs := fs.NewOSFileSystem()
	cfg := config.DefaultConfig()
	checksumMgr := hash.NewChecksumManager()
	gitignoreSvc, _ := git.NewService(workspaceRoot, fs)
	userInterface := mock.NewMockUI()
	commandExecutor := executor.NewOSCommandExecutor(cfg)
	pathResolver := path.NewResolver(workspaceRoot)
	dockerConfig := shell.DockerConfig{
		CheckCommand: []string{"docker", "info"},
		StartCommand: []string{"docker", "desktop", "start"},
	}

	// Initialize all real tools
	listTool := directory.NewListDirectoryTool(fs, gitignoreSvc, cfg, pathResolver)
	findTool := directory.NewFindFileTool(fs, commandExecutor, cfg, pathResolver)
	readTool := file.NewReadFileTool(fs, checksumMgr, cfg, pathResolver)
	writeTool := file.NewWriteFileTool(fs, checksumMgr, cfg, pathResolver)
	editTool := file.NewEditFileTool(fs, checksumMgr, cfg, pathResolver)
	searchTool := search.NewSearchContentTool(fs, commandExecutor, cfg, pathResolver)
	shellTool := shell.NewShellTool(fs, commandExecutor, cfg, dockerConfig, pathResolver)
	todoReadTool := todo.NewReadTodosTool(todo.NewInMemoryTodoStore(), cfg)
	todoWriteTool := todo.NewWriteTodosTool(todo.NewInMemoryTodoStore(), cfg)

	// Initialize all adapters
	tools := []orchadapter.Tool{
		orchadapter.NewListDirectoryAdapter(listTool),
		orchadapter.NewFindFileAdapter(findTool),
		orchadapter.NewReadFileAdapter(readTool),
		orchadapter.NewWriteFileAdapter(writeTool),
		orchadapter.NewEditFileAdapter(editTool),
		orchadapter.NewSearchContentAdapter(searchTool),
		orchadapter.NewShellAdapter(shellTool),
		orchadapter.NewReadTodosAdapter(todoReadTool),
		orchadapter.NewWriteTodosAdapter(todoWriteTool),
	}

	// Provider mock that just exits
	mockProvider := mock.NewMockProvider()
	mockProvider.GenerateFunc = func(ctx context.Context, req *pmodel.GenerateRequest) (*pmodel.GenerateResponse, error) {
		return &pmodel.GenerateResponse{
			Content: pmodel.ResponseContent{
				Type: pmodel.ResponseTypeText,
				Text: "Exiting",
			},
		}, nil
	}

	policySvc := NewPolicyService(&orchmodel.Policy{}, userInterface)

	// Create cancellable context
	runCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Make UI return "exit" and cancel context to stop the loop
	userInterface.InputFunc = func(ctx context.Context, prompt string) (string, error) {
		cancel()
		return "exit", nil
	}

	orchestrator := New(cfg, mockProvider, policySvc, userInterface, tools)

	// Just verify it starts up without panic
	err := orchestrator.Run(runCtx, "do everything")
	if err != nil && err != context.Canceled {
		assert.NoError(t, err)
	}
}
