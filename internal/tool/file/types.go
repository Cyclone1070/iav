package file

import (
	"os"

	"github.com/Cyclone1070/iav/internal/config"
	"github.com/Cyclone1070/iav/internal/tool/pathutil"
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

// ReadFileDTO is the wire format for ReadFile operation
type ReadFileDTO struct {
	Path   string `json:"path"`
	Offset *int64 `json:"offset,omitempty"`
	Limit  *int64 `json:"limit,omitempty"`
}

// ReadFileRequest is the validated domain entity for ReadFile operation
type ReadFileRequest struct {
	absPath string
	relPath string
	offset  int64
	limit   int64
}

// NewReadFileRequest creates a validated ReadFileRequest from a DTO
func NewReadFileRequest(
	dto ReadFileDTO,
	cfg *config.Config,
	workspaceRoot string,
	fs interface {
		Lstat(path string) (os.FileInfo, error)
		Readlink(path string) (string, error)
		UserHomeDir() (string, error)
	},
) (*ReadFileRequest, error) {
	// Constructor validation - everything we can know from inputs
	if dto.Path == "" {
		return nil, &PathRequiredError{}
	}

	var offset int64
	if dto.Offset != nil {
		if *dto.Offset < 0 {
			return nil, &NegativeOffsetError{Value: *dto.Offset}
		}
		offset = *dto.Offset
	}

	var limit int64
	if dto.Limit != nil {
		if *dto.Limit < 0 {
			return nil, &NegativeLimitError{Value: *dto.Limit}
		}
		limit = *dto.Limit
	}

	// Path resolution (I/O-based validation)
	abs, rel, err := resolvePathWithFS(workspaceRoot, fs, dto.Path)
	if err != nil {
		return nil, err
	}

	return &ReadFileRequest{
		absPath: abs,
		relPath: rel,
		offset:  offset,
		limit:   limit,
	}, nil
}

// AbsPath returns the absolute path
func (r *ReadFileRequest) AbsPath() string {
	return r.absPath
}

// RelPath returns the relative path
func (r *ReadFileRequest) RelPath() string {
	return r.relPath
}

// Offset returns the offset
func (r *ReadFileRequest) Offset() int64 {
	return r.offset
}

// Limit returns the limit
func (r *ReadFileRequest) Limit() int64 {
	return r.limit
}

// resolvePathWithFS is a helper that calls pathutil.Resolve with the given filesystem
func resolvePathWithFS(workspaceRoot string, fs interface {
	Lstat(path string) (os.FileInfo, error)
	Readlink(path string) (string, error)
	UserHomeDir() (string, error)
}, path string) (abs string, rel string, err error) {
	// Cast to pathutil.FileSystem (the interface is identical)
	fsImpl := fs.(pathutil.FileSystem)
	return pathutil.Resolve(workspaceRoot, fsImpl, path)
}

// WriteFileDTO is the wire format for WriteFile operation
type WriteFileDTO struct {
	Path    string       `json:"path"`
	Content string       `json:"content"`
	Perm    *os.FileMode `json:"perm,omitempty"`
}

// WriteFileRequest is the validated domain entity for WriteFile operation
type WriteFileRequest struct {
	absPath string
	relPath string
	content string
	perm    os.FileMode
}

// NewWriteFileRequest creates a validated WriteFileRequest from a DTO
func NewWriteFileRequest(
	dto WriteFileDTO,
	cfg *config.Config,
	workspaceRoot string,
	fs interface {
		Lstat(path string) (os.FileInfo, error)
		Readlink(path string) (string, error)
		UserHomeDir() (string, error)
	},
) (*WriteFileRequest, error) {
	// Constructor validation
	if dto.Path == "" {
		return nil, &PathRequiredError{}
	}

	if dto.Content == "" {
		return nil, &ContentRequiredError{}
	}

	perm := os.FileMode(0644) // default
	if dto.Perm != nil {
		if *dto.Perm&^os.FileMode(0777) != 0 {
			return nil, &InvalidPermissionError{Perm: uint32(*dto.Perm)}
		}
		perm = *dto.Perm & 0777
	}

	// Path resolution
	abs, rel, err := resolvePathWithFS(workspaceRoot, fs, dto.Path)
	if err != nil {
		return nil, err
	}

	return &WriteFileRequest{
		absPath: abs,
		relPath: rel,
		content: dto.Content,
		perm:    perm,
	}, nil
}

// AbsPath returns the absolute path
func (r *WriteFileRequest) AbsPath() string {
	return r.absPath
}

// RelPath returns the relative path
func (r *WriteFileRequest) RelPath() string {
	return r.relPath
}

// Content returns the content
func (r *WriteFileRequest) Content() string {
	return r.content
}

// Perm returns the file permissions
func (r *WriteFileRequest) Perm() os.FileMode {
	return r.perm
}

// EditFileDTO is the wire format for EditFile operation
type EditFileDTO struct {
	Path       string      `json:"path"`
	Operations []Operation `json:"operations"`
}

// EditFileRequest is the validated domain entity for EditFile operation
type EditFileRequest struct {
	absPath    string
	relPath    string
	operations []Operation
}

// NewEditFileRequest creates a validated EditFileRequest from a DTO
func NewEditFileRequest(
	dto EditFileDTO,
	cfg *config.Config,
	workspaceRoot string,
	fs interface {
		Lstat(path string) (os.FileInfo, error)
		Readlink(path string) (string, error)
		UserHomeDir() (string, error)
	},
) (*EditFileRequest, error) {
	// Constructor validation
	if dto.Path == "" {
		return nil, &PathRequiredError{}
	}

	if len(dto.Operations) == 0 {
		return nil, &OperationsRequiredError{}
	}

	for i, op := range dto.Operations {
		if op.Before == "" {
			return nil, &BeforeRequiredError{Index: i + 1}
		}
		if op.ExpectedReplacements < 0 {
			return nil, &NegativeExpectedReplacementsError{Index: i + 1, Value: op.ExpectedReplacements}
		}
	}

	// Path resolution
	abs, rel, err := resolvePathWithFS(workspaceRoot, fs, dto.Path)
	if err != nil {
		return nil, err
	}

	return &EditFileRequest{
		absPath:    abs,
		relPath:    rel,
		operations: dto.Operations,
	}, nil
}

// AbsPath returns the absolute path
func (r *EditFileRequest) AbsPath() string {
	return r.absPath
}

// RelPath returns the relative path
func (r *EditFileRequest) RelPath() string {
	return r.relPath
}

// Operations returns the operations
func (r *EditFileRequest) Operations() []Operation {
	return r.operations
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
