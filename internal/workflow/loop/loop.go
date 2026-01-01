package loop

import (
	"context"
	"fmt"

	"github.com/Cyclone1070/iav/internal/provider"
	"github.com/Cyclone1070/iav/internal/workflow"
)

type Loop struct {
	provider      llmProvider
	tools         toolManager
	events        chan<- workflow.Event
	maxIterations int
}

func NewLoop(provider llmProvider, tools toolManager, events chan<- workflow.Event, maxIterations int) *Loop {
	return &Loop{
		provider:      provider,
		tools:         tools,
		events:        events,
		maxIterations: maxIterations,
	}
}

func (l *Loop) Run(ctx context.Context, initialMessage string) error {
	messages := []provider.Message{
		{Role: provider.RoleUser, Content: initialMessage},
	}

	defer func() {
		if l.events != nil {
			l.events <- workflow.DoneEvent{}
		}
	}()

	for i := 0; i < l.maxIterations; i++ {
		if err := ctx.Err(); err != nil {
			messages = append(messages, provider.Message{
				Role:    provider.RoleUser,
				Content: "[Session cancelled by user]",
			})
			return err
		}

		if l.events != nil {
			l.events <- workflow.ThinkingEvent{}
		}

		resp, err := l.provider.Generate(ctx, messages, l.tools.Declarations())
		if err != nil {
			return fmt.Errorf("provider.Generate: %w", err)
		}

		messages = append(messages, *resp)

		if resp.Content != "" && l.events != nil {
			l.events <- workflow.TextEvent{Text: resp.Content}
		}

		if len(resp.ToolCalls) == 0 {
			return nil
		}

		for _, tc := range resp.ToolCalls {
			toolResp, err := l.tools.Execute(ctx, tc, l.events)
			if err != nil {
				return fmt.Errorf("tools.Execute (%s): %w", tc.Function.Name, err)
			}
			messages = append(messages, toolResp)
		}
	}

	messages = append(messages, provider.Message{
		Role:    provider.RoleUser,
		Content: "[Max iterations reached]",
	})
	return fmt.Errorf("max iterations (%d) reached", l.maxIterations)
}
