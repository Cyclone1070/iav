package path

import (
	"errors"
)

// -- Sentinels --

var (
	ErrOutsideWorkspace = errors.New("path is outside workspace root")
)
