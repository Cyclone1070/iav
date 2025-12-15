package fsutil

import (
	"bytes"
	"errors"
	"io"
	"os"
	"testing"
)

// mockWriteSyncCloser implements writeSyncCloser for testing
type mockWriteSyncCloser struct {
	buffer      *bytes.Buffer
	name        string
	writeErr    error
	syncErr     error
	closeErr    error
	writeCalled bool
	syncCalled  bool
	closeCalled bool
}

func newMockWriteSyncCloser(name string) *mockWriteSyncCloser {
	return &mockWriteSyncCloser{
		buffer: new(bytes.Buffer),
		name:   name,
	}
}

func (m *mockWriteSyncCloser) Write(p []byte) (n int, err error) {
	m.writeCalled = true
	if m.writeErr != nil {
		return 0, m.writeErr
	}
	return m.buffer.Write(p)
}

func (m *mockWriteSyncCloser) Sync() error {
	m.syncCalled = true
	if m.syncErr != nil {
		return m.syncErr
	}
	return nil
}

func (m *mockWriteSyncCloser) Close() error {
	m.closeCalled = true
	if m.closeErr != nil {
		return m.closeErr
	}
	return nil
}

func (m *mockWriteSyncCloser) Name() string {
	return m.name
}

func TestWriteFileAtomic(t *testing.T) {
	t.Run("crash during createTemp - no side effects", func(t *testing.T) {
		fs := NewOSFileSystem()

		// Override createTemp to fail
		fs.createTemp = func(dir, pattern string) (writeSyncCloser, error) {
			return nil, errors.New("disk full")
		}

		err := fs.WriteFileAtomic("/test/file.txt", []byte("content"), 0644)
		if err == nil {
			t.Fatal("expected error from createTemp failure")
		}
		if !errors.Is(err, io.EOF) && err.Error() != "failed to create temp file: disk full" {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("crash during Write - temp cleaned up", func(t *testing.T) {
		fs := NewOSFileSystem()

		mockFile := newMockWriteSyncCloser("/tmp/test-123")
		mockFile.writeErr = errors.New("write failed")

		removeCalled := false

		// Override createTemp to return mock
		fs.createTemp = func(dir, pattern string) (writeSyncCloser, error) {
			return mockFile, nil
		}

		// Override remove to verify cleanup
		fs.remove = func(name string) error {
			if name == mockFile.name {
				removeCalled = true
			}
			return nil
		}

		err := fs.WriteFileAtomic("/test/file.txt", []byte("content"), 0644)
		if err == nil {
			t.Fatal("expected error from Write failure")
		}

		if !mockFile.writeCalled {
			t.Error("Write should have been called")
		}

		if !removeCalled {
			t.Error("temp file should have been cleaned up")
		}
	})

	t.Run("crash during Sync - temp cleaned up", func(t *testing.T) {
		fs := NewOSFileSystem()

		mockFile := newMockWriteSyncCloser("/tmp/test-456")
		mockFile.syncErr = errors.New("sync failed")

		removeCalled := false

		fs.createTemp = func(dir, pattern string) (writeSyncCloser, error) {
			return mockFile, nil
		}

		fs.remove = func(name string) error {
			if name == mockFile.name {
				removeCalled = true
			}
			return nil
		}

		err := fs.WriteFileAtomic("/test/file.txt", []byte("content"), 0644)
		if err == nil {
			t.Fatal("expected error from Sync failure")
		}

		if !mockFile.writeCalled {
			t.Error("Write should have been called")
		}

		if !mockFile.syncCalled {
			t.Error("Sync should have been called")
		}

		if !removeCalled {
			t.Error("temp file should have been cleaned up")
		}
	})

	t.Run("crash during Close - temp cleaned up", func(t *testing.T) {
		fs := NewOSFileSystem()

		mockFile := newMockWriteSyncCloser("/tmp/test-789")
		mockFile.closeErr = errors.New("close failed")

		removeCalled := false

		fs.createTemp = func(dir, pattern string) (writeSyncCloser, error) {
			return mockFile, nil
		}

		fs.remove = func(name string) error {
			if name == mockFile.name {
				removeCalled = true
			}
			return nil
		}

		err := fs.WriteFileAtomic("/test/file.txt", []byte("content"), 0644)
		if err == nil {
			t.Fatal("expected error from Close failure")
		}

		if !mockFile.closeCalled {
			t.Error("Close should have been called")
		}

		if !removeCalled {
			t.Error("temp file should have been cleaned up")
		}
	})

	t.Run("crash during Rename - temp cleaned up", func(t *testing.T) {
		fs := NewOSFileSystem()

		mockFile := newMockWriteSyncCloser("/tmp/test-abc")
		removeCalled := false

		fs.createTemp = func(dir, pattern string) (writeSyncCloser, error) {
			return mockFile, nil
		}

		// Override rename to fail
		fs.rename = func(oldpath, newpath string) error {
			return errors.New("rename failed")
		}

		fs.remove = func(name string) error {
			if name == mockFile.name {
				removeCalled = true
			}
			return nil
		}

		err := fs.WriteFileAtomic("/test/file.txt", []byte("content"), 0644)
		if err == nil {
			t.Fatal("expected error from Rename failure")
		}

		if !mockFile.closeCalled {
			t.Error("Close should have been called before rename")
		}

		if !removeCalled {
			t.Error("temp file should have been cleaned up after rename failure")
		}
	})

	t.Run("crash during Chmod - file exists but wrong permissions", func(t *testing.T) {
		fs := NewOSFileSystem()

		mockFile := newMockWriteSyncCloser("/tmp/test-def")
		renameCalled := false

		fs.createTemp = func(dir, pattern string) (writeSyncCloser, error) {
			return mockFile, nil
		}

		fs.rename = func(oldpath, newpath string) error {
			renameCalled = true
			return nil
		}

		// Override chmod to fail
		fs.chmod = func(name string, mode os.FileMode) error {
			return errors.New("chmod failed")
		}

		err := fs.WriteFileAtomic("/test/file.txt", []byte("content"), 0644)
		if err == nil {
			t.Fatal("expected error from Chmod failure")
		}

		if !renameCalled {
			t.Error("Rename should have been called before chmod")
		}
	})

	t.Run("successful atomic write", func(t *testing.T) {
		fs := NewOSFileSystem()

		mockFile := newMockWriteSyncCloser("/tmp/test-success")
		renameCalled := false
		chmodCalled := false
		removeCalled := false

		fs.createTemp = func(dir, pattern string) (writeSyncCloser, error) {
			return mockFile, nil
		}

		fs.rename = func(oldpath, newpath string) error {
			if oldpath != mockFile.name {
				t.Errorf("expected rename from %s, got %s", mockFile.name, oldpath)
			}
			if newpath != "/test/file.txt" {
				t.Errorf("expected rename to /test/file.txt, got %s", newpath)
			}
			renameCalled = true
			return nil
		}

		fs.chmod = func(name string, mode os.FileMode) error {
			if name != "/test/file.txt" {
				t.Errorf("expected chmod on /test/file.txt, got %s", name)
			}
			if mode != 0644 {
				t.Errorf("expected mode 0644, got %o", mode)
			}
			chmodCalled = true
			return nil
		}

		fs.remove = func(name string) error {
			removeCalled = true
			return nil
		}

		err := fs.WriteFileAtomic("/test/file.txt", []byte("content"), 0644)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !mockFile.writeCalled {
			t.Error("Write should have been called")
		}

		if !mockFile.syncCalled {
			t.Error("Sync should have been called")
		}

		if !mockFile.closeCalled {
			t.Error("Close should have been called")
		}

		if !renameCalled {
			t.Error("Rename should have been called")
		}

		if !chmodCalled {
			t.Error("Chmod should have been called")
		}

		if removeCalled {
			t.Error("Remove should NOT have been called on success")
		}

		if mockFile.buffer.String() != "content" {
			t.Errorf("expected content to be written, got %q", mockFile.buffer.String())
		}
	})
}
