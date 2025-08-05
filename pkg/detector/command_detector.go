// Package detector provides generic command detection logic with configurable rules
package detector

import (
	"slices"
	"strings"

	"mvdan.cc/sh/v3/syntax"

	"github.com/krmcbride/claudecode-hooks/pkg/shellparse"
)

// SecurityLevel defines the security analysis depth
type SecurityLevel string

const (
	SecurityBasic    SecurityLevel = "basic"    // Pattern matching only
	SecurityAdvanced SecurityLevel = "advanced" // + obfuscation detection
	SecurityParanoid SecurityLevel = "paranoid" // + all dynamic content blocking
)

// CommandRule defines what commands and patterns to detect
type CommandRule struct {
	Command         string   // Primary command (git, aws, kubectl)
	BlockedPatterns []string // Subcommand patterns to block
	AllowExceptions []string // Patterns to allow despite blocks
	Description     string   // Human readable description
}

// CommandDetector provides comprehensive command detection with configurable rules
type CommandDetector struct {
	commandRules  []CommandRule
	securityLevel SecurityLevel
	issues        []string
	maxDepth      int
	currentDepth  int
}

// NewCommandDetector creates a new detector with specified rules and security level
func NewCommandDetector(rules []CommandRule, securityLevel SecurityLevel, maxDepth int) *CommandDetector {
	if maxDepth <= 0 {
		maxDepth = 10 // Default safe recursion limit
	}

	return &CommandDetector{
		commandRules:  rules,
		securityLevel: securityLevel,
		issues:        make([]string, 0),
		maxDepth:      maxDepth,
		currentDepth:  0,
	}
}

// GetIssues returns all detected issues
func (d *CommandDetector) GetIssues() []string {
	return d.issues
}

// AnalyzeCommand is the main entry point for command analysis
func (d *CommandDetector) AnalyzeCommand(command string) bool {
	d.currentDepth = 0
	d.issues = d.issues[:0] // Clear previous issues
	return d.analyzeCommandRecursive(command)
}

// analyzeCommandRecursive performs comprehensive analysis with recursion tracking
func (d *CommandDetector) analyzeCommandRecursive(command string) bool {
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
	return slices.ContainsFunc(calls, d.analyzeCallExpr)
}

// analyzeCallExpr analyzes a single call expression for configured command patterns
func (d *CommandDetector) analyzeCallExpr(call *syntax.CallExpr) bool {
	if len(call.Args) == 0 {
		return false
	}

	// Get command with enhanced resolution
	cmd, cmdIsStatic := shellparse.ResolveStaticWord(call.Args[0])

	// If command is dynamic, check security level
	if !cmdIsStatic {
		if d.securityLevel == SecurityAdvanced || d.securityLevel == SecurityParanoid {
			d.issues = append(d.issues, "Dynamic command detected (potential obfuscation)")
			return true
		}
	}

	// Check for direct command matches
	if d.checkDirectCommand(call, cmd) {
		return true
	}

	// Advanced security checks (skip for basic level)
	if d.securityLevel != SecurityBasic {
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
	}

	return false
}

// checkDirectCommand checks for direct command matches with configured rules
func (d *CommandDetector) checkDirectCommand(call *syntax.CallExpr, cmd string) bool {
	// Check each command rule
	for _, rule := range d.commandRules {
		if !d.isMatchingCommand(cmd, rule.Command) {
			continue
		}

		// Need at least command + subcommand for pattern matching
		if len(call.Args) < 2 {
			continue
		}

		// Get the subcommand or arguments
		args := make([]string, 0, len(call.Args)-1)
		for _, arg := range call.Args[1:] {
			argVal, argIsStatic := shellparse.ResolveStaticWord(arg)

			// For paranoid level, block any dynamic subcommands
			if !argIsStatic && d.securityLevel == SecurityParanoid {
				d.issues = append(d.issues, "Dynamic "+rule.Command+" subcommand detected")
				return true
			}

			if argIsStatic {
				args = append(args, argVal)
			}
		}

		// Check if command arguments match blocked patterns
		fullArgs := strings.Join(args, " ")

		// First check allow exceptions
		if d.hasAllowException(fullArgs, rule.AllowExceptions) {
			continue
		}

		// Then check blocked patterns
		if d.hasBlockedPattern(args, fullArgs, rule) {
			return true
		}
	}

	return false
}

