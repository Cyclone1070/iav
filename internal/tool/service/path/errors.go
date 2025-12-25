package path

import (
	"errors"
)

// -- Sentinels --

var (
	ErrOutsideWorkspace    = errors.New("path is outside workspace root")
	ErrWorkspaceRootNotSet = errors.New("workspace root not set")
	ErrNotADirectory       = errors.New("not a directory")
)
