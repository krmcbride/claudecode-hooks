// Package shellparse provides shell command parsing utilities.
package shellparse

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"mvdan.cc/sh/v3/syntax"
)

// ParseCommand parses a shell command and extracts command calls
func ParseCommand(command string) ([]*syntax.CallExpr, error) {
	parser := syntax.NewParser()
	node, err := parser.Parse(strings.NewReader(command), "")
	if err != nil {
		return nil, fmt.Errorf("failed to parse command: %w", err)
	}

	var calls []*syntax.CallExpr
	syntax.Walk(node, func(node syntax.Node) bool {
		if call, ok := node.(*syntax.CallExpr); ok {
			calls = append(calls, call)
		}
		return true
	})

	return calls, nil
}

// WordToString converts a syntax.Word to string
func WordToString(word *syntax.Word) string {
	if word == nil {
		return ""
	}
	var result strings.Builder
	for _, part := range word.Parts {
		switch p := part.(type) {
		case *syntax.Lit:
			result.WriteString(p.Value)
		case *syntax.SglQuoted:
			result.WriteString(p.Value)
		case *syntax.DblQuoted:
			for _, dqPart := range p.Parts {
				if lit, ok := dqPart.(*syntax.Lit); ok {
					result.WriteString(lit.Value)
				}
			}
		}
	}
	return result.String()
}

// GetCommandName extracts the command name from a CallExpr
func GetCommandName(call *syntax.CallExpr) string {
	if len(call.Args) == 0 {
		return ""
	}
	return WordToString(call.Args[0])
}

// GetCommandArgs extracts command arguments from a CallExpr
func GetCommandArgs(call *syntax.CallExpr) []string {
	if len(call.Args) <= 1 {
		return nil
	}

	args := make([]string, 0, len(call.Args)-1)
	for _, arg := range call.Args[1:] {
		args = append(args, WordToString(arg))
	}
	return args
}

// ResolveStaticWord attempts to resolve a word into a static string.
// It returns the resolved string and a boolean indicating if the resolution is complete
// (i.e., the word contained no dynamic parts like variables or command substitutions).
func ResolveStaticWord(word *syntax.Word) (val string, isStatic bool) {
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
					// but for security, we'll mark it as dynamic
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

// IsGitCommand checks if a command name refers to git, handling various forms
func IsGitCommand(cmd string) bool {
	// Handle direct git command
	if cmd == "git" {
		return true
	}

	// Handle full paths like /usr/bin/git, /usr/local/bin/git, ./git
	if strings.HasSuffix(cmd, "/git") {
		return true
	}

	// Handle Windows paths (git.exe)
	if strings.HasSuffix(cmd, "git.exe") || strings.HasSuffix(cmd, "/git.exe") {
		return true
	}

	return false
}

// NormalizeCommandPath normalizes a command path for comparison
func NormalizeCommandPath(cmd string) string {
	// Clean the path
	cleaned := filepath.Clean(cmd)

	// Extract just the command name for comparison
	base := filepath.Base(cleaned)

	// Remove .exe extension if present (Windows)
	base = strings.TrimSuffix(base, ".exe")

	return base
}

// ExtractShellCommands extracts shell commands from common shell interpreter patterns
// Returns commands found and a boolean indicating if dynamic content was detected
func ExtractShellCommands(call *syntax.CallExpr) ([]string, bool) {
	if len(call.Args) < 2 {
		return nil, false
	}

	cmd, cmdIsStatic := ResolveStaticWord(call.Args[0])
	if !cmdIsStatic {
		return nil, true // Dynamic command itself
	}

	// Normalize command name
	cmdName := NormalizeCommandPath(cmd)

	// Check if this is a shell interpreter
	if !isShellInterpreter(cmdName) {
		return nil, false
	}

	var commands []string
	hasDynamicContent := false

	// Look for -c flag
	for i := 1; i < len(call.Args); i++ {
		arg, argIsStatic := ResolveStaticWord(call.Args[i])
		if !argIsStatic {
			// SECURITY: Don't skip dynamic arguments - they could contain malicious content
			hasDynamicContent = true
			continue
		}

		// If we find -c, the next argument should be the command
		if arg == "-c" && i+1 < len(call.Args) {
			cmdStr, cmdStrIsStatic := ResolveStaticWord(call.Args[i+1])
			if !cmdStrIsStatic {
				// SECURITY: Dynamic shell command content is extremely dangerous
				hasDynamicContent = true
			} else if cmdStr != "" {
				commands = append(commands, cmdStr)
			}
			break
		}
	}

	return commands, hasDynamicContent
}

// isShellInterpreter checks if a command is a shell interpreter
func isShellInterpreter(cmd string) bool {
	shellCommands := []string{
		"sh", "bash", "zsh", "dash", "ksh", "csh", "tcsh", "fish",
	}

	return slices.Contains(shellCommands, cmd)
}

// AnalyzeEvalCommand analyzes eval commands for suspicious content
func AnalyzeEvalCommand(call *syntax.CallExpr) []string {
	if len(call.Args) < 2 {
		return nil
	}

	cmd, cmdIsStatic := ResolveStaticWord(call.Args[0])
	if !cmdIsStatic || cmd != "eval" {
		return nil
	}

	var evalContent []string

	// Collect all arguments to eval (they get concatenated)
	for i := 1; i < len(call.Args); i++ {
		arg, argIsStatic := ResolveStaticWord(call.Args[i])
		if argIsStatic && arg != "" {
			evalContent = append(evalContent, arg)
		}
	}

	return evalContent
}

// DetectObfuscation performs basic obfuscation detection on a string
func DetectObfuscation(s string) (bool, []string) {
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
	// ${} patterns that look suspicious
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
