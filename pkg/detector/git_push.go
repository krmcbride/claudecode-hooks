// Package detector provides git push detection logic
package detector

import (
	"slices"
	"strings"

	"mvdan.cc/sh/v3/syntax"

	"github.com/krmcbride/claudecode-hooks/pkg/shellparse"
)

// GitPushDetector provides comprehensive git push detection with recursion limits
type GitPushDetector struct {
	issues       []string
	maxDepth     int
	currentDepth int
}

// NewGitPushDetector creates a new detector with default settings
func NewGitPushDetector() *GitPushDetector {
	return &GitPushDetector{
		issues:       make([]string, 0),
		maxDepth:     10, // Prevent DoS through deep recursion
		currentDepth: 0,
	}
}

// GetIssues returns all detected issues
func (d *GitPushDetector) GetIssues() []string {
	return d.issues
}

// AnalyzeCommand is the main entry point for command analysis
func (d *GitPushDetector) AnalyzeCommand(command string) bool {
	d.currentDepth = 0
	return d.analyzeCommandRecursive(command)
}

// analyzeCommandRecursive performs comprehensive analysis with recursion tracking
func (d *GitPushDetector) analyzeCommandRecursive(command string) bool {
	// Check recursion depth
	d.currentDepth++
	if d.currentDepth > d.maxDepth {
		d.issues = append(d.issues, "Max recursion depth exceeded - potential DoS attempt")
		return true
	}
	defer func() { d.currentDepth-- }()

	// Parse the main command
	calls, err := shellparse.ParseCommand(command)
	if err != nil {
		// If we can't parse it, be conservative and block
		d.issues = append(d.issues, "Failed to parse command: "+err.Error())
		return true
	}

	// Analyze each call expression
	for _, call := range calls {
		if d.analyzeCallExpr(call) {
			return true
		}
	}

	return false // Simplified logic - only return true if we found something
}

// analyzeCallExpr analyzes a single call expression for git push patterns
func (d *GitPushDetector) analyzeCallExpr(call *syntax.CallExpr) bool {
	if len(call.Args) == 0 {
		return false
	}

	// Get command with enhanced resolution
	cmd, cmdIsStatic := shellparse.ResolveStaticWord(call.Args[0])

	// If command is dynamic, it's suspicious
	if !cmdIsStatic {
		d.issues = append(d.issues, "Dynamic command detected (potential obfuscation)")
		return true
	}

	// Check for direct git push
	if d.checkDirectGitPush(call, cmd) {
		return true
	}

	// Check for shell interpreter patterns (sh -c, bash -c)
	if d.checkShellInterpreter(call, cmd) {
		return true
	}

	// Check for eval patterns
	if d.checkEvalCommand(call, cmd) {
		return true
	}

	// Check for other execution patterns
	if d.checkExecutionPatterns(call, cmd) {
		return true
	}

	// Check for obfuscation patterns
	if d.checkObfuscationPatterns(call) {
		return true
	}

	return false
}

// checkDirectGitPush checks for direct git push commands
func (d *GitPushDetector) checkDirectGitPush(call *syntax.CallExpr, cmd string) bool {
	if !shellparse.IsGitCommand(cmd) {
		return false
	}

	// Need at least git + subcommand
	if len(call.Args) < 2 {
		return false
	}

	// Get the subcommand
	subCmd, subCmdIsStatic := shellparse.ResolveStaticWord(call.Args[1])

	// If subcommand is dynamic, it could be "$SUBCMD" where SUBCMD=push
	if !subCmdIsStatic {
		d.issues = append(d.issues, "Dynamic git subcommand detected")
		return true
	}

	// Direct git push
	if subCmd == "push" {
		d.issues = append(d.issues, "Direct 'git push' command detected")
		return true
	}

	return false
}

// checkShellInterpreter checks for shell interpreter patterns
func (d *GitPushDetector) checkShellInterpreter(call *syntax.CallExpr, _ string) bool {
	// Extract shell commands using enhanced parsing
	shellCommands, hasDynamicContent := shellparse.ExtractShellCommands(call)

	// SECURITY: Block if dynamic content detected (prevents command substitution bypass)
	if hasDynamicContent {
		d.issues = append(d.issues, "Dynamic shell command content detected - potential command substitution")
		return true
	}

	for _, shellCmd := range shellCommands {
		// Recursively analyze the shell command
		if d.analyzeCommandRecursive(shellCmd) {
			d.issues = append(d.issues, "Git push detected in shell command: "+shellCmd)
			return true
		}

		// Also check for simple string patterns as fallback
		if d.containsGitPushPattern(shellCmd) {
			d.issues = append(d.issues, "Git push pattern detected in shell command")
			return true
		}
	}

	return false
}

