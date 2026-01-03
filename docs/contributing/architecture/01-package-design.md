# 1. Package Design

**Goal**: High cohesion. Code related to a feature are placed close together, usually in the same package.

*   **Feature-Based Packages**: A package must represent a **feature** or **domain concept**, not a layer (controller, service, model).
    *   **Why**: Feature packages contain all related code (types, logic, handlers) in one place. Layer packages scatter related code across the codebase.

> [!NOTE]
> **Multiple implementations**: If a package has multiple implementations (e.g., different storage backends), use sub-packages for each implementation, with shared types and errors in the parent.
>
> **Example**:
> ```text
> internal/storage/
>    ├── memory/      # In-memory storage implementation
>    ├── file/        # File-based storage implementation
>    ├── types.go   # Shared types
>    └── errors.go   # Shared errors
> ```

*   **Package Naming**: Package structure and naming should provide enough context to understand the content and purpose of each package.
    *   **Why**: Clear names enable discoverability and prevent packages from becoming dumping grounds. Generic names lead to junk drawers grouping unrelated logic.
    *   **Guideline**: Names like `helper/`, `service/`, or `util/` are acceptable as parent directories when their children and/or parent have specific, descriptive names.

> [!NOTE]
> **Example**: Acceptable Structure
>
> ```text
> internal/tool/
>   ├── helper/
>   │   ├── pagination/   # Specific: handles offset/limit logic
>   │   └── content/      # Specific: binary detection
>   └── service/
>       ├── fs/           # Specific: filesystem operations
>       └── executor/     # Specific: command execution
> ```
>
> The parent directories (`helper/`, `service/`) provide organizational context, while the actual packages have descriptive names.

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

*   **Splitting Rule**: If a package grows to 10-15 files (excluding tests), it is likely too big. Break it into focused sub-packages.
    *   **Why**: Large packages become hard to navigate and test.
    *   **Action**: Split by domain (e.g., `internal/user/`, `internal/order/`), not by layer.

> [!NOTE]
> **Single-File Directories Are Acceptable**
>
> When extracting shared code to prevent circular dependencies, a directory with one file is fine. Correct structure matters more than file count.
