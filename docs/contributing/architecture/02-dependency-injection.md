# 2. Dependency Injection

**Goal**: Explicit, testable dependencies.

*   **Strict DI**: Dependencies MUST be passed via constructor.
    *   **Why**: Explicit dependencies make code testable and prevent hidden coupling.

*   **No Globals**: Never use global state for dependencies.
    *   **Why**: Globals create hidden dependencies, prevent parallel tests, and make code unpredictable.

*   **Pure Helpers vs Dependencies**:
    *   **Pure helpers** (stateless, no I/O, no side effects): Import directly. No interface needed.
    *   **Dependencies** (stateful, does I/O, has side effects): Define interface in consumer, inject via constructor.
    *   **Why**: DI is for swappable/mockable behavior. Pure functions don't need mocking.

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
