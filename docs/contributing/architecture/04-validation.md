# 5. Validation Strategy

**Goal**: Trust your objects.

*   **Constructor Validation**: Validate everything you KNOW at construction time.
    *   **Where**: `New...` function.
    *   **What**: Input parameters and their relationships (e.g., "path cannot be empty", "offset >= 0", "limit <= maxLimit").
    *   **Guarantee**: It is impossible to create an invalid instance.
    *   **Why**: Callers cannot forget to validate. Invalid objects cannot exist.

*   **Runtime Validation**: Validate what you DON'T KNOW until the method runs.
    *   **Where**: First lines of the method, clearly commented as `// Runtime Validation`.
    *   **What**: External state requiring I/O or side effects (e.g., "file exists", "user is unique in DB").
    *   **Why**: These checks require runtime operations. You cannot know the answer without performing the operations.

> [!TIP]
> **Rule of Thumb**: If validation requires a runtime operation (filesystem, network, database), it belongs in the method. Everything else belongs in the constructor.

**Example**:

```go
// Constructor validation - what we CAN know from inputs
func NewReadFileRequest(path string, offset, limit int64) (*ReadFileRequest, error) {
    if path == "" {
        return nil, errors.New("path is required")
    }
    if offset < 0 {
        return nil, errors.New("offset cannot be negative")
    }
    if limit < 0 {
        return nil, errors.New("limit cannot be negative")
    }
    return &ReadFileRequest{path: path, offset: offset, limit: limit}, nil
}

// Runtime validation - what we CAN'T know until we check
func (t *ReadFileTool) Run(ctx context.Context, req *ReadFileRequest) (*ReadFileResponse, error) {
    // Runtime Validation
    abs, err := t.pathResolver.Resolve(req.path)
    if err != nil {
        return nil, err // path outside workspace
    }
    info, err := t.fs.Stat(abs)
    if err != nil {
        return nil, err // file doesn't exist
    }
    if info.IsDir() {
        return nil, &IsDirectoryError{Path: abs}
    }

    // 2. Implementation
    content, err := t.fs.ReadFile(abs)
    ...
}
```
