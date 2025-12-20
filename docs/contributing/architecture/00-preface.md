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
- [ ] All exported errors defined in the `shared/` package
- [ ] Cross-package error checks use **Sentinel Errors** (via `errors.Is`)
