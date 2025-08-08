// Package detector provides pattern matching utilities for command detection
package detector

import "strings"

// hasBlockedPattern checks if text matches any blocked patterns
func hasBlockedPattern(text string, patterns []string) bool {
	if len(patterns) == 0 {
		return false
	}

	textLower := strings.ToLower(text)
	for _, pattern := range patterns {
		// Handle wildcard pattern - blocks everything
		if pattern == "*" {
			return true
		}

		// Handle glob patterns (e.g., "delete-*", "terminate-*")
		if strings.Contains(pattern, "*") {
			prefix := strings.TrimSuffix(strings.ToLower(pattern), "*")
			if strings.HasPrefix(textLower, prefix) {
				return true
			}
			// Also check if it appears as a word (for "aws delete-bucket")
			if strings.Contains(textLower, " "+prefix) {
				return true
			}
		} else {
			// Simple substring matching for exact patterns
			if strings.Contains(textLower, strings.ToLower(pattern)) {
				return true
			}
		}
	}
	return false
}
