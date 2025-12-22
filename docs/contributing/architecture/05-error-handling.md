# 7. Error Handling

**Goal**: Errors live with the code that returns them.

*   **Sentinel Errors**: Use sentinels for standard domain conditions ("Not Found", "Invalid Input").
    *   **Mechanism**: `var ErrNotFound = errors.New("not found")` in the same package as the code.

*   **Error Structs**: Use structs only when context (paths, values) is required for error handling logic.
    *   **Mechanism**: `type PathError struct { Path string }` in the same package.

> [!NOTE]
> **Multiple Implementations**: If multiple packages return the same error (e.g., `storage/memory/` and `storage/file/` both return `ErrNotFound`), define the error in the parent package.

> [!CAUTION]
> **FORBIDDEN ERROR PATTERNS**
>
> | Pattern | Why Bad |
> |---------|---------|
> | **Behavioral Interfaces** | Using `interface { NotFound() bool }` leads to boilerplate explosion. |
> | **Raw errors.New output** | `return errors.New("fail")`. Untestable. Use a sentinel instead. |

*   **Error Wrapping**: Always wrap errors to add context.
    *   **How**: `fmt.Errorf("operation failed: %w", err)`
    *   **Checking**: Use `errors.Is(err, pkg.ErrX)` for sentinels. Use `errors.As(err, &target)` for structs.

**Example**:

```go
// package file - errors live with the implementation
package file

var ErrNotFound = errors.New("file not found")

func (t *ReadTool) Run() error {
    return fmt.Errorf("read: %w", ErrNotFound)
}

// Consumer imports the implementation package
import "iav/internal/tool/file"

func handle(err error) {
    if errors.Is(err, file.ErrNotFound) {
        // Handle
    }
}
```

