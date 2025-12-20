# 7. Error Handling

**Goal**: Decoupling, Type Safety, and Scalability.

*   **Shared Error Domain**: All cross-boundary Sentinel Errors and Public Struct Errors MUST be defined in the `shared/` package (e.g., `internal/tool/shared`).
    *   **Structure**: `internal/pkggroup/shared/errors.go` contains sentinels and error structs.
    *   **Topology**: The `shared/` package MUST be a **Leaf Node** and a **Sibling** to the packages that use it.
    *   **Why**: Fully decouples Consumers from Provider implementations. Prevents circular dependencies.
    *   **Usage**: Producers return `shared.ErrX`. Consumers check `errors.Is(e, shared.ErrX)`.

*   **Sentinel Errors**: Use Sentinels for all standard domain conditions ("Not Found", "Invalid Input").
    *   **Mechanism**: `var ErrNotFound = errors.New("not found")` defined in the `shared/` package.

*   **Error Structs**: Use Structs only when context (paths, values) is required for error handling logic.
    *   **Mechanism**: `type PathError struct { ... }` defined in the `shared/` package.

> [!CAUTION]
> **FORBIDDEN ERROR PATTERNS**
>
> | Pattern | Why Bad |
> |---------|---------|
> | **Local Sentinel Errors** | Defining `ErrX` in `service` forces `handler` to import `service` just to check an error. This couples Logic packages. |
> | **Behavioral Interfaces** | Using `interface { NotFound() bool }` leads to boilerplate explosion and obscures simple error checks. |
> | **Raw errors.New output** | `return errors.New("fail")`. **Untestable**. Use a Sentinel instead. |

*   **Error Wrapping**: Always wrap errors to add context.
    *   **How**: `fmt.Errorf("operation failed: %w", e)`
    *   **Checking**: Use `errors.Is(e, shared.ErrSentinel)` to check nature. Use `errors.As(e, &targetStruct)` to check data.

**Example**:

```go
// Shared Definitions (package shared, e.g., internal/tool/shared)
var ErrNotFound = errors.New("file not found")

// Provider (package fs)
import "iav/internal/tool/shared"
func Open() error {
    return fmt.Errorf("fs: %w", shared.ErrNotFound)
}

// Consumer (package usecase)
import "iav/internal/tool/shared"
func Do() {
    if e := fs.Open(); e != nil {
        if errors.Is(e, shared.ErrNotFound) {
            // Handle specific error without importing 'fs'
            return
        }
    }
}
```
