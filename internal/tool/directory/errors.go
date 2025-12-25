package directory

import (
	"errors"
)

// -- Sentinels --

var (
	ErrFileMissing     = errors.New("file or path does not exist")
	ErrNotADirectory   = errors.New("not a directory")
	ErrPathRequired    = errors.New("path is required")
	ErrPatternRequired = errors.New("pattern is required")
	ErrInvalidPattern  = errors.New("invalid pattern")
)
