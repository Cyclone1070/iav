# tool Package

## Responsibility

The `tool` package defines tool types and display data structures. It is the foundational layer that other packages import for type definitions.

**Owns:**
- `Declaration` — Tool schema (name, description, parameters) sent to LLM
- `Schema` — JSON Schema type definitions for parameters
- `ToolDisplay` — Interface for UI display types
- `StringDisplay`, `DiffDisplay`, `ShellDisplay` — Concrete display types

**Does NOT own:**
- Tool execution logic (individual tools in subpackages)
- Tool registry or orchestration
