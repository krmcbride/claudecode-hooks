// Package detector - string literal analysis for embedded commands
package detector

import (
	"strings"

	"mvdan.cc/sh/v3/syntax"
)

// analyzeStringLiterals performs deep inspection of string literals that might
// contain embedded commands. This is crucial for detecting commands hidden in:
//   - Shell interpreter arguments (bash -c "git push")
//   - Eval statements (eval "aws delete")
//   - Echo piped to shell (echo "git push" | sh)
//
// Only analyzes strings for commands known to execute their string arguments.
func (d *CommandDetector) analyzeStringLiterals(call *syntax.CallExpr) bool {
	// First check what command we're dealing with
	cmd, _ := resolveStaticWord(call.Args[0])
	normalizedCmd := normalizeCommand(cmd)

	// For certain commands, we should analyze their string arguments
	// These are commands that typically execute strings as shell code
	shouldAnalyzeStrings := false
	switch normalizedCmd {
	case "sh", "bash", "zsh", "ksh", "dash", "fish":
		// Shell interpreters with -c flag
		shouldAnalyzeStrings = true
	case "eval", "source", ".":
		// Commands that evaluate strings as shell code
		shouldAnalyzeStrings = true
	case "echo", "printf":
		// These commands output text - only analyze if piped to shell
		// We'll analyze their strings but with more conservative filtering
		shouldAnalyzeStrings = true
	}

	if !shouldAnalyzeStrings {
		return false
	}

	// Skip the first argument (the command itself) and analyze the rest
	for i := 1; i < len(call.Args); i++ {
		arg := call.Args[i]

		// Extract all string literals from this argument
		if strings := extractStringLiterals(arg); len(strings) > 0 {
			for _, str := range strings {
				// For echo/printf, be more conservative - only check if it really looks like a command
				if normalizedCmd == "echo" || normalizedCmd == "printf" {
					if !d.definitelyLooksLikeCommand(str) {
						continue
					}
				} else {
					// For shell interpreters and eval, check more broadly
					if !d.looksLikeCommand(str) {
						continue
					}
				}

				// Try to parse and analyze each string as a shell expression
				if d.analyzeShellExprRecursive(str) {
					d.addIssue("Blocked command found in string: " + str)
					return true // BLOCK
				}
			}
		}
	}
	return false
}

// looksLikeCommand heuristically determines if a string contains executable commands.
// Used to filter out simple arguments before expensive parsing.
// Returns false for:
//   - Flags (strings starting with -)
//   - Single words without shell metacharacters
//
// Returns true if the string appears to contain commands or command chains.
func (d *CommandDetector) looksLikeCommand(str string) bool {
	// Skip if it's just a flag (starts with -)
	if strings.HasPrefix(str, "-") {
		return false
	}

	// Skip if it's a single word without spaces (likely just an argument)
	if !strings.Contains(str, " ") && !strings.Contains(str, ";") &&
		!strings.Contains(str, "|") && !strings.Contains(str, "&") {
		return false
	}

	// Additional check: if the string starts with a known blocked command, it's likely a command
	// This helps with cases like "git push origin" in various contexts
	for _, rule := range d.commandRules {
		if strings.HasPrefix(str, rule.BlockedCommand+" ") {
			return true
		}
	}

	return true
}

// definitelyLooksLikeCommand applies strict heuristics for echo/printf content.
// Since echo/printf are often used legitimately, this function reduces false
// positives by only flagging strings that:
//   - Start with a known blocked command
//   - Contain shell metacharacters indicating command chaining (;, &&, ||, |)
//
// This avoids blocking legitimate output like: echo "Use git for version control"
func (d *CommandDetector) definitelyLooksLikeCommand(str string) bool {
	// Must start with a known blocked command to be considered
	startsWithCommand := false
	for _, rule := range d.commandRules {
		if strings.HasPrefix(str, rule.BlockedCommand+" ") {
			startsWithCommand = true
			break
		}
	}

	if !startsWithCommand {
		// Check for shell metacharacters that indicate command execution
		if strings.Contains(str, ";") || strings.Contains(str, "&&") ||
			strings.Contains(str, "||") || strings.Contains(str, "|") {
			// Could be a command chain
			return true
		}
		return false
	}

	return true
}

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
