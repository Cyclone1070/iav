package fsutil

import (
	"sync"
	"testing"
)

func TestChecksumManager(t *testing.T) {
	manager := NewChecksumManager()
	manager.Clear()

	path := "/test/path.txt"
	checksum := "abc123"

	// Test Get on empty cache
	_, ok := manager.Get(path)
	if ok {
		t.Error("cache should be empty")
	}

	// Test Update
	manager.Update(path, checksum)

	// Test Get after update
	retrievedChecksum, ok := manager.Get(path)
	if !ok {
		t.Error("cache should contain the entry")
	}
	if retrievedChecksum != checksum {
		t.Errorf("expected checksum %s, got %s", checksum, retrievedChecksum)
	}

	// Test concurrent access
	var wg sync.WaitGroup
	for i := range 10 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			manager.Update(path, checksum)
			manager.Get(path)
		}(i)
	}
	wg.Wait()

	// Verify final state
	finalChecksum, ok := manager.Get(path)
	if !ok {
		t.Error("cache entry should still exist after concurrent access")
	}
	if finalChecksum != checksum {
		t.Errorf("checksum should remain %s after concurrent access", checksum)
	}
}

func TestChecksumManagerClear(t *testing.T) {
	manager := NewChecksumManager()
	manager.Clear()

	// Add some entries
	manager.Update("/file1.txt", "hash1")
	manager.Update("/file2.txt", "hash2")

	// Clear
	manager.Clear()

	// Verify entries are gone
	_, ok := manager.Get("/file1.txt")
	if ok {
		t.Error("cache should be empty after Clear")
	}
}
