package tools

import (
	"sync"
)

// checksumCache stores file checksums keyed by absolute path.
// It is thread-safe and can be used concurrently.
type checksumCache struct {
	mu    sync.RWMutex
	store map[string]string
}

// NewChecksumCache creates a new thread-safe checksum cache instance.
func NewChecksumCache() ChecksumStore {
	return &checksumCache{
		store: make(map[string]string),
	}
}

// Update stores or updates checksum for a file path
func (c *checksumCache) Update(path string, checksum string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.store[path] = checksum
}

// Get retrieves checksum for a file path
func (c *checksumCache) Get(path string) (checksum string, ok bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	checksum, ok = c.store[path]
	if !ok {
		return "", false
	}
	return checksum, true
}

// Clear removes all cached checksums (useful for testing)
func (c *checksumCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.store = make(map[string]string)
}

