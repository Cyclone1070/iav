package fs

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/Cyclone1070/iav/internal/config"
	"github.com/Cyclone1070/iav/internal/tool/helper/content"
)

// OSFileSystem implements filesystem operations using the local OS filesystem primitives.
type OSFileSystem struct {
	config *config.Config
}

// NewOSFileSystem creates a new OSFileSystem.
func NewOSFileSystem(cfg *config.Config) *OSFileSystem {
	return &OSFileSystem{config: cfg}
}

// ReadFileLinesResult holds the result of a line-based file read.
type ReadFileLinesResult struct {
	Content    string // RAW content (no line numbers - formatting is caller's job)
	TotalLines int
	StartLine  int // actual start (1-indexed)
	EndLine    int // actual end (1-indexed)
}

// ReadFileLines reads lines from a file with safety limits.
// startLine: 1-indexed, 0 means 1
// endLine: 1-indexed, 0 means read to EOF
func (fs *OSFileSystem) ReadFileLines(path string, startLine, endLine int) (*ReadFileLinesResult, error) {
	if startLine <= 0 {
		startLine = 1
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return nil, err
	}

	if info.Size() > fs.config.Tools.MaxFileSize {
		return nil, fmt.Errorf("file %s exceeds max size (%d bytes)", path, fs.config.Tools.MaxFileSize)
	}

	// Read first 8KB for binary check
	sample := make([]byte, 8192)
	n, err := file.Read(sample)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("read sample: %w", err)
	}
	if content.IsBinaryContent(sample[:n]) {
		return nil, fmt.Errorf("binary file: %s", path)
	}

	// Seek back to start for Pass 1 (counting lines)
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("seek to start: %w", err)
	}

	// Pass 1: Count total lines
	totalLines := 0
	// Use a fixed buffer to avoid loading large lines if we don't need them yet
	// But we need to count them. bufio.Scanner can handle lines up to 64KB by default.
	// If a line is longer, it will return an error.
	// Actually, let's use a simpler line counter that doesn't care about line length.
	totalLines, err = countLines(file)
	if err != nil {
		return nil, fmt.Errorf("count lines: %w", err)
	}

	// Seek back to start for Pass 2 (reading content)
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("seek to start: %w", err)
	}

	if startLine > totalLines {
		return &ReadFileLinesResult{
			Content:    "",
			TotalLines: totalLines,
			StartLine:  startLine,
			EndLine:    0,
		}, nil
	}

	// Pass 2: Skip to startLine and read until endLine
	var buffer bytes.Buffer
	currentLine := 1
	actualEndLine := 0

	reader := bufio.NewReader(file)
	for {
		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return nil, fmt.Errorf("read line %d: %w", currentLine, err)
		}

		if currentLine >= startLine {
			if endLine == 0 || currentLine <= endLine {
				buffer.WriteString(line)
				actualEndLine = currentLine
			}
		}

		if endLine > 0 && currentLine >= endLine {
			break
		}

		if err == io.EOF {
			break
		}
		currentLine++
	}

	return &ReadFileLinesResult{
		Content:    buffer.String(),
		TotalLines: totalLines,
		StartLine:  startLine,
		EndLine:    actualEndLine,
	}, nil
}

// countLines counts newlines in a file efficiently.
func countLines(r io.Reader) (int, error) {
	buf := make([]byte, 32*1024)
	count := 0
	hasContent := false
	lastByte := byte(0)

	for {
		n, err := r.Read(buf)
		if n > 0 {
			hasContent = true
			for i := 0; i < n; i++ {
				if buf[i] == '\n' {
					count++
				}
				lastByte = buf[i]
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, err
		}
	}

	// If the file is not empty and doesn't end with a newline, it still has 1 line.
	// E.g., "hello" (no \n) -> count is 0, but it's 1 line.
	// "hello\n" -> count is 1, it's 1 line.
	// "hello\nworld" -> count is 1, it's 2 lines.
	if hasContent && lastByte != '\n' {
		count++
	}

	return count, nil
}

// WriteFileAtomic writes content to a file atomically using temp file + rename pattern.
// This ensures that if the process crashes mid-write, the original file remains intact.
// The temp file is created in the same directory as the target to ensure atomic rename.
func (fs *OSFileSystem) WriteFileAtomic(path string, content []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)

	tmpFile, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return fmt.Errorf("create temp in %s: %w", dir, err)
	}

	tmpPath := tmpFile.Name()
	needsCleanup := true

	defer func() {
		if tmpFile != nil {
			_ = tmpFile.Close()
		}
		if needsCleanup {
			_ = os.Remove(tmpPath)
		}
	}()

	if _, err := tmpFile.Write(content); err != nil {
		return fmt.Errorf("write temp %s: %w", tmpPath, err)
	}

	if err := tmpFile.Sync(); err != nil {
		return fmt.Errorf("sync temp %s: %w", tmpPath, err)
	}

	if err := tmpFile.Chmod(perm); err != nil {
		return fmt.Errorf("chmod temp %s: %w", tmpPath, err)
	}

	// Close file before rename (required on some systems)
	if err := tmpFile.Close(); err != nil {
		tmpFile = nil
		return fmt.Errorf("close temp %s: %w", tmpPath, err)
	}
	tmpFile = nil

	// Atomic rename is the critical operation that ensures consistency
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("rename %s to %s: %w", tmpPath, path, err)
	}
	needsCleanup = false

	return nil
}

// EnsureDirs creates parent directories recursively if they don't exist.
func (fs *OSFileSystem) EnsureDirs(path string) error {
	return os.MkdirAll(path, 0o755)
}

// Readlink reads the target of a symlink.
func (fs *OSFileSystem) Readlink(path string) (string, error) {
	return os.Readlink(path)
}

// UserHomeDir returns the current user's home directory.
func (fs *OSFileSystem) UserHomeDir() (string, error) {
	return os.UserHomeDir()
}

// ListDir lists the contents of a directory.
// Returns a slice of FileInfo for each entry in the directory.
func (fs *OSFileSystem) ListDir(path string) ([]os.FileInfo, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	infos := make([]os.FileInfo, 0, len(entries))
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			return nil, err
		}
		infos = append(infos, info)
	}

	return infos, nil
}
