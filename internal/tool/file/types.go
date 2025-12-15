package file

import (
	"fmt"
	"os"

	"github.com/Cyclone1070/iav/internal/config"
)

// Operation represents a single edit operation for EditFile.
// Before must be a non-empty literal snippet that exists in the file.
// ExpectedReplacements must match the exact number of occurrences of Before in the file.
// If ExpectedReplacements is omitted (0), it defaults to 1.
type Operation struct {
	Before               string // required, non-empty literal snippet
	After                string // required
	ExpectedReplacements int    // optional, defaults to 1 if omitted; must be >= 1
}

// ReadFileRequest contains parameters for ReadFile operation
type ReadFileRequest struct {
	Path   string `json:"path"`
	Offset *int64 `json:"offset,omitempty"`
	Limit  *int64 `json:"limit,omitempty"`
}

// Validate validates the ReadFileRequest
func (r ReadFileRequest) Validate(cfg *config.Config) error {
	if r.Path == "" {
		return fmt.Errorf("path is required")
	}
	if r.Offset != nil && *r.Offset < 0 {
		return fmt.Errorf("offset cannot be negative")
	}
	if r.Limit != nil && *r.Limit < 0 {
		return fmt.Errorf("limit cannot be negative")
	}
	return nil
}

// WriteFileRequest contains parameters for WriteFile operation
type WriteFileRequest struct {
	Path    string       `json:"path"`
	Content string       `json:"content"`
	Perm    *os.FileMode `json:"perm,omitempty"`
}

// Validate validates the WriteFileRequest
func (r WriteFileRequest) Validate(cfg *config.Config) error {
	if r.Path == "" {
		return fmt.Errorf("path is required")
	}
	if r.Content == "" {
		return fmt.Errorf("content is required")
	}
	if r.Perm != nil && *r.Perm&^os.FileMode(0777) != 0 {
		return fmt.Errorf("invalid permissions: only standard bits (0-0777) allowed")
	}
	return nil
}

// EditFileRequest contains parameters for EditFile operation
type EditFileRequest struct {
	Path       string      `json:"path"`
	Operations []Operation `json:"operations"`
}

// Validate validates the EditFileRequest
func (r EditFileRequest) Validate(cfg *config.Config) error {
	if r.Path == "" {
		return fmt.Errorf("path is required")
	}
	if len(r.Operations) == 0 {
		return fmt.Errorf("operations cannot be empty")
	}
	for i, op := range r.Operations {
		if op.Before == "" {
			return fmt.Errorf("operation %d: Before must be non-empty", i+1)
		}
		if op.ExpectedReplacements < 0 {
			return fmt.Errorf("operation %d: ExpectedReplacements cannot be negative", i+1)
		}
	}
	return nil
}

// ReadFileResponse contains the result of a ReadFile operation
type ReadFileResponse struct {
	AbsolutePath string
	RelativePath string
	Size         int64
	Content      string
}

// WriteFileResponse contains the result of a WriteFile operation
type WriteFileResponse struct {
	AbsolutePath string
	RelativePath string
	BytesWritten int
	FileMode     uint32
}

// EditFileResponse contains the result of an EditFile operation
type EditFileResponse struct {
	AbsolutePath      string
	RelativePath      string
	OperationsApplied int
	FileSize          int64
}
