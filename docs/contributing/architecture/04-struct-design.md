# 4. Struct Design: Encapsulation vs Public Fields

**Goal**: Use the right pattern for the struct's purpose.

## Rule

| Struct Type                    | Fields  | Constructor       | Validation          |
| ------------------------------ | ------- | ----------------- | ------------------- |
| **Services & Domain Entities** | Private | `NewT()` required | At construction     |
| **DTOs & Data Holders**        | Public  | Optional          | `Validate()` method |

## Services & Domain Entities

Use **private fields + constructor** when:

- Fields must be valid together (invariants)
- Dependencies must not be nil
- State should be immutable after creation

```go
// Private fields - constructor enforces invariants
type Resolver struct {
    workspaceRoot string      // must be non-empty
    fs            FileSystem  // must not be nil
}

func NewResolver(root string, fs FileSystem) *Resolver {
    return &Resolver{workspaceRoot: root, fs: fs}
}
```

> [!NOTE]
> Getters are acceptable when external code needs to read a value.
> Avoid setters — prefer immutability.

## DTOs & Data Holders

Use **public fields** when:

- Struct is a data container (dto, request, response, config, etc)
- JSON/YAML marshaling is needed
- All field combinations are valid states

> [!NOTE]
> If you need to validate the struct after construction, use a `Validate()` method.

```go
// Public fields - validation via method
type ReadFileRequest struct {
    Path   string
    Offset int64
    Limit  int64
}

func (r *ReadFileRequest) Validate(cfg *config.Config) error {
    if r.Path == "" {
        return ErrEmptyPath
    }
    return nil
}
```