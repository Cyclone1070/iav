# Go Architecture & Design Guidelines

> [!IMPORTANT]
> **Core Principle**: Make invalid states unrepresentable. Design your code so that it is impossible to misuse.
>
> These principles are strict guidelines to ensure maintainable, testable, and robust Go code. Any violations, no matter how small, will be rejected immediately during code review.

## Pre-Commit Checklist

Before submitting code, verify **every** item.

### Package Design
- [ ] No generic subdirectories (`model/`, `service/`, `utils/`, `types/`)
- [ ] No package exceeds 10-15 files (excluding `*_test.go`)
- [ ] No circular dependencies
- [ ] No junk drawer packages mixing unrelated logic

### Dependency Injection
- [ ] Pure helpers imported directly (no interface needed)
- [ ] Dependencies injected via constructor with consumer-defined interface
- [ ] No global state for dependencies

### Interfaces
- [ ] All interfaces defined in the **consumer** package, not the implementer
- [ ] No unused methods in interfaces (grep to verify each method is called)
- [ ] Exception: import interface from helper package if already coupled

### Structs & Encapsulation
- [ ] All domain entity fields are **private**
- [ ] All domain entities have `New*` constructors
- [ ] No direct struct initialization with `{}`
- [ ] DTOs have **no methods** attached
- [ ] No `json:`/`yaml:`/ORM tags on domain entities

### Validation
- [ ] Constructor validation for everything knowable from inputs
- [ ] Runtime validation at method start (clearly commented `// Runtime Validation`)

### Testing
- [ ] All mocks defined locally in `*_test.go` files (no shared `mock/` package)
- [ ] All test helpers defined locally in test files
### Error Handling
- [ ] No shared error packages (e.g., `errutil`)
- [ ] Errors defined in the package that raises them
- [ ] Cross-package error checks use **Behavioral Interfaces** (no imports)

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
> internal/feature/
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
    *   **Why**: Large packages become hard to navigate and test. The urge to create `models/` or `services/` is a symptom of bloat.
    *   **Action**: Split by domain (e.g., `internal/user/`, `internal/order/`), not by layer.

> [!CAUTION]
> **ANTI-PATTERN**: Flatten and Bloat
>
> When removing generic subdirectories, do NOT blindly flatten all files into the parent package.
>
> ```text
> # BEFORE: Internal layering (WRONG)
> internal/feature/
>   ├── models/     (8 files)
>   ├── services/   (12 files)
>   └── handlers/   (5 files)
>
> # WRONG FIX: Flatten everything (now 25 files!)
> internal/feature/
>   ├── user.go
>   ├── order.go
>   └── ... (25 files)
>
> # CORRECT FIX: Split by domain
> internal/
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

## 2. Dependency Injection
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

---

## 3. Interfaces: Consumer-Defined
**Goal**: Decoupling and testability.

*   **Define Where Used**: Do NOT define interfaces in the implementing package. Define them in the consumer package.
    *   **Why**: The consumer knows what it needs. The implementer should not dictate the contract.
    *   **Benefit**: You can swap implementations without touching the consumer. You can mock easily in tests.

> [!CAUTION]
> **ANTI-PATTERN**: Copy Exact Interface Methods
>
> *   **Bad**: Consumer interface declares methods it never calls (e.g., `Rename()` exists but is never invoked).
> *   **Why**: The interface exposes internal implementation details or dependencies of dependencies.
> *   **How It Happens**: Copying methods from the implementer instead of auditing actual usage.
> *   **Solution**: Check your package for each interface method. If unused, remove it.


*   **No Shared Interfaces**: Interfaces are local to the package that uses them. NOT shared across packages, even siblings.
    *   **Why**: Sibling packages should not know each other exist. Each defines its own interface with only the methods IT needs.
    *   **Trade-off**: This creates duplication. You accept small duplication in exchange for massive decoupling. This is correct.

> [!CAUTION]
> **ANTI-PATTERN**: Shared Interface Library
>
> *   **Bad**: Creating `internal/interfaces/filesystem.go` with a 10-method interface everyone imports.
> *   **Why**: This is `model/` in disguise. It couples all consumers and forces implementers to satisfy methods they don't need.
> *   **Solution**: Each consumer defines its own minimal interface. Duplication is acceptable. Coupling is not.

> [!CAUTION]
> **ANTI-PATTERN**: Confusing Interface Sharing with Implementation Sharing
>
> *   **Interfaces**: Consumer-defined, NOT shared across packages.
> *   **Implementations**: CAN be shared in dedicated packages (via dependency injection or direct import for pure helpers).
> *   **Mistake**: Duplicating implementations because you're avoiding shared interfaces. Share the concrete type, not the interface.

> [!TIP]
> **Exception: Helper Package Interfaces**
>
> If you already import a helper package (e.g., `pathutil`) and call its functions, you are coupled to it.
> In this case, **import its interface directly** rather than redefining an identical interface locally.
>
> *   **Bad**: Redefining `type pathResolver interface { Lstat()... }` when you already import `pathutil.Resolve`.
> *   **Good**: Use `pathutil.FileSystem` directly since coupling already exists.
> *   **Why**: Redefining the interface is noise. It disguises where the requirement comes from.

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

---

## 4. Structs & Encapsulation
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

## 5. Validation Strategy
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
        return nil, errors.New("path is a directory, not a file")
    }

    // 2. Implementation
    content, err := t.fs.ReadFile(abs)
    ...
}
```

