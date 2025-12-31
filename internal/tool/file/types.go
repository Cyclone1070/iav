package file

import (
	"fmt"

	"github.com/Cyclone1070/iav/internal/config"
)

// -- Read File --

type ReadFileRequest struct {
	Path      string `json:"path"`
	StartLine int    `json:"start_line,omitempty"`
	EndLine   int    `json:"end_line,omitempty"`
}

func (r *ReadFileRequest) Validate() error {
	if r.Path == "" {
		return fmt.Errorf("path is required")
	}
	if r.StartLine <= 0 {
		r.StartLine = 1
	}
	if r.EndLine < 0 {
		r.EndLine = 0
	}
	if r.EndLine > 0 && r.EndLine < r.StartLine {
		r.EndLine = r.StartLine
	}
	return nil
}

type ReadFileResponse struct {
	Content      string
	AbsolutePath string
	RelativePath string
	Size         int64
	StartLine    int
	EndLine      int
	TotalLines   int
}

// -- Write File --

type WriteFileRequest struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

func (r *WriteFileRequest) Validate(cfg *config.Config) error {
	if r.Path == "" {
		return fmt.Errorf("path is required")
	}
	if r.Content == "" {
		return fmt.Errorf("content is required")
	}
	if int64(len(r.Content)) > cfg.Tools.MaxFileSize {
		return fmt.Errorf("content too large: %d bytes exceeds limit %d", len(r.Content), cfg.Tools.MaxFileSize)
	}
	return nil
}

type WriteFileResponse struct {
	AbsolutePath string
	RelativePath string
	BytesWritten int
}

// -- Edit File --

type EditOperation struct {
	Before               string `json:"before"`
	After                string `json:"after"`
	ExpectedReplacements int    `json:"expected_replacements,omitempty"`
}

type EditFileRequest struct {
	Path       string          `json:"path"`
	Operations []EditOperation `json:"operations"`
}

func (r *EditFileRequest) Validate() error {
	if r.Path == "" {
		return fmt.Errorf("path is required")
	}
	if len(r.Operations) == 0 {
		return fmt.Errorf("operations are required")
	}
	for i := range r.Operations {
		if r.Operations[i].ExpectedReplacements <= 0 {
			r.Operations[i].ExpectedReplacements = 1
		}
	}
	return nil
}

type EditFileResponse struct {
	AbsolutePath      string
	RelativePath      string
	OperationsApplied int
	FileSize          int64
}