// isMatchingCommand checks if a command matches the rule's command pattern
func (d *CommandDetector) isMatchingCommand(cmd, ruleCmd string) bool {
	// Direct match
	if cmd == ruleCmd {
		return true
	}

	// Use shellparse to normalize the command path (handles Windows paths properly)
	normalizedCmd := shellparse.NormalizeCommandPath(cmd)
	if normalizedCmd == ruleCmd {
		return true
	}

	// Handle full paths like /usr/bin/git, /usr/local/bin/git, ./git
	if strings.HasSuffix(cmd, "/"+ruleCmd) {
		return true
	}

	// Handle Windows paths (cmd.exe)
	if strings.HasSuffix(cmd, ruleCmd+".exe") || strings.HasSuffix(cmd, "/"+ruleCmd+".exe") || strings.HasSuffix(cmd, "\\"+ruleCmd+".exe") {
		return true
	}

	return false
}

// hasAllowException checks if the command matches any allow exceptions
func (d *CommandDetector) hasAllowException(fullArgs string, allowExceptions []string) bool {
	for _, exception := range allowExceptions {
		if exception != "" && strings.Contains(strings.ToLower(fullArgs), strings.ToLower(exception)) {
			return true
		}
	}
	return false
}

// hasBlockedPattern checks if args match any blocked patterns
func (d *CommandDetector) hasBlockedPattern(args []string, fullArgs string, rule CommandRule) bool {
	for _, pattern := range rule.BlockedPatterns {
		if pattern == "" {
			continue
		}

		// Check if pattern matches
		if d.matchesPattern(args, fullArgs, pattern, rule.Command) {
			d.issues = append(d.issues, "Blocked "+rule.Command+" pattern detected: "+pattern)
			return true
		}
	}
	return false
}

// matchesPattern checks if the command arguments match a blocked pattern
func (d *CommandDetector) matchesPattern(args []string, fullArgs, pattern, command string) bool {
	lowerFullArgs := strings.ToLower(fullArgs)
	lowerPattern := strings.ToLower(pattern)

	// Direct substring match
	if strings.Contains(lowerFullArgs, lowerPattern) {
		return true
	}

	// Check individual args for exact matches
	for _, arg := range args {
		if strings.ToLower(arg) == lowerPattern {
			return true
		}
	}

	// For compound patterns like "git push", check proximity
	patternWords := strings.Fields(lowerPattern)
	if len(patternWords) > 1 {
		return d.containsPatternWords(lowerFullArgs, patternWords)
	}

	return false
}

// containsPatternWords checks if all pattern words exist in reasonable proximity
func (d *CommandDetector) containsPatternWords(text string, words []string) bool {
	// Check if all words exist
	for _, word := range words {
		if !strings.Contains(text, word) {
			return false
		}
	}

	// Simple proximity check - if words are within reasonable distance
	if len(words) == 2 {
		firstIndex := strings.Index(text, words[0])
		secondIndex := strings.Index(text, words[1])
		if firstIndex >= 0 && secondIndex >= 0 {
			distance := secondIndex - firstIndex
			if distance > 0 && distance < 20 {
				return true
			}
		}
	}

	return false
}

// checkShellInterpreter checks for shell interpreter patterns (advanced security only)
func (d *CommandDetector) checkShellInterpreter(call *syntax.CallExpr, _ string) bool {
	// Extract shell commands using enhanced parsing
	shellCommands, hasDynamicContent := shellparse.ExtractShellCommands(call)

	// SECURITY: Block if dynamic content detected (prevents command substitution bypass)
	if hasDynamicContent && d.securityLevel == SecurityParanoid {
		d.issues = append(d.issues, "Dynamic shell command content detected - potential command substitution")
		return true
	}

	for _, shellCmd := range shellCommands {
		// Recursively analyze the shell command
		if d.analyzeCommandRecursive(shellCmd) {
			d.issues = append(d.issues, "Blocked command detected in shell command: "+shellCmd)
			return true
		}

		// Also check for simple string patterns as fallback
		if d.containsAnyCommandPattern(shellCmd) {
			d.issues = append(d.issues, "Blocked pattern detected in shell command")
			return true
		}
	}

	return false
}

// checkEvalCommand checks for eval command patterns (advanced security only)
func (d *CommandDetector) checkEvalCommand(call *syntax.CallExpr, cmd string) bool {
	if cmd != "eval" {
		return false
	}

	evalContent := shellparse.AnalyzeEvalCommand(call)

	for _, content := range evalContent {
		// Recursively analyze eval content
		if d.analyzeCommandRecursive(content) {
			d.issues = append(d.issues, "Blocked command detected in eval command")
			return true
		}

		// Check for string patterns
		if d.containsAnyCommandPattern(content) {
			d.issues = append(d.issues, "Blocked pattern detected in eval")
			return true
		}
	}

	return false
}

