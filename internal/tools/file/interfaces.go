package file

import "os"

// fileSystem defines the minimal filesystem interface needed by file tools.
// This is a consumer-defined interface per architecture guidelines §2.
type fileSystem interface {
	Stat(path string) (os.FileInfo, error)
	Lstat(path string) (os.FileInfo, error)
	Readlink(path string) (string, error)
	UserHomeDir() (string, error)
	ReadFileRange(path string, offset, limit int64) ([]byte, error)
	EnsureDirs(path string) error
	WriteFileAtomic(path string, content []byte, perm os.FileMode) error
	Remove(name string) error
	Rename(oldpath, newpath string) error
	Chmod(name string, mode os.FileMode) error
}

// binaryDetector defines the interface for binary content detection.
type binaryDetector interface {
	IsBinaryContent(content []byte) bool
}

// checksumManager defines the interface for file checksum management.
type checksumManager interface {
	Compute(data []byte) string
	Get(path string) (checksum string, ok bool)
	Update(path string, checksum string)
}

// pathResolver defines the interface for path resolution.
type pathResolver interface {
	Resolve(workspaceRoot string, fs fileSystem, path string) (abs string, rel string, err error)
}
