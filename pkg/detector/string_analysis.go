// Package detector - string analysis utilities for command detection
package detector

import (
	"strings"

	"mvdan.cc/sh/v3/syntax"
)

// extractStringLiterals extracts all string literals from a syntax word
func extractStringLiterals(word *syntax.Word) []string {
	if word == nil {
		return nil
	}

	var strings []string

	// Walk through all parts of the word
	for _, part := range word.Parts {
		switch p := part.(type) {
		case *syntax.Lit:
			// Plain literal - could be a string
			if p.Value != "" {
				strings = append(strings, p.Value)
			}
		case *syntax.SglQuoted:
			// Single-quoted string
			if p.Value != "" {
				strings = append(strings, p.Value)
			}
		case *syntax.DblQuoted:
			// Double-quoted string - may contain expansions
			if quotedStr := extractFromDoubleQuoted(p); quotedStr != "" {
				strings = append(strings, quotedStr)
			}
		}
	}

	return strings
}

// extractFromDoubleQuoted extracts the string content from a double-quoted expression
func extractFromDoubleQuoted(dq *syntax.DblQuoted) string {
	if dq == nil {
		return ""
	}

	var result strings.Builder
	for _, part := range dq.Parts {
		switch p := part.(type) {
		case *syntax.Lit:
			result.WriteString(p.Value)
		case *syntax.SglQuoted:
			result.WriteString(p.Value)
			// Skip expansions - we'll analyze the literal parts
		}
	}

	return result.String()
}
