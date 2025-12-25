package shell

import (
	"errors"
)

// -- Shell Tool Sentinels --

var (
	ErrCommandRequired = errors.New("command cannot be empty")
	ErrInvalidTimeout  = errors.New("timeout_seconds cannot be negative")
	ErrEnvFileParse    = errors.New("invalid line in env file")
)
