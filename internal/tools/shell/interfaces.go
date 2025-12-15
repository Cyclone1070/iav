package shell

import (
	"context"
	"io"
	"os"
)

// fileSystem defines the minimal filesystem interface needed by shell tools.
// This is a consumer-defined interface per architecture guidelines §2.
type fileSystem interface {
	Stat(path string) (os.FileInfo, error)
	Lstat(path string) (os.FileInfo, error)
	Readlink(path string) (string, error)
	UserHomeDir() (string, error)
	ReadFileRange(path string, offset, limit int64) ([]byte, error)
}

// binaryDetector defines the interface for binary content detection.
type binaryDetector interface {
	IsBinaryContent(content []byte) bool
}

// Process defines the minimal process interface needed by shell tools.
type Process interface {
	Wait() error
	Kill() error
	Signal(sig os.Signal) error
}

// commandExecutor defines the interface for executing shell commands.
type commandExecutor interface {
	Start(ctx context.Context, cmd []string, opts ProcessOptions) (Process, io.Reader, io.Reader, error)
	Run(ctx context.Context, cmd []string) ([]byte, error)
}

// pathResolver defines the interface for path resolution.
type pathResolver interface {
	Resolve(workspaceRoot string, fs fileSystem, path string) (abs string, rel string, err error)
}
