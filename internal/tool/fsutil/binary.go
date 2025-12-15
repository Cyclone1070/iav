package fsutil

// SystemBinaryDetector implements binary content detection using null byte detection.
// It checks for null bytes in the first N bytes of content, with special handling for UTF BOMs.
type SystemBinaryDetector struct {
	SampleSize int // Number of bytes to sample for binary detection
}

// NewSystemBinaryDetector creates a new SystemBinaryDetector with the specified sample size.
func NewSystemBinaryDetector(sampleSize int) *SystemBinaryDetector {
	return &SystemBinaryDetector{
		SampleSize: sampleSize,
	}
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
