package file

import (
	"errors"
)

// -- Sentinels --

var (
	ErrFileMissing              = errors.New("file or path does not exist")
	ErrFileExists               = errors.New("file already exists")
	ErrBinaryFile               = errors.New("file is binary")
	ErrFileTooLarge             = errors.New("file too large")
	ErrOperationsRequired       = errors.New("operations cannot be empty")
	ErrSnippetNotFound          = errors.New("snippet not found")
	ErrReplacementCountMismatch = errors.New("replacement count mismatch")
	ErrEditConflict             = errors.New("edit conflict")
	ErrIsDirectory              = errors.New("path is a directory")
	ErrPathRequired             = errors.New("path is required")
	ErrContentRequiredForWrite  = errors.New("content is required for write operation")
)
