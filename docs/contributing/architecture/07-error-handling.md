# 7. Error Handling

**Goal**: Decoupling. Consumers can check errors without importing the producer package.

*   **Errors in Parent Package**: Sentinel errors and error structs belong in the parent package (`errors.go`), not in sub-packages.
    *   **Structure**: `internal/tool/errors.go` contains all sentinels and error structs for the `tool` feature group.
    *   **Why**: Without shared errors, checking an error means importing the producer. Shared errors let consumers check errors without coupling to who produced them.
    *   **Usage**: Producers return `tool.ErrX`. Consumers check `errors.Is(err, tool.ErrX)`.

*   **Sentinel Errors**: Use sentinels for standard domain conditions ("Not Found", "Invalid Input").
    *   **Mechanism**: `var ErrNotFound = errors.New("not found")` in the parent package.

*   **Error Structs**: Use structs only when context (paths, values) is required for error handling logic.
    *   **Mechanism**: `type PathError struct { Path string }` in the parent package.

> [!CAUTION]
> **FORBIDDEN ERROR PATTERNS**
>
> | Pattern | Why Bad |
> |---------|---------|
> | **Local Sentinel Errors** | Defining `ErrX` in `file/` forces consumers to import `file` just to check an error. |
> | **Behavioral Interfaces** | Using `interface { NotFound() bool }` leads to boilerplate explosion. |
> | **Raw errors.New output** | `return errors.New("fail")`. Untestable. Use a sentinel instead. |

*   **Error Wrapping**: Always wrap errors to add context.
    *   **How**: `fmt.Errorf("operation failed: %w", err)`
    *   **Checking**: Use `errors.Is(err, tool.ErrX)` for sentinels. Use `errors.As(err, &target)` for structs.

**Example**:

```go
// Parent package (internal/tool/errors.go)
package tool

var ErrNotFound = errors.New("file not found")

// Producer (internal/tool/file/read.go)
import "iav/internal/tool"

func (t *ReadTool) Run() error {
    return fmt.Errorf("read: %w", tool.ErrNotFound)
}

// Consumer (internal/orchestrator)
import "iav/internal/tool"

func handle(err error) {
    if errors.Is(err, tool.ErrNotFound) {
        // Handle without importing 'file'
    }
}
```
