# 5. Error Handling

> [!IMPORTANT]
> **Minimize Error Returns**
> 
> Every returned error forces the caller to handle it, adding complexity. Before returning an error, ask:
> * Can this be handled internally (clamp, default, fallback)?
> * Can the caller actually do anything different?
> * Is this truly exceptional, or just an edge case we can normalize?
> 
> Only return errors that the caller can meaningfully act upon. Some errors provide critical information to the caller and must be returned.

**Goal**: Errors live with the code that returns them.

## Choosing Error Types

*   **Sentinel**: Error provides distinctly actionable information—caller takes a different action based on this specific error.
    *   `var ErrNotFound = errors.New("not found")`

*   **Struct**: Error is complex and caller needs to extract context fields (path, value) for handling.
    *   `type PathError struct { Path string; Cause error }`

*   **`fmt.Errorf`**: Errors we can't handle internally and caller also has no choice on how to handle and have to pass it up, or can only proceed in one way regardless of details (e.g., I/O failures, permission errors, unexpected errors — user fixes externally).
    *   `fmt.Errorf("stat %s: %w", path, err)`

> [!TIP]
> **Merging Errors**: If multiple distinct errors lead to the same handling sequence, merge them into one sentinel or use `fmt.Errorf` wrapping. Don't create separate error types unless they drive different caller behavior.

## Error Wrapping

Always wrap errors with context using `%w`:
*   `fmt.Errorf("read file: %w", err)`

Checking wrapped errors:
*   **Sentinel**: `errors.Is(err, pkg.ErrNotFound)`
*   **Struct**: `errors.As(err, &pathErr)`

> [!NOTE]
> **Multiple Implementations**: If there are multiple implementations (e.g., different storage backends), define errors in the parent package and all implementations import.

> [!CAUTION]
> **FORBIDDEN ERROR PATTERNS**
>
> | Pattern | Why Bad |
> |---------|---------|
> | **Behavioral Interfaces** | Using `interface { NotFound() bool }` leads to boilerplate explosion. |
> | **Raw errors.New output** | `return errors.New("fail")`. Untestable. Use a sentinel instead. |



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

