package todo

import (
	"errors"
)

// -- Sentinels --

var (
	ErrInvalidStatus    = errors.New("invalid status")
	ErrEmptyDescription = errors.New("description cannot be empty")
)
