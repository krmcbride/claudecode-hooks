// Package detector - obfuscation detection strategies
package detector

import (
	"strings"

	"mvdan.cc/sh/v3/syntax"
)

// checkObfuscation detects various obfuscation techniques used to hide commands:
//   - Base64 encoding
//   - Hex encoding
//   - Echo with escape sequences (\x codes)
//   - Character substitution patterns
//
// These techniques are commonly used to bypass simple string matching.
func (d *CommandDetector) checkObfuscation(call *syntax.CallExpr) bool {
	// Collect all static string content
	content := d.collectStaticContent(call)

	// Check for base64/hex encoding
	if obfuscated, obfIssues := detectObfuscation(content); obfuscated {
		d.issues = append(d.issues, obfIssues...)
		return true // BLOCK
	}

	// Check echo with escape sequences
	return d.checkEchoEscapes(call)
}

// collectStaticContent aggregates all statically resolvable string content
// from a command call. Used for obfuscation detection where we need to
// analyze the command as a whole rather than individual arguments.
func (d *CommandDetector) collectStaticContent(call *syntax.CallExpr) string {
	var allContent strings.Builder
	for _, arg := range call.Args {
		val, isStatic := resolveStaticWord(arg)
		if isStatic && val != "" {
			allContent.WriteString(val)
			allContent.WriteString(" ")
		}
	}
	return allContent.String()
}

// checkEchoEscapes detects echo commands using escape sequences to construct
// hidden commands. Examples:
//   - echo -e "\x67\x69\x74" (hex for "git")
//   - echo $'\147\151\164' (octal for "git")
//
// These can be piped to shell interpreters to execute obfuscated commands.
func (d *CommandDetector) checkEchoEscapes(call *syntax.CallExpr) bool {
	cmd, _ := resolveStaticWord(call.Args[0])
	if !isEchoCommand(cmd) {
		return false
	}

	for _, arg := range call.Args[1:] {
		argStr, _ := resolveStaticWord(arg)

		// Check for hex escapes
		if strings.Contains(argStr, "\\x") || strings.Contains(argStr, "\\0") {
			d.addIssue("echo with escape sequences detected (possible obfuscation)")
			return true // BLOCK
		}

		// Check for -e flag
		if argStr == "-e" {
			d.addIssue("echo -e detected which enables escape sequences")
			return true // BLOCK
		}
	}
	return false
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
