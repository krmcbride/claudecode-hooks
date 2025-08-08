// Package detector provides pattern matching utilities for command detection
package detector

import "strings"

// hasAllowedException checks if text matches any allowed patterns
func hasAllowedException(text string, allowedPatterns []string) bool {
	if len(allowedPatterns) == 0 {
		return false
	}

	textLower := strings.ToLower(text)
	for _, pattern := range allowedPatterns {
		if matchesAllWords(textLower, pattern) {
			return true
		}
	}
	return false
}

// hasBlockedPattern checks if text matches any blocked patterns
func hasBlockedPattern(text string, patterns []string) bool {
	if len(patterns) == 0 {
		return false
	}

	textLower := strings.ToLower(text)
	for _, pattern := range patterns {
		// Simple substring matching
		if strings.Contains(textLower, strings.ToLower(pattern)) {
			return true
		}
	}
	return false
}

// matchesAllWords checks if all words in pattern exist in text
// This replaces the proximity-based matching with simple word existence
func matchesAllWords(text, pattern string) bool {
	words := strings.Fields(strings.ToLower(pattern))
	for _, word := range words {
		if !strings.Contains(text, word) {
			return false
		}
	}
	return true
}
