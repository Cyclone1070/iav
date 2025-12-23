package shell

import (
	"context"
	"time"

	"github.com/Cyclone1070/iav/internal/tool/executor"
)

// envFileReader defines the minimal filesystem interface needed for reading environment files.
type envFileReader interface {
	ReadFileRange(path string, offset, limit int64) ([]byte, error)
}

// pathResolver defines workspace path resolution operations.
type pathResolver interface {
	Abs(path string) (string, error)
	Rel(path string) (string, error)
}

// commandExecutor defines the interface for executing shell commands.
type commandExecutor interface {
	Run(ctx context.Context, cmd []string, dir string, env []string) (*executor.Result, error)
	RunWithTimeout(ctx context.Context, cmd []string, dir string, env []string, timeout time.Duration) (*executor.Result, error)
}
