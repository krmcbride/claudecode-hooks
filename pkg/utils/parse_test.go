package utils

import (
	"reflect"
	"testing"
)

func TestParseCommaSeparated(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "Empty string",
			input:    "",
			expected: []string{},
		},
		{
			name:     "Single value",
			input:    "value",
			expected: []string{"value"},
		},
		{
			name:     "Multiple values",
			input:    "a,b,c",
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "Values with spaces",
			input:    "a, b , c",
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "Empty values filtered",
			input:    "a,,b",
			expected: []string{"a", "b"},
		},
		{
			name:     "Only commas",
			input:    ",,,",
			expected: []string{},
		},
		{
			name:     "Leading and trailing commas",
			input:    ",a,b,",
			expected: []string{"a", "b"},
		},
		{
			name:     "File extensions",
			input:    ".go,.js,.py",
			expected: []string{".go", ".js", ".py"},
		},
		{
			name:     "Command patterns",
			input:    "push,force-push,delete",
			expected: []string{"push", "force-push", "delete"},
		},
		{
			name:     "AWS commands",
			input:    "delete-bucket,terminate-instances",
			expected: []string{"delete-bucket", "terminate-instances"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseCommaSeparated(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("ParseCommaSeparated(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}