// checkEvalCommand checks for eval command patterns
func (d *GitPushDetector) checkEvalCommand(call *syntax.CallExpr, cmd string) bool {
	if cmd != "eval" {
		return false
	}

	evalContent := shellparse.AnalyzeEvalCommand(call)

	for _, content := range evalContent {
		// Recursively analyze eval content
		if d.analyzeCommandRecursive(content) {
			d.issues = append(d.issues, "Git push detected in eval command")
			return true
		}

		// Check for string patterns
		if d.containsGitPushPattern(content) {
			d.issues = append(d.issues, "Git push pattern detected in eval")
			return true
		}
	}

	return false
}

// checkExecutionPatterns checks for other command execution patterns
func (d *GitPushDetector) checkExecutionPatterns(call *syntax.CallExpr, cmd string) bool {
	// List of commands that can execute other commands
	execCommands := []string{
		"xargs", "find", "parallel", "env", "nohup",
		"timeout", "time", "watch", "script",
	}

	if !slices.Contains(execCommands, cmd) {
		return false
	}

	// Look for git push in arguments
	for i := 1; i < len(call.Args); i++ {
		arg, argIsStatic := shellparse.ResolveStaticWord(call.Args[i])
		if !argIsStatic {
			// Dynamic content in execution command
			d.issues = append(d.issues, "Dynamic content in "+cmd+" command")
			return true
		}

		if d.containsGitPushPattern(arg) {
			d.issues = append(d.issues, "Git push pattern detected in "+cmd+" arguments")
			return true
		}
	}

	return false
}

// checkObfuscationPatterns checks for common obfuscation patterns
func (d *GitPushDetector) checkObfuscationPatterns(call *syntax.CallExpr) bool {
	// Collect all static string content for analysis
	var allContent strings.Builder

	for _, arg := range call.Args {
		val, isStatic := shellparse.ResolveStaticWord(arg)
		if isStatic && val != "" {
			allContent.WriteString(val)
			allContent.WriteString(" ")
		}
	}

	content := allContent.String()

	// Use common obfuscation detection
	if obfuscated, issues := shellparse.DetectObfuscation(content); obfuscated {
		d.issues = append(d.issues, issues...)
		// If obfuscated AND contains git-related terms, block it
		lowerContent := strings.ToLower(content)
		if strings.Contains(lowerContent, "git") || strings.Contains(lowerContent, "push") {
			return true
		}
	}

	// Check for specific obfuscated patterns
	obfuscatedPatterns := []string{
		"gi\"t pu\"sh", "gi't pu'sh", "gi\\t pu\\sh",
		"g'i't p'u's'h", "g\"i\"t p\"u\"s\"h",
		"gi${x}t pu${x}sh", "gi*t pu*sh",
	}

	for _, pattern := range obfuscatedPatterns {
		if strings.Contains(content, pattern) {
			d.issues = append(d.issues, "Obfuscated git push pattern detected")
			return true
		}
	}

	return false
}

// containsGitPushPattern performs simple string pattern matching
func (d *GitPushDetector) containsGitPushPattern(s string) bool {
	lowerStr := strings.ToLower(s)

	// Direct patterns
	patterns := []string{
		"git push",
		"git  push", // Multiple spaces
		"git\tpush", // Tab
		"git\npush", // Newline
	}

	for _, pattern := range patterns {
		if strings.Contains(lowerStr, pattern) {
			return true
		}
	}

	// Check if both words exist in reasonable proximity
	if strings.Contains(lowerStr, "git") && strings.Contains(lowerStr, "push") {
		// Simple proximity check - if both words are within 20 characters
		gitIndex := strings.Index(lowerStr, "git")
		pushIndex := strings.Index(lowerStr, "push")
		if gitIndex >= 0 && pushIndex >= 0 {
			distance := pushIndex - gitIndex
			if distance > 0 && distance < 20 {
				return true
			}
		}
	}

	return false
}
