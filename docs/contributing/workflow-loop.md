# workflow/loop Package

## Responsibility

The `loop` package orchestrates the agent's think-act cycle. It coordinates between the LLM provider and tool manager.

**Owns:**
- Main agent loop: send messages → get response → handle tool calls → repeat
- Coordination between `llmProvider` and `toolManager` interfaces
- Emitting loop-level events: `EventThinking`, `EventText`, `EventDone`

**Does NOT own:**
- Tool registry or parsing (delegated to `toolmanager`)
- LLM communication details (delegated to `provider`)
- Tool-specific events (emitted by `toolmanager`)
