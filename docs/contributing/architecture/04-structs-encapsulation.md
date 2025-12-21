# 4. Structs & Encapsulation

**Goal**: Control state and enforce invariants.

> [!NOTE]
> Error structs are not covered by this section, these rules don't apply to errors. See [Error Handling](./05-error-handling.md).

*   **Private Fields**: All domain entity fields MUST be private.
    *   **Why**: Private fields prevent external code from putting the object into an invalid state.

*   **Public Constructor**: You MUST provide a `New...` constructor for every domain entity.
    *   **Why**: Constructors are the single point of validation. If you add validation later, you won't need to refactor every usage.

> [!NOTE]
> **STRICT RULE**: Direct initialization with `{}` is strictly forbidden with **no exception**.
> 
> Even if there's no validation yet and it appears to be boilerplate. Even if it's a private struct used in a single place with a single primitive field. The struct will grow soon and refactoring is unavoidable. Future proofing is more important than initial, short-lived convenience. 
>
> It might sound obvious, but go encourages public fields with direct initialization for simple structs, trading future proofing for initial convenience. This is not a trade off we want to make.

*   **DTOs**: Data Transfer Objects are an exception. They are allowed to have public fields but NO methods.
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
