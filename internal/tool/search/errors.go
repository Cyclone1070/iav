package search

import (
	"errors"
)

// -- Sentinels --

var (
	ErrQueryRequired = errors.New("query is required")
	ErrFileMissing   = errors.New("file or path does not exist")
	ErrNotADirectory = errors.New("path is not a directory")
)
