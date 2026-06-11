# Architectural Decision Records (ADRs)

Rules are strictly enforced.

## 1. Horizontal Isolation (No Peer Imports)
*   **Rule**: `internal/` packages must never import peer packages. Only `cmd/` can import multiple internal packages to wire dependencies.
*   **Enforcement**: `internal/arch_test.go`
*   **Don't**: `import "github.com/Cyclone1070/iav/internal/sandbox"` inside `internal/stages/docker`.
*   **Do**: Declare local consumer interfaces and inject implementations at composition root (`cmd/`).

## 2. Pure Data Domain Contracts
*   **Rule**: `internal/domain` is strictly for logic-free, interface-free data structures.
*   **Don't**: Define interfaces or helper functions in `internal/domain`.
*   **Do**: Keep `domain` limited to pure structs and configuration schemas.

## 3. Consumer-Owned Abstractions (DIP)
*   **Rule**: Interfaces belong to the package that consumes them, not the package that implements them.
*   **Don't**: Declare a central interface package.
*   **Do**: Define small, targeted interfaces locally inside the consuming package.

## 4. Compile-Time Type Safety (Zero `any`)
*   **Rule**: Avoid `any` or `interface{}` return types. Use structural/marker interfaces.
*   **Don't**: Return `any` from configuration parsers.
*   **Do**: Return a local interface that implements a signature marker like `ConfigType() string`.

## 5. Dependency-Injected Structs
*   **Rule**: Avoid exposing package-level global functions for services/parsers. Expose constructors returning structs.
*   **Don't**: Define package-level globals like `func Parse(...)`.
*   **Do**: Define `type Parser struct` with dependencies inside it, instantiated with a `New()` constructor.

## 6. DI Mocked Unit Tests
*   **Rule**: Unit tests must run in total isolation. No disk/network/process calls.
*   **Don't**: Run real Docker commands or read real files in unit tests.
*   **Do**: Inject mocks of local consumer interfaces to simulate all execution paths.

## 7. Shared Domain Structs Only
*   **Rule**: Every struct defined in `internal/domain` must be imported and used by 2 or more packages (excluding `internal/domain` itself).
*   **Don't**: Put package-specific configuration structs or models in the shared domain package.
*   **Do**: Place shared, module-wide schemas and DTOs in `domain`.

## 8. Symmetrical Test File Naming
*   **Rule**: Every test file `xxx_test.go` must have a corresponding implementation file `xxx.go` in the same package (excluding `integration_test.go`, `arch_test.go`, and `mock_test.go`).
*   **Don't**: Create feature-based test files like `parse_test.go` or `sanitise_test.go` without a matching implementation file.
*   **Do**: Ensure `implement_test.go` maps 1:1 with `implement.go`.
