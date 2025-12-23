package executor

import (
	"errors"
	"fmt"
)

// ErrTimeout is returned when a command exceeds its timeout.
var ErrTimeout = errors.New("command timeout")

// CommandError represents generic command execution failures (start, output, wait).
type CommandError struct {
	Cmd   string
	Cause error
	Stage string // "start", "read output", "execution"
}

func (e *CommandError) Error() string {
	return fmt.Sprintf("command %s failed at %s: %v", e.Cmd, e.Stage, e.Cause)
}
func (e *CommandError) Unwrap() error { return e.Cause }
