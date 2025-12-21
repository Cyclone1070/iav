# 1. Package Design

**Goal**: Decoupled packages. Someone working on or modifying a package shouldn't need to know about other packages.

*   **Feature-Based Packages**: A package must represent a **feature** or **domain concept**, not a layer (controller, service, model).
    *   **Why**: Feature packages contain all related code (types, logic, handlers) in one place. Layer packages scatter related code across the codebase.

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

> [!NOTE]
> **Single-File Directories Are Acceptable**
>
> When extracting shared code to prevent circular dependencies, a directory with one file is fine. Correct structure matters more than file count.

*   **Parent Package Role**: The parent package holds **shared types and errors** for its children.
    *   **Contains**: Shared types (`types.go`) and errors (`errors.go`). These can be split into smaller files if too big.
    *   **Import Rule**: Parent must **NEVER** import its sub-packages.
    *   **Why**: Without shared types, touching the data means importing the producer package. Shared types let you work with the data without coupling to who produced it.

> [!NOTE]
> **Shared Types (Parent vs. Local)**: How do you know where to place what struct if the interfaces are consumer defined?
>
> Decide based on **who the type is meant for**:
> - **Meant for consumers → Parent**: Types in public method signatures (Requests, Responses, Errors) are the Contract.
> - **Meant for wiring → Local**: Types for construction/configuration (Config, Options) and private types, stay in the sub-package.
>
> Consumers need the Contract. Wiring (`main.go`) can import sub-packages directly without breaking decoupling.

> [!CAUTION]
> **ANTI-PATTERN**: Junk Drawer
>
> *   **Bad**: `feature/utils` or `feature/common` containing mixed logic (strings, encryption, formatting).
> *   **Why**: Violates cohesion. Becomes a dumping ground where dependencies tangle.
> *   **Solution**: Group by what it operates on:
>     *   String helpers → `feature/text` or `internal/strutil`
>     *   Time helpers → `feature/timeext`
>     *   Domain logic → `feature/auth/hashing` (NOT `feature/auth/utils`)

*   **Naming Convention**: Use suffixes to signal package type at a glance.
    *   `*serv` — Dependency with I/O (inject via interface): `fsserv`, `pathserv`, `gitserv`
    *   `*util` — Pure stateless helper (import directly): `hashutil`, `contentutil`
    *   No suffix — core feature: `file`, `shell`, `directory`
