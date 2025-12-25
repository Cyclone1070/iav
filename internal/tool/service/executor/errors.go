package executor

import (
	"errors"
)

// ErrTimeout is returned when a command exceeds its timeout.
var ErrTimeout = errors.New("command timeout")
