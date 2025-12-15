package shell

import "bytes"

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
