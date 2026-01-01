# provider Package

## Responsibility

The `provider` package abstracts LLM communication. It defines message types and provider implementations.

**Owns:**
- `Message` — Messages exchanged with LLM (user, assistant, tool)
- `ToolCall` — Tool invocation request from LLM (name + JSON arguments)
- Provider implementations (e.g., Gemini client)

**Does NOT own:**
- Tool execution or display types
- Workflow orchestration
