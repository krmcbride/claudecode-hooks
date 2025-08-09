// Package detector - internal shell parsing utilities
package detector

import (
	"fmt"
	"strings"

	"mvdan.cc/sh/v3/syntax"
)

// parseShellExpression parses a shell expression into an Abstract Syntax Tree.
// The input shellExpr can be a simple command ("ls -la") or a complex expression
// with pipes, conditionals, loops, and subshells ("cd /tmp && git pull || echo failed").
// Returns the AST root node which can be traversed to extract various elements
// like command calls, redirections, variables, etc.
func parseShellExpression(shellExpr string) (syntax.Node, error) {
	parser := syntax.NewParser()
	node, err := parser.Parse(strings.NewReader(shellExpr), "")
	if err != nil {
		return nil, fmt.Errorf("failed to parse shell expression: %w", err)
	}
	return node, nil
}

// extractCallExprs walks the AST and collects all command call expressions.
// These represent actual command invocations (e.g., "git push", "echo hello").
// The traversal is depth-first, capturing commands in nested structures like
// subshells, conditionals, and loops.
func extractCallExprs(node syntax.Node) []*syntax.CallExpr {
	var calls []*syntax.CallExpr
	syntax.Walk(node, func(n syntax.Node) bool {
		if call, ok := n.(*syntax.CallExpr); ok {
			calls = append(calls, call)
		}
		return true // Continue traversing into child nodes
	})
	return calls
}

// resolveStaticWord attempts to resolve a word into a static string.
// It returns the resolved string and a boolean indicating if the resolution is complete
// (i.e., the word contained no dynamic parts like variables or command substitutions).
func resolveStaticWord(word *syntax.Word) (val string, isStatic bool) {
	if word == nil {
		return "", true
	}

	var sb strings.Builder
	isStatic = true

	for _, part := range word.Parts {
		switch p := part.(type) {
		case *syntax.Lit:
			sb.WriteString(p.Value)
		case *syntax.SglQuoted:
			sb.WriteString(p.Value)
		case *syntax.DblQuoted:
			// Handle parts inside double quotes
			for _, subPart := range p.Parts {
				switch sp := subPart.(type) {
				case *syntax.Lit:
					sb.WriteString(sp.Value)
				case *syntax.ParamExp:
					// Variable expansion makes it dynamic
					isStatic = false
					// For partial resolution, we could try to handle simple cases
					// but for safety, we'll mark it as dynamic
				case *syntax.CmdSubst:
					// Command substitution makes it dynamic
					isStatic = false
				case *syntax.ArithmExp:
					// Arithmetic expansion makes it dynamic
					isStatic = false
				default:
					// Any other dynamic element
					isStatic = false
				}
			}
		case *syntax.ParamExp:
			// Variable expansion outside quotes
			isStatic = false
		case *syntax.CmdSubst:
			// Command substitution outside quotes
			isStatic = false
		case *syntax.ArithmExp:
			// Arithmetic expansion outside quotes
			isStatic = false
		case *syntax.ProcSubst:
			// Process substitution
			isStatic = false
		default:
			// Any other dynamic element
			isStatic = false
		}
	}

	return sb.String(), isStatic
}

// detectObfuscation performs basic obfuscation detection on a string
func detectObfuscation(s string) (bool, []string) {
	var issues []string
	detected := false

	// Check for base64 encoding patterns
	if isLikelyBase64(s) {
		issues = append(issues, "Possible base64 encoded content")
		detected = true
	}

	// Check for hex encoding patterns
	if isLikelyHexEncoded(s) {
		issues = append(issues, "Possible hex encoded content")
		detected = true
	}

	// Check for reverse string patterns
	if containsReversePattern(s) {
		issues = append(issues, "Possible reverse string obfuscation")
		detected = true
	}

	// Check for character substitution patterns
	if containsSubstitutionPattern(s) {
		issues = append(issues, "Possible character substitution obfuscation")
		detected = true
	}

	return detected, issues
}

// isLikelyBase64 checks if a string looks like base64 encoding
func isLikelyBase64(s string) bool {
	// Must be at least 8 characters for meaningful content
	if len(s) < 8 {
		return false
	}

	// Check for base64 character set and proper padding
	base64Chars := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/="

	// Count valid base64 characters
	validChars := 0
	for _, char := range s {
		for _, validChar := range base64Chars {
			if char == validChar {
				validChars++
				break
			}
		}
	}

	// If more than 90% of characters are valid base64, it's likely encoded
	return float64(validChars)/float64(len(s)) > 0.9
}

// isLikelyHexEncoded checks if a string looks like hex encoding
func isLikelyHexEncoded(s string) bool {
	// Must be at least 6 characters and even length
	if len(s) < 6 || len(s)%2 != 0 {
		return false
	}

	// Check if all characters are hex digits
	for _, char := range s {
		if (char < '0' || char > '9') && (char < 'a' || char > 'f') && (char < 'A' || char > 'F') {
			return false
		}
	}

	return true
}

// containsReversePattern checks for reverse string patterns
func containsReversePattern(s string) bool {
	// Look for common reverse patterns
	reversePatterns := []string{
		"rev", "tac", "hsup", "tig", // "push" reversed, "git" reversed
	}

	lowerS := strings.ToLower(s)
	for _, pattern := range reversePatterns {
		if strings.Contains(lowerS, pattern) {
			return true
		}
	}

	return false
}

// containsSubstitutionPattern checks for character substitution obfuscation
func containsSubstitutionPattern(s string) bool {
	// Look for patterns with excessive variable substitutions
	// ${} patterns that may indicate obfuscation
	if strings.Count(s, "${") > 2 && strings.Contains(s, "}") {
		return true
	}

	// Multiple quoted segments that could be obfuscation
	if strings.Count(s, "\"") > 4 || strings.Count(s, "'") > 4 {
		// Check if it contains git-related parts
		if strings.Contains(strings.ToLower(s), "git") || strings.Contains(strings.ToLower(s), "push") {
			return true
		}
	}

	return false
}
