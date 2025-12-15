package search

import (
	"context"
	"io"
	"os"

	"github.com/Cyclone1070/iav/internal/tools/shell"
)

// fileSystem defines the minimal filesystem interface needed by search tools.
// This is a consumer-defined interface per architecture guidelines §2.
type fileSystem interface {
	Stat(path string) (os.FileInfo, error)
	Lstat(path string) (os.FileInfo, error)
	Readlink(path string) (string, error)
	UserHomeDir() (string, error)
}

// commandExecutor defines the interface for executing shell commands.
type commandExecutor interface {
	Start(ctx context.Context, cmd []string, opts shell.ProcessOptions) (shell.Process, io.Reader, io.Reader, error)
}

// pathResolver defines the interface for path resolution.
type pathResolver interface {
	Resolve(workspaceRoot string, fs fileSystem, path string) (abs string, rel string, err error)
}
