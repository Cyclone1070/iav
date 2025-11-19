package tools

import (
	"crypto/sha256"
	"encoding/hex"
	"sync"
)

// checksumManagerImpl is the concrete implementation of ChecksumManager.
// It is thread-safe and can be used concurrently.
type checksumManagerImpl struct {
	mu    sync.RWMutex
	store map[string]string
}

// NewChecksumManager creates a new thread-safe checksum manager instance.
func NewChecksumManager() ChecksumManager {
	return &checksumManagerImpl{
		store: make(map[string]string),
	}
}

// Compute computes SHA-256 checksum of data
func (m *checksumManagerImpl) Compute(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// Get retrieves checksum for a file path
func (m *checksumManagerImpl) Get(path string) (checksum string, ok bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	checksum, ok = m.store[path]
	return checksum, ok
}

// Update stores or updates checksum for a file path
func (m *checksumManagerImpl) Update(path string, checksum string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.store[path] = checksum
}

// Clear removes all cached checksums (useful for testing)
func (m *checksumManagerImpl) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.store = make(map[string]string)
}

