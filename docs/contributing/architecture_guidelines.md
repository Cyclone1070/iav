# Go Architecture & Design Guidelines

> [!IMPORTANT]
> **Core Principle**: Make invalid states unrepresentable. Design your code so that it is impossible to misuse.
>
> These principles are strict guidelines to ensure maintainable, testable, and robust Go code. Any violations, no matter how small, will be rejected immediately during code review.

## Pre-Commit Checklist

Before submitting code, verify **every** item. A single unchecked box = rejection.

### Package Design
- [ ] No generic subdirectories (`model/`, `service/`, `utils/`, `types/`)
- [ ] No package exceeds 10-15 files (excluding `*_test.go`)
- [ ] No circular dependencies
- [ ] No junk drawer packages mixing unrelated logic

### Interfaces
- [ ] All interfaces defined in the **consumer** package, not the implementer
- [ ] All interfaces are minimal (≤5 methods)

### Structs & Encapsulation
- [ ] All domain entity fields are **private**
- [ ] All domain entities have `New*` constructors
- [ ] No direct struct initialization with `{}`
- [ ] DTOs have **no methods** attached
- [ ] No `json:`/`yaml:`/ORM tags on domain entities

### Validation & DI
- [ ] Static validation inside constructors
- [ ] Dynamic validation at start of method body (clearly commented)
- [ ] All dependencies injected via constructor
- [ ] No global state for dependencies

---

## 1. Package Design
**Goal**: Small, focused, reusable components.

*   **Small & Focused**: Packages should do one thing and do it well.
    *   **Why**: Single responsibility makes code easier to understand, test, and maintain.

*   **File Organization**: Do not create generic sub-directories like `model/`, `service/`, or `types/` inside your package.
    *   **Correct**: `feature/types.go`, `feature/service.go`
    *   **Incorrect**: `feature/models/types.go`, `feature/services/service.go`
    *   **Why**: Generic layers group by what code IS, not what it DOES. This scatters related logic across directories.

> [!CAUTION]
> **ANTI-PATTERN**: Layered Organization
>
> ```text
> # WRONG: Grouping by layer
> internal/
>   ├── controllers/
>   ├── services/
>   └── models/
>
> # WRONG: Internal layering inside package
> feature/user/
>   ├── models/user.go
>   └── services/register.go
>
> # CORRECT: Grouping by domain with flat files
> internal/
>   ├── order/
>   ├── payment/
>   └── customer/
> ```

*   **Splitting Rule**: If a package grows to 10-15 files, it is too big. Break it into focused sub-packages.
    *   **Why**: Large packages become hard to navigate and test. The urge to create `models/` is a symptom of bloat.
    *   **Action**: Split by domain (e.g., `feature/user/`, `feature/order/`), not by layer.

> [!CAUTION]
> **ANTI-PATTERN**: Flatten and Bloat
>
> When removing generic subdirectories, do NOT blindly flatten all files into the parent package.
>
> ```text
> # BEFORE: Internal layering (WRONG)
> feature/
>   ├── models/     (8 files)
>   ├── services/   (12 files)
>   └── handlers/   (5 files)
>
> # WRONG FIX: Flatten everything (now 25 files!)
> feature/
>   ├── user.go
>   ├── order.go
>   └── ... (25 files)
>
> # CORRECT FIX: Split by domain
> feature/
>   ├── user/       (types.go, service.go, handler.go)
>   ├── order/      (types.go, service.go, handler.go)
>   └── payment/    (types.go, service.go, handler.go)
> ```
>
> *   **Why**: You've traded one anti-pattern for another. Both violate "small and focused."
> *   **Rule**: If flattening exceeds 10-15 files, split into domain sub-packages.

*   **Hierarchy**: Nested packages are permitted for grouping related sub-features.
    *   **Why**: Hierarchy provides logical organization without violating single responsibility.

*   **Parent Package Directionality**: A parent package must pick ONE interaction direction with its sub-packages.
    *   **Option A (Composition Root)**: Parent imports sub-packages to wire them together. Sub-packages NEVER import the parent.
    *   **Option B (Shared Definition)**: Parent contains shared types/interfaces. Sub-packages import the parent. Parent NEVER imports sub-packages.
    *   **Why**: Mixing these directions guarantees circular dependencies. If `tools/` imports `tools/file`, then `tools/file` cannot import `tools/`.

*   **No Circular Dependencies**: If you hit a circular dependency, your design is wrong.
    *   **Why**: Circular deps create tight coupling and make testing impossible.
    *   **Solution**: Extract common definitions to a third package or decouple via interfaces.

> [!NOTE]
> **Single-File Directories Are Acceptable**
>
> When extracting shared code to prevent circular dependencies, a directory with one file is fine. Correct structure matters more than file count.
>
> *   ✅ `feature/errors/errors.go` (prevents circular deps between parent and children)
> *   ✅ `feature/pagination/pagination.go` (descriptive helper package)
> *   ❌ `feature/utils/errors.go` (generic junk drawer)

