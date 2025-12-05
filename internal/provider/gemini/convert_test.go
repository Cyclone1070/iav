package gemini

import (
	"testing"
	"time"

	provider "github.com/Cyclone1070/iav/internal/provider/models"
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

// =============================================================================
// SCHEMA CONVERSION TESTS
// =============================================================================
// These tests verify toGeminiSchema handles complex nested structures correctly.

func TestToGeminiSchema_DeeplyNested(t *testing.T) {
	// Schema representing: { outer: { middle: { inner: string } } }
	// 3 levels of object nesting
	schema := &provider.Schema{
		Type: "object",
		Properties: map[string]provider.Schema{
			"outer": {
				Type:        "object",
				Description: "Outer object",
				Properties: map[string]provider.Schema{
					"middle": {
						Type:        "object",
						Description: "Middle object",
						Properties: map[string]provider.Schema{
							"inner": {
								Type:        "string",
								Description: "Inner string value",
							},
						},
						Required: []string{"inner"},
					},
				},
				Required: []string{"middle"},
			},
		},
		Required: []string{"outer"},
	}

	result := toGeminiSchema(schema)

	// Verify root level
	if result == nil {
		t.Fatal("Result is nil")
	}
	if result.Type != genai.TypeObject {
		t.Errorf("Root type: expected object, got %v", result.Type)
	}
	if len(result.Required) != 1 || result.Required[0] != "outer" {
		t.Errorf("Root required: expected [outer], got %v", result.Required)
	}

	// Verify outer level
	outer := result.Properties["outer"]
	if outer == nil {
		t.Fatal("Missing 'outer' property")
	}
	if outer.Type != genai.TypeObject {
		t.Errorf("Outer type: expected object, got %v", outer.Type)
	}
	if outer.Description != "Outer object" {
		t.Errorf("Outer description: expected 'Outer object', got %q", outer.Description)
	}

	// Verify middle level
	middle := outer.Properties["middle"]
	if middle == nil {
		t.Fatal("Missing 'middle' property")
	}
	if middle.Type != genai.TypeObject {
		t.Errorf("Middle type: expected object, got %v", middle.Type)
	}
	if len(middle.Required) != 1 || middle.Required[0] != "inner" {
		t.Errorf("Middle required: expected [inner], got %v", middle.Required)
	}

	// Verify inner level
	inner := middle.Properties["inner"]
	if inner == nil {
		t.Fatal("Missing 'inner' property")
	}
	if inner.Type != genai.TypeString {
		t.Errorf("Inner type: expected string, got %v", inner.Type)
	}
	if inner.Description != "Inner string value" {
		t.Errorf("Inner description: expected 'Inner string value', got %q", inner.Description)
	}
}

func TestToGeminiSchema_ArrayOfObjectsWithProperties(t *testing.T) {
	// Schema representing edit_file operations:
	// { operations: [{ before: string, after: string, count: integer }] }
	schema := &provider.Schema{
		Type: "object",
		Properties: map[string]provider.Schema{
			"operations": {
				Type:        "array",
				Description: "List of edit operations to apply",
				Items: &provider.Schema{
					Type:        "object",
					Description: "Single edit operation",
					Properties: map[string]provider.Schema{
						"before": {
							Type:        "string",
							Description: "Text to find and replace",
						},
						"after": {
							Type:        "string",
							Description: "Replacement text",
						},
						"expected_count": {
							Type:        "integer",
							Description: "Expected number of replacements",
						},
					},
					Required: []string{"before", "after"},
				},
			},
		},
		Required: []string{"operations"},
	}

	result := toGeminiSchema(schema)

	// Verify operations array
	ops := result.Properties["operations"]
	if ops == nil {
		t.Fatal("Missing 'operations' property")
	}
	if ops.Type != genai.TypeArray {
		t.Errorf("Operations type: expected array, got %v", ops.Type)
	}
	if ops.Description != "List of edit operations to apply" {
		t.Errorf("Operations description mismatch: %q", ops.Description)
	}

	// Verify items schema
	items := ops.Items
	if items == nil {
		t.Fatal("Missing items schema for operations array")
	}
	if items.Type != genai.TypeObject {
		t.Errorf("Items type: expected object, got %v", items.Type)
	}

	// Verify item properties
	if len(items.Properties) != 3 {
		t.Errorf("Expected 3 item properties, got %d", len(items.Properties))
	}

	before := items.Properties["before"]
	if before == nil || before.Type != genai.TypeString {
		t.Error("'before' property missing or wrong type")
	}

	after := items.Properties["after"]
	if after == nil || after.Type != genai.TypeString {
		t.Error("'after' property missing or wrong type")
	}

	count := items.Properties["expected_count"]
	if count == nil || count.Type != genai.TypeInteger {
		t.Error("'expected_count' property missing or wrong type")
	}

	// Verify required fields on items
	if len(items.Required) != 2 {
		t.Errorf("Expected 2 required item fields, got %d", len(items.Required))
	}
}

func TestToGeminiSchema_MixedNesting(t *testing.T) {
	// Complex schema: { config: { items: [{ nested: { value: bool } }] } }
	// Tests: object -> object -> array -> object -> object -> primitive
	schema := &provider.Schema{
		Type: "object",
		Properties: map[string]provider.Schema{
			"config": {
				Type: "object",
				Properties: map[string]provider.Schema{
					"items": {
						Type: "array",
						Items: &provider.Schema{
							Type: "object",
							Properties: map[string]provider.Schema{
								"nested": {
									Type: "object",
									Properties: map[string]provider.Schema{
										"value": {
											Type:        "boolean",
											Description: "Deep boolean value",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	result := toGeminiSchema(schema)

	// Navigate the entire path: config -> items -> [item] -> nested -> value
	config := result.Properties["config"]
	if config == nil {
		t.Fatal("Missing 'config'")
	}

	items := config.Properties["items"]
	if items == nil {
		t.Fatal("Missing 'items'")
	}
	if items.Type != genai.TypeArray {
		t.Errorf("Items should be array, got %v", items.Type)
	}

	itemSchema := items.Items
	if itemSchema == nil {
		t.Fatal("Missing items schema")
	}

	nested := itemSchema.Properties["nested"]
	if nested == nil {
		t.Fatal("Missing 'nested'")
	}
	if nested.Type != genai.TypeObject {
		t.Errorf("Nested should be object, got %v", nested.Type)
	}

	value := nested.Properties["value"]
	if value == nil {
		t.Fatal("Missing 'value'")
	}
	if value.Type != genai.TypeBoolean {
		t.Errorf("Value should be boolean, got %v", value.Type)
	}
	if value.Description != "Deep boolean value" {
		t.Errorf("Value description mismatch: %q", value.Description)
	}
}

func TestToGeminiSchema_AllTypes(t *testing.T) {
	// Test all supported JSON Schema types
	schema := &provider.Schema{
		Type: "object",
		Properties: map[string]provider.Schema{
			"str_field":     {Type: "string", Description: "A string"},
			"num_field":     {Type: "number", Description: "A float"},
			"int_field":     {Type: "integer", Description: "An integer"},
			"bool_field":    {Type: "boolean", Description: "A boolean"},
			"arr_field":     {Type: "array", Items: &provider.Schema{Type: "string"}},
			"obj_field":     {Type: "object", Description: "An object"},
			"unknown_field": {Type: "unknown_type"}, // Should default to string
		},
	}

	result := toGeminiSchema(schema)

	expectations := map[string]genai.Type{
		"str_field":     genai.TypeString,
		"num_field":     genai.TypeNumber,
		"int_field":     genai.TypeInteger,
		"bool_field":    genai.TypeBoolean,
		"arr_field":     genai.TypeArray,
		"obj_field":     genai.TypeObject,
		"unknown_field": genai.TypeString, // Default fallback
	}

	for name, expectedType := range expectations {
		prop := result.Properties[name]
		if prop == nil {
			t.Errorf("Missing property: %s", name)
			continue
		}
		if prop.Type != expectedType {
			t.Errorf("%s: expected %v, got %v", name, expectedType, prop.Type)
		}
	}

	// Verify array items are also converted
	arrField := result.Properties["arr_field"]
	if arrField.Items == nil {
		t.Error("Array field missing items schema")
	} else if arrField.Items.Type != genai.TypeString {
		t.Errorf("Array items: expected string, got %v", arrField.Items.Type)
	}
}

func TestToGeminiSchema_WithEnums(t *testing.T) {
	schema := &provider.Schema{
		Type: "object",
		Properties: map[string]provider.Schema{
			"status": {
				Type:        "string",
				Description: "Todo status",
				Enum:        []string{"pending", "in_progress", "completed", "cancelled"},
			},
		},
	}

	result := toGeminiSchema(schema)

	status := result.Properties["status"]
	if status == nil {
		t.Fatal("Missing 'status' property")
	}
	if len(status.Enum) != 4 {
		t.Errorf("Expected 4 enum values, got %d", len(status.Enum))
	}
	expectedEnums := []string{"pending", "in_progress", "completed", "cancelled"}
	for i, expected := range expectedEnums {
		if status.Enum[i] != expected {
			t.Errorf("Enum[%d]: expected %q, got %q", i, expected, status.Enum[i])
		}
	}
}

func TestToGeminiSchema_NilInput(t *testing.T) {
	result := toGeminiSchema(nil)
	if result != nil {
		t.Error("Expected nil result for nil input")
	}
}
