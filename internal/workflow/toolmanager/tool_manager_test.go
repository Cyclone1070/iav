package toolmanager

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Cyclone1070/iav/internal/provider"
	"github.com/Cyclone1070/iav/internal/tool"
	"github.com/Cyclone1070/iav/internal/workflow"
	"github.com/stretchr/testify/assert"
)

type mockResult struct {
	llmContent string
	display    tool.ToolDisplay
}

func (m *mockResult) LLMContent() string        { return m.llmContent }
func (m *mockResult) Display() tool.ToolDisplay { return m.display }

type mockInput struct {
	Value string `json:"value"`
}

func (m *mockInput) String() string { return m.Value }

type mockTool struct {
	name        string
	declaration tool.Declaration
	executeFunc func(ctx context.Context, input any) (toolResult, error)
}

func (m *mockTool) Name() string                  { return m.name }
func (m *mockTool) Declaration() tool.Declaration { return m.declaration }
func (m *mockTool) Input() any                    { return &mockInput{} }
func (m *mockTool) Execute(ctx context.Context, input any) (toolResult, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, input)
	}
	return &mockResult{llmContent: "ok", display: tool.StringDisplay("ok")}, nil
}

func TestRegister_AddsTool(t *testing.T) {
	tm := NewToolManager()
	mt := &mockTool{name: "test-tool", declaration: tool.Declaration{Name: "test-tool"}}
	tm.Register(mt)

	decls := tm.Declarations()
	assert.Len(t, decls, 1)
	assert.Equal(t, "test-tool", decls[0].Name)
}

func TestRegister_DuplicateName(t *testing.T) {
	tm := NewToolManager()
	mt1 := &mockTool{name: "test-tool", declaration: tool.Declaration{Name: "test-tool", Description: "v1"}}
	mt2 := &mockTool{name: "test-tool", declaration: tool.Declaration{Name: "test-tool", Description: "v2"}}

	tm.Register(mt1)
	tm.Register(mt2)

	decls := tm.Declarations()
	assert.Len(t, decls, 1)
	assert.Equal(t, "v2", decls[0].Description)
}

func TestDeclarations_SortedByName(t *testing.T) {
	tm := NewToolManager()
	tm.Register(&mockTool{name: "z", declaration: tool.Declaration{Name: "z"}})
	tm.Register(&mockTool{name: "a", declaration: tool.Declaration{Name: "a"}})
	tm.Register(&mockTool{name: "m", declaration: tool.Declaration{Name: "m"}})

	decls := tm.Declarations()
	assert.Len(t, decls, 3)
	assert.Equal(t, "a", decls[0].Name)
	assert.Equal(t, "m", decls[1].Name)
	assert.Equal(t, "z", decls[2].Name)
}

func TestExecute_UnknownTool_ReturnsMessageToLLM(t *testing.T) {
	tm := NewToolManager()
	res, err := tm.Execute(context.Background(), provider.ToolCall{
		ID:       "tc-123",
		Function: provider.FunctionCall{Name: "unknown"},
	}, nil)

	assert.NoError(t, err)
	assert.Equal(t, "tc-123", res.ToolCallID)
	assert.Contains(t, res.Content, "Error: tool \"unknown\" does not exist")
}

func TestExecute_ValidJSON_ParsesCorrectly(t *testing.T) {
	tm := NewToolManager()
	var capturedInput *mockInput
	tm.Register(&mockTool{
		name: "test",
		executeFunc: func(ctx context.Context, input any) (toolResult, error) {
			capturedInput = input.(*mockInput)
			return &mockResult{llmContent: "ok"}, nil
		},
	})

	events := make(chan workflow.Event, 10)
	res, err := tm.Execute(context.Background(), provider.ToolCall{
		ID: "tc-456",
		Function: provider.FunctionCall{
			Name:      "test",
			Arguments: json.RawMessage(`{"value": "hello"}`),
		},
	}, events)

	assert.NoError(t, err)
	assert.Equal(t, "hello", capturedInput.Value)
	assert.Equal(t, "tc-456", res.ToolCallID)
}

