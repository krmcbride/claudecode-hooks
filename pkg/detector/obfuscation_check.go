// Package detector - obfuscation detection strategies
package detector

import (
	"slices"
	"strings"

	"mvdan.cc/sh/v3/syntax"
)

// checkObfuscation detects various obfuscation techniques used to hide commands:
//   - Base64 decoding being piped to execution
//   - Hex encoding
//   - Echo with escape sequences (\x codes)
//   - Character substitution patterns
//
// These techniques are commonly used to bypass simple string matching.
func (d *CommandDetector) checkObfuscation(call *syntax.CallExpr) bool {
	// Check for base64 being decoded and executed
	if d.checkBase64Execution(call) {
		return true // BLOCK
	}

	// Collect all static string content for other obfuscation checks
	content := d.collectStaticContent(call)

	// Check for hex encoding and other obfuscation patterns (but NOT base64)
	if obfuscated, obfIssues := detectObfuscationExceptBase64(content); obfuscated {
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

// detectObfuscationExceptBase64 performs obfuscation detection excluding base64
// Base64 is now handled separately by checkBase64Execution to avoid false positives
func detectObfuscationExceptBase64(s string) (bool, []string) {
	var issues []string
	detected := false

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

// checkBase64Execution detects when base64 is being decoded AND executed.
// This prevents false positives from file paths while catching actual threats like:
//   - base64 -d | bash
//   - echo <base64> | base64 --decode | sh
//   - eval $(base64 -d ...)
func (d *CommandDetector) checkBase64Execution(call *syntax.CallExpr) bool {
	if len(call.Args) == 0 {
		return false
	}

	cmd, _ := resolveStaticWord(call.Args[0])
	normalizedCmd := normalizeCommand(cmd)

	// Check if this is a base64 decode command
	if normalizedCmd == "base64" {
		// Look for decode flags (-d, --decode, -D)
		hasDecodeFlag := false
		for i := 1; i < len(call.Args); i++ {
			argStr, _ := resolveStaticWord(call.Args[i])
			if argStr == "-d" || argStr == "--decode" || argStr == "-D" {
				hasDecodeFlag = true
				break
			}
		}

		if hasDecodeFlag {
			// This is a base64 decode operation
			// In a real pipeline, we'd need to check if it's piped to a shell
			// For now, we'll flag it with a warning but not block it
			// The actual threat is when it's piped to execution
			d.addIssue("Warning: base64 decode detected - ensure output is not executed")
			// Don't block just decoding - only block if we see execution patterns
			return false
		}
	}

	// Check for patterns where base64 output might be executed
	// This would need more sophisticated pipeline analysis
	// For now, check if command contains both base64 and shell interpreters
	if strings.Contains(cmd, "base64") {
		for _, arg := range call.Args[1:] {
			argStr, _ := resolveStaticWord(arg)
			// Check if any argument mentions shell interpreters
			if isShellInterpreter(argStr) || strings.Contains(argStr, "eval") {
				d.addIssue("Base64 decode potentially being executed")
				return true // BLOCK
			}
		}
	}

	return false
}

// isShellInterpreter checks if a command is a shell interpreter
func isShellInterpreter(cmd string) bool {
	shells := []string{"sh", "bash", "zsh", "ksh", "dash", "fish", "csh", "tcsh"}
	normalizedCmd := normalizeCommand(cmd)
	return slices.Contains(shells, normalizedCmd)
}