// checkExecutionPatterns checks for other command execution patterns (advanced security only)
func (d *CommandDetector) checkExecutionPatterns(call *syntax.CallExpr, cmd string) bool {
	// List of commands that can execute other commands
	execCommands := []string{
		"xargs", "find", "parallel", "env", "nohup",
		"timeout", "time", "watch", "script",
	}

	if !slices.Contains(execCommands, cmd) {
		return false
	}

	// Look for blocked patterns in arguments
	for i := 1; i < len(call.Args); i++ {
		arg, argIsStatic := shellparse.ResolveStaticWord(call.Args[i])
		if !argIsStatic && d.securityLevel == SecurityParanoid {
			// Dynamic content in execution command
			d.issues = append(d.issues, "Dynamic content in "+cmd+" command")
			return true
		}

		if argIsStatic && d.containsAnyCommandPattern(arg) {
			d.issues = append(d.issues, "Blocked pattern detected in "+cmd+" arguments")
			return true
		}
	}

	return false
}

// checkObfuscationPatterns checks for common obfuscation patterns (advanced security only)
func (d *CommandDetector) checkObfuscationPatterns(call *syntax.CallExpr) bool {
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
		// If obfuscated AND contains blocked command terms, block it
		if d.containsAnyCommandPattern(content) {
			return true
		}
	}

	// Check for specific obfuscated patterns for each command rule
	for _, rule := range d.commandRules {
		if d.containsObfuscatedPatterns(content, rule) {
			d.issues = append(d.issues, "Obfuscated "+rule.Command+" pattern detected")
			return true
		}
	}

	return false
}

// containsObfuscatedPatterns checks for obfuscated versions of command patterns
func (d *CommandDetector) containsObfuscatedPatterns(content string, rule CommandRule) bool {
	for _, pattern := range rule.BlockedPatterns {
		if pattern == "" {
			continue
		}

		// Generate common obfuscation variants
		obfuscatedPatterns := d.generateObfuscatedPatterns(rule.Command, pattern)

		for _, obfPattern := range obfuscatedPatterns {
			if strings.Contains(content, obfPattern) {
				return true
			}
		}
	}
	return false
}

// generateObfuscatedPatterns creates common obfuscation variants
func (d *CommandDetector) generateObfuscatedPatterns(command, pattern string) []string {
	patterns := []string{}

	// For simple patterns, create quote-based obfuscation
	if !strings.Contains(pattern, " ") {
		// Single word pattern obfuscation
		word := pattern
		patterns = append(patterns,
			"\""+word+"\"", "'"+word+"'",
			string(word[0])+"\""+word[1:]+"\"",
			string(word[0])+"'"+word[1:]+"'",
		)
	} else {
		// Multi-word pattern obfuscation
		fullPattern := command + " " + pattern
		patterns = append(patterns,
			strings.ReplaceAll(fullPattern, " ", "\" \""),
			strings.ReplaceAll(fullPattern, " ", "' '"),
			strings.ReplaceAll(fullPattern, " ", "\\ "),
		)
	}

	return patterns
}

// containsAnyCommandPattern checks if text contains any configured command patterns
func (d *CommandDetector) containsAnyCommandPattern(text string) bool {
	lowerText := strings.ToLower(text)

	for _, rule := range d.commandRules {
		lowerCommand := strings.ToLower(rule.Command)

		for _, pattern := range rule.BlockedPatterns {
			if pattern == "" {
				continue
			}

			lowerPattern := strings.ToLower(pattern)

			// Direct pattern match
			fullPattern := lowerCommand + " " + lowerPattern
			if strings.Contains(lowerText, fullPattern) {
				return true
			}

			// Check proximity if both command and pattern exist
			if strings.Contains(lowerText, lowerCommand) && strings.Contains(lowerText, lowerPattern) {
				cmdIndex := strings.Index(lowerText, lowerCommand)
				patternIndex := strings.Index(lowerText, lowerPattern)
				if cmdIndex >= 0 && patternIndex >= 0 {
					distance := patternIndex - cmdIndex
					if distance > 0 && distance < 20 {
						return true
					}
				}
			}
		}
	}

	return false
}