func TestExecute_MalformedJSON_ReturnsMessageToLLM(t *testing.T) {
	tm := NewToolManager()
	tm.Register(&mockTool{name: "test"})

	res, err := tm.Execute(context.Background(), provider.ToolCall{
		ID: "tc-789",
		Function: provider.FunctionCall{
			Name:      "test",
			Arguments: json.RawMessage(`{invalid}`),
		},
	}, nil)

	assert.NoError(t, err)
	assert.Equal(t, "tc-789", res.ToolCallID)
	assert.Contains(t, res.Content, "Error: invalid arguments for tool \"test\"")
}

func TestExecute_EmitsToolEvents(t *testing.T) {
	tm := NewToolManager()
	tm.Register(&mockTool{
		name: "test",
		executeFunc: func(ctx context.Context, input any) (toolResult, error) {
			return &mockResult{llmContent: "ok", display: tool.StringDisplay("result")}, nil
		},
	})

	events := make(chan workflow.Event, 10)
	_, err := tm.Execute(context.Background(), provider.ToolCall{
		Function: provider.FunctionCall{
			Name:      "test",
			Arguments: json.RawMessage(`{"value": "hello"}`),
		},
	}, events)

	assert.NoError(t, err)

	e1 := <-events
	start, ok := e1.(workflow.ToolStartEvent)
	assert.True(t, ok)
	assert.Equal(t, "test", start.ToolName)
	assert.Equal(t, "hello", start.RequestDisplay)

	e2 := <-events
	end, ok := e2.(workflow.ToolEndEvent)
	assert.True(t, ok)
	assert.Equal(t, "test", end.ToolName)
	assert.Equal(t, tool.StringDisplay("result"), end.Display)
}

func TestExecute_Shell_StreamsAndEnds(t *testing.T) {
	tm := NewToolManager()
	tm.Register(&mockTool{
		name: "shell",
		executeFunc: func(ctx context.Context, input any) (toolResult, error) {
			return &mockResult{
				llmContent: "Command finished",
				display: tool.ShellDisplay{
					Command:    "ls",
					WorkingDir: "/",
					Output:     strings.NewReader("file1\nfile2\n"),
					Wait: func() int {
						return 0
					},
				},
			}, nil
		},
	})

	events := make(chan workflow.Event, 10)
	_, err := tm.Execute(context.Background(), provider.ToolCall{
		Function: provider.FunctionCall{
			Name:      "shell",
			Arguments: json.RawMessage(`{}`),
		},
	}, events)

	assert.NoError(t, err)

	e1 := <-events
	assert.IsType(t, workflow.ToolStartEvent{}, e1)

	var streamOutput strings.Builder
loop:
	for {
		e := <-events
		switch ev := e.(type) {
		case workflow.ToolStreamEvent:
			streamOutput.WriteString(ev.Chunk)
		case workflow.ShellEndEvent:
			assert.Equal(t, 0, ev.ExitCode)
			break loop
		case workflow.ToolEndEvent:
			t.Fatalf("received ToolEndEvent instead of ShellEndEvent")
		}
	}

	assert.Equal(t, "file1\nfile2\n", streamOutput.String())
}

func TestExecute_ContextCancelled_StopsStreaming(t *testing.T) {
	tm := NewToolManager()
	tm.Register(&mockTool{
		name: "shell",
		executeFunc: func(ctx context.Context, input any) (toolResult, error) {
			pipeR, pipeW := io.Pipe()
			go func() {
				<-ctx.Done()
				pipeW.Close()
			}()
			return &mockResult{
				display: tool.ShellDisplay{
					Output: pipeR,
					Wait:   func() int { return 0 },
				},
			}, nil
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	events := make(chan workflow.Event, 10)

	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	_, err := tm.Execute(ctx, provider.ToolCall{
		Function: provider.FunctionCall{Name: "shell", Arguments: json.RawMessage(`{}`)},
	}, events)

	assert.Error(t, err)
}

func TestExecute_ConcurrentCalls_NoRace(t *testing.T) {
	tm := NewToolManager()
	tm.Register(&mockTool{name: "tool"})

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = tm.Execute(context.Background(), provider.ToolCall{
				Function: provider.FunctionCall{Name: "tool", Arguments: json.RawMessage(`{}`)},
			}, nil)
		}()
	}
	wg.Wait()
}
