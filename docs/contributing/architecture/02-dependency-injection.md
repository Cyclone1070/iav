# 2. Dependency Injection

**Goal**: Explicit, testable dependencies.

*   **Strict DI**: Dependencies MUST be passed via constructor.
    *   **Why**: Explicit dependencies make code testable and prevent hidden coupling.

*   **Pure Helpers vs Dependencies**:
    *   **Pure helpers**: Simple, static functions. Import directly, including interfaces, structs and errors. No interface needed.
    *   **Dependencies**: Complex logic. Define interface in consumer, inject via constructor. Wiring in main.
    *   **Why**: DI is for swappable/mockable behavior. Pure functions don't need mocking.

*   **No Globals**: Never use global state for dependencies.
    *   **Why**: Globals create hidden dependencies, prevent parallel tests, and make code unpredictable.

**Example**:

```go
// Service with injected dependency
type FileProcessor struct {
    fs FileSystem
}

func NewFileProcessor(fs FileSystem) *FileProcessor {
    return &FileProcessor{fs: fs}
}
```
