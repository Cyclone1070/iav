// Package fs provides a thin wrapper around os and io/fs operations,
// exposing an interface so callers can inject mocks in tests.
package fs

import (
	"io/fs"
	"os"
	"path/filepath"
)

// RealFS implements filesystem operations using the real operating system filesystem.
type RealFS struct{}

func (RealFS) Open(name string) (fs.File, error) {
	//nolint:gosec // G304: File path is parameterized by design
	return os.Open(name)
}

func (RealFS) EvalSymlinks(path string) (string, error) {
	return filepath.EvalSymlinks(path)
}

func (RealFS) Stat(name string) (fs.FileInfo, error) {
	return os.Stat(name)
}

func (RealFS) WalkDir(root string, fn fs.WalkDirFunc) error {
	return filepath.WalkDir(root, fn)
}

func (RealFS) Abs(path string) (string, error) {
	return filepath.Abs(path)
}
