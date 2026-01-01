# tool Package

## Responsibility

The `tool` package defines tool types and display data structures. It is the foundational layer that other packages import for type definitions.

**Owns:**
- `Declaration` — Tool schema (name, description, parameters) sent to LLM
- `Schema` — JSON Schema type definitions for parameters
- `ToolDisplay` — Interface for UI display types
- `StringDisplay`, `DiffDisplay`, `ShellDisplay` — Concrete display types

**Does NOT own:**
- Tool execution logic (individual tools in subpackages)
- Tool registry or orchestration

---

## Error Handling Contract

Tools must follow this error handling pattern:

| Error Type                  | Return                                | Example                                                       |
| --------------------------- | ------------------------------------- | ------------------------------------------------------------- |
| **Tool failure** (expected) | `nil` error, encode in `LLMContent()` | File not found, permission denied, validation error, exit ≠ 0 |
| **Infra error** (context)   | Return error                          | `context.Canceled`, `context.DeadlineExceeded`                |

**Rationale:** Tool failures are recoverable — the agent sees the error in `LLMContent()` and decides what to do (retry, ask user). Infra errors stop the loop.

```go
func (t *MyTool) Execute(ctx context.Context, req any) (toolResult, error) {
    // Always check context first
    if ctx.Err() != nil {
        return nil, ctx.Err()  // Infra → return error, stops loop
    }

    result, err := doWork()
    if err != nil {
        // Tool failure → encode in result, loop continues
        return &MyResult{
            Content: fmt.Sprintf("Error: %v", err),
        }, nil
    }

    return &MyResult{Content: result}, nil
}
```
