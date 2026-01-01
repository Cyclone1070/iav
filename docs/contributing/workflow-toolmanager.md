# workflow/toolmanager Package

## Responsibility

The `toolmanager` package manages tool registry and execution. It translates between provider types and internal tool types.

**Owns:**
- Tool registry: maps tool names → tool implementations
- JSON parsing: `provider.ToolCall.Arguments` → typed request structs
- Event emission: `EventToolStart` (with request display) and `EventToolEnd` (with result display)
- Response construction: returns `provider.Message` with LLM content

**Does NOT own:**
- Individual tool implementations (in `tool/` subpackages)
- Loop orchestration (that's `loop`)
