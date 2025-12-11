package mocks

// MockBinaryDetector implements models.BinaryDetector with configurable behaviour
type MockBinaryDetector struct {
	IsBinaryContentFunc func(content []byte) bool
}

// NewMockBinaryDetector creates a new mock binary detector
func NewMockBinaryDetector() *MockBinaryDetector {
	return &MockBinaryDetector{}
}

func (f *MockBinaryDetector) IsBinaryContent(content []byte) bool {
	if f.IsBinaryContentFunc != nil {
		return f.IsBinaryContentFunc(content)
	}
	// Default to false (not binary) if no func provided
	return false
}
