package shell

import (
	"fmt"
	"time"
)

// TimeoutError is returned when a shell command exceeds its timeout.
type TimeoutError struct {
	Command  []string
	Duration time.Duration
}

func (e *TimeoutError) Error() string {
	return fmt.Sprintf("shell command %v timed out after %v", e.Command, e.Duration)
}

func (e *TimeoutError) Timeout() bool {
	return true
}

// EnvFileReadError is returned when reading an env file fails.
type EnvFileReadError struct {
	Path  string
	Cause error
}

func (e *EnvFileReadError) Error() string {
	return fmt.Sprintf("failed to read env file %s: %v", e.Path, e.Cause)
}

func (e *EnvFileReadError) Unwrap() error {
	return e.Cause
}

func (e *EnvFileReadError) IOError() bool {
	return true
}

// EnvFileParseError is returned when an env file has an invalid format.
type EnvFileParseError struct {
	Path    string
	Line    int
	Content string
}

func (e *EnvFileParseError) Error() string {
	return fmt.Sprintf("invalid line %d in env file %s: %s", e.Line, e.Path, e.Content)
}

func (e *EnvFileParseError) InvalidInput() bool {
	return true
}

// EnvFileScanError is returned when scanning an env file fails.
type EnvFileScanError struct {
	Path  string
	Cause error
}

func (e *EnvFileScanError) Error() string {
	return fmt.Sprintf("error reading env file %s: %v", e.Path, e.Cause)
}

func (e *EnvFileScanError) Unwrap() error {
	return e.Cause
}

func (e *EnvFileScanError) IOError() bool {
	return true
}

// CommandRequiredError is returned when a command is missing.
type CommandRequiredError struct{}

func (e *CommandRequiredError) Error() string {
	return "command cannot be empty"
}

func (e *CommandRequiredError) InvalidInput() bool {
	return true
}

// NegativeTimeoutError is returned when a timeout is negative.
type NegativeTimeoutError struct {
	Value int
}

func (e *NegativeTimeoutError) Error() string {
	return fmt.Sprintf("timeout_seconds cannot be negative: %d", e.Value)
}

func (e *NegativeTimeoutError) InvalidInput() bool {
	return true
}
