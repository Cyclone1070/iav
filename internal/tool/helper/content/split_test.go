package content

import (
	"reflect"
	"testing"
)

func TestSplitLines(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "single line LF",
			input:    "line1",
			expected: []string{"line1"},
		},
		{
			name:     "multiple lines LF",
			input:    "line1\nline2\nline3",
			expected: []string{"line1", "line2", "line3"},
		},
		{
			name:     "trailing newline LF",
			input:    "line1\n",
			expected: []string{"line1"},
		},
		{
			name:     "empty string",
			input:    "",
			expected: nil,
		},
		{
			name:     "only newline LF",
			input:    "\n",
			expected: []string{""},
		},
		{
			name:     "multiple lines CRLF",
			input:    "line1\r\nline2\r\nline3",
			expected: []string{"line1", "line2", "line3"},
		},
		{
			name:     "trailing newline CRLF",
			input:    "line1\r\n",
			expected: []string{"line1"},
		},
		{
			name:     "mixed endings",
			input:    "line1\nline2\r\nline3",
			expected: []string{"line1", "line2", "line3"},
		},
		{
			name:     "dangling CR",
			input:    "line1\rline2", // treat \r as content if not followed by \n
			expected: []string{"line1\rline2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SplitLines(tt.input); !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("SplitLines(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}
