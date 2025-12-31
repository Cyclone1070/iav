package shell

import (
	"context"
	"time"

	"github.com/Cyclone1070/iav/internal/tool/service/executor"
	"github.com/Cyclone1070/iav/internal/tool/service/fs"
)

// envFileOps defines the minimal filesystem interface needed for reading environment files.
type envFileOps interface {
	ReadFileLines(path string, startLine, endLine int) (*fs.ReadFileLinesResult, error)
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
