package mocks

// MockBinaryDetector implements models.BinaryDetector with configurable behaviour
type MockBinaryDetector struct {
	IsBinaryContentFunc func(content []byte) bool
	SampleSize          int // Kept for test compatibility, unused by logic
}

// NewMockBinaryDetector creates a new mock binary detector
func NewMockBinaryDetector() *MockBinaryDetector {
	return &MockBinaryDetector{
		SampleSize: 4096,
	}
}

func (f *MockBinaryDetector) IsBinaryContent(content []byte) bool {
	if f.IsBinaryContentFunc != nil {
		return f.IsBinaryContentFunc(content)
	}
	// Default to false (not binary) if no func provided
	return false
}
