// Package utils provides common utility functions for Claude Code hooks.
package utils

import "strings"

// ParseCommaSeparated splits a comma-separated string into a slice of trimmed, non-empty strings.
// Returns an empty slice for empty input.
//
// Examples:
//   - "a,b,c" -> ["a", "b", "c"]
//   - "a, b , c" -> ["a", "b", "c"]
//   - "a,,b" -> ["a", "b"] (empty values filtered out)
//   - "" -> []
func ParseCommaSeparated(input string) []string {
	if input == "" {
		return []string{}
	}

	parts := strings.Split(input, ",")
	result := make([]string, 0, len(parts))

	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return result
}
