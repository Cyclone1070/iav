# Shell Tool

The Shell Tool (`internal/tools/shell.go`) allows the agent to execute commands on the local machine within the workspace boundary. It is designed with strict security policies and robust output handling.

## Features

- **Workspace Confinement**: Commands are executed within the defined workspace root. Attempts to access files outside the workspace (e.g., via `../` or symlinks) are blocked.
- **Policy Enforcement**: Commands are validated against a configurable policy (`CommandPolicy`):
  - **Allow list**: Commands automatically approved for execution
  - **Deny list**: Commands automatically rejected
  - **Default**: Commands not in either list require user approval (ask)
  - **SessionAllow**: Runtime overrides that allow previously-approved commands for the session
- **Docker Readiness**: Automatically checks if Docker is running before executing Docker commands, and attempts to start it if configured.
- **Output Collection**: Captures stdout and stderr with size limits (`MaxBytes`) to prevent memory exhaustion. Detects binary output and truncates it.
- **Timeout Management**: Enforces execution time limits (`TimeoutSeconds`) and ensures processes are killed (SIGTERM -> SIGKILL) on timeout.
- **PTY Support**: Supports pseudo-terminal execution for interactive commands (if requested).

## Usage

```go
req := models.ShellRequest{
    Command:        []string{"ls", "-la"},
    WorkingDir:     "src",
    TimeoutSeconds: 10,
}
resp, err := shellTool.Run(ctx, wCtx, req)
```

## Configuration

The tool relies on `models.WorkspaceContext` for configuration:
- `WorkspaceRoot`: The absolute path to the workspace.
- `CommandPolicy`: Defines allowed and approved commands.
- `DockerConfig`: Configuration for Docker readiness checks.

## Security

- **Path Traversal**: All paths are resolved and validated to be within `WorkspaceRoot`.
- **Command Injection**: Commands are executed as slice of arguments, avoiding shell injection vulnerabilities (unless `sh -c` is explicitly allowed and used).
- **Resource Limits**: Output size and execution time are strictly limited.
