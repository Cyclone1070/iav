# 4. Structs & Encapsulation

**Goal**: Control state and enforce invariants.

*   **Private Fields**: All domain entity fields MUST be private.
    *   **Why**: Private fields prevent external code from putting the object into an invalid state.

*   **Public Constructor**: You MUST provide a `New...` constructor for every domain entity.
    *   **Why**: Constructors are the single point of validation. If you add validation later, you won't need to refactor every usage.
    *   **Strict Rule**: Direct initialization with `{}` is forbidden, even if there's no validation yet.

> [!CAUTION]
> **ANTI-PATTERN**: Constructor Bypass
>
> *   **Bad**: `user := &User{email: "..."}`
> *   **Good**: `user := NewUser("...")`
> *   **Why**: Bypassing the constructor skips validation and makes invariants impossible to guarantee.

*   **DTOs**: Use Data Transfer Objects with public fields for serialization (JSON, API). DTOs have NO methods.
    *   **Why**: DTOs are pure data carriers. Behavior belongs in domain entities.

> [!CAUTION]
> **ANTI-PATTERN**: Tag Pollution
>
> *   **Rule**: NEVER add `json:`, `yaml:`, or ORM tags to domain entities.
> *   **Why**: Couples business logic to external interfaces.
> *   **Solution**: Use dedicated DTOs for serialization and separate DB models for persistence.

**Example**:

```go
// Domain Entity - private fields, constructor enforces invariants
type User struct {
    id    string
    email string
}

func NewUser(id, email string) (*User, error) {
    if id == "" {
        return nil, errors.New("id required")
    }
    return &User{id: id, email: email}, nil
}

// DTO - public fields, no methods, serialization tags
type UserDTO struct {
    ID    string `json:"id"`
    Email string `json:"email"`
}
```