---

---

## 6. Testing
**Goal**: Deterministic, isolated, self-contained tests.

*   **Mocking**: Use mocks for all dependencies in unit tests.
    *   **Why**: Real dependencies (databases, filesystems) make tests slow, flaky, and non-deterministic.

*   **No Temp Files/Dirs**: Do not touch the filesystem in unit tests. Mock the interface.
    *   **Why**: Filesystem operations are slow and create test pollution across runs.

*   **Local Mocks**: Define mocks inside the `*_test.go` file that uses them. No shared `mock/` package.
    *   **Why**: Consumer-defined interfaces mean each test defines its own interface. The mock implements THAT interface. Mocks can't drift. No import cycles.

*   **Local Helpers**: Test helper functions should be defined in the test file that uses them.
    *   **Why**: Keeps tests self-contained and readable.

*   **Exception – `internal/testutils`**: Truly generic helpers MAY be placed here.
    *   `testutils` MUST NOT import anything from the codebase (`internal/*`, `cmd/*`).
    *   Only standard library and external dependencies allowed. Use sparingly.

> [!CAUTION]
> **ANTI-PATTERN**: Shared Mock Package
>
> *   **Bad**: `internal/testing/mock/filesystem.go` with a "god mock" used everywhere.
> *   **Why**: Creates coupling, import cycles, and mocks that implement methods no single consumer needs.
> *   **Solution**: Define `mockFileSystem` inside `file/read_test.go` with only the methods `file.fileSystem` requires.


---

## 7. Error Handling
**Goal**: Decoupling and robustness.

*   **Behavioral Interfaces**: Check *what* the error does, not *who* it is.
    *   **Provider**: Implement behavioral methods on error structs (e.g., `NotFound() bool { return true }`).
    *   **Consumer**: Define a local interface for the behavior you need to check.
    *   **Why**: Completely removes import dependencies between consumer and provider. The consumer doesn't need to know the provider exists to handle its errors.

> [!CAUTION]
> **FORBIDDEN ERROR PATTERNS**
>
> The following error patterns are **strictly prohibited**:
>
> | Pattern | Example | Why Bad |
> |---------|---------|---------|
> | **Sentinel Errors** | `var ErrNotFound = errors.New("not found")` | Cannot carry context, forces `==` checks that couple packages |
> | **fmt.Errorf** | `return fmt.Errorf("failed: %w", err)` | Anonymous errors, no behavioral method, untestable |
> | **errors.New inline** | `return errors.New("something failed")` | Same as sentinel - no context, no behavior |
> | **pkg/errors** | `errors.Wrap(err, "context")` | Deprecated pattern, use behavioral errors |
>
> **REQUIRED**: All errors MUST be behavioral error types:
> ```go
> // ✅ CORRECT: Behavioral error type
> type NotFoundError struct {
>     Path string
> }
> func (e *NotFoundError) Error() string { return "not found: " + e.Path }
> func (e *NotFoundError) NotFound() bool { return true }
>
> // ❌ WRONG: Sentinel error
> var ErrNotFound = errors.New("not found")
>
> // ❌ WRONG: fmt.Errorf
> return fmt.Errorf("file not found: %s", path)
> ```

*   **Local Error Definitions**: Define errors in the package that raises them.
    *   **Why**: Keeps packages self-contained.

> [!CAUTION]
> **ANTI-PATTERN**: Shared Error Package
>
> *   **Bad**: `internal/errutil` containing all system errors.
> *   **Why**: This is a "junk drawer" that couples every package to every other package.
> *   **Solution**: Delete it. Define errors locally and use behavioral interfaces.


**Example**:

```go
// Provider (package fs)
type NotFoundError struct { name string }
func (e *NotFoundError) Error() string { return "not found: " + e.name }
func (e *NotFoundError) NotFound() bool { return true } // <--- THE BEHAVIOR

// 2. Consumer (package usecase) - NO import of "fs" needed!
type notFound interface {
    NotFound() bool
}

func (u *UseCase) Do() {
    if err := u.opener.Open("file.txt"); err != nil {
        // Check behavior via type assertion
        if e, ok := err.(notFound); ok && e.NotFound() {
            // Handle missing file
            return
        }
    }
}
```
