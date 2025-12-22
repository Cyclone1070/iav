package file

import (
	"os"

	"github.com/Cyclone1070/iav/internal/config"
)

// -- Read File --

type ReadFileRequest struct {
	Path   string `json:"path"`
	Offset *int64 `json:"offset,omitempty"`
	Limit  *int64 `json:"limit,omitempty"`
}

func (r *ReadFileRequest) Validate(cfg *config.Config) error {
	if r.Path == "" {
		return ErrPathRequired
	}
	if r.Offset != nil && *r.Offset < 0 {
		return ErrInvalidOffset
	}
	if r.Limit != nil && *r.Limit < 0 {
		return ErrInvalidLimit
	}
	return nil
}

type ReadFileResponse struct {
	Content      string
	AbsolutePath string
	RelativePath string
	Size         int64
}

// -- Write File --

type WriteFileRequest struct {
	Path    string       `json:"path"`
	Content string       `json:"content"`
	Perm    *os.FileMode `json:"perm,omitempty"`
}

func (r *WriteFileRequest) Validate(cfg *config.Config) error {
	if r.Path == "" {
		return ErrPathRequired
	}
	if r.Content == "" {
		return ErrContentRequiredForWrite
	}
	if r.Perm != nil && *r.Perm > 0777 {
		return ErrInvalidPermissions
	}
	if int64(len(r.Content)) > cfg.Tools.MaxFileSize {
		return ErrFileTooLarge
	}
	return nil
}

type WriteFileResponse struct {
	AbsolutePath string
	RelativePath string
	BytesWritten int
	FileMode     uint32
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

func (r *EditFileRequest) Validate(cfg *config.Config) error {
	if r.Path == "" {
		return ErrPathRequired
	}
	if len(r.Operations) == 0 {
		return ErrOperationsRequired
	}
	for _, op := range r.Operations {
		if op.Before == "" {
			return ErrEmptyBefore
		}
		if op.ExpectedReplacements < 0 {
			return ErrInvalidExpectedReplacements
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
