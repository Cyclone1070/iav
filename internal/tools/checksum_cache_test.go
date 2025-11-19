package tools

import (
	"sync"
	"testing"
)

func TestChecksumCache(t *testing.T) {
	cache := NewChecksumCache()
	cache.Clear()

	path := "/test/path.txt"
	checksum := "abc123"

	// Test Get on empty cache
	_, ok := cache.Get(path)
	if ok {
		t.Error("cache should be empty")
	}

	// Test Update
	cache.Update(path, checksum)

	// Test Get after update
	retrievedChecksum, ok := cache.Get(path)
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
			cache.Update(path, checksum)
			cache.Get(path)
		}(i)
	}
	wg.Wait()

	// Verify final state
	finalChecksum, ok := cache.Get(path)
	if !ok {
		t.Error("cache entry should still exist after concurrent access")
	}
	if finalChecksum != checksum {
		t.Errorf("checksum should remain %s after concurrent access", checksum)
	}
}

func TestChecksumCacheClear(t *testing.T) {
	cache := NewChecksumCache()
	cache.Clear()

	// Add some entries
	cache.Update("/file1.txt", "hash1")
	cache.Update("/file2.txt", "hash2")

	// Clear
	cache.Clear()

	// Verify entries are gone
	_, ok := cache.Get("/file1.txt")
	if ok {
		t.Error("cache should be empty after Clear")
	}
}
