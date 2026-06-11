// Package workflow orchestrates the full IaV run — discovers test scripts,
// parses configs, sandbox-validates paths, and executes via Docker or Compose.
package workflow

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
)

type FileSystem interface {
	Abs(path string) (string, error)
	Stat(path string) (fs.FileInfo, error)
	WalkDir(root string, fn fs.WalkDirFunc) error
}

type Discovery struct {
	fs     FileSystem
}

func NewDiscovery(fs FileSystem) *Discovery {
	return &Discovery{fs: fs}
}

func (d *Discovery) Run(targetPath string) ([]string, error) {
	targetAbs, err := d.fs.Abs(targetPath)
	if err != nil {
		return nil, fmt.Errorf("invalid path %q: %w", targetPath, err)
	}

	fileInfo, err := d.fs.Stat(targetAbs)
	if err != nil {
		return nil, fmt.Errorf("failed to access %q: %w", targetPath, err)
	}

	if !fileInfo.IsDir() {
		return []string{targetAbs}, nil
	}

	var testScripts []string
	err = d.fs.WalkDir(targetAbs, d.walkDirCallback(targetAbs, &testScripts))
	if err != nil {
		return nil, fmt.Errorf("failed to scan directory: %w", err)
	}
	if len(testScripts) == 0 {
		return nil, fmt.Errorf("no valid test compose files found in %q", targetPath)
	}

	return testScripts, nil
}

func (d *Discovery) walkDirCallback(targetAbs string, testScripts *[]string) fs.WalkDirFunc {
	return func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if path != targetAbs {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(path, ".test.yml") || strings.HasSuffix(path, ".test.yaml") {
			*testScripts = append(*testScripts, path)
		}
		return nil
	}
}
