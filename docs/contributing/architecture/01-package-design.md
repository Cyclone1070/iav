# 1. Package Design

**Goal**: Decoupled packages. Someone working on or modifying a package shouldn't need to know about other packages.

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

*   **File Organization**: Do not create generic sub-packages like `model/`, `service/`, or `types/` inside your package.
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

> [!NOTE]
> **Grouping Directories vs Packages**
>
> A **grouping directory** is a folder that contains other packages but has **no `.go` files itself**. Naming for these is less strict because they have minimal effect on actual code readability. These should be named so that navigating and discovering packages is easy and natural.
>
> ```text
> # ACCEPTABLE: Grouping directory with generic name if it makes sense
> internal/tool/
>   └── helper/           # No .go files - just a folder
>       ├── hash/       # package hash
>       ├── pagination/ # package pagination
>       └── content/    # package content
>
> # NOT ACCEPTABLE: Package with generic name
> internal/tool/util/util.go  # package util - BAD
> ```
>

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
>     *   Domain logic → `feature/auth/hashing`
