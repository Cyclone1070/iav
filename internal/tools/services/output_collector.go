package services

import (
	"bytes"
)

// Collector captures command output with size limits and binary content detection.
// It implements io.Writer and can be used to collect stdout/stderr from processes.
type Collector struct {
	Buffer    bytes.Buffer
	MaxBytes  int
	Truncated bool
	IsBinary  bool

	// Internal state for binary detection
	bytesChecked int
	sampleSize   int // Number of bytes to check for binary content
}

// NewCollector creates a new output collector with the specified maximum byte limit and binary detection sample size.
func NewCollector(maxBytes int, sampleSize int) *Collector {
	return &Collector{
		MaxBytes:   maxBytes,
		sampleSize: sampleSize,
	}
}

// Write implements io.Writer for collecting process output.
// It detects binary content and enforces size limits, truncating if necessary.
func (c *Collector) Write(p []byte) (n int, err error) {
	if c.IsBinary {
		return len(p), nil // Discard rest if binary
	}

	// Check for binary content in the first N bytes
	if c.bytesChecked < c.sampleSize {
		remainingCheck := c.sampleSize - c.bytesChecked
		toCheck := p
		if len(toCheck) > remainingCheck {
			toCheck = toCheck[:remainingCheck]
		}

		if bytes.IndexByte(toCheck, 0) != -1 {
			c.IsBinary = true
			c.Truncated = true // Treated as truncated since we stop collecting
			return len(p), nil
		}
		c.bytesChecked += len(toCheck)
	}

	// Check if we have space
	remainingSpace := c.MaxBytes - c.Buffer.Len()
	if remainingSpace <= 0 {
		c.Truncated = true
		return len(p), nil
	}

	toWrite := p
	if len(toWrite) > remainingSpace {
		toWrite = toWrite[:remainingSpace]
		c.Truncated = true
	}

	written, err := c.Buffer.Write(toWrite)
	if err != nil {
		return written, err
	}

	// We always return len(p) to satisfy io.Writer contract, even if we truncated
	return len(p), nil
}

// String returns the collected output as a string.
// Returns "[Binary Content]" if binary data was detected.
func (c *Collector) String() string {
	if c.IsBinary {
		return "[Binary Content]"
	}
	return c.Buffer.String()
}

// SystemBinaryDetector implements BinaryDetector using null byte detection.
// It checks for null bytes in the first 4KB of content, with special handling for UTF BOMs.
type SystemBinaryDetector struct {
	SampleSize int // Number of bytes to sample for binary detection
}

// IsBinaryContent checks if content bytes contain binary data by looking for null bytes.
// It handles UTF-16 and UTF-32 BOMs specially to avoid false positives.
func (r *SystemBinaryDetector) IsBinaryContent(content []byte) bool {
	// Check for common text file BOMs (UTF-16, UTF-32)
	if len(content) >= 2 {
		if (content[0] == 0xFF && content[1] == 0xFE) ||
			(content[0] == 0xFE && content[1] == 0xFF) {
			return false // UTF-16 BOM - treat as text, skip null check
		}
	}
	if len(content) >= 4 {
		if (content[0] == 0xFF && content[1] == 0xFE && content[2] == 0x00 && content[3] == 0x00) ||
			(content[0] == 0x00 && content[1] == 0x00 && content[2] == 0xFE && content[3] == 0xFF) {
			return false // UTF-32 BOM - treat as text, skip null check
		}
	}

	// Check for null bytes in configured sample size
	sampleSize := min(len(content), r.SampleSize)
	for i := range sampleSize {
		if content[i] == 0 {
			return true
		}
	}
	return false
}
