package file

import "os"

// fileOps defines the filesystem operations needed for file read/write.
// This is a consumer-defined interface per architecture guidelines §2.
type fileOps interface {
	Stat(path string) (os.FileInfo, error)
	ReadFileRange(path string, offset, limit int64) ([]byte, error)
	WriteFileAtomic(path string, content []byte, perm os.FileMode) error
	EnsureDirs(path string) error
}

// pathResolver defines the filesystem operations needed for path resolution.
// This interface is passed to pathutil.Resolve which requires these methods.
type pathResolver interface {
	Lstat(path string) (os.FileInfo, error)
	Readlink(path string) (string, error)
	UserHomeDir() (string, error)
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
