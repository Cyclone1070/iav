package services

import (
	"github.com/Cyclone1070/deployforme/internal/tools/models"
)

// SystemBinaryDetector implements BinaryDetector using local heuristics
type SystemBinaryDetector struct{}

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

	// Check for null bytes in first 4KB for files without BOM
	sampleSize := min(len(content), models.BinaryDetectionSampleSize)
	for i := range sampleSize {
		if content[i] == 0 {
			return true
		}
	}
	return false
}

