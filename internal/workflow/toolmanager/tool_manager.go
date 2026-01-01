package toolmanager

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"

	"github.com/Cyclone1070/iav/internal/provider"
	"github.com/Cyclone1070/iav/internal/tool"
	"github.com/Cyclone1070/iav/internal/workflow"
)

type ToolManager struct {
	registry map[string]toolImpl
}

func NewToolManager(tools ...toolImpl) *ToolManager {
	tm := &ToolManager{
		registry: make(map[string]toolImpl),
	}
	for _, t := range tools {
		tm.Register(t)
	}
	return tm
}

func (m *ToolManager) Register(t toolImpl) {
	m.registry[t.Name()] = t
}

func (m *ToolManager) Declarations() []tool.Declaration {
	decls := make([]tool.Declaration, 0, len(m.registry))
	for _, t := range m.registry {
		decls = append(decls, t.Declaration())
	}
	sort.Slice(decls, func(i, j int) bool {
		return decls[i].Name < decls[j].Name
	})
	return decls
}

func (m *ToolManager) Execute(ctx context.Context, tc provider.ToolCall, events chan<- workflow.Event) (provider.Message, error) {
	t, ok := m.registry[tc.Function.Name]
	if !ok {
		decls := m.Declarations()
		declsJSON, _ := json.MarshalIndent(decls, "", "  ")
		errMsg := fmt.Sprintf("Error: tool %q does not exist.\n\nAvailable tools:\n%s", tc.Function.Name, declsJSON)

		if events != nil {
			events <- workflow.ToolStartEvent{
				ToolName:       tc.Function.Name,
				RequestDisplay: "",
			}
			events <- workflow.ToolEndEvent{
				ToolName: tc.Function.Name,
				Display:  tool.StringDisplay("Invalid tool request"),
			}
		}

		return provider.Message{
			Role:       provider.RoleTool,
			ToolCallID: tc.ID,
			Content:    errMsg,
		}, nil
	}

	req := t.Input()
	if err := json.Unmarshal(tc.Function.Arguments, req); err != nil {
		declJSON, _ := json.MarshalIndent(t.Declaration(), "", "  ")
		errMsg := fmt.Sprintf("Error: invalid arguments for tool %q: %v\n\nExpected schema:\n%s", tc.Function.Name, err, declJSON)

		if events != nil {
			events <- workflow.ToolStartEvent{
				ToolName:       tc.Function.Name,
				RequestDisplay: "",
			}
			events <- workflow.ToolEndEvent{
				ToolName: tc.Function.Name,
				Display:  tool.StringDisplay("Invalid tool request"),
			}
		}

		return provider.Message{
			Role:       provider.RoleTool,
			ToolCallID: tc.ID,
			Content:    errMsg,
		}, nil
	}

	if events != nil {
		display := ""
		if s, ok := req.(fmt.Stringer); ok {
			display = s.String()
		}
		events <- workflow.ToolStartEvent{
			ToolName:       tc.Function.Name,
			RequestDisplay: display,
		}
	}

	res, err := t.Execute(ctx, req)
	if err != nil {
		// Per contract, tools only return errors for infrastructure issues (context cancellation)
		if events != nil {
			events <- workflow.ToolEndEvent{
				ToolName: tc.Function.Name,
				Display:  tool.StringDisplay("Cancelled"),
			}
		}
		return provider.Message{}, err
	}

	display := res.Display()

	// Special handling for shell streaming
	if sh, ok := display.(tool.ShellDisplay); ok {
		buf := make([]byte, 4096)
		for {
			select {
			case <-ctx.Done():
				if events != nil {
					events <- workflow.ShellEndEvent{
						ToolName: tc.Function.Name,
						ExitCode: -1,
					}
				}
				return provider.Message{}, ctx.Err()
			default:
			}

			n, err := sh.Output.Read(buf)
			if n > 0 && events != nil {
				events <- workflow.ToolStreamEvent{
					ToolName: tc.Function.Name,
					Chunk:    string(buf[:n]),
				}
			}
			if err == io.EOF {
				break
			}
			if err != nil {
				// Stop streaming on read error, but don't return error to loop
				break
			}
		}

		exitCode := sh.Wait()
		if events != nil {
			events <- workflow.ShellEndEvent{
				ToolName: tc.Function.Name,
				ExitCode: exitCode,
			}
		}
	} else {
		if events != nil {
			events <- workflow.ToolEndEvent{
				ToolName: tc.Function.Name,
				Display:  display,
			}
		}
	}

	if err := ctx.Err(); err != nil {
		return provider.Message{}, err
	}

	return provider.Message{
		Role:       provider.RoleTool,
		ToolCallID: tc.ID,
		Content:    res.LLMContent(),
	}, nil
}
