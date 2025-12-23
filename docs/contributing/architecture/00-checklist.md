# Go Architecture & Design Guidelines

> [!IMPORTANT]
> **Design Goals**: Future proofing code over initial convenience. Decoupling is the key to maintainable, testable, and robust Go code.
>
> These principles are strict guidelines to ensure maintainable, testable, and robust Go code. Any violations, no matter how small, will be rejected immediately during code review.

## Pre-Commit Checklist

Before submitting code, verify **every** item.

### Package Design
- [ ] No package exceeds 10-15 files (excluding `*_test.go`)
- [ ] Parent package does NOT import its sub-packages
- [ ] Types and errors live with their implementation package
- [ ] Exception: Shared types/errors in parent ONLY for multiple implementations

### Dependency Injection
- [ ] Pure helpers imported directly (including their interfaces, structs, errors)
- [ ] Dependencies injected via constructor with consumer-defined interface

### Interfaces
- [ ] All interfaces defined in the **consumer** package, not the implementer
- [ ] No unused methods in interfaces (grep to verify each method is called)
- [ ] Exception: import interface from helper package if already coupled

### Struct Design
- [ ] Services & domain entities use private fields + `NewT()` constructor
- [ ] DTOs & data holders use public fields + `Validate()` method if needed

### Error Handling
- [ ] Errors defined in the same package as the code that returns them
- [ ] Exception: Shared errors in parent ONLY for multiple implementations
- [ ] Cross-package error checks use **Sentinel Errors** (via `errors.Is`)

### Testing
- [ ] All mocks defined locally in `*_test.go` files (no shared `mock/` package)
- [ ] All test helpers defined locally in test files
