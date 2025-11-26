package gemini

import (
	"testing"
	"time"

	"google.golang.org/genai"
)

func TestParseRetryAfter(t *testing.T) {
	tests := []struct {
		name     string
		apiErr   *genai.APIError
		expected *time.Duration
	}{
		{
			name:     "nil error",
			apiErr:   nil,
			expected: nil,
		},
		{
			name: "empty details",
			apiErr: &genai.APIError{
				Code:    429,
				Details: []map[string]any{},
			},
			expected: nil,
		},
		{
			name: "retryDelay as int",
			apiErr: &genai.APIError{
				Code: 429,
				Details: []map[string]any{
					{"retryDelay": 120},
				},
			},
			expected: durationPtr(120 * time.Second),
		},
		{
			name: "retryDelay as int64",
			apiErr: &genai.APIError{
				Code: 429,
				Details: []map[string]any{
					{"retryDelay": int64(60)},
				},
			},
			expected: durationPtr(60 * time.Second),
		},
		{
			name: "retryDelay as float64 (JSON number)",
			apiErr: &genai.APIError{
				Code: 429,
				Details: []map[string]any{
					{"retryDelay": 30.0},
				},
			},
			expected: durationPtr(30 * time.Second),
		},
		{
			name: "retry_after field (snake_case)",
			apiErr: &genai.APIError{
				Code: 429,
				Details: []map[string]any{
					{"retry_after": 90},
				},
			},
			expected: durationPtr(90 * time.Second),
		},
		{
			name: "retryAfter field (camelCase)",
			apiErr: &genai.APIError{
				Code: 429,
				Details: []map[string]any{
					{"retryAfter": 45},
				},
			},
			expected: durationPtr(45 * time.Second),
		},
		{
			name: "Retry-After field (HTTP header style)",
			apiErr: &genai.APIError{
				Code: 429,
				Details: []map[string]any{
					{"Retry-After": 180},
				},
			},
			expected: durationPtr(180 * time.Second),
		},
		{
			name: "string number of seconds",
			apiErr: &genai.APIError{
				Code: 429,
				Details: []map[string]any{
					{"retryDelay": "75"},
				},
			},
			expected: durationPtr(75 * time.Second),
		},
		{
			name: "Google duration format in details",
			apiErr: &genai.APIError{
				Code: 429,
				Details: []map[string]any{
					{
						"retryDelay": map[string]any{
							"seconds": 150,
						},
					},
				},
			},
			expected: durationPtr(150 * time.Second),
		},
		{
			name: "nested in metadata",
			apiErr: &genai.APIError{
				Code: 429,
				Details: []map[string]any{
					{
						"metadata": map[string]any{
							"retryDelay": 100,
						},
					},
				},
			},
			expected: durationPtr(100 * time.Second),
		},
		{
			name: "multiple details, retry in second one",
			apiErr: &genai.APIError{
				Code: 429,
				Details: []map[string]any{
					{"someOtherField": "value"},
					{"retryDelay": 50},
				},
			},
			expected: durationPtr(50 * time.Second),
		},
		{
			name: "no retry field present",
			apiErr: &genai.APIError{
				Code: 429,
				Details: []map[string]any{
					{"someField": "value"},
					{"anotherField": 123},
				},
			},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseRetryAfter(tt.apiErr)

			if tt.expected == nil {
				if result != nil {
					t.Errorf("expected nil, got %v", *result)
				}
				return
			}

			if result == nil {
				t.Errorf("expected %v, got nil", *tt.expected)
				return
			}

			if *result != *tt.expected {
				t.Errorf("expected %v, got %v", *tt.expected, *result)
			}
		})
	}
}

func TestParseRetryValue(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		expected *time.Duration
	}{
		{
			name:     "int value",
			value:    60,
			expected: durationPtr(60 * time.Second),
		},
		{
			name:     "int64 value",
			value:    int64(120),
			expected: durationPtr(120 * time.Second),
		},
		{
			name:     "float64 value",
			value:    45.0,
			expected: durationPtr(45 * time.Second),
		},
		{
			name:     "string number",
			value:    "30",
			expected: durationPtr(30 * time.Second),
		},
		{
			name:     "string float",
			value:    "2.5",
			expected: durationPtr(2500 * time.Millisecond),
		},
		{
			name: "Google duration format - seconds only",
			value: map[string]any{
				"seconds": 120,
			},
			expected: durationPtr(120 * time.Second),
		},
		{
			name: "Google duration format - seconds and nanos",
			value: map[string]any{
				"seconds": 5,
				"nanos":   500000000, // 0.5 seconds
			},
			expected: durationPtr(5*time.Second + 500*time.Millisecond),
		},
		{
			name: "Google duration format - nanos only",
			value: map[string]any{
				"nanos": 250000000, // 0.25 seconds
			},
			expected: durationPtr(250 * time.Millisecond),
		},
		{
			name: "Google duration format - int64 seconds",
			value: map[string]any{
				"seconds": int64(90),
			},
			expected: durationPtr(90 * time.Second),
		},
		{
			name: "Google duration format - float64 seconds",
			value: map[string]any{
				"seconds": 75.0,
			},
			expected: durationPtr(75 * time.Second),
		},
		{
			name: "Google duration format - string seconds",
			value: map[string]any{
				"seconds": "60",
			},
			expected: durationPtr(60 * time.Second),
		},
		{
			name:     "empty string",
			value:    "",
			expected: nil,
		},
		{
			name:     "invalid string",
			value:    "not-a-number",
			expected: nil,
		},
		{
			name:     "unsupported type",
			value:    true,
			expected: nil,
		},
		{
			name:     "nil value",
			value:    nil,
			expected: nil,
		},
		{
			name: "empty duration map",
			value: map[string]any{
				"other": "field",
			},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseRetryValue(tt.value)

			if tt.expected == nil {
				if result != nil {
					t.Errorf("expected nil, got %v", *result)
				}
				return
			}

			if result == nil {
				t.Errorf("expected %v, got nil", *tt.expected)
				return
			}

			if *result != *tt.expected {
				t.Errorf("expected %v, got %v", *tt.expected, *result)
			}
		})
	}
}

// Helper function to create duration pointer
func durationPtr(d time.Duration) *time.Duration {
	return &d
}