> [!CAUTION]
> **ANTI-PATTERN**: Junk Drawer
>
> *   **Bad**: `feature/utils` or `feature/common` containing mixed logic (strings, encryption, formatting).
> *   **Why**: Violates cohesion. Becomes a dumping ground where dependencies tangle.
> *   **Solution**: Group by what it operates on:
>     *   String helpers → `feature/text` or `internal/strutil`
>     *   Time helpers → `feature/timeext`
>     *   Domain logic → `feature/auth/hashing` (NOT `feature/auth/utils`)
> *   **Exception**: A `utils` package is permissible ONLY IF it uses strictly the standard library.

---

## 2. Interfaces: Consumer-Defined
**Goal**: Decoupling and testability.

*   **Define Where Used**: Do NOT define interfaces in the implementing package. Define them in the consumer package.
    *   **Why**: The consumer knows what it needs. The implementer should not dictate the contract.
    *   **Benefit**: You can swap implementations without touching the consumer. You can mock easily in tests.

*   **Small Interfaces**: Keep interfaces minimal (`Reader` vs `ReadWriteCloser`).
    *   **Why**: Large interfaces force implementers to provide methods they don't use.
    *   **Rule**: If an interface has more than 5 methods, split it by use case.

*   **No Shared Interfaces**: Interfaces are local to the package that uses them. NOT shared across packages, even siblings.
    *   **Why**: Sibling packages should not know each other exist. Each defines its own interface with only the methods IT needs.
    *   **Trade-off**: This creates duplication. You accept small duplication in exchange for massive decoupling. This is correct.

> [!CAUTION]
> **ANTI-PATTERN**: Shared Interface Library
>
> *   **Bad**: Creating `internal/interfaces/filesystem.go` with a 10-method interface everyone imports.
> *   **Why**: This is `model/` in disguise. It couples all consumers and forces implementers to satisfy methods they don't need.
> *   **Solution**: Each consumer defines its own minimal interface. Duplication is acceptable. Coupling is not.

**Example**:

```go
// package service - defines only what IT needs
type UserRepository interface {
    Find(id string) (*User, error)
}

type Service struct {
    repo UserRepository
}
```

**Sibling Isolation Example**:

```go
// package file - defines only what IT needs
type fileSystem interface {
    Stat(path string) (FileInfo, error)
    ReadFileRange(path string, offset, limit int64) ([]byte, error)
}

// package directory - different package, own interface
type fileSystem interface {
    Stat(path string) (FileInfo, error)
    ListDir(path string) ([]FileInfo, error)
}

// Both satisfied by the same OSFileSystem, but neither knows about the other.
```

> [!CAUTION]
> **ANTI-PATTERN**: Leaky Interfaces
>
> *   **Bad**: `Save(u *User) (sql.Result, error)` — ties interface to SQL.
> *   **Good**: `Save(u *User) (string, error)` — returns what you need without leaking implementation.
> *   **Why**: Leaky interfaces prevent alternative implementations (file system, memory store).

> [!CAUTION]
> **ANTI-PATTERN**: Confusing Interface Sharing with Implementation Sharing
>
> *   **Interfaces**: Consumer-defined, NOT shared across packages.
> *   **Implementations**: CAN be shared in dedicated packages (via dependency injection or direct import for pure helpers).
> *   **Mistake**: Duplicating implementations because you're avoiding shared interfaces. Share the concrete type, not the interface.

---

## 3. Structs & Encapsulation
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

---

## 4. Validation Strategy
**Goal**: Trust your objects.

*   **Static Validation (Invariants)**: Validate inside the constructor.
    *   **Where**: `New...` function.
    *   **Guarantee**: It is impossible to create an invalid instance.
    *   **Scope**: Internal consistency (e.g., "ID cannot be empty", "email must have @").

*   **Dynamic Validation (Business Rules)**: Validate at the start of the method body.
    *   **Where**: First lines of the method, clearly commented.
    *   **Scope**: External state (e.g., "file exists", "user is unique in DB").
    *   **Why**: Separating static and dynamic validation makes code predictable and testable.

**Example**:

```go
// Static validation in constructor
func NewUser(id, email string) (*User, error) {
    if id == "" {
        return nil, errors.New("id is required")
    }
    if !strings.Contains(email, "@") {
        return nil, errors.New("invalid email")
    }
    return &User{id: id, email: email}, nil
}

// Dynamic validation in method
func (u *User) Save(repo UserRepository) error {
    // 1. Dynamic Validation
    if exists := repo.Exists(u.id); exists {
        return errors.New("user already exists")
    }

    // 2. Implementation
    return repo.Save(u)
}
```

---

## 5. Dependency Injection & Testing
**Goal**: Deterministic, isolated tests.

*   **Strict DI**: Dependencies MUST be passed via constructor.
    *   **Why**: Explicit dependencies make code testable and prevent hidden coupling.

*   **No Globals**: Never use global state for dependencies.
    *   **Why**: Globals create hidden dependencies, prevent parallel tests, and make code unpredictable.

*   **Mocking**: Use mocks for all dependencies in unit tests.
    *   **Why**: Real dependencies (databases, filesystems) make tests slow, flaky, and non-deterministic.

*   **No Temp Files/Dirs**: Do not touch the filesystem in unit tests. Mock the interface.
    *   **Why**: Filesystem operations are slow and create test pollution across runs.

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

// Test with mock - no disk access
func TestFileProcessor(t *testing.T) {
    mockFS := new(MockFileSystem)
    processor := NewFileProcessor(mockFS)
    // Test logic without touching disk
}
```
