package directory

import (
	"context"
	"os"

	"github.com/Cyclone1070/iav/internal/tool/executor"
)

// pathResolver defines workspace path resolution operations.
type pathResolver interface {
	Abs(path string) (string, error)
	Rel(path string) (string, error)
}

// dirFinder defines the filesystem operations needed for finding files.
// Note: Does NOT include ListDir - this tool uses the fd command instead.
type dirFinder interface {
	Stat(path string) (os.FileInfo, error)
}

// commandExecutor defines the interface for executing find commands.
// This is a consumer-defined interface per architecture guidelines §2.
type commandExecutor interface {
	Run(ctx context.Context, cmd []string, dir string, env []string) (*executor.Result, error)
}
